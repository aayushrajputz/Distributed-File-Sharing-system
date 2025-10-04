package service

import (
	"context"
	"fmt"
	"time"

	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/channels"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/models"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/repository"
	"github.com/sirupsen/logrus"
)

// NotificationService handles multi-channel notification logic
type NotificationService struct {
	notifRepo        *repository.NotificationRepository
	preferencesRepo  *repository.UserPreferencesRepository
	channelManager   channels.ChannelManager
	logger           *logrus.Logger
}

// NewNotificationService creates a new notification service
func NewNotificationService(
	notifRepo *repository.NotificationRepository,
	preferencesRepo *repository.UserPreferencesRepository,
	channelManager channels.ChannelManager,
	logger *logrus.Logger,
) *NotificationService {
	return &NotificationService{
		notifRepo:       notifRepo,
		preferencesRepo: preferencesRepo,
		channelManager:  channelManager,
		logger:          logger,
	}
}

// SendMultiChannelNotification sends notification via multiple channels
func (s *NotificationService) SendMultiChannelNotification(ctx context.Context, req *channels.NotificationRequest) ([]*channels.DeliveryResult, error) {
	// Get user preferences
	preferences, err := s.preferencesRepo.GetByUserID(ctx, req.UserID)
	if err != nil {
		if err == repository.ErrUserPreferencesNotFound {
			// Create default preferences
			preferences = s.preferencesRepo.GetDefaultPreferences(req.UserID)
			if err := s.preferencesRepo.Create(ctx, preferences); err != nil {
				s.logger.WithError(err).Error("Failed to create default user preferences")
			}
		} else {
			return nil, fmt.Errorf("failed to get user preferences: %w", err)
		}
	}

	// Filter channels based on user preferences
	enabledChannels := s.filterChannelsByPreferences(req.Channels, preferences)
	if len(enabledChannels) == 0 {
		return nil, fmt.Errorf("no enabled channels for user %s", req.UserID)
	}

	// Update request with user contact information
	s.enrichRequestWithUserData(req, preferences)

	// Send via channels
	results, err := s.channelManager.SendMultiChannel(ctx, req)
	if err != nil {
		s.logger.WithError(err).WithField("user_id", req.UserID).Error("Failed to send multi-channel notification")
		return results, err
	}

	// Store notification in database
	notification := s.createNotificationFromRequest(req, results)
	if err := s.notifRepo.Create(ctx, notification); err != nil {
		s.logger.WithError(err).Error("Failed to store notification in database")
		// Don't return error as notification was sent successfully
	}

	return results, nil
}

// SendWithFallback sends notification with fallback mechanism
func (s *NotificationService) SendWithFallback(ctx context.Context, req *channels.NotificationRequest) ([]*channels.DeliveryResult, error) {
	// Get user preferences
	preferences, err := s.preferencesRepo.GetByUserID(ctx, req.UserID)
	if err != nil {
		if err == repository.ErrUserPreferencesNotFound {
			preferences = s.preferencesRepo.GetDefaultPreferences(req.UserID)
			if err := s.preferencesRepo.Create(ctx, preferences); err != nil {
				s.logger.WithError(err).Error("Failed to create default user preferences")
			}
		} else {
			return nil, fmt.Errorf("failed to get user preferences: %w", err)
		}
	}

	// Filter channels based on user preferences
	enabledChannels := s.filterChannelsByPreferences(req.Channels, preferences)
	if len(enabledChannels) == 0 {
		return nil, fmt.Errorf("no enabled channels for user %s", req.UserID)
	}

	// Update request with user contact information
	s.enrichRequestWithUserData(req, preferences)

	// Send with fallback
	results, err := s.channelManager.SendWithFallback(ctx, req)
	if err != nil {
		s.logger.WithError(err).WithField("user_id", req.UserID).Error("Failed to send notification with fallback")
		return results, err
	}

	// Store notification in database
	notification := s.createNotificationFromRequest(req, results)
	if err := s.notifRepo.Create(ctx, notification); err != nil {
		s.logger.WithError(err).Error("Failed to store notification in database")
	}

	return results, nil
}

// UpdateUserPreferences updates user notification preferences
func (s *NotificationService) UpdateUserPreferences(ctx context.Context, userID string, preferences *channels.UserPreferences) error {
	preferences.UserID = userID
	preferences.UpdatedAt = time.Now()

	if err := s.preferencesRepo.Upsert(ctx, preferences); err != nil {
		return fmt.Errorf("failed to update user preferences: %w", err)
	}

	s.logger.WithField("user_id", userID).Info("User preferences updated")
	return nil
}

// GetUserPreferences gets user notification preferences
func (s *NotificationService) GetUserPreferences(ctx context.Context, userID string) (*channels.UserPreferences, error) {
	preferences, err := s.preferencesRepo.GetByUserID(ctx, userID)
	if err != nil {
		if err == repository.ErrUserPreferencesNotFound {
			// Return default preferences
			return s.preferencesRepo.GetDefaultPreferences(userID), nil
		}
		return nil, fmt.Errorf("failed to get user preferences: %w", err)
	}

	return preferences, nil
}

// filterChannelsByPreferences filters channels based on user preferences
func (s *NotificationService) filterChannelsByPreferences(requestedChannels []string, preferences *channels.UserPreferences) []string {
	var enabledChannels []string

	for _, channel := range requestedChannels {
		// Check if channel is enabled in user preferences
		if enabled, exists := preferences.ChannelSettings[channel]; exists && enabled {
			enabledChannels = append(enabledChannels, channel)
		} else if !exists {
			// If not in preferences, check if it's in enabled channels list
			for _, enabledChannel := range preferences.EnabledChannels {
				if enabledChannel == channel {
					enabledChannels = append(enabledChannels, channel)
					break
				}
			}
		}
	}

	return enabledChannels
}

// enrichRequestWithUserData enriches the request with user contact information
func (s *NotificationService) enrichRequestWithUserData(req *channels.NotificationRequest, preferences *channels.UserPreferences) {
	if req.Email == "" && preferences.Email != "" {
		req.Email = preferences.Email
	}
	if req.Phone == "" && preferences.Phone != "" {
		req.Phone = preferences.Phone
	}
	if req.PushToken == "" && preferences.PushToken != "" {
		req.PushToken = preferences.PushToken
	}
}

// createNotificationFromRequest creates a notification model from request
func (s *NotificationService) createNotificationFromRequest(req *channels.NotificationRequest, results []*channels.DeliveryResult) *models.Notification {
	// Convert delivery results
	deliveryResults := make([]models.DeliveryResult, len(results))
	for i, result := range results {
		deliveryResults[i] = models.DeliveryResult{
			Channel:      result.Channel,
			Success:      result.Success,
			ErrorMessage: result.ErrorMessage,
			DeliveredAt:  result.DeliveredAt,
			MessageID:    result.MessageID,
		}
	}

	// Convert map[string]string to map[string]interface{}
	metadata := make(map[string]interface{})
	for k, v := range req.Metadata {
		metadata[k] = v
	}

	return &models.Notification{
		UserID:     req.UserID,
		EventType:  models.EventType(req.Type),
		Channel:    models.NotificationChannel(req.Channels[0]), // Use first channel as primary
		Title:      req.Title,
		Message:    req.Body,
		Priority:   models.Priority(req.Priority),
		Metadata:   metadata,
		Status:     models.StatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

// GetChannelStatus returns the status of all notification channels
func (s *NotificationService) GetChannelStatus() map[string]bool {
	return s.channelManager.GetChannelStatus()
}

// ValidateChannels validates that all specified channels are available
func (s *NotificationService) ValidateChannels(channelNames []string) error {
	return s.channelManager.ValidateChannels(channelNames)
}
