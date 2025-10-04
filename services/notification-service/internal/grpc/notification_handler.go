package grpc

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/models"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/services"
	notificationv1 "github.com/yourusername/distributed-file-sharing/services/notification-service/pkg/pb/notification/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NotificationGRPCServer implements the gRPC server for notification service
type NotificationGRPCServer struct {
	notificationv1.UnimplementedNotificationServiceServer
	notifSvc *services.NotificationService
	logger   *logrus.Logger
}

// NewNotificationGRPCServer creates a new NotificationGRPCServer
func NewNotificationGRPCServer(notifSvc *services.NotificationService, logger *logrus.Logger) *NotificationGRPCServer {
	return &NotificationGRPCServer{
		notifSvc: notifSvc,
		logger:   logger,
	}
}

// SendNotification handles incoming gRPC requests to send a notification
func (s *NotificationGRPCServer) SendNotification(ctx context.Context, req *notificationv1.SendNotificationRequest) (*notificationv1.SendNotificationResponse, error) {
	s.logger.WithFields(logrus.Fields{
		"user_id":    req.UserId,
		"event_type": req.EventType.String(),
		"title":      req.Title,
		"channel":    req.Channel.String(),
	}).Info("Received gRPC SendNotification request")

	// Convert gRPC request to internal model
	internalReq := &models.NotificationRequest{
		UserID:    req.UserId,
		EventType: models.EventType(req.EventType.String()),
		Channel:   models.ChannelEmail, // Default to email for now
		Title:     req.Title,
		Message:   req.Message,
		Priority:  models.PriorityNormal, // Default priority
		Metadata:  make(map[string]interface{}),
	}

	// Convert metadata
	for key, value := range req.Metadata {
		internalReq.Metadata[key] = value
	}

	// Send notification using the core service
	resp, err := s.notifSvc.SendNotification(ctx, internalReq)
	if err != nil {
		s.logger.WithError(err).Error("Failed to send notification via gRPC")
		return nil, status.Errorf(codes.Internal, "failed to send notification: %v", err)
	}

	// Convert internal response to gRPC response
	grpcResp := &notificationv1.SendNotificationResponse{
		Id:      resp.ID,
		Status:  notificationv1.NotificationStatus_NOTIFICATION_STATUS_SENT,
		Channel: req.Channel,
		Error:   "",
	}

	s.logger.WithField("notification_id", resp.ID).Info("gRPC SendNotification request processed")
	return grpcResp, nil
}

// GetUnreadCount handles incoming gRPC requests to get unread notification count
func (s *NotificationGRPCServer) GetUnreadCount(ctx context.Context, req *notificationv1.GetUnreadCountRequest) (*notificationv1.GetUnreadCountResponse, error) {
	s.logger.WithField("user_id", req.UserId).Info("Received gRPC GetUnreadCount request")

	// Get unread count using the core service
	count, err := s.notifSvc.GetUnreadCount(ctx, req.UserId)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get unread count via gRPC")
		return nil, status.Errorf(codes.Internal, "failed to get unread count: %v", err)
	}

	// Convert internal response to gRPC response
	grpcResp := &notificationv1.GetUnreadCountResponse{
		Count: count,
	}

	s.logger.WithFields(logrus.Fields{
		"user_id": req.UserId,
		"count":   count,
	}).Info("gRPC GetUnreadCount request processed")
	return grpcResp, nil
}
