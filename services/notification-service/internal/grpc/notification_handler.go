package grpc

import (
	"context"
	"errors"
	"log"

	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/kafka"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/models"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/repository"
	notificationv1 "github.com/yourusername/distributed-file-sharing/services/notification-service/pkg/pb/notification/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type NotificationHandler struct {
	notificationv1.UnimplementedNotificationServiceServer
	notifRepo    *repository.NotificationRepository
	streamBroker *kafka.StreamBroker
}

func NewNotificationHandler(
	notifRepo *repository.NotificationRepository,
	streamBroker *kafka.StreamBroker,
) *NotificationHandler {
	return &NotificationHandler{
		notifRepo:    notifRepo,
		streamBroker: streamBroker,
	}
}

func (h *NotificationHandler) GetNotifications(ctx context.Context, req *notificationv1.GetNotificationsRequest) (*notificationv1.GetNotificationsResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	page := req.Page
	if page < 1 {
		page = 1
	}

	limit := req.Limit
	if limit < 1 || limit > 100 {
		limit = 20
	}

	notifications, total, unreadCount, err := h.notifRepo.FindByUser(ctx, req.UserId, page, limit, req.UnreadOnly)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get notifications")
	}

	protoNotifications := make([]*notificationv1.Notification, 0, len(notifications))
	for _, notif := range notifications {
		protoNotifications = append(protoNotifications, h.modelToProto(notif))
	}

	return &notificationv1.GetNotificationsResponse{
		Notifications: protoNotifications,
		Total:         total,
		UnreadCount:   unreadCount,
		Page:          page,
		Limit:         limit,
	}, nil
}

func (h *NotificationHandler) MarkAsRead(ctx context.Context, req *notificationv1.MarkAsReadRequest) (*notificationv1.MarkAsReadResponse, error) {
	if req.NotificationId == "" || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "notification_id and user_id are required")
	}

	if err := h.notifRepo.MarkAsRead(ctx, req.NotificationId, req.UserId); err != nil {
		if errors.Is(err, repository.ErrNotificationNotFound) {
			return nil, status.Error(codes.NotFound, "notification not found")
		}
		return nil, status.Error(codes.Internal, "failed to mark notification as read")
	}

	return &notificationv1.MarkAsReadResponse{
		Message: "Notification marked as read",
	}, nil
}

func (h *NotificationHandler) MarkAllAsRead(ctx context.Context, req *notificationv1.MarkAllAsReadRequest) (*notificationv1.MarkAllAsReadResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	count, err := h.notifRepo.MarkAllAsRead(ctx, req.UserId)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to mark all notifications as read")
	}

	return &notificationv1.MarkAllAsReadResponse{
		Message: "All notifications marked as read",
		Count:   count,
	}, nil
}

func (h *NotificationHandler) DeleteNotification(ctx context.Context, req *notificationv1.DeleteNotificationRequest) (*notificationv1.DeleteNotificationResponse, error) {
	if req.NotificationId == "" || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "notification_id and user_id are required")
	}

	if err := h.notifRepo.Delete(ctx, req.NotificationId, req.UserId); err != nil {
		if errors.Is(err, repository.ErrNotificationNotFound) {
			return nil, status.Error(codes.NotFound, "notification not found")
		}
		return nil, status.Error(codes.Internal, "failed to delete notification")
	}

	return &notificationv1.DeleteNotificationResponse{
		Message: "Notification deleted",
	}, nil
}

func (h *NotificationHandler) GetUnreadCount(ctx context.Context, req *notificationv1.GetUnreadCountRequest) (*notificationv1.GetUnreadCountResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	count, err := h.notifRepo.GetUnreadCount(ctx, req.UserId)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get unread count")
	}

	return &notificationv1.GetUnreadCountResponse{
		Count: count,
	}, nil
}

func (h *NotificationHandler) SendNotification(ctx context.Context, req *notificationv1.SendNotificationRequest) (*notificationv1.SendNotificationResponse, error) {
	if req.UserId == "" || req.Title == "" || req.Body == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id, title, and body are required")
	}

	notification := &models.Notification{
		UserID:   req.UserId,
		Type:     models.NotificationType(req.Type.String()),
		Title:    req.Title,
		Body:     req.Body, // Changed from Message to Body
		Link:     req.Link,
		Metadata: req.Metadata,
	}

	if err := h.notifRepo.Create(ctx, notification); err != nil {
		return nil, status.Error(codes.Internal, "failed to create notification")
	}

	// Broadcast to streaming subscribers
	h.streamBroker.Broadcast(notification)

	return &notificationv1.SendNotificationResponse{
		NotificationId: notification.ID.Hex(),
		Message:        "Notification sent successfully",
	}, nil
}

// SubscribeStream implements server-side streaming for real-time notifications
func (h *NotificationHandler) SubscribeStream(req *notificationv1.SubscribeStreamRequest, stream notificationv1.NotificationService_SubscribeStreamServer) error {
	if req.UserId == "" {
		return status.Error(codes.InvalidArgument, "user_id is required")
	}

	log.Printf("User %s subscribed to notification stream", req.UserId)

	// Subscribe to stream broker
	ch := h.streamBroker.Subscribe(req.UserId)
	defer h.streamBroker.Unsubscribe(req.UserId, ch)

	// Send notifications from channel to stream
	for {
		select {
		case <-stream.Context().Done():
			log.Printf("User %s unsubscribed from notification stream", req.UserId)
			return nil
		case notification := <-ch:
			if err := stream.Send(h.modelToProto(notification)); err != nil {
				log.Printf("Error sending notification to stream: %v", err)
				return err
			}
		}
	}
}

func (h *NotificationHandler) modelToProto(notification *models.Notification) *notificationv1.Notification {
	return &notificationv1.Notification{
		NotificationId: notification.ID.Hex(),
		UserId:         notification.UserID,
		Type:           h.typeToProto(notification.Type),
		Title:          notification.Title,
		Body:           notification.Body, // Changed from Message to Body
		Link:           notification.Link,
		IsRead:         notification.IsRead, // Changed from Read to IsRead
		Metadata:       notification.Metadata,
		CreatedAt:      timestamppb.New(notification.CreatedAt),
	}
}

func (h *NotificationHandler) typeToProto(t models.NotificationType) notificationv1.NotificationType {
	switch t {
	case models.NotificationTypeFileUploaded:
		return notificationv1.NotificationType_NOTIFICATION_TYPE_FILE_UPLOADED
	case models.NotificationTypeFileShared:
		return notificationv1.NotificationType_NOTIFICATION_TYPE_FILE_SHARED
	case models.NotificationTypeFileDeleted:
		return notificationv1.NotificationType_NOTIFICATION_TYPE_FILE_DELETED
	case models.NotificationTypeShareRevoked:
		return notificationv1.NotificationType_NOTIFICATION_TYPE_SHARE_REVOKED
	case models.NotificationTypeSystem:
		return notificationv1.NotificationType_NOTIFICATION_TYPE_SYSTEM
	default:
		return notificationv1.NotificationType_NOTIFICATION_TYPE_UNSPECIFIED
	}
}
