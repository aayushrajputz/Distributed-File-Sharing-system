package services

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/models"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/repository"
)

// NotificationService is the main service that orchestrates all notification operations
type NotificationService struct {
	notifRepo     *repository.NotificationRepository
	preferenceSvc *PreferenceService
	templateSvc   *TemplateService
	batchSvc      *BatchService
	dlqSvc        *DLQService
	retrySvc      *RetryService
	handlers      map[models.NotificationChannel]NotificationHandler
	config        *ServiceConfig
	logger        *logrus.Logger
}

// ServiceConfig contains service configuration
type ServiceConfig struct {
	EnableBatching   bool
	EnableRetry      bool
	EnableDLQ        bool
	DefaultChannel   models.NotificationChannel
	FallbackChannels []models.NotificationChannel
}

// NotificationHandler interface for different notification channels
type NotificationHandler interface {
	Send(ctx context.Context, req *models.NotificationRequest) (*models.NotificationResponse, error)
	Validate(req *models.NotificationRequest) error
	GetName() string
	IsEnabled() bool
	TestConnection(ctx context.Context) error
}

// NewNotificationService creates a new notification service
func NewNotificationService(
	notifRepo *repository.NotificationRepository,
	preferenceSvc *PreferenceService,
	templateSvc *TemplateService,
	batchSvc *BatchService,
	dlqSvc *DLQService,
	retrySvc *RetryService,
	config *ServiceConfig,
	logger *logrus.Logger,
) *NotificationService {
	service := &NotificationService{
		notifRepo:     notifRepo,
		preferenceSvc: preferenceSvc,
		templateSvc:   templateSvc,
		batchSvc:      batchSvc,
		dlqSvc:        dlqSvc,
		retrySvc:      retrySvc,
		handlers:      make(map[models.NotificationChannel]NotificationHandler),
		config:        config,
		logger:        logger,
	}

	return service
}

// RegisterHandler registers a notification handler
func (s *NotificationService) RegisterHandler(channel models.NotificationChannel, handler NotificationHandler) {
	s.handlers[channel] = handler
	s.logger.WithField("channel", channel).Info("Registered notification handler")
}

// SendNotification sends a notification through the optimal channel
func (s *NotificationService) SendNotification(ctx context.Context, req *models.NotificationRequest) (*models.NotificationResponse, error) {
	// Validate request
	if err := s.validateRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Check if user is subscribed to this event type
	subscribed, err := s.preferenceSvc.IsEventSubscribed(ctx, req.UserID, req.EventType)
	if err != nil {
		return nil, fmt.Errorf("failed to check event subscription: %w", err)
	}
	if !subscribed {
		s.logger.WithFields(logrus.Fields{
			"user_id":    req.UserID,
			"event_type": req.EventType,
		}).Debug("User not subscribed to event type")
		return &models.NotificationResponse{
			Status:  models.StatusFailed,
			Channel: req.Channel,
			Error:   "user not subscribed to event type",
		}, nil
	}

	// Get optimal channel if not specified
	if req.Channel == "" {
		channel, err := s.preferenceSvc.GetOptimalChannel(ctx, req.UserID, req.EventType)
		if err != nil {
			return nil, fmt.Errorf("failed to get optimal channel: %w", err)
		}
		req.Channel = channel
	}

	// Check if channel is enabled for user
	channelEnabled, err := s.preferenceSvc.IsChannelEnabled(ctx, req.UserID, req.Channel)
	if err != nil {
		return nil, fmt.Errorf("failed to check channel status: %w", err)
	}
	if !channelEnabled {
		s.logger.WithFields(logrus.Fields{
			"user_id": req.UserID,
			"channel": req.Channel,
		}).Debug("Channel not enabled for user")
		return &models.NotificationResponse{
			Status:  models.StatusFailed,
			Channel: req.Channel,
			Error:   "channel not enabled for user",
		}, nil
	}

	// Check quiet hours (unless bypassed)
	if !req.BypassQuietHours {
		inQuietHours, err := s.preferenceSvc.IsInQuietHours(ctx, req.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to check quiet hours: %w", err)
		}
		if inQuietHours {
			s.logger.WithFields(logrus.Fields{
				"user_id": req.UserID,
				"channel": req.Channel,
			}).Debug("User in quiet hours, skipping notification")
			return &models.NotificationResponse{
				Status:  models.StatusFailed,
				Channel: req.Channel,
				Error:   "user in quiet hours",
			}, nil
		}
	}

	// Apply template if not bypassed
	if !req.BypassBatching {
		templateData := s.templateSvc.CreateTemplateData(req, nil)
		req, err = s.templateSvc.RenderNotification(ctx, req, templateData)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to render notification template")
		}
	}

	// Check if notification should be batched
	if s.config.EnableBatching && !req.BypassBatching && !s.shouldBypassBatching(req.EventType) {
		return s.sendBatchedNotification(ctx, req)
	}

	// Send immediately
	return s.sendImmediateNotification(ctx, req)
}

// sendBatchedNotification sends a notification through batching
func (s *NotificationService) sendBatchedNotification(ctx context.Context, req *models.NotificationRequest) (*models.NotificationResponse, error) {
	// Add to batch
	if err := s.batchSvc.AddToBatch(ctx, req); err != nil {
		s.logger.WithError(err).WithField("user_id", req.UserID).Error("Failed to add notification to batch")
		return &models.NotificationResponse{
			Status:  models.StatusFailed,
			Channel: req.Channel,
			Error:   "failed to add to batch",
		}, err
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":    req.UserID,
		"event_type": req.EventType,
		"channel":    req.Channel,
	}).Debug("Notification added to batch")

	return &models.NotificationResponse{
		Status:  models.StatusSent,
		Channel: req.Channel,
	}, nil
}

// sendImmediateNotification sends a notification immediately
func (s *NotificationService) sendImmediateNotification(ctx context.Context, req *models.NotificationRequest) (*models.NotificationResponse, error) {
	// Create notification record
	notification := &models.Notification{
		UserID:    req.UserID,
		EventType: req.EventType,
		Channel:   req.Channel,
		Title:     req.Title,
		Message:   req.Message,
		Status:    models.StatusPending,
		Priority:  req.Priority,
		Metadata:  req.Metadata,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Store notification
	if err := s.notifRepo.Create(ctx, notification); err != nil {
		return nil, fmt.Errorf("failed to create notification: %w", err)
	}

	// Get handler for channel
	handler, exists := s.handlers[req.Channel]
	if !exists {
		return nil, fmt.Errorf("no handler found for channel: %s", req.Channel)
	}

	// Check if handler is enabled
	if !handler.IsEnabled() {
		return nil, fmt.Errorf("handler for channel %s is not enabled", req.Channel)
	}

	// Send notification
	response, err := handler.Send(ctx, req)
	if err != nil {
		// Update notification status to failed
		s.notifRepo.UpdateStatus(ctx, notification.ID.Hex(), models.StatusFailed, err.Error())

		// Add to DLQ if enabled
		if s.config.EnableDLQ {
			originalEvent := map[string]interface{}{
				"user_id":    req.UserID,
				"event_type": string(req.EventType),
				"channel":    string(req.Channel),
				"title":      req.Title,
				"message":    req.Message,
				"priority":   string(req.Priority),
				"metadata":   req.Metadata,
			}
			s.dlqSvc.AddToDLQ(ctx, notification, originalEvent, err.Error())
		}

		return response, err
	}

	// Update notification status
	if response.Status == models.StatusSent {
		s.notifRepo.UpdateStatus(ctx, notification.ID.Hex(), models.StatusSent, "")
	} else {
		s.notifRepo.UpdateStatus(ctx, notification.ID.Hex(), models.StatusFailed, response.Error)
	}

	return response, nil
}

// SendWithFallback sends a notification with fallback channels
func (s *NotificationService) SendWithFallback(ctx context.Context, req *models.NotificationRequest) (*models.NotificationResponse, error) {
	// Get fallback channels
	fallbackChannels, err := s.preferenceSvc.GetFallbackChannels(ctx, req.UserID, req.EventType)
	if err != nil {
		return nil, fmt.Errorf("failed to get fallback channels: %w", err)
	}

	// Try each channel in order
	for _, channel := range fallbackChannels {
		req.Channel = channel

		// Check if channel is enabled
		enabled, err := s.preferenceSvc.IsChannelEnabled(ctx, req.UserID, channel)
		if err != nil {
			s.logger.WithError(err).WithField("channel", channel).Warn("Failed to check channel status")
			continue
		}
		if !enabled {
			continue
		}

		// Try to send notification
		response, err := s.sendImmediateNotification(ctx, req)
		if err == nil && response.Status == models.StatusSent {
			s.logger.WithFields(logrus.Fields{
				"user_id":    req.UserID,
				"event_type": req.EventType,
				"channel":    channel,
			}).Info("Notification sent successfully with fallback")
			return response, nil
		}

		s.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":    req.UserID,
			"event_type": req.EventType,
			"channel":    channel,
		}).Warn("Failed to send notification via channel, trying next")
	}

	// All channels failed
	return &models.NotificationResponse{
		Status:  models.StatusFailed,
		Channel: req.Channel,
		Error:   "all channels failed",
	}, fmt.Errorf("all channels failed")
}

// ProcessKafkaEvent processes a Kafka event
func (s *NotificationService) ProcessKafkaEvent(ctx context.Context, event *models.KafkaFileEvent) error {
	// Convert Kafka event to notification request
	req := &models.NotificationRequest{
		UserID:    event.UserID,
		EventType: s.mapEventType(event.Type, event.Success),
		Title:     s.getEventTitle(event.Type, event.Success),
		Message:   s.getEventMessage(event),
		Priority:  s.getEventPriority(event.Type, event.Success),
		Metadata: map[string]interface{}{
			"file_id":      event.FileID,
			"file_name":    event.FileName,
			"file_size":    event.FileSize,
			"success":      event.Success,
			"error_reason": event.ErrorReason,
		},
	}

	// Send notification
	_, err := s.SendNotification(ctx, req)
	return err
}

// validateRequest validates a notification request
func (s *NotificationService) validateRequest(req *models.NotificationRequest) error {
	if req.UserID == "" {
		return fmt.Errorf("user ID is required")
	}
	if req.EventType == "" {
		return fmt.Errorf("event type is required")
	}
	if req.Title == "" {
		return fmt.Errorf("title is required")
	}
	if req.Message == "" {
		return fmt.Errorf("message is required")
	}
	return nil
}

// shouldBypassBatching checks if an event type should bypass batching
func (s *NotificationService) shouldBypassBatching(eventType models.EventType) bool {
	criticalTypes := []models.EventType{
		models.EventTypeQuotaExceeded,
		models.EventTypeSecurityAlert,
		models.EventTypeSystemMaintenance,
	}

	for _, criticalType := range criticalTypes {
		if eventType == criticalType {
			return true
		}
	}

	return false
}

// mapEventType maps Kafka event type to notification event type
func (s *NotificationService) mapEventType(eventType string, success bool) models.EventType {
	switch eventType {
	case "file.uploaded":
		if success {
			return models.EventTypeFileUploaded
		} else {
			return models.EventTypeFileUploadFailed
		}
	case "file.deleted":
		return models.EventTypeFileDeleted
	case "file.shared":
		return models.EventTypeFileShared
	default:
		return models.EventTypeFileUploaded
	}
}

// getEventTitle gets the title for an event
func (s *NotificationService) getEventTitle(eventType string, success bool) string {
	switch eventType {
	case "file.uploaded":
		if success {
			return "File Upload Complete"
		} else {
			return "File Upload Failed"
		}
	case "file.deleted":
		return "File Deleted"
	case "file.shared":
		return "File Shared"
	default:
		return "Notification"
	}
}

// getEventMessage gets the message for an event
func (s *NotificationService) getEventMessage(event *models.KafkaFileEvent) string {
	switch event.Type {
	case "file.uploaded":
		if event.Success {
			return fmt.Sprintf("Your file '%s' has been uploaded successfully", event.FileName)
		} else {
			return fmt.Sprintf("Failed to upload file '%s': %s", event.FileName, event.ErrorReason)
		}
	case "file.deleted":
		return fmt.Sprintf("Your file '%s' has been deleted", event.FileName)
	case "file.shared":
		return fmt.Sprintf("A file '%s' has been shared with you", event.FileName)
	default:
		return "You have a new notification"
	}
}

// getEventPriority gets the priority for an event
func (s *NotificationService) getEventPriority(eventType string, success bool) models.Priority {
	if !success {
		return models.PriorityHigh
	}

	switch eventType {
	case "file.uploaded":
		return models.PriorityNormal
	case "file.deleted":
		return models.PriorityNormal
	case "file.shared":
		return models.PriorityNormal
	default:
		return models.PriorityNormal
	}
}

// GetNotificationStats gets notification statistics
func (s *NotificationService) GetNotificationStats(ctx context.Context, userID string, startDate, endDate time.Time) (map[string]int64, error) {
	return s.notifRepo.GetNotificationStats(ctx, userID, startDate, endDate)
}

// GetServiceHealth gets the health status of the service
func (s *NotificationService) GetServiceHealth(ctx context.Context) (map[string]interface{}, error) {
	health := map[string]interface{}{
		"service":   "notification-service",
		"status":    "healthy",
		"timestamp": time.Now(),
		"config": map[string]interface{}{
			"batching_enabled": s.config.EnableBatching,
			"retry_enabled":    s.config.EnableRetry,
			"dlq_enabled":      s.config.EnableDLQ,
			"default_channel":  s.config.DefaultChannel,
		},
		"handlers": make(map[string]interface{}),
	}

	// Check handler health
	for channel, handler := range s.handlers {
		handlerHealth := map[string]interface{}{
			"enabled": handler.IsEnabled(),
		}

		// Test connection
		if err := handler.TestConnection(ctx); err != nil {
			handlerHealth["error"] = err.Error()
		} else {
			handlerHealth["status"] = "healthy"
		}

		health["handlers"].(map[string]interface{})[string(channel)] = handlerHealth
	}

	return health, nil
}

// StartBackgroundProcesses starts all background processes
func (s *NotificationService) StartBackgroundProcesses(ctx context.Context) {
	// Start batch processor
	if s.config.EnableBatching && s.batchSvc != nil {
		go s.batchSvc.StartBatchProcessor(ctx)
	}

	// Start retry processor
	if s.config.EnableRetry && s.retrySvc != nil {
		go s.retrySvc.StartRetryProcessor(ctx)
	}

	// Start DLQ processor
	if s.config.EnableDLQ && s.dlqSvc != nil {
		go s.dlqSvc.StartDLQProcessor(ctx)
	}

	s.logger.Info("Background processes started")
}

// GetNotifications gets notifications for a user with pagination and filtering
func (s *NotificationService) GetNotifications(ctx context.Context, userID string, page, limit int, statusFilter *models.NotificationStatus, eventTypeFilter *models.EventType) ([]*models.Notification, int64, error) {
	return s.notifRepo.GetByUserID(ctx, userID, page, limit, statusFilter, eventTypeFilter)
}

// GetNotification gets a specific notification by ID
func (s *NotificationService) GetNotification(ctx context.Context, notificationID, userID string) (*models.Notification, error) {
	return s.notifRepo.GetByIDAndUserID(ctx, notificationID, userID)
}

// MarkAsRead marks a notification as read
func (s *NotificationService) MarkAsRead(ctx context.Context, notificationID, userID string) error {
	return s.notifRepo.MarkAsRead(ctx, notificationID, userID)
}

// MarkAllAsRead marks all notifications as read for a user
func (s *NotificationService) MarkAllAsRead(ctx context.Context, userID string) (int64, error) {
	return s.notifRepo.MarkAllAsRead(ctx, userID)
}

// DeleteNotification deletes a notification
func (s *NotificationService) DeleteNotification(ctx context.Context, notificationID, userID string) error {
	return s.notifRepo.DeleteByIDAndUserID(ctx, notificationID, userID)
}

// GetUnreadCount gets the unread notification count for a user
func (s *NotificationService) GetUnreadCount(ctx context.Context, userID string) (int64, error) {
	return s.notifRepo.GetUnreadCount(ctx, userID)
}
