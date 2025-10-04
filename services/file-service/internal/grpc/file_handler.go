package grpc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"sync"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sony/gobreaker"
	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/cache"
	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/config"
	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/kafka"
	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/models"
	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/repository"
	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/storage"
	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/validation"
	filev1 "github.com/yourusername/distributed-file-sharing/services/file-service/pkg/pb/file/v1"
	"golang.org/x/time/rate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// BillingClient interface for billing service communication
type BillingClient interface {
	UpdateUsage(ctx context.Context, userID string, usedBytes int64, fileCount int64, operation string) error
	CheckQuota(ctx context.Context, userID string, fileSizeBytes int64) (bool, string, int64, error)
}

type FileHandler struct {
	filev1.UnimplementedFileServiceServer
	fileRepo       *repository.FileRepository
	storageRepo    *repository.StorageRepository
	storage        *storage.MinioStorage
	producer       *kafka.Producer
	config         *config.Config
	logger         *logrus.Logger
	kafkaBreaker   *gobreaker.CircuitBreaker
	minioBreaker   *gobreaker.CircuitBreaker
	uploadLimiters map[string]*rate.Limiter
	limiterMu      sync.RWMutex
	cache          *cache.RedisCache
	billingClient  BillingClient
}

func NewFileHandler(
	fileRepo *repository.FileRepository,
	storageRepo *repository.StorageRepository,
	storage *storage.MinioStorage,
	producer *kafka.Producer,
	cfg *config.Config,
	logger *logrus.Logger,
	redisCache *cache.RedisCache,
	billingClient BillingClient,
) *FileHandler {
	return &FileHandler{
		fileRepo:    fileRepo,
		storageRepo: storageRepo,
		storage:     storage,
		producer:    producer,
		config:      cfg,
		logger:      logger,
		kafkaBreaker: gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:        "kafka",
			MaxRequests: cfg.CircuitBreakerMaxReq,
			Interval:    time.Minute,
			Timeout:     cfg.CircuitBreakerTimeout,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
				return counts.Requests >= 3 && failureRatio >= 0.6
			},
			OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
				logger.WithFields(logrus.Fields{
					"circuit_breaker": name,
					"from_state":      from.String(),
					"to_state":        to.String(),
				}).Warn("Circuit breaker state changed")
			},
		}),
		minioBreaker: gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:        "minio",
			MaxRequests: cfg.CircuitBreakerMaxReq,
			Interval:    time.Minute,
			Timeout:     cfg.CircuitBreakerTimeout,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
				return counts.Requests >= 3 && failureRatio >= 0.6
			},
			OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
				logger.WithFields(logrus.Fields{
					"circuit_breaker": name,
					"from_state":      from.String(),
					"to_state":        to.String(),
				}).Warn("Circuit breaker state changed")
			},
		}),
		uploadLimiters: make(map[string]*rate.Limiter),
		cache:          redisCache,
		billingClient:  billingClient,
	}
}

// getUserIDFromContext extracts user ID from context (set by auth middleware)
func (h *FileHandler) getUserIDFromContext(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}

	userIDs := md.Get("user_id")
	if len(userIDs) == 0 {
		return "", status.Error(codes.Unauthenticated, "user_id not found in metadata")
	}

	userID := userIDs[0]
	if userID == "" {
		return "", status.Error(codes.Unauthenticated, "empty user_id in metadata")
	}

	return userID, nil
}

// getRequestID extracts or generates request ID for tracing
func (h *FileHandler) getRequestID(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		if reqIDs := md.Get("request_id"); len(reqIDs) > 0 {
			return reqIDs[0]
		}
	}
	return uuid.New().String()
}

// getUploadLimiter returns rate limiter for user
func (h *FileHandler) getUploadLimiter(userID string) *rate.Limiter {
	h.limiterMu.Lock()
	defer h.limiterMu.Unlock()

	limiter, exists := h.uploadLimiters[userID]
	if !exists {
		// Rate limiting: uploads per minute
		limiter = rate.NewLimiter(
			rate.Every(time.Minute/time.Duration(h.config.UploadRatePerMinute)),
			h.config.UploadRateBurst,
		)
		h.uploadLimiters[userID] = limiter
	}
	return limiter
}

func (h *FileHandler) UploadFile(ctx context.Context, req *filev1.UploadFileRequest) (*filev1.UploadFileResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, h.config.OperationTimeout)
	defer cancel()

	requestID := h.getRequestID(ctx)
	logger := h.logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"method":     "UploadFile",
		"file_name":  req.Name,
		"file_size":  req.Size,
	})

	// Get authenticated user ID from context
	userID, err := h.getUserIDFromContext(ctx)
	if err != nil {
		logger.WithError(err).Warn("Authentication failed")
		return nil, err
	}

	logger = logger.WithField("user_id", userID)

	// Rate limiting
	limiter := h.getUploadLimiter(userID)
	if !limiter.Allow() {
		logger.Warn("Upload rate limit exceeded")
		return nil, status.Error(codes.ResourceExhausted, "upload rate limit exceeded, please try again later")
	}

	// Validate user ID format
	if err := validation.ValidateObjectID(userID); err != nil {
		logger.WithError(err).Error("Invalid user ID format")
		return nil, status.Error(codes.InvalidArgument, "invalid user ID")
	}

	// Validate required fields
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "file name is required")
	}

	// Sanitize filename and prevent path traversal
	safeName, err := validation.SanitizeFileName(req.Name)
	if err != nil {
		logger.WithError(err).Warn("Invalid filename")
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid filename: %v", err))
	}

	// Validate file size
	if err := validation.ValidateFileSize(req.Size, h.config.MinFileSize, h.config.MaxFileSize); err != nil {
		logger.WithError(err).Warn("Invalid file size")
		return nil, status.Errorf(codes.InvalidArgument, "file size must be between %d bytes and %d bytes", h.config.MinFileSize, h.config.MaxFileSize)
	}

	// TODO: Re-enable storage quota checking after billing integration is restored

	// Validate MIME type
	if err := validation.ValidateMimeType(req.MimeType, h.config.AllowedMimeTypes); err != nil {
		logger.WithError(err).Warn("Unsupported MIME type")
		return nil, status.Error(codes.InvalidArgument, "unsupported file type")
	}

	// Check storage quota before upload
	if err := h.checkStorageQuota(ctx, userID, req.Size); err != nil {
		logger.WithError(err).Warn("Storage quota exceeded")
		return nil, status.Error(codes.ResourceExhausted, "storage limit reached. Please upgrade your plan.")
	}

	// Generate safe storage path
	storagePath, err := validation.GenerateSafeStoragePath(userID, safeName)
	if err != nil {
		logger.WithError(err).Error("Failed to generate storage path")
		return nil, status.Error(codes.Internal, "unable to process request")
	}

	// Create file record
	now := time.Now()
	file := &models.File{
		Name:        safeName,
		Description: req.Description,
		Size:        req.Size,
		MimeType:    req.MimeType,
		OwnerID:     userID,
		StoragePath: storagePath,
		ContentHash: "", // Will be set in CompleteUpload
		Status:      models.FileStatusUploading,
		CreatedAt:   now,
		UpdatedAt:   now,
		Metadata: map[string]string{
			"upload_url_expires_at": now.Add(h.config.PresignedURLExpiry).Format(time.RFC3339),
			"request_id":            requestID,
		},
	}

	if err := h.fileRepo.Create(ctx, file); err != nil {
		logger.WithError(err).Error("Failed to create file record")
		return nil, status.Error(codes.Internal, "unable to process request")
	}

	logger = logger.WithField("file_id", file.ID.Hex())

	// Generate presigned upload URL with circuit breaker
	var uploadURL string
	_, err = h.minioBreaker.Execute(func() (interface{}, error) {
		var urlErr error
		uploadURL, urlErr = h.storage.GeneratePresignedUploadURL(ctx, file.StoragePath, h.config.PresignedURLExpiry)
		return uploadURL, urlErr
	})

	if err != nil {
		logger.WithError(err).Error("Failed to generate presigned URL")
		return nil, status.Error(codes.Internal, "unable to generate upload URL")
	}

	// Start goroutine to cleanup stale uploads
	go h.cleanupStaleUpload(file.ID.Hex(), h.config.PresignedURLExpiry+5*time.Minute)

	logger.Info("File upload initiated successfully")

	return &filev1.UploadFileResponse{
		FileId:    file.ID.Hex(),
		UploadUrl: uploadURL,
		Message:   "Upload URL generated. Use PUT request to upload file.",
	}, nil
}

func (h *FileHandler) CompleteUpload(ctx context.Context, req *filev1.CompleteUploadRequest) (*filev1.CompleteUploadResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, h.config.OperationTimeout)
	defer cancel()

	requestID := h.getRequestID(ctx)
	logger := h.logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"method":     "CompleteUpload",
		"file_id":    req.FileId,
	})

	// Get authenticated user ID
	userID, err := h.getUserIDFromContext(ctx)
	if err != nil {
		logger.WithError(err).Warn("Authentication failed")
		return nil, err
	}

	logger = logger.WithField("user_id", userID)

	// Validate input
	if req.FileId == "" {
		return nil, status.Error(codes.InvalidArgument, "file_id is required")
	}

	file, err := h.fileRepo.FindByID(ctx, req.FileId)
	if err != nil {
		if errors.Is(err, repository.ErrFileNotFound) {
			return nil, status.Error(codes.NotFound, "file not found")
		}
		logger.WithError(err).Error("Failed to find file")
		return nil, status.Error(codes.Internal, "unable to process request")
	}

	// Check ownership
	if file.OwnerID != userID {
		logger.Warn("Unauthorized access attempt")
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}

	// Verify checksum if provided
	if req.Checksum != "" {
		objectInfo, err := h.storage.GetFileInfo(ctx, file.StoragePath)
		if err != nil {
			logger.WithError(err).Warn("Failed to get file info for checksum verification")
			file.Status = models.FileStatusError
			h.fileRepo.Update(ctx, file)
			return nil, status.Error(codes.Internal, "file verification failed")
		}

		// Compare checksums (ETag is MD5 for single-part uploads)
		if objectInfo.ETag != req.Checksum {
			logger.WithFields(logrus.Fields{
				"expected_checksum": objectInfo.ETag,
				"provided_checksum": req.Checksum,
			}).Warn("Checksum mismatch")

			file.Status = models.FileStatusError
			h.fileRepo.Update(ctx, file)

			return nil, status.Error(codes.InvalidArgument, "checksum verification failed")
		}
	}

	// Update file status
	file.Status = models.FileStatusAvailable
	file.Checksum = req.Checksum
	file.UpdatedAt = time.Now()
	if file.ContentHash == "" {
		file.ContentHash = req.Checksum
	}

	if err := h.fileRepo.Update(ctx, file); err != nil {
		logger.WithError(err).Error("Failed to update file status")
		return nil, status.Error(codes.Internal, "unable to process request")
	}

	// Update storage usage in local storage repository
	if err := h.storageRepo.AddUsage(ctx, userID, file.Size); err != nil {
		logger.WithError(err).Warn("Failed to update local storage usage")
		// Don't fail the request if storage update fails
	} else {
		logger.WithFields(logrus.Fields{
			"user_id":   userID,
			"file_size": file.Size,
		}).Info("Storage usage updated successfully")
	}

	// Also update billing service if available
	if h.billingClient != nil {
		err = h.billingClient.UpdateUsage(ctx, userID, file.Size, 1, "ADD")
		if err != nil {
			logger.WithError(err).Warn("Failed to update billing service usage")
			// Don't fail the request if billing update fails
		}
	}

	// Publish file uploaded event with circuit breaker
	uploadEvent := kafka.NewFileUploadedEvent(
		file.ID.Hex(),
		file.OwnerID,
		file.Name,
		file.MimeType,
		file.Size,
		"{}", // Empty metadata for now
	)

	_, err = h.kafkaBreaker.Execute(func() (interface{}, error) {
		return nil, h.producer.PublishFileUploadedEvent(ctx, uploadEvent)
	})

	if err != nil {
		logger.WithError(err).Warn("Failed to publish file upload event (circuit breaker may be open)")
		// Don't fail the request if event publishing fails
	}

	// Also publish file version event for version tracking
	versionEvent := kafka.NewFileVersionedEvent(
		file.ID.Hex(),
		file.OwnerID,
		file.Name,
		file.MimeType,
		file.StoragePath,
		file.Checksum,
		"{}", // Empty metadata for now
		file.Size,
		1, // First version
	)

	_, err = h.kafkaBreaker.Execute(func() (interface{}, error) {
		return nil, h.producer.PublishFileVersionedEvent(ctx, versionEvent)
	})

	if err != nil {
		logger.WithError(err).Warn("Failed to publish file version event (circuit breaker may be open)")
		// Don't fail the request if event publishing fails
	}

	logger.Info("File upload completed successfully")

	return &filev1.CompleteUploadResponse{
		File:    h.modelToProto(file),
		Message: "File uploaded successfully",
	}, nil
}

func (h *FileHandler) GetFile(ctx context.Context, req *filev1.GetFileRequest) (*filev1.GetFileResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, h.config.QueryTimeout)
	defer cancel()

	requestID := h.getRequestID(ctx)
	logger := h.logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"method":     "GetFile",
		"file_id":    req.FileId,
	})

	// Get authenticated user ID
	userID, err := h.getUserIDFromContext(ctx)
	if err != nil {
		logger.WithError(err).Warn("Authentication failed")
		return nil, err
	}

	logger = logger.WithField("user_id", userID)

	if req.FileId == "" {
		return nil, status.Error(codes.InvalidArgument, "file_id is required")
	}

	file, err := h.fileRepo.FindByID(ctx, req.FileId)
	if err != nil {
		if errors.Is(err, repository.ErrFileNotFound) {
			return nil, status.Error(codes.NotFound, "file not found")
		}
		logger.WithError(err).Error("Failed to find file")
		return nil, status.Error(codes.Internal, "unable to process request")
	}

	// Check permissions (owner or shared with user)
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

	logger.Info("File retrieved successfully")

	return &filev1.GetFileResponse{
		File: h.modelToProto(file),
	}, nil
}

func (h *FileHandler) ListFiles(ctx context.Context, req *filev1.ListFilesRequest) (*filev1.ListFilesResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, h.config.QueryTimeout)
	defer cancel()

	requestID := h.getRequestID(ctx)
	logger := h.logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"method":     "ListFiles",
	})

	// Get authenticated user ID
	userID, err := h.getUserIDFromContext(ctx)
	if err != nil {
		logger.WithError(err).Warn("Authentication failed")
		return nil, err
	}

	logger = logger.WithField("user_id", userID)

	// Validate pagination
	page, limit, err := validation.ValidatePagination(req.Page, req.Limit, h.config.MaxPageSize)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	files, total, err := h.fileRepo.FindByOwner(ctx, userID, page, limit)
	if err != nil {
		logger.WithError(err).Error("Failed to list files")
		return nil, status.Error(codes.Internal, "unable to process request")
	}

	protoFiles := make([]*filev1.File, 0, len(files))
	for _, file := range files {
		protoFiles = append(protoFiles, h.modelToProto(file))
	}

	logger.WithFields(logrus.Fields{
		"count": len(files),
		"total": total,
		"page":  page,
	}).Info("Files listed successfully")

	return &filev1.ListFilesResponse{
		Files: protoFiles,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (h *FileHandler) GetDownloadURL(ctx context.Context, req *filev1.GetDownloadURLRequest) (*filev1.GetDownloadURLResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, h.config.OperationTimeout)
	defer cancel()

	requestID := h.getRequestID(ctx)
	logger := h.logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"method":     "GetDownloadURL",
		"file_id":    req.FileId,
	})

	// Get authenticated user ID
	userID, err := h.getUserIDFromContext(ctx)
	if err != nil {
		logger.WithError(err).Warn("Authentication failed")
		return nil, err
	}

	logger = logger.WithField("user_id", userID)

	if req.FileId == "" {
		return nil, status.Error(codes.InvalidArgument, "file_id is required")
	}

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

	// Generate download URL with circuit breaker
	var downloadURL string
	_, err = h.minioBreaker.Execute(func() (interface{}, error) {
		var urlErr error
		downloadURL, urlErr = h.storage.GeneratePresignedDownloadURL(ctx, file.StoragePath, h.config.PresignedURLExpiry)
		return downloadURL, urlErr
	})

	if err != nil {
		logger.WithError(err).Error("Failed to generate download URL")
		return nil, status.Error(codes.Internal, "unable to generate download URL")
	}

	// Publish file download event
	downloadEvent := kafka.NewFileDownloadedEvent(
		file.ID.Hex(),
		userID,
		file.Name,
		"{}", // Empty metadata for now
	)

	_, err = h.kafkaBreaker.Execute(func() (interface{}, error) {
		return nil, h.producer.PublishFileDownloadedEvent(ctx, downloadEvent)
	})

	if err != nil {
		logger.WithError(err).Warn("Failed to publish file download event")
		// Don't fail the request if event publishing fails
	}

	logger.Info("Download URL generated successfully")

	return &filev1.GetDownloadURLResponse{
		DownloadUrl: downloadURL,
		ExpiresIn:   int64(h.config.PresignedURLExpiry.Seconds()),
	}, nil
}

func (h *FileHandler) DeleteFile(ctx context.Context, req *filev1.DeleteFileRequest) (*filev1.DeleteFileResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, h.config.OperationTimeout)
	defer cancel()

	requestID := h.getRequestID(ctx)
	logger := h.logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"method":     "DeleteFile",
		"file_id":    req.FileId,
	})

	// Get authenticated user ID
	userID, err := h.getUserIDFromContext(ctx)
	if err != nil {
		logger.WithError(err).Warn("Authentication failed")
		return nil, err
	}

	logger = logger.WithField("user_id", userID)

	if req.FileId == "" {
		return nil, status.Error(codes.InvalidArgument, "file_id is required")
	}

	file, err := h.fileRepo.FindByID(ctx, req.FileId)
	if err != nil {
		if errors.Is(err, repository.ErrFileNotFound) {
			return nil, status.Error(codes.NotFound, "file not found")
		}
		logger.WithError(err).Error("Failed to find file")
		return nil, status.Error(codes.Internal, "unable to process request")
	}

	// Check ownership (only owner can delete)
	if file.OwnerID != userID {
		logger.Warn("Unauthorized delete attempt")
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}

	// Permanently delete from database (no trash functionality)
	if err := h.fileRepo.PermanentDeleteDirect(ctx, req.FileId); err != nil {
		logger.WithError(err).Error("Failed to permanently delete file")
		return nil, status.Error(codes.Internal, "unable to delete file")
	}

	// Decrease storage usage
	if err := h.storageRepo.RemoveUsage(ctx, userID, file.Size); err != nil {
		logger.WithError(err).Warn("Failed to update storage usage")
	}

	logger.WithFields(logrus.Fields{
		"file_id":   req.FileId,
		"user_id":   userID,
		"file_size": file.Size,
	}).Info("File permanently deleted - storage usage decreased")

	// Delete from MinIO storage
	_, err = h.minioBreaker.Execute(func() (interface{}, error) {
		return nil, h.storage.DeleteFile(ctx, file.StoragePath)
	})
	if err != nil {
		logger.WithError(err).Warn("Failed to delete file from storage")
		// Don't fail the request if storage deletion fails
	}

	// Publish file deleted event
	deleteEvent := kafka.NewFileDeletedEvent(
		file.ID.Hex(),
		file.OwnerID,
		file.Name,
		"{}", // Empty metadata for now
	)

	_, err = h.kafkaBreaker.Execute(func() (interface{}, error) {
		return nil, h.producer.PublishFileDeletedEvent(ctx, deleteEvent)
	})

	if err != nil {
		logger.WithError(err).Warn("Failed to publish file deletion event")
	}

	logger.Info("File permanently deleted successfully")

	return &filev1.DeleteFileResponse{
		Message: "File permanently deleted",
	}, nil
}

func (h *FileHandler) ShareFile(ctx context.Context, req *filev1.ShareFileRequest) (*filev1.ShareFileResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, h.config.OperationTimeout)
	defer cancel()

	requestID := h.getRequestID(ctx)
	logger := h.logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"method":     "ShareFile",
		"file_id":    req.FileId,
	})

	// Get authenticated user ID
	userID, err := h.getUserIDFromContext(ctx)
	if err != nil {
		logger.WithError(err).Warn("Authentication failed")
		return nil, err
	}

	logger = logger.WithField("user_id", userID)

	if req.FileId == "" {
		return nil, status.Error(codes.InvalidArgument, "file_id is required")
	}

	// Allow sharing with no emails (link-only sharing)
	if len(req.SharedWithEmails) == 0 && req.ExpiryTime == "" {
		return nil, status.Error(codes.InvalidArgument, "either shared_with_emails or expiry_time must be provided")
	}

	file, err := h.fileRepo.FindByID(ctx, req.FileId)
	if err != nil {
		if errors.Is(err, repository.ErrFileNotFound) {
			return nil, status.Error(codes.NotFound, "file not found")
		}
		logger.WithError(err).Error("Failed to find file")
		return nil, status.Error(codes.Internal, "unable to process request")
	}

	// Check ownership
	if file.OwnerID != userID {
		logger.Warn("Unauthorized share attempt")
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}

	// Parse expiry time
	var expiryTime *time.Time
	if req.ExpiryTime != "" {
		parsedTime, err := time.Parse(time.RFC3339, req.ExpiryTime)
		if err != nil {
			logger.WithError(err).WithField("expiry_time", req.ExpiryTime).Warn("Invalid expiry time format")
			return nil, status.Error(codes.InvalidArgument, "invalid expiry_time format, expected RFC3339")
		}
		expiryTime = &parsedTime
	}

	// Generate share link
	baseURL := h.config.FrontendURL
	if baseURL == "" {
		baseURL = "http://localhost:3000" // fallback
	}
	shareLink := fmt.Sprintf("%s/shared/%s", baseURL, req.FileId)

	// Create shares
	var protoShares []*filev1.FileShare
	var shareLinkGenerated bool

	// If no emails provided, create a link-only share (public share)
	if len(req.SharedWithEmails) == 0 {
		share := &models.FileShare{
			FileID:          req.FileId,
			OwnerID:         userID,
			SharedWithID:    "", // Empty for public shares
			SharedWithEmail: "", // Empty for public shares
			Permission:      models.Permission(req.Permission.String()),
			ExpiryTime:      expiryTime,
			ShareLink:       shareLink,
			IsActive:        true,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		if err := h.fileRepo.CreateShare(ctx, share); err != nil {
			logger.WithError(err).Error("Failed to create link-only share")
			return nil, status.Error(codes.Internal, "unable to create share")
		}

		var expiryTimestamp *timestamppb.Timestamp
		if share.ExpiryTime != nil {
			expiryTimestamp = timestamppb.New(*share.ExpiryTime)
		}

		protoShares = append(protoShares, &filev1.FileShare{
			ShareId:         share.ID.Hex(),
			FileId:          share.FileID,
			OwnerId:         share.OwnerID,
			SharedWithEmail: "", // Empty for link-only shares
			Permission:      req.Permission,
			ExpiryTime:      expiryTimestamp,
			ShareLink:       share.ShareLink,
			IsActive:        share.IsActive,
			CreatedAt:       timestamppb.New(share.CreatedAt),
			UpdatedAt:       timestamppb.New(share.UpdatedAt),
		})
		shareLinkGenerated = true
	} else {
		// Create shares for each email
		for _, email := range req.SharedWithEmails {
			// Validate email
			if err := validation.ValidateEmail(email); err != nil {
				logger.WithError(err).WithField("email", email).Warn("Invalid email")
				continue
			}

			share := &models.FileShare{
				FileID:          req.FileId,
				OwnerID:         userID,
				SharedWithEmail: email,
				Permission:      models.Permission(req.Permission.String()),
				ExpiryTime:      expiryTime,
				ShareLink:       shareLink,
				IsActive:        true,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			}

			if err := h.fileRepo.CreateShare(ctx, share); err != nil {
				logger.WithError(err).WithField("email", email).Error("Failed to create share")
				continue
			}

			var expiryTimestamp *timestamppb.Timestamp
			if share.ExpiryTime != nil {
				expiryTimestamp = timestamppb.New(*share.ExpiryTime)
			}

			protoShares = append(protoShares, &filev1.FileShare{
				ShareId:         share.ID.Hex(),
				FileId:          share.FileID,
				OwnerId:         share.OwnerID,
				SharedWithEmail: share.SharedWithEmail,
				Permission:      req.Permission,
				ExpiryTime:      expiryTimestamp,
				ShareLink:       share.ShareLink,
				IsActive:        share.IsActive,
				CreatedAt:       timestamppb.New(share.CreatedAt),
				UpdatedAt:       timestamppb.New(share.UpdatedAt),
			})

			// Publish file shared event
			event := kafka.FileEvent{
				Type:      kafka.EventFileShared,
				FileID:    file.ID.Hex(),
				FileName:  file.Name,
				OwnerID:   file.OwnerID,
				Metadata:  map[string]string{"shared_with": email, "permission": req.Permission.String()},
				Timestamp: time.Now().Format(time.RFC3339),
			}

			_, err := h.kafkaBreaker.Execute(func() (interface{}, error) {
				return nil, h.producer.PublishFileEvent(ctx, event)
			})

			if err != nil {
				logger.WithError(err).Warn("Failed to publish Kafka event")
			}
		}
		shareLinkGenerated = true
	}

	logger.WithFields(logrus.Fields{
		"share_count": len(protoShares),
		"share_link":  shareLinkGenerated,
	}).Info("File shared successfully")

	response := &filev1.ShareFileResponse{
		Shares:  protoShares,
		Message: "File shared successfully",
	}

	if shareLinkGenerated {
		response.ShareLink = shareLink
	}

	return response, nil
}

func (h *FileHandler) UnshareFile(ctx context.Context, req *filev1.UnshareFileRequest) (*filev1.UnshareFileResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, h.config.OperationTimeout)
	defer cancel()

	requestID := h.getRequestID(ctx)
	logger := h.logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"method":     "UnshareFile",
		"file_id":    req.FileId,
		"share_id":   req.ShareId,
	})

	// Get authenticated user ID
	userID, err := h.getUserIDFromContext(ctx)
	if err != nil {
		logger.WithError(err).Warn("Authentication failed")
		return nil, err
	}

	logger = logger.WithField("user_id", userID)

	if req.FileId == "" || req.ShareId == "" {
		return nil, status.Error(codes.InvalidArgument, "file_id and share_id are required")
	}

	file, err := h.fileRepo.FindByID(ctx, req.FileId)
	if err != nil {
		if errors.Is(err, repository.ErrFileNotFound) {
			return nil, status.Error(codes.NotFound, "file not found")
		}
		logger.WithError(err).Error("Failed to find file")
		return nil, status.Error(codes.Internal, "unable to process request")
	}

	// Check ownership
	if file.OwnerID != userID {
		logger.Warn("Unauthorized unshare attempt")
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}

	if err := h.fileRepo.DeleteShare(ctx, req.ShareId); err != nil {
		logger.WithError(err).Error("Failed to delete share")
		return nil, status.Error(codes.Internal, "unable to process request")
	}

	logger.Info("Share removed successfully")

	return &filev1.UnshareFileResponse{
		Message: "Share removed successfully",
	}, nil
}

func (h *FileHandler) ListSharedFiles(ctx context.Context, req *filev1.ListSharedFilesRequest) (*filev1.ListSharedFilesResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, h.config.QueryTimeout)
	defer cancel()

	requestID := h.getRequestID(ctx)
	logger := h.logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"method":     "ListSharedFiles",
	})

	// Get authenticated user ID
	userID, err := h.getUserIDFromContext(ctx)
	if err != nil {
		logger.WithError(err).Warn("Authentication failed")
		return nil, err
	}

	logger = logger.WithField("user_id", userID)

	// Validate pagination
	page, limit, err := validation.ValidatePagination(req.Page, req.Limit, h.config.MaxPageSize)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	files, total, err := h.fileRepo.FindSharedWithUser(ctx, userID, page, limit)
	if err != nil {
		logger.WithError(err).Error("Failed to list shared files")
		return nil, status.Error(codes.Internal, "unable to process request")
	}

	protoFiles := make([]*filev1.File, 0, len(files))
	for _, file := range files {
		protoFiles = append(protoFiles, h.modelToProto(file))
	}

	logger.WithFields(logrus.Fields{
		"count": len(files),
		"total": total,
		"page":  page,
	}).Info("Shared files listed successfully")

	return &filev1.ListSharedFilesResponse{
		Files: protoFiles,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (h *FileHandler) UpdateFile(ctx context.Context, req *filev1.UpdateFileRequest) (*filev1.UpdateFileResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, h.config.OperationTimeout)
	defer cancel()

	requestID := h.getRequestID(ctx)
	logger := h.logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"method":     "UpdateFile",
		"file_id":    req.FileId,
	})

	// Get authenticated user ID
	userID, err := h.getUserIDFromContext(ctx)
	if err != nil {
		logger.WithError(err).Warn("Authentication failed")
		return nil, err
	}

	logger = logger.WithField("user_id", userID)

	if req.FileId == "" {
		return nil, status.Error(codes.InvalidArgument, "file_id is required")
	}

	file, err := h.fileRepo.FindByID(ctx, req.FileId)
	if err != nil {
		if errors.Is(err, repository.ErrFileNotFound) {
			return nil, status.Error(codes.NotFound, "file not found")
		}
		logger.WithError(err).Error("Failed to find file")
		return nil, status.Error(codes.Internal, "unable to process request")
	}

	// Check ownership
	if file.OwnerID != userID {
		logger.Warn("Unauthorized update attempt")
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}

	// Update fields
	if req.Name != "" {
		safeName, err := validation.SanitizeFileName(req.Name)
		if err != nil {
			logger.WithError(err).Warn("Invalid filename")
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid filename: %v", err))
		}
		file.Name = safeName
	}

	if req.Description != "" {
		file.Description = req.Description
	}

	// Update timestamp
	file.UpdatedAt = time.Now()

	if err := h.fileRepo.Update(ctx, file); err != nil {
		logger.WithError(err).Error("Failed to update file")
		return nil, status.Error(codes.Internal, "unable to process request")
	}

	logger.Info("File updated successfully")

	return &filev1.UpdateFileResponse{
		File:    h.modelToProto(file),
		Message: "File updated successfully",
	}, nil
}

func (h *FileHandler) GetStorageUsage(ctx context.Context, req *filev1.GetStorageUsageRequest) (*filev1.GetStorageUsageResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, h.config.QueryTimeout)
	defer cancel()

	requestID := h.getRequestID(ctx)
	logger := h.logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"method":     "GetStorageUsage",
	})

	// Get authenticated user ID
	userID, err := h.getUserIDFromContext(ctx)
	if err != nil {
		logger.WithError(err).Warn("Authentication failed")
		return nil, err
	}

	logger = logger.WithField("user_id", userID)

	// Get or calculate storage usage
	stats, err := h.storageRepo.CalculateUsageFromFiles(ctx, userID, h.fileRepo)
	if err != nil {
		logger.WithError(err).Error("Failed to calculate storage usage")
		return nil, status.Error(codes.Internal, "unable to calculate storage usage")
	}

	logger.WithFields(logrus.Fields{
		"used_bytes":  stats.UsedBytes,
		"quota_bytes": stats.QuotaBytes,
		"file_count":  stats.FileCount,
	}).Info("Storage usage retrieved successfully")

	return &filev1.GetStorageUsageResponse{
		UsedBytes:       stats.UsedBytes,
		QuotaBytes:      stats.QuotaBytes,
		FileCount:       stats.FileCount,
		UsedGb:          stats.GetUsedGB(),
		QuotaGb:         stats.GetQuotaGB(),
		UsagePercentage: stats.GetUsagePercentage(),
	}, nil
}

// checkStorageQuota checks if user has enough storage quota for the file
func (h *FileHandler) checkStorageQuota(ctx context.Context, userID string, fileSize int64) error {
	// Use billing service for quota check if available
	if h.billingClient != nil {
		canUpload, message, _, err := h.billingClient.CheckQuota(ctx, userID, fileSize)
		if err != nil {
			h.logger.WithError(err).Warn("Failed to check quota with billing service, falling back to local calculation")
			// Fall back to local calculation
		} else {
			if !canUpload {
				return fmt.Errorf("storage quota exceeded: %s", message)
			}
			return nil
		}
	}

	// Fallback to local storage calculation
	stats, err := h.storageRepo.CalculateUsageFromFiles(ctx, userID, h.fileRepo)
	if err != nil {
		h.logger.WithError(err).Error("Failed to calculate storage usage for quota check")
		return fmt.Errorf("unable to check storage quota")
	}

	// Check if adding this file would exceed quota
	if (stats.UsedBytes + fileSize) > stats.QuotaBytes {
		return fmt.Errorf("storage quota exceeded: used %d bytes, quota %d bytes, file size %d bytes",
			stats.UsedBytes, stats.QuotaBytes, fileSize)
	}

	return nil
}

// cleanupStaleUpload marks files as error if upload not completed in time
func (h *FileHandler) cleanupStaleUpload(fileID string, timeout time.Duration) {
	time.Sleep(timeout)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	file, err := h.fileRepo.FindByID(ctx, fileID)
	if err != nil {
		h.logger.WithError(err).WithField("file_id", fileID).Warn("Failed to find file for cleanup")
		return
	}

	// If still in uploading state, mark as error
	if file.Status == models.FileStatusUploading {
		file.Status = models.FileStatusError
		if err := h.fileRepo.Update(ctx, file); err != nil {
			h.logger.WithError(err).WithField("file_id", fileID).Error("Failed to mark stale upload as error")
		} else {
			h.logger.WithField("file_id", fileID).Info("Marked stale upload as error")
		}
	}
}

func (h *FileHandler) modelToProto(file *models.File) *filev1.File {
	// Ensure timestamps are valid - use current time as fallback for zero values
	createdAt := file.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	updatedAt := file.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}

	return &filev1.File{
		FileId:      file.ID.Hex(),
		Name:        file.Name,
		Description: file.Description,
		Size:        file.Size,
		MimeType:    file.MimeType,
		OwnerId:     file.OwnerID,
		StoragePath: file.StoragePath,
		Checksum:    file.Checksum,
		Status:      h.statusToProto(file.Status),
		CreatedAt:   timestamppb.New(createdAt),
		UpdatedAt:   timestamppb.New(updatedAt),
	}
}

func (h *FileHandler) statusToProto(status models.FileStatus) filev1.FileStatus {
	switch status {
	case models.FileStatusUploading:
		return filev1.FileStatus_FILE_STATUS_UPLOADING
	case models.FileStatusAvailable:
		return filev1.FileStatus_FILE_STATUS_AVAILABLE
	case models.FileStatusProcessing:
		return filev1.FileStatus_FILE_STATUS_PROCESSING
	case models.FileStatusError:
		return filev1.FileStatus_FILE_STATUS_ERROR
	default:
		return filev1.FileStatus_FILE_STATUS_UNSPECIFIED
	}
}

// AddToFavorites adds a file to user's favorites
func (h *FileHandler) AddToFavorites(ctx context.Context, req *filev1.FavoriteRequest) (*filev1.FavoriteResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, h.config.OperationTimeout)
	defer cancel()

	requestID := h.getRequestID(ctx)
	logger := h.logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"method":     "AddToFavorites",
		"file_id":    req.FileId,
	})

	// Get authenticated user ID
	userID, err := h.getUserIDFromContext(ctx)
	if err != nil {
		logger.WithError(err).Warn("Authentication failed")
		return nil, err
	}

	logger = logger.WithField("user_id", userID)

	if req.FileId == "" {
		return nil, status.Error(codes.InvalidArgument, "file_id is required")
	}

	// Verify file exists and user has access
	file, err := h.fileRepo.FindByID(ctx, req.FileId)
	if err != nil {
		if errors.Is(err, repository.ErrFileNotFound) {
			return nil, status.Error(codes.NotFound, "file not found")
		}
		logger.WithError(err).Error("Failed to find file")
		return nil, status.Error(codes.Internal, "unable to process request")
	}

	// Check if user owns the file or has access to it
	if file.OwnerID != userID {
		hasAccess, err := h.fileRepo.CheckShareAccess(ctx, req.FileId, userID)
		if err != nil {
			logger.WithError(err).Error("Failed to check share access")
			return nil, status.Error(codes.Internal, "unable to process request")
		}
		if !hasAccess {
			logger.Warn("Unauthorized favorite attempt")
			return nil, status.Error(codes.PermissionDenied, "access denied")
		}
	}

	// Add to favorites
	err = h.fileRepo.AddToFavorites(ctx, userID, req.FileId)
	if err != nil {
		logger.WithError(err).Error("Failed to add to favorites")
		return nil, status.Error(codes.Internal, "unable to add to favorites")
	}

	logger.Info("File added to favorites successfully")

	return &filev1.FavoriteResponse{
		Message:    "File added to favorites",
		IsFavorite: true,
	}, nil
}

// RemoveFromFavorites removes a file from user's favorites
func (h *FileHandler) RemoveFromFavorites(ctx context.Context, req *filev1.FavoriteRequest) (*filev1.FavoriteResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, h.config.OperationTimeout)
	defer cancel()

	requestID := h.getRequestID(ctx)
	logger := h.logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"method":     "RemoveFromFavorites",
		"file_id":    req.FileId,
	})

	// Get authenticated user ID
	userID, err := h.getUserIDFromContext(ctx)
	if err != nil {
		logger.WithError(err).Warn("Authentication failed")
		return nil, err
	}

	logger = logger.WithField("user_id", userID)

	if req.FileId == "" {
		return nil, status.Error(codes.InvalidArgument, "file_id is required")
	}

	// Remove from favorites
	err = h.fileRepo.RemoveFromFavorites(ctx, userID, req.FileId)
	if err != nil {
		logger.WithError(err).Error("Failed to remove from favorites")
		return nil, status.Error(codes.Internal, "unable to remove from favorites")
	}

	logger.Info("File removed from favorites successfully")

	return &filev1.FavoriteResponse{
		Message:    "File removed from favorites",
		IsFavorite: false,
	}, nil
}

// ListFavorites lists user's favorite files
func (h *FileHandler) ListFavorites(ctx context.Context, req *filev1.ListFavoritesRequest) (*filev1.ListFilesResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, h.config.QueryTimeout)
	defer cancel()

	requestID := h.getRequestID(ctx)
	logger := h.logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"method":     "ListFavorites",
	})

	// Get authenticated user ID
	userID, err := h.getUserIDFromContext(ctx)
	if err != nil {
		logger.WithError(err).Warn("Authentication failed")
		return nil, err
	}

	logger = logger.WithField("user_id", userID)

	// Validate pagination
	page, limit, err := validation.ValidatePagination(req.Page, req.Limit, h.config.MaxPageSize)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	files, total, err := h.fileRepo.FindFavoritesByUser(ctx, userID, page, limit)
	if err != nil {
		logger.WithError(err).Error("Failed to list favorite files")
		return nil, status.Error(codes.Internal, "unable to process request")
	}

	protoFiles := make([]*filev1.File, 0, len(files))
	for _, file := range files {
		protoFiles = append(protoFiles, h.modelToProto(file))
	}

	logger.WithFields(logrus.Fields{
		"count": len(files),
		"total": total,
		"page":  page,
	}).Info("Favorite files listed successfully")

	return &filev1.ListFilesResponse{
		Files: protoFiles,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}
