package services

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/models"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/repository"
)

// DLQService handles Dead Letter Queue processing
type DLQService struct {
	dlqRepo       *repository.DLQRepository
	notifRepo     *repository.NotificationRepository
	preferenceSvc *PreferenceService
	templateSvc   *TemplateService
	config        *DLQConfig
	logger        *logrus.Logger
}

// DLQConfig contains DLQ service configuration
type DLQConfig struct {
	MaxRetries      int
	RetryInterval   time.Duration
	CleanupInterval time.Duration
	BatchSize       int
}

// NewDLQService creates a new DLQ service
func NewDLQService(
	dlqRepo *repository.DLQRepository,
	notifRepo *repository.NotificationRepository,
	preferenceSvc *PreferenceService,
	templateSvc *TemplateService,
	config *DLQConfig,
	logger *logrus.Logger,
) *DLQService {
	return &DLQService{
		dlqRepo:       dlqRepo,
		notifRepo:     notifRepo,
		preferenceSvc: preferenceSvc,
		templateSvc:   templateSvc,
		config:        config,
		logger:        logger,
	}
}

// AddToDLQ adds a failed notification to the Dead Letter Queue
func (s *DLQService) AddToDLQ(ctx context.Context, notification *models.Notification, originalEvent map[string]interface{}, failureReason string) error {
	// Create DLQ entry
	entry := &models.DeadLetterQueueEntry{
		OriginalEvent:  originalEvent,
		NotificationID: notification.ID,
		UserID:         notification.UserID,
		EventType:      notification.EventType,
		FailureReason:  failureReason,
		RetryHistory:   make([]models.RetryAttempt, 0),
		MaxRetries:     s.config.MaxRetries,
		CreatedAt:      time.Now(),
		IsProcessed:    false,
	}

	// Add to DLQ
	if err := s.dlqRepo.Create(ctx, entry); err != nil {
		return fmt.Errorf("failed to add notification to DLQ: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"notification_id": notification.ID.Hex(),
		"user_id":         notification.UserID,
		"event_type":      notification.EventType,
		"failure_reason":  failureReason,
	}).Warn("Notification added to DLQ")

	return nil
}

// ProcessDLQ processes entries in the Dead Letter Queue
func (s *DLQService) ProcessDLQ(ctx context.Context) error {
	// Get entries ready for retry
	entries, err := s.dlqRepo.GetReadyForRetry(ctx, s.config.BatchSize)
	if err != nil {
		return fmt.Errorf("failed to get DLQ entries: %w", err)
	}

	for _, entry := range entries {
		if err := s.processDLQEntry(ctx, entry); err != nil {
			s.logger.WithError(err).WithField("dlq_id", entry.ID.Hex()).Error("Failed to process DLQ entry")
			continue
		}
	}

	return nil
}

// processDLQEntry processes a single DLQ entry
func (s *DLQService) processDLQEntry(ctx context.Context, entry *models.DeadLetterQueueEntry) error {
	// Check if entry has exceeded max retries
	if len(entry.RetryHistory) >= entry.MaxRetries {
		s.logger.WithFields(logrus.Fields{
			"dlq_id":      entry.ID.Hex(),
			"user_id":     entry.UserID,
			"retry_count": len(entry.RetryHistory),
			"max_retries": entry.MaxRetries,
		}).Warn("DLQ entry exceeded max retries, marking as processed")

		// Mark as processed
		return s.dlqRepo.MarkAsProcessed(ctx, entry.ID.Hex())
	}

	// Create retry attempt
	retryAttempt := models.RetryAttempt{
		AttemptedAt: time.Now(),
		Success:     false,
		Duration:    0,
	}

	// Try to reprocess the notification
	start := time.Now()
	success, errorReason := s.reprocessNotification(ctx, entry)
	retryAttempt.Duration = time.Since(start).Milliseconds()

	if success {
		retryAttempt.Success = true
		s.logger.WithFields(logrus.Fields{
			"dlq_id":      entry.ID.Hex(),
			"user_id":     entry.UserID,
			"retry_count": len(entry.RetryHistory) + 1,
		}).Info("DLQ entry reprocessed successfully")

		// Mark as processed
		return s.dlqRepo.MarkAsProcessed(ctx, entry.ID.Hex())
	} else {
		retryAttempt.ErrorReason = errorReason
		s.logger.WithFields(logrus.Fields{
			"dlq_id":       entry.ID.Hex(),
			"user_id":      entry.UserID,
			"retry_count":  len(entry.RetryHistory) + 1,
			"error_reason": errorReason,
		}).Warn("DLQ entry retry failed")
	}

	// Update retry info
	nextRetryAt := time.Now().Add(s.config.RetryInterval)
	if err := s.dlqRepo.UpdateRetryInfo(ctx, entry.ID.Hex(), &retryAttempt, &nextRetryAt); err != nil {
		return fmt.Errorf("failed to update retry info: %w", err)
	}

	return nil
}

// reprocessNotification attempts to reprocess a failed notification
func (s *DLQService) reprocessNotification(ctx context.Context, entry *models.DeadLetterQueueEntry) (bool, string) {
	// Get the original notification
	notification, err := s.notifRepo.GetByID(ctx, entry.NotificationID.Hex())
	if err != nil {
		return false, fmt.Sprintf("failed to get notification: %v", err)
	}

	// Check if user still exists and has valid preferences
	_, err = s.preferenceSvc.GetUserPreferences(ctx, entry.UserID)
	if err != nil {
		return false, fmt.Sprintf("failed to get user preferences: %v", err)
	}

	// Check if user is still subscribed to this event type
	subscribed, err := s.preferenceSvc.IsEventSubscribed(ctx, entry.UserID, entry.EventType)
	if err != nil {
		return false, fmt.Sprintf("failed to check event subscription: %v", err)
	}
	if !subscribed {
		return false, "user no longer subscribed to event type"
	}

	// Check if channel is still enabled
	channelEnabled, err := s.preferenceSvc.IsChannelEnabled(ctx, entry.UserID, notification.Channel)
	if err != nil {
		return false, fmt.Sprintf("failed to check channel status: %v", err)
	}
	if !channelEnabled {
		return false, "channel no longer enabled for user"
	}

	// Try to send the notification again
	// This would integrate with the main notification service
	// For now, we'll simulate success
	s.logger.WithFields(logrus.Fields{
		"notification_id": notification.ID.Hex(),
		"user_id":         notification.UserID,
		"event_type":      notification.EventType,
		"channel":         notification.Channel,
	}).Info("Reprocessing notification from DLQ")

	// Update notification status
	if err := s.notifRepo.UpdateStatus(ctx, notification.ID.Hex(), models.StatusSent, ""); err != nil {
		return false, fmt.Sprintf("failed to update notification status: %v", err)
	}

	return true, ""
}

// GetDLQStats gets DLQ statistics
func (s *DLQService) GetDLQStats(ctx context.Context) (map[string]int64, error) {
	return s.dlqRepo.GetStats(ctx)
}

// GetDLQStatsByEventType gets DLQ statistics by event type
func (s *DLQService) GetDLQStatsByEventType(ctx context.Context) (map[string]int64, error) {
	return s.dlqRepo.GetStatsByEventType(ctx)
}

// GetFailedEntries gets entries that have exceeded max retries
func (s *DLQService) GetFailedEntries(ctx context.Context, limit int) ([]*models.DeadLetterQueueEntry, error) {
	return s.dlqRepo.GetFailedEntries(ctx, limit)
}

// RetryDLQEntry manually retries a DLQ entry
func (s *DLQService) RetryDLQEntry(ctx context.Context, dlqID string) error {
	// Get DLQ entry
	entry, err := s.dlqRepo.GetByID(ctx, dlqID)
	if err != nil {
		return fmt.Errorf("failed to get DLQ entry: %w", err)
	}

	// Process the entry
	return s.processDLQEntry(ctx, entry)
}

// DeleteDLQEntry deletes a DLQ entry
func (s *DLQService) DeleteDLQEntry(ctx context.Context, dlqID string) error {
	return s.dlqRepo.Delete(ctx, dlqID)
}

// CleanupOldEntries cleans up old processed entries
func (s *DLQService) CleanupOldEntries(ctx context.Context) (int64, error) {
	// Clean up entries older than 7 days
	olderThan := time.Now().AddDate(0, 0, -7)
	return s.dlqRepo.CleanupOldEntries(ctx, olderThan)
}

// StartDLQProcessor starts the DLQ processor goroutine
func (s *DLQService) StartDLQProcessor(ctx context.Context) {
	ticker := time.NewTicker(s.config.RetryInterval)
	defer ticker.Stop()

	cleanupTicker := time.NewTicker(s.config.CleanupInterval)
	defer cleanupTicker.Stop()

	s.logger.Info("DLQ processor started")

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("DLQ processor stopped")
			return
		case <-ticker.C:
			if err := s.ProcessDLQ(ctx); err != nil {
				s.logger.WithError(err).Error("Failed to process DLQ")
			}
		case <-cleanupTicker.C:
			if count, err := s.CleanupOldEntries(ctx); err != nil {
				s.logger.WithError(err).Error("Failed to cleanup old DLQ entries")
			} else if count > 0 {
				s.logger.WithField("count", count).Info("Cleaned up old DLQ entries")
			}
		}
	}
}

// GetDLQEntries gets DLQ entries with pagination
func (s *DLQService) GetDLQEntries(ctx context.Context, page, limit int, processed *bool) ([]*models.DeadLetterQueueEntry, int64, error) {
	return s.dlqRepo.GetAll(ctx, page, limit, processed)
}

// GetDLQEntry gets a specific DLQ entry
func (s *DLQService) GetDLQEntry(ctx context.Context, dlqID string) (*models.DeadLetterQueueEntry, error) {
	return s.dlqRepo.GetByID(ctx, dlqID)
}

// GetDLQEntryByNotificationID gets a DLQ entry by notification ID
func (s *DLQService) GetDLQEntryByNotificationID(ctx context.Context, notificationID string) (*models.DeadLetterQueueEntry, error) {
	// This would require converting string to ObjectID
	// For now, return an error
	return nil, fmt.Errorf("not implemented")
}

// ExportDLQEntries exports DLQ entries for analysis
func (s *DLQService) ExportDLQEntries(ctx context.Context, startDate, endDate time.Time) ([]*models.DeadLetterQueueEntry, error) {
	// This would require additional repository methods to filter by date
	// For now, return all entries
	entries, _, err := s.dlqRepo.GetAll(ctx, 1, 1000, nil)
	return entries, err
}

// GetDLQHealth checks the health of the DLQ system
func (s *DLQService) GetDLQHealth(ctx context.Context) (map[string]interface{}, error) {
	stats, err := s.GetDLQStats(ctx)
	if err != nil {
		return nil, err
	}

	eventStats, err := s.GetDLQStatsByEventType(ctx)
	if err != nil {
		return nil, err
	}

	health := map[string]interface{}{
		"total_entries":     stats["pending"] + stats["processed"],
		"pending_entries":   stats["pending"],
		"processed_entries": stats["processed"],
		"event_breakdown":   eventStats,
		"max_retries":       s.config.MaxRetries,
		"retry_interval":    s.config.RetryInterval.String(),
		"cleanup_interval":  s.config.CleanupInterval.String(),
	}

	return health, nil
}

// GetAll gets DLQ entries with pagination and filtering
func (s *DLQService) GetAll(ctx context.Context, page, limit int, processed *bool) ([]*models.DeadLetterQueueEntry, int64, error) {
	return s.dlqRepo.GetAll(ctx, page, limit, processed)
}
