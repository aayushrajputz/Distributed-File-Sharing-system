package grpc

import (
	"context"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/models"
	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/service"
	filev1 "github.com/yourusername/distributed-file-sharing/services/file-service/pkg/pb/file/v1"
)

// PrivateFolderHandler handles private folder gRPC requests
type PrivateFolderHandler struct {
	filev1.UnimplementedPrivateFolderServiceServer
	service *service.PrivateFolderService
	logger  *logrus.Logger
}

// NewPrivateFolderHandler creates a new private folder handler
func NewPrivateFolderHandler(service *service.PrivateFolderService, logger *logrus.Logger) *PrivateFolderHandler {
	return &PrivateFolderHandler{
		service: service,
		logger:  logger,
	}
}

// SetPIN sets or updates a user's PIN
func (h *PrivateFolderHandler) SetPIN(ctx context.Context, req *filev1.SetPINRequest) (*filev1.SetPINResponse, error) {
	h.logger.WithFields(logrus.Fields{
		"user_id": req.UserId,
		"method":  "SetPIN",
	}).Info("Setting PIN for user")

	err := h.service.SetPIN(ctx, req.UserId, req.Pin)
	if err != nil {
		h.logger.WithError(err).Error("Failed to set PIN")
		return &filev1.SetPINResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &filev1.SetPINResponse{
		Success: true,
		Message: "PIN set successfully",
	}, nil
}

// ValidatePIN validates a user's PIN
func (h *PrivateFolderHandler) ValidatePIN(ctx context.Context, req *filev1.ValidatePINRequest) (*filev1.ValidatePINResponse, error) {
	h.logger.WithFields(logrus.Fields{
		"user_id": req.UserId,
		"method":  "ValidatePIN",
	}).Info("Validating PIN for user")

	pinReq := &models.PINValidationRequest{
		UserID:    req.UserId,
		PIN:       req.Pin,
		IPAddress: getClientIP(ctx),
		UserAgent: getUserAgent(ctx),
	}

	resp, err := h.service.ValidatePIN(ctx, pinReq)
	if err != nil {
		h.logger.WithError(err).Error("Failed to validate PIN")
		return &filev1.ValidatePINResponse{
			Success: false,
			Message: "Internal server error",
		}, status.Error(codes.Internal, "failed to validate PIN")
	}

	return &filev1.ValidatePINResponse{
		Success:      resp.Success,
		Message:      resp.Message,
		AttemptsLeft: int32(resp.AttemptsLeft),
		LockedUntil:  resp.LockedUntil,
	}, nil
}

// MakeFilePrivate moves a file to private folder
func (h *PrivateFolderHandler) MakeFilePrivate(ctx context.Context, req *filev1.MakeFilePrivateRequest) (*filev1.MakeFilePrivateResponse, error) {
	h.logger.WithFields(logrus.Fields{
		"user_id": req.UserId,
		"file_id": req.FileId,
		"method":  "MakeFilePrivate",
	}).Info("Making file private")

	makePrivateReq := &models.MakePrivateRequest{
		UserID: req.UserId,
		FileID: req.FileId,
		PIN:    req.Pin,
	}

	resp, err := h.service.MakeFilePrivate(ctx, makePrivateReq)
	if err != nil {
		h.logger.WithError(err).Error("Failed to make file private")
		return &filev1.MakeFilePrivateResponse{
			Success: false,
			Message: "Internal server error",
		}, status.Error(codes.Internal, "failed to make file private")
	}

	return &filev1.MakeFilePrivateResponse{
		Success: resp.Success,
		Message: resp.Message,
		FileId:  resp.FileID,
	}, nil
}

// RemoveFileFromPrivate removes a file from private folder
func (h *PrivateFolderHandler) RemoveFileFromPrivate(ctx context.Context, req *filev1.RemoveFileFromPrivateRequest) (*filev1.RemoveFileFromPrivateResponse, error) {
	h.logger.WithFields(logrus.Fields{
		"user_id": req.UserId,
		"file_id": req.FileId,
		"method":  "RemoveFileFromPrivate",
	}).Info("Removing file from private folder")

	resp, err := h.service.RemoveFileFromPrivate(ctx, req.UserId, req.FileId, req.Pin)
	if err != nil {
		h.logger.WithError(err).Error("Failed to remove file from private folder")
		return &filev1.RemoveFileFromPrivateResponse{
			Success: false,
			Message: "Internal server error",
		}, status.Error(codes.Internal, "failed to remove file from private folder")
	}

	return &filev1.RemoveFileFromPrivateResponse{
		Success: resp.Success,
		Message: resp.Message,
		FileId:  resp.FileID,
	}, nil
}

// GetPrivateFiles retrieves all private files for a user
func (h *PrivateFolderHandler) GetPrivateFiles(ctx context.Context, req *filev1.GetPrivateFilesRequest) (*filev1.GetPrivateFilesResponse, error) {
	h.logger.WithFields(logrus.Fields{
		"user_id": req.UserId,
		"method":  "GetPrivateFiles",
	}).Info("Getting private files for user")

	resp, err := h.service.GetPrivateFiles(ctx, req.UserId, req.Limit, req.Offset)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get private files")
		return &filev1.GetPrivateFilesResponse{
			Success: false,
			Message: "Internal server error",
		}, status.Error(codes.Internal, "failed to get private files")
	}

	// Convert to protobuf format
	var files []*filev1.PrivateFileInfo
	for _, file := range resp.Files {
		files = append(files, &filev1.PrivateFileInfo{
			FileId:         file.FileID,
			FileName:       file.FileName,
			FileSize:       file.FileSize,
			ContentType:    file.ContentType,
			MovedAt:        file.MovedAt.Unix(),
			OriginalFolder: file.OriginalFolder,
		})
	}

	return &filev1.GetPrivateFilesResponse{
		Success: true,
		Message: "Private files retrieved successfully",
		Files:   files,
		Total:   resp.Total,
	}, nil
}

// GetAccessLogs retrieves access logs for a user
func (h *PrivateFolderHandler) GetAccessLogs(ctx context.Context, req *filev1.GetAccessLogsRequest) (*filev1.GetAccessLogsResponse, error) {
	h.logger.WithFields(logrus.Fields{
		"user_id": req.UserId,
		"method":  "GetAccessLogs",
	}).Info("Getting access logs for user")

	logs, err := h.service.GetAccessLogs(ctx, req.UserId, req.Limit)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get access logs")
		return &filev1.GetAccessLogsResponse{
			Success: false,
			Message: "Internal server error",
		}, status.Error(codes.Internal, "failed to get access logs")
	}

	// Convert to protobuf format
	var accessLogs []*filev1.AccessLog
	for _, log := range logs {
		accessLogs = append(accessLogs, &filev1.AccessLog{
			Id:            log.ID.Hex(),
			UserId:        log.UserID,
			FileId:        log.FileID,
			Action:        log.Action,
			IpAddress:     log.IPAddress,
			UserAgent:     log.UserAgent,
			Success:       log.Success,
			FailureReason: log.FailureReason,
			CreatedAt:     log.CreatedAt.Unix(),
		})
	}

	return &filev1.GetAccessLogsResponse{
		Success:    true,
		Message:    "Access logs retrieved successfully",
		AccessLogs: accessLogs,
	}, nil
}

// CheckFileAccess checks if user can access a private file
func (h *PrivateFolderHandler) CheckFileAccess(ctx context.Context, req *filev1.CheckFileAccessRequest) (*filev1.CheckFileAccessResponse, error) {
	h.logger.WithFields(logrus.Fields{
		"user_id": req.UserId,
		"file_id": req.FileId,
		"method":  "CheckFileAccess",
	}).Info("Checking file access")

	hasAccess, err := h.service.CheckFileAccess(ctx, req.UserId, req.FileId)
	if err != nil {
		h.logger.WithError(err).Error("Failed to check file access")
		return &filev1.CheckFileAccessResponse{
			HasAccess: false,
			Message:   "Internal server error",
		}, status.Error(codes.Internal, "failed to check file access")
	}

	message := "Access granted"
	if !hasAccess {
		message = "PIN required for private file access"
	}

	return &filev1.CheckFileAccessResponse{
		HasAccess: hasAccess,
		Message:   message,
	}, nil
}

// Helper functions to extract client information from context
func getClientIP(ctx context.Context) string {
	// Extract IP from gRPC metadata or context
	// This is a simplified implementation
	return "127.0.0.1"
}

func getUserAgent(ctx context.Context) string {
	// Extract User-Agent from gRPC metadata or context
	// This is a simplified implementation
	return "grpc-client"
}
