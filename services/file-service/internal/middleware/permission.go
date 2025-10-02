package middleware

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/models"
	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PermissionChecker handles permission validation for file operations
type PermissionChecker struct {
	fileRepo *repository.FileRepository
	logger   *logrus.Logger
}

// NewPermissionChecker creates a new permission checker
func NewPermissionChecker(fileRepo *repository.FileRepository, logger *logrus.Logger) *PermissionChecker {
	return &PermissionChecker{
		fileRepo: fileRepo,
		logger:   logger,
	}
}

// CheckFileAccess validates if a user has access to a file with the required permission
func (pc *PermissionChecker) CheckFileAccess(ctx context.Context, fileID, userID string, requiredPermission models.Permission) error {
	logger := pc.logger.WithFields(logrus.Fields{
		"file_id":             fileID,
		"user_id":             userID,
		"required_permission": requiredPermission,
	})

	// Check if user is the owner
	file, err := pc.fileRepo.FindByID(ctx, fileID)
	if err != nil {
		if repository.IsErrFileNotFound(err) {
			return status.Error(codes.NotFound, "file not found")
		}
		logger.WithError(err).Error("Failed to find file")
		return status.Error(codes.Internal, "unable to process request")
	}

	// Owner has all permissions
	if file.OwnerID == userID {
		logger.Debug("User is file owner, access granted")
		return nil
	}

	// Check shared access
	hasAccess, permission, err := pc.fileRepo.CheckShareAccessWithPermission(ctx, fileID, userID)
	if err != nil {
		logger.WithError(err).Error("Failed to check share access")
		return status.Error(codes.Internal, "unable to process request")
	}

	if !hasAccess {
		logger.Warn("User does not have access to file")
		return status.Error(codes.PermissionDenied, "access denied")
	}

	// Check if share has expired
	share, err := pc.fileRepo.GetActiveShare(ctx, fileID, userID)
	if err != nil {
		logger.WithError(err).Error("Failed to get share details")
		return status.Error(codes.Internal, "unable to process request")
	}

	if share != nil && share.ExpiryTime != nil && time.Now().After(*share.ExpiryTime) {
		logger.Warn("Share has expired")
		return status.Error(codes.PermissionDenied, "share has expired")
	}

	// Check if share is active
	if share != nil && !share.IsActive {
		logger.Warn("Share is not active")
		return status.Error(codes.PermissionDenied, "share is not active")
	}

	// Check permission level
	if !pc.hasRequiredPermission(permission, requiredPermission) {
		logger.WithFields(logrus.Fields{
			"user_permission":     permission,
			"required_permission": requiredPermission,
		}).Warn("Insufficient permissions")
		return status.Error(codes.PermissionDenied, "insufficient permissions")
	}

	logger.Debug("Access granted")
	return nil
}

// hasRequiredPermission checks if the user's permission level meets the required permission
func (pc *PermissionChecker) hasRequiredPermission(userPermission, requiredPermission models.Permission) bool {
	// Define permission hierarchy
	permissionLevels := map[models.Permission]int{
		models.PermissionRead:  1,
		models.PermissionWrite: 2,
		models.PermissionAdmin: 3,
	}

	userLevel, exists := permissionLevels[userPermission]
	if !exists {
		return false
	}

	requiredLevel, exists := permissionLevels[requiredPermission]
	if !exists {
		return false
	}

	return userLevel >= requiredLevel
}

// CheckReadAccess checks if user can read the file
func (pc *PermissionChecker) CheckReadAccess(ctx context.Context, fileID, userID string) error {
	return pc.CheckFileAccess(ctx, fileID, userID, models.PermissionRead)
}

// CheckWriteAccess checks if user can write/edit the file
func (pc *PermissionChecker) CheckWriteAccess(ctx context.Context, fileID, userID string) error {
	return pc.CheckFileAccess(ctx, fileID, userID, models.PermissionWrite)
}

// CheckAdminAccess checks if user has admin access to the file
func (pc *PermissionChecker) CheckAdminAccess(ctx context.Context, fileID, userID string) error {
	return pc.CheckFileAccess(ctx, fileID, userID, models.PermissionAdmin)
}

// CheckDeleteAccess checks if user can delete the file (admin only)
func (pc *PermissionChecker) CheckDeleteAccess(ctx context.Context, fileID, userID string) error {
	return pc.CheckFileAccess(ctx, fileID, userID, models.PermissionAdmin)
}

// CheckShareAccess checks if user can share the file (admin only)
func (pc *PermissionChecker) CheckShareAccess(ctx context.Context, fileID, userID string) error {
	return pc.CheckFileAccess(ctx, fileID, userID, models.PermissionAdmin)
}




