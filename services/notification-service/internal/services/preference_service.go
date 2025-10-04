package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/models"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/repository"
	"github.com/sirupsen/logrus"
)

// PreferenceService handles user notification preferences
type PreferenceService struct {
	preferencesRepo *repository.PreferencesRepository
	logger          *logrus.Logger
}

// NewPreferenceService creates a new preference service
func NewPreferenceService(preferencesRepo *repository.PreferencesRepository, logger *logrus.Logger) *PreferenceService {
	return &PreferenceService{
		preferencesRepo: preferencesRepo,
		logger:          logger,
	}
}

// GetUserPreferences gets user notification preferences
func (s *PreferenceService) GetUserPreferences(ctx context.Context, userID string) (*models.UserNotificationPreferences, error) {
	preferences, err := s.preferencesRepo.GetByUserID(ctx, userID)
	if err != nil {
		if err == repository.ErrPreferencesNotFound {
			// Return default preferences
			defaultPrefs := s.preferencesRepo.GetDefaultPreferences(userID)
			s.logger.WithField("user_id", userID).Info("Using default preferences for new user")
			return defaultPrefs, nil
		}
		return nil, fmt.Errorf("failed to get user preferences: %w", err)
	}

	return preferences, nil
}

// UpdateUserPreferences updates user notification preferences
func (s *PreferenceService) UpdateUserPreferences(ctx context.Context, userID string, preferences *models.UserNotificationPreferences) error {
	preferences.UserID = userID
	preferences.UpdatedAt = time.Now()

	// Validate preferences
	if err := s.validatePreferences(preferences); err != nil {
		return fmt.Errorf("invalid preferences: %w", err)
	}

	// Upsert preferences
	if err := s.preferencesRepo.Upsert(ctx, preferences); err != nil {
		return fmt.Errorf("failed to update user preferences: %w", err)
	}

	s.logger.WithField("user_id", userID).Info("User preferences updated successfully")
	return nil
}

// IsChannelEnabled checks if a channel is enabled for a user
func (s *PreferenceService) IsChannelEnabled(ctx context.Context, userID string, channel models.NotificationChannel) (bool, error) {
	return s.preferencesRepo.IsChannelEnabled(ctx, userID, channel)
}

// IsEventSubscribed checks if a user is subscribed to an event type
func (s *PreferenceService) IsEventSubscribed(ctx context.Context, userID string, eventType models.EventType) (bool, error) {
	return s.preferencesRepo.IsEventSubscribed(ctx, userID, eventType)
}

// GetChannelPriorities gets channel priorities for a user and event type
func (s *PreferenceService) GetChannelPriorities(ctx context.Context, userID string, eventType models.EventType) ([]models.NotificationChannel, error) {
	return s.preferencesRepo.GetChannelPriorities(ctx, userID, eventType)
}

// IsInQuietHours checks if a user is currently in quiet hours
func (s *PreferenceService) IsInQuietHours(ctx context.Context, userID string) (bool, error) {
	preferences, err := s.GetUserPreferences(ctx, userID)
	if err != nil {
		return false, err
	}

	if !preferences.QuietHoursEnabled {
		return false, nil
	}

	now := time.Now()
	currentTime := now.Format("15:04")

	// Check if current time is within quiet hours
	return s.isTimeInQuietHours(currentTime, preferences.QuietHoursStart, preferences.QuietHoursEnd), nil
}

// isTimeInQuietHours checks if current time is within quiet hours
func (s *PreferenceService) isTimeInQuietHours(currentTime, startTime, endTime string) bool {
	if startTime == "" || endTime == "" {
		return false
	}

	// Parse times
	current, err := time.Parse("15:04", currentTime)
	if err != nil {
		return false
	}

	start, err := time.Parse("15:04", startTime)
	if err != nil {
		return false
	}

	end, err := time.Parse("15:04", endTime)
	if err != nil {
		return false
	}

	// Check if quiet hours cross midnight
	if start.After(end) {
		// Quiet hours cross midnight (e.g., 22:00 to 08:00)
		return current.After(start) || current.Before(end) || current.Equal(start) || current.Equal(end)
	} else {
		// Quiet hours don't cross midnight (e.g., 22:00 to 23:00)
		return (current.After(start) || current.Equal(start)) && (current.Before(end) || current.Equal(end))
	}
}

// ShouldSendNotification checks if a notification should be sent based on user preferences
func (s *PreferenceService) ShouldSendNotification(ctx context.Context, userID string, eventType models.EventType, channel models.NotificationChannel, bypassQuietHours bool) (bool, error) {
	// Check if user is subscribed to this event type
	subscribed, err := s.IsEventSubscribed(ctx, userID, eventType)
	if err != nil {
		return false, fmt.Errorf("failed to check event subscription: %w", err)
	}
	if !subscribed {
		s.logger.WithFields(logrus.Fields{
			"user_id":    userID,
			"event_type": eventType,
		}).Debug("User not subscribed to event type")
		return false, nil
	}

	// Check if channel is enabled for user
	channelEnabled, err := s.IsChannelEnabled(ctx, userID, channel)
	if err != nil {
		return false, fmt.Errorf("failed to check channel status: %w", err)
	}
	if !channelEnabled {
		s.logger.WithFields(logrus.Fields{
			"user_id": userID,
			"channel": channel,
		}).Debug("Channel not enabled for user")
		return false, nil
	}

	// Check quiet hours (unless bypassed)
	if !bypassQuietHours {
		inQuietHours, err := s.IsInQuietHours(ctx, userID)
		if err != nil {
			return false, fmt.Errorf("failed to check quiet hours: %w", err)
		}
		if inQuietHours {
			s.logger.WithFields(logrus.Fields{
				"user_id": userID,
				"channel": channel,
			}).Debug("User in quiet hours, skipping notification")
			return false, nil
		}
	}

	return true, nil
}

// GetOptimalChannel gets the optimal channel for sending a notification to a user
func (s *PreferenceService) GetOptimalChannel(ctx context.Context, userID string, eventType models.EventType) (models.NotificationChannel, error) {
	priorities, err := s.GetChannelPriorities(ctx, userID, eventType)
	if err != nil {
		return "", fmt.Errorf("failed to get channel priorities: %w", err)
	}

	// Find the first enabled channel in priority order
	for _, channel := range priorities {
		enabled, err := s.IsChannelEnabled(ctx, userID, channel)
		if err != nil {
			s.logger.WithError(err).WithFields(logrus.Fields{
				"user_id": userID,
				"channel": channel,
			}).Warn("Failed to check channel status")
			continue
		}
		if enabled {
			return channel, nil
		}
	}

	return "", fmt.Errorf("no enabled channels found for user %s and event type %s", userID, eventType)
}

// GetFallbackChannels gets fallback channels for a user and event type
func (s *PreferenceService) GetFallbackChannels(ctx context.Context, userID string, eventType models.EventType) ([]models.NotificationChannel, error) {
	priorities, err := s.GetChannelPriorities(ctx, userID, eventType)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel priorities: %w", err)
	}

	var fallbackChannels []models.NotificationChannel
	for _, channel := range priorities {
		enabled, err := s.IsChannelEnabled(ctx, userID, channel)
		if err != nil {
			s.logger.WithError(err).WithFields(logrus.Fields{
				"user_id": userID,
				"channel": channel,
			}).Warn("Failed to check channel status")
			continue
		}
		if enabled {
			fallbackChannels = append(fallbackChannels, channel)
		}
	}

	return fallbackChannels, nil
}

// validatePreferences validates user preferences
func (s *PreferenceService) validatePreferences(preferences *models.UserNotificationPreferences) error {
	// Validate email format if provided
	if preferences.Email != "" && !s.isValidEmail(preferences.Email) {
		return fmt.Errorf("invalid email format: %s", preferences.Email)
	}

	// Validate phone number format if provided
	if preferences.PhoneNumber != "" && !s.isValidPhoneNumber(preferences.PhoneNumber) {
		return fmt.Errorf("invalid phone number format: %s", preferences.PhoneNumber)
	}

	// Validate quiet hours format
	if preferences.QuietHoursEnabled {
		if preferences.QuietHoursStart != "" && !s.isValidTimeFormat(preferences.QuietHoursStart) {
			return fmt.Errorf("invalid quiet hours start format: %s", preferences.QuietHoursStart)
		}
		if preferences.QuietHoursEnd != "" && !s.isValidTimeFormat(preferences.QuietHoursEnd) {
			return fmt.Errorf("invalid quiet hours end format: %s", preferences.QuietHoursEnd)
		}
	}

	// Validate event subscriptions
	for _, eventType := range preferences.EventSubscriptions {
		if !s.isValidEventType(eventType) {
			return fmt.Errorf("invalid event type: %s", eventType)
		}
	}

	return nil
}

// isValidEmail validates email format
func (s *PreferenceService) isValidEmail(email string) bool {
	// Simple email validation
	return strings.Contains(email, "@") && strings.Contains(email, ".")
}

// isValidPhoneNumber validates phone number format
func (s *PreferenceService) isValidPhoneNumber(phone string) bool {
	// Simple phone validation - should start with + and contain only digits and +
	phone = strings.TrimSpace(phone)
	if !strings.HasPrefix(phone, "+") {
		return false
	}
	
	// Remove + and check if remaining characters are digits
	digits := phone[1:]
	for _, char := range digits {
		if char < '0' || char > '9' {
			return false
		}
	}
	
	return len(digits) >= 10 // Minimum 10 digits
}

// isValidTimeFormat validates time format (HH:MM)
func (s *PreferenceService) isValidTimeFormat(timeStr string) bool {
	_, err := time.Parse("15:04", timeStr)
	return err == nil
}

// isValidEventType validates event type
func (s *PreferenceService) isValidEventType(eventType models.EventType) bool {
	validTypes := []models.EventType{
		models.EventTypeFileUploaded,
		models.EventTypeFileUploadFailed,
		models.EventTypeFileDeleted,
		models.EventTypeFileShared,
		models.EventTypeQuotaWarning80,
		models.EventTypeQuotaWarning90,
		models.EventTypeQuotaExceeded,
		models.EventTypeSecurityAlert,
		models.EventTypeSystemMaintenance,
	}

	for _, validType := range validTypes {
		if eventType == validType {
			return true
		}
	}
	return false
}

// GetUsersInQuietHours gets users who are currently in quiet hours
func (s *PreferenceService) GetUsersInQuietHours(ctx context.Context) ([]string, error) {
	return s.preferencesRepo.GetUsersInQuietHours(ctx)
}

// GetUsersByEventType gets users who are subscribed to a specific event type
func (s *PreferenceService) GetUsersByEventType(ctx context.Context, eventType models.EventType) ([]string, error) {
	return s.preferencesRepo.GetUsersByEventType(ctx, eventType)
}

// GetUsersByChannel gets users who have a specific channel enabled
func (s *PreferenceService) GetUsersByChannel(ctx context.Context, channel models.NotificationChannel) ([]string, error) {
	return s.preferencesRepo.GetUsersByChannel(ctx, channel)
}

// ResetToDefaults resets user preferences to default values
func (s *PreferenceService) ResetToDefaults(ctx context.Context, userID string) error {
	defaultPrefs := s.preferencesRepo.GetDefaultPreferences(userID)
	return s.UpdateUserPreferences(ctx, userID, defaultPrefs)
}

// GetPreferenceStats gets preference statistics
func (s *PreferenceService) GetPreferenceStats(ctx context.Context) (map[string]int64, error) {
	// This would require additional repository methods to get statistics
	// For now, return empty stats
	return map[string]int64{}, nil
}

// GetDefaultPreferences returns default preferences for a user
func (s *PreferenceService) GetDefaultPreferences(userID string) *models.UserNotificationPreferences {
	return s.preferencesRepo.GetDefaultPreferences(userID)
}