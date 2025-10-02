package grpc

import (
	"context"
	"errors"
	"time"

	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/cache"
	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/repository"
	filev1 "github.com/yourusername/distributed-file-sharing/services/file-service/pkg/pb/file/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetFile with Redis caching
func (h *FileHandler) GetFileWithCache(ctx context.Context, req *filev1.GetFileRequest) (*filev1.GetFileResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, h.config.QueryTimeout)
	defer cancel()

	requestID := h.getRequestID(ctx)
	logger := h.logger.WithFields(map[string]interface{}{
		"request_id": requestID,
		"method":     "GetFile",
		"file_id":    req.FileId,
	})

	// Get authenticated user ID
	userID, err := h.getUserIDFromContext(ctx)
	if err != nil {
		logger.Warn("Authentication failed")
		return nil, err
	}

	logger = logger.WithField("user_id", userID)

	if req.FileId == "" {
		return nil, status.Error(codes.InvalidArgument, "file_id is required")
	}

	// Try cache first
	if h.cache != nil && h.cache.IsEnabled() {
		cachedFile, err := h.cache.GetFileMetadata(ctx, req.FileId)
		if err == nil {
			logger.Debug("Cache hit for file metadata")

			// Still need to check permissions
			if cachedFile.OwnerID != userID {
				hasAccess, err := h.fileRepo.CheckShareAccess(ctx, req.FileId, userID)
				if err != nil {
					logger.WithError(err).Error("Failed to check share access")
					return nil, status.Error(codes.Internal, "unable to process request")
				}

				if !hasAccess {
					logger.Warn("Unauthorized access attempt")
					return nil, status.Error(codes.PermissionDenied, "access denied")
				}
			}

			logger.Info("File retrieved successfully from cache")
			return &filev1.GetFileResponse{
				File: h.modelToProto(cachedFile),
			}, nil
		} else if err != cache.ErrCacheMiss && err != cache.ErrCacheDisabled {
			logger.WithError(err).Warn("Cache error, falling back to database")
		}
	}

	// Cache miss or disabled, fetch from database
	logger.Debug("Cache miss, querying database")
	file, err := h.fileRepo.FindByID(ctx, req.FileId)
	if err != nil {
		if errors.Is(err, repository.ErrFileNotFound) {
			return nil, status.Error(codes.NotFound, "file not found")
		}
		logger.WithError(err).Error("Failed to find file")
		return nil, status.Error(codes.Internal, "unable to process request")
	}

	// Check permissions
	if file.OwnerID != userID {
		hasAccess, err := h.fileRepo.CheckShareAccess(ctx, req.FileId, userID)
		if err != nil {
			logger.WithError(err).Error("Failed to check share access")
			return nil, status.Error(codes.Internal, "unable to process request")
		}

		if !hasAccess {
			logger.Warn("Unauthorized access attempt")
			return nil, status.Error(codes.PermissionDenied, "access denied")
		}
	}

	// Cache the result
	if h.cache != nil && h.cache.IsEnabled() {
		if err := h.cache.SetFileMetadata(ctx, file); err != nil {
			logger.WithError(err).Warn("Failed to cache file metadata")
		}
	}

	logger.Info("File retrieved successfully from database")

	return &filev1.GetFileResponse{
		File: h.modelToProto(file),
	}, nil
}

// invalidateFileCache invalidates all cache entries for a file
func (h *FileHandler) invalidateFileCache(ctx context.Context, fileID, ownerID string) {
	if h.cache == nil || !h.cache.IsEnabled() {
		return
	}

	if err := h.cache.InvalidateAllFileCache(ctx, fileID, ownerID); err != nil {
		h.logger.WithError(err).WithFields(map[string]interface{}{
			"file_id":  fileID,
			"owner_id": ownerID,
		}).Warn("Failed to invalidate file cache")
	}
}

// cachePresignedURL caches presigned URL data
func (h *FileHandler) cachePresignedURL(ctx context.Context, fileID, userID, url string, expiresAt time.Time) {
	if h.cache == nil || !h.cache.IsEnabled() {
		return
	}

	urlData := &cache.PresignedURLData{
		URL:       url,
		ExpiresAt: expiresAt,
		FileID:    fileID,
		UserID:    userID,
	}

	if err := h.cache.SetPresignedURLData(ctx, fileID, urlData); err != nil {
		h.logger.WithError(err).WithField("file_id", fileID).Warn("Failed to cache presigned URL data")
	}
}
