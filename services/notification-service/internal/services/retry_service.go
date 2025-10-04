package services

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/models"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/repository"
	"github.com/sirupsen/logrus"
)

// RetryService handles retry logic with exponential backoff
type RetryService struct {
	notifRepo *repository.NotificationRepository
	dlqSvc    *DLQService
	config    *RetryConfig
	logger    *logrus.Logger
}

// RetryConfig contains retry service configuration
type RetryConfig struct {
	MaxRetries      int
	BaseDelay       time.Duration
	MaxDelay        time.Duration
	Multiplier      float64
	Jitter          bool
	RetryInterval   time.Duration
	BatchSize       int
}

// NewRetryService creates a new retry service
func NewRetryService(
	notifRepo *repository.NotificationRepository,
	dlqSvc *DLQService,
	config *RetryConfig,
	logger *logrus.Logger,
) *RetryService {
	return &RetryService{
		notifRepo: notifRepo,
		dlqSvc:    dlqSvc,
		config:    config,
		logger:    logger,
	}
}

// RetryNotification retries a failed notification
func (s *RetryService) RetryNotification(ctx context.Context, notification *models.Notification, retryFunc func(context.Context, *models.Notification) error) error {
	// Check if notification has exceeded max retries
	if notification.RetryCount >= s.config.MaxRetries {
		s.logger.WithFields(logrus.Fields{
			"notification_id": notification.ID.Hex(),
			"user_id":         notification.UserID,
			"retry_count":     notification.RetryCount,
			"max_retries":     s.config.MaxRetries,
		}).Warn("Notification exceeded max retries, moving to DLQ")
		
		// Move to DLQ
		return s.moveToDLQ(ctx, notification, "exceeded max retries")
	}

	// Calculate retry delay
	delay := s.calculateRetryDelay(notification.RetryCount)
	
	// Update retry info
	nextRetryAt := time.Now().Add(delay)
	if err := s.notifRepo.UpdateRetryInfo(ctx, notification.ID.Hex(), notification.RetryCount+1, &nextRetryAt, ""); err != nil {
		return fmt.Errorf("failed to update retry info: %w", err)
	}

	// Wait for retry delay
	time.Sleep(delay)

	// Attempt retry
	start := time.Now()
	err := retryFunc(ctx, notification)
	duration := time.Since(start)

	// Create delivery attempt
	attempt := models.DeliveryAttempt{
		AttemptedAt: time.Now(),
		Success:     err == nil,
		Duration:    duration.Milliseconds(),
	}

	if err != nil {
		attempt.ErrorReason = err.Error()
		s.logger.WithError(err).WithFields(logrus.Fields{
			"notification_id": notification.ID.Hex(),
			"user_id":         notification.UserID,
			"retry_count":     notification.RetryCount + 1,
			"duration_ms":     duration.Milliseconds(),
		}).Warn("Notification retry failed")
	} else {
		s.logger.WithFields(logrus.Fields{
			"notification_id": notification.ID.Hex(),
			"user_id":         notification.UserID,
			"retry_count":     notification.RetryCount + 1,
			"duration_ms":     duration.Milliseconds(),
		}).Info("Notification retry successful")
	}

	// Add delivery attempt
	if err := s.notifRepo.AddDeliveryAttempt(ctx, notification.ID.Hex(), &attempt); err != nil {
		s.logger.WithError(err).WithField("notification_id", notification.ID.Hex()).Error("Failed to add delivery attempt")
	}

	// Update notification status
	if err != nil {
		// Retry failed, update status
		if err := s.notifRepo.UpdateStatus(ctx, notification.ID.Hex(), models.StatusFailed, err.Error()); err != nil {
			s.logger.WithError(err).WithField("notification_id", notification.ID.Hex()).Error("Failed to update notification status")
		}
		
		// Check if we should move to DLQ
		if notification.RetryCount+1 >= s.config.MaxRetries {
			return s.moveToDLQ(ctx, notification, err.Error())
		}
		
		return err
	} else {
		// Retry successful, update status
		if err := s.notifRepo.UpdateStatus(ctx, notification.ID.Hex(), models.StatusSent, ""); err != nil {
			s.logger.WithError(err).WithField("notification_id", notification.ID.Hex()).Error("Failed to update notification status")
		}
		return nil
	}
}

// ProcessRetries processes all notifications that are ready for retry
func (s *RetryService) ProcessRetries(ctx context.Context) error {
	// Get notifications ready for retry
	notifications, err := s.notifRepo.GetPendingRetries(ctx, s.config.BatchSize)
	if err != nil {
		return fmt.Errorf("failed to get pending retries: %w", err)
	}

	for _, notification := range notifications {
		// This would integrate with the main notification service
		// For now, we'll just log the retry attempt
		s.logger.WithFields(logrus.Fields{
			"notification_id": notification.ID.Hex(),
			"user_id":         notification.UserID,
			"event_type":      notification.EventType,
			"channel":         notification.Channel,
			"retry_count":     notification.RetryCount,
		}).Info("Processing notification retry")
	}

	return nil
}

// calculateRetryDelay calculates the delay for the next retry using exponential backoff
func (s *RetryService) calculateRetryDelay(retryCount int) time.Duration {
	// Calculate exponential delay
	delay := float64(s.config.BaseDelay) * math.Pow(s.config.Multiplier, float64(retryCount))
	
	// Apply maximum delay limit
	if delay > float64(s.config.MaxDelay) {
		delay = float64(s.config.MaxDelay)
	}
	
	// Convert to duration
	retryDelay := time.Duration(delay)
	
	// Add jitter if enabled
	if s.config.Jitter {
		// Add random jitter up to 25% of the delay
		jitter := time.Duration(float64(retryDelay) * 0.25 * (0.5 - 0.5*math.Sin(float64(time.Now().UnixNano()))))
		retryDelay += jitter
	}
	
	return retryDelay
}

// moveToDLQ moves a notification to the Dead Letter Queue
func (s *RetryService) moveToDLQ(ctx context.Context, notification *models.Notification, reason string) error {
	// Create original event data
	originalEvent := map[string]interface{}{
		"user_id":    notification.UserID,
		"event_type": string(notification.EventType),
		"channel":    string(notification.Channel),
		"title":      notification.Title,
		"message":    notification.Message,
		"priority":   string(notification.Priority),
		"metadata":   notification.Metadata,
		"created_at": notification.CreatedAt,
	}

	// Add to DLQ
	return s.dlqSvc.AddToDLQ(ctx, notification, originalEvent, reason)
}

// GetRetryStats gets retry statistics
func (s *RetryService) GetRetryStats(ctx context.Context) (map[string]int64, error) {
	// Get failed notifications
	failed, err := s.notifRepo.GetFailedNotifications(ctx, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to get failed notifications: %w", err)
	}

	// Get pending retries
	pending, err := s.notifRepo.GetPendingRetries(ctx, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending retries: %w", err)
	}

	// Calculate statistics
	stats := map[string]int64{
		"failed_notifications": int64(len(failed)),
		"pending_retries":      int64(len(pending)),
		"max_retries":          int64(s.config.MaxRetries),
	}

	// Calculate retry count distribution
	retryCounts := make(map[int]int64)
	for _, notif := range failed {
		retryCounts[notif.RetryCount]++
	}
	for _, notif := range pending {
		retryCounts[notif.RetryCount]++
	}

	// Add retry count distribution to stats
	for count, num := range retryCounts {
		stats[fmt.Sprintf("retry_count_%d", count)] = num
	}

	return stats, nil
}

// GetRetryHealth checks the health of the retry system
func (s *RetryService) GetRetryHealth(ctx context.Context) (map[string]interface{}, error) {
	stats, err := s.GetRetryStats(ctx)
	if err != nil {
		return nil, err
	}

	health := map[string]interface{}{
		"failed_notifications": stats["failed_notifications"],
		"pending_retries":      stats["pending_retries"],
		"max_retries":          stats["max_retries"],
		"base_delay":           s.config.BaseDelay.String(),
		"max_delay":            s.config.MaxDelay.String(),
		"multiplier":           s.config.Multiplier,
		"jitter_enabled":       s.config.Jitter,
		"retry_interval":       s.config.RetryInterval.String(),
		"batch_size":           s.config.BatchSize,
	}

	return health, nil
}

// StartRetryProcessor starts the retry processor goroutine
func (s *RetryService) StartRetryProcessor(ctx context.Context) {
	ticker := time.NewTicker(s.config.RetryInterval)
	defer ticker.Stop()

	s.logger.Info("Retry processor started")

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Retry processor stopped")
			return
		case <-ticker.C:
			if err := s.ProcessRetries(ctx); err != nil {
				s.logger.WithError(err).Error("Failed to process retries")
			}
		}
	}
}

// RetryNotificationImmediately retries a notification immediately without delay
func (s *RetryService) RetryNotificationImmediately(ctx context.Context, notification *models.Notification, retryFunc func(context.Context, *models.Notification) error) error {
	// Check if notification has exceeded max retries
	if notification.RetryCount >= s.config.MaxRetries {
		return s.moveToDLQ(ctx, notification, "exceeded max retries")
	}

	// Attempt retry immediately
	start := time.Now()
	err := retryFunc(ctx, notification)
	duration := time.Since(start)

	// Create delivery attempt
	attempt := models.DeliveryAttempt{
		AttemptedAt: time.Now(),
		Success:     err == nil,
		Duration:    duration.Milliseconds(),
	}

	if err != nil {
		attempt.ErrorReason = err.Error()
	}

	// Add delivery attempt
	if err := s.notifRepo.AddDeliveryAttempt(ctx, notification.ID.Hex(), &attempt); err != nil {
		s.logger.WithError(err).WithField("notification_id", notification.ID.Hex()).Error("Failed to add delivery attempt")
	}

	// Update retry count
	if err := s.notifRepo.UpdateRetryInfo(ctx, notification.ID.Hex(), notification.RetryCount+1, nil, ""); err != nil {
		s.logger.WithError(err).WithField("notification_id", notification.ID.Hex()).Error("Failed to update retry info")
	}

	// Update notification status
	if err != nil {
		if err := s.notifRepo.UpdateStatus(ctx, notification.ID.Hex(), models.StatusFailed, err.Error()); err != nil {
			s.logger.WithError(err).WithField("notification_id", notification.ID.Hex()).Error("Failed to update notification status")
		}
		return err
	} else {
		if err := s.notifRepo.UpdateStatus(ctx, notification.ID.Hex(), models.StatusSent, ""); err != nil {
			s.logger.WithError(err).WithField("notification_id", notification.ID.Hex()).Error("Failed to update notification status")
		}
		return nil
	}
}

// GetRetryDelayForAttempt calculates the retry delay for a specific attempt
func (s *RetryService) GetRetryDelayForAttempt(attempt int) time.Duration {
	return s.calculateRetryDelay(attempt)
}

// GetNextRetryTime calculates when the next retry should occur
func (s *RetryService) GetNextRetryTime(retryCount int) time.Time {
	delay := s.calculateRetryDelay(retryCount)
	return time.Now().Add(delay)
}

// IsRetryable checks if a notification should be retried
func (s *RetryService) IsRetryable(notification *models.Notification) bool {
	// Check if notification has exceeded max retries
	if notification.RetryCount >= s.config.MaxRetries {
		return false
	}

	// Check if notification is in failed status
	if notification.Status != models.StatusFailed {
		return false
	}

	// Check if next retry time has passed
	if notification.NextRetryAt != nil && notification.NextRetryAt.After(time.Now()) {
		return false
	}

	return true
}

// GetRetryableNotifications gets notifications that are ready for retry
func (s *RetryService) GetRetryableNotifications(ctx context.Context, limit int) ([]*models.Notification, error) {
	return s.notifRepo.GetPendingRetries(ctx, limit)
}
