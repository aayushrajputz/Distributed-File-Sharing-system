package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/models"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/repository"
)

// BatchService handles batch notification processing
type BatchService struct {
	redisClient   *redis.Client
	batchRepo     *repository.BatchRepository
	notifRepo     *repository.NotificationRepository
	preferenceSvc *PreferenceService
	templateSvc   *TemplateService
	config        *BatchConfig
	logger        *logrus.Logger
}

// BatchConfig contains batch service configuration
type BatchConfig struct {
	WindowDuration time.Duration
	MaxSize        int
	FlushInterval  time.Duration
	RedisKeyPrefix string
}

// BatchItem represents an item in a batch
type BatchItem struct {
	UserID    string                     `json:"user_id"`
	EventType models.EventType           `json:"event_type"`
	Channel   models.NotificationChannel `json:"channel"`
	Title     string                     `json:"title"`
	Message   string                     `json:"message"`
	Priority  models.Priority            `json:"priority"`
	Metadata  map[string]interface{}     `json:"metadata"`
	Timestamp time.Time                  `json:"timestamp"`
}

// BatchKey represents a Redis key for batching
type BatchKey struct {
	UserID    string                     `json:"user_id"`
	EventType models.EventType           `json:"event_type"`
	Channel   models.NotificationChannel `json:"channel"`
}

// NewBatchService creates a new batch service
func NewBatchService(
	redisClient *redis.Client,
	batchRepo *repository.BatchRepository,
	notifRepo *repository.NotificationRepository,
	preferenceSvc *PreferenceService,
	templateSvc *TemplateService,
	config *BatchConfig,
	logger *logrus.Logger,
) *BatchService {
	return &BatchService{
		redisClient:   redisClient,
		batchRepo:     batchRepo,
		notifRepo:     notifRepo,
		preferenceSvc: preferenceSvc,
		templateSvc:   templateSvc,
		config:        config,
		logger:        logger,
	}
}

// AddToBatch adds a notification to the batch
func (s *BatchService) AddToBatch(ctx context.Context, req *models.NotificationRequest) error {
	// Check if notification should bypass batching
	if req.BypassBatching {
		return s.sendImmediately(ctx, req)
	}

	// Check if event type should bypass batching (critical notifications)
	if s.shouldBypassBatching(req.EventType) {
		return s.sendImmediately(ctx, req)
	}

	// Get user preferences to determine channels
	_, err := s.preferenceSvc.GetUserPreferences(ctx, req.UserID)
	if err != nil {
		s.logger.WithError(err).WithField("user_id", req.UserID).Warn("Failed to get user preferences, sending immediately")
		return s.sendImmediately(ctx, req)
	}

	// Get optimal channel for this event type
	channel, err := s.preferenceSvc.GetOptimalChannel(ctx, req.UserID, req.EventType)
	if err != nil {
		s.logger.WithError(err).WithField("user_id", req.UserID).Warn("Failed to get optimal channel, sending immediately")
		return s.sendImmediately(ctx, req)
	}

	// Create batch item
	item := BatchItem{
		UserID:    req.UserID,
		EventType: req.EventType,
		Channel:   channel,
		Title:     req.Title,
		Message:   req.Message,
		Priority:  req.Priority,
		Metadata:  req.Metadata,
		Timestamp: time.Now(),
	}

	// Add to Redis batch
	return s.addToRedisBatch(ctx, item)
}

// addToRedisBatch adds an item to Redis batch
func (s *BatchService) addToRedisBatch(ctx context.Context, item BatchItem) error {
	// Create batch key
	batchKey := BatchKey{
		UserID:    item.UserID,
		EventType: item.EventType,
		Channel:   item.Channel,
	}

	key := s.getBatchKey(batchKey)

	// Marshal item to JSON
	itemJSON, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("failed to marshal batch item: %w", err)
	}

	// Add to Redis sorted set with timestamp as score
	score := float64(item.Timestamp.Unix())
	err = s.redisClient.ZAdd(ctx, key, &redis.Z{
		Score:  score,
		Member: itemJSON,
	}).Err()

	if err != nil {
		return fmt.Errorf("failed to add item to batch: %w", err)
	}

	// Set expiration for the batch key
	expiration := s.config.WindowDuration + time.Minute // Add buffer
	s.redisClient.Expire(ctx, key, expiration)

	s.logger.WithFields(logrus.Fields{
		"user_id":    item.UserID,
		"event_type": item.EventType,
		"channel":    item.Channel,
		"key":        key,
	}).Debug("Added item to batch")

	return nil
}

// ProcessBatches processes all pending batches
func (s *BatchService) ProcessBatches(ctx context.Context) error {
	// Get all batch keys
	pattern := s.config.RedisKeyPrefix + "*"
	keys, err := s.redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to get batch keys: %w", err)
	}

	for _, key := range keys {
		if err := s.processBatch(ctx, key); err != nil {
			s.logger.WithError(err).WithField("key", key).Error("Failed to process batch")
			continue
		}
	}

	return nil
}

// processBatch processes a single batch
func (s *BatchService) processBatch(ctx context.Context, key string) error {
	// Get all items in the batch
	items, err := s.redisClient.ZRange(ctx, key, 0, -1).Result()
	if err != nil {
		return fmt.Errorf("failed to get batch items: %w", err)
	}

	if len(items) == 0 {
		// Empty batch, remove key
		s.redisClient.Del(ctx, key)
		return nil
	}

	// Parse batch key to get user and event info
	batchKey, err := s.parseBatchKey(key)
	if err != nil {
		return fmt.Errorf("failed to parse batch key: %w", err)
	}

	// Parse items
	var batchItems []BatchItem
	for _, itemJSON := range items {
		var item BatchItem
		if err := json.Unmarshal([]byte(itemJSON), &item); err != nil {
			s.logger.WithError(err).WithField("item", itemJSON).Warn("Failed to unmarshal batch item")
			continue
		}
		batchItems = append(batchItems, item)
	}

	// Create batch notification
	batchNotification := s.createBatchNotification(batchKey, batchItems)

	// Store in database
	if err := s.batchRepo.Create(ctx, batchNotification); err != nil {
		return fmt.Errorf("failed to create batch notification: %w", err)
	}

	// Send batch notification
	if err := s.sendBatchNotification(ctx, batchNotification); err != nil {
		s.logger.WithError(err).WithField("batch_id", batchNotification.ID.Hex()).Error("Failed to send batch notification")
		// Update status to failed
		s.batchRepo.UpdateStatus(ctx, batchNotification.ID.Hex(), models.StatusFailed)
	} else {
		// Update status to sent
		s.batchRepo.UpdateStatus(ctx, batchNotification.ID.Hex(), models.StatusSent)
	}

	// Remove batch from Redis
	s.redisClient.Del(ctx, key)

	s.logger.WithFields(logrus.Fields{
		"user_id":    batchKey.UserID,
		"event_type": batchKey.EventType,
		"channel":    batchKey.Channel,
		"count":      len(batchItems),
	}).Info("Processed batch notification")

	return nil
}

// createBatchNotification creates a batch notification from items
func (s *BatchService) createBatchNotification(batchKey BatchKey, items []BatchItem) *models.BatchNotification {
	// Group items by success/failure
	successItems := make([]models.BatchItem, 0)
	failureItems := make([]models.BatchItem, 0)

	for _, item := range items {
		batchItem := models.BatchItem{
			FileName:    s.getStringFromMetadata(item.Metadata, "file_name", "Unknown"),
			FileSize:    s.getInt64FromMetadata(item.Metadata, "file_size", 0),
			Success:     item.EventType != models.EventTypeFileUploadFailed,
			ErrorReason: s.getStringFromMetadata(item.Metadata, "error_message", ""),
			Metadata:    item.Metadata,
			Timestamp:   item.Timestamp,
		}

		if batchItem.Success {
			successItems = append(successItems, batchItem)
		} else {
			failureItems = append(failureItems, batchItem)
		}
	}

	// Create title and message
	title, message := s.createBatchTitleAndMessage(successItems, failureItems)

	// Combine all items
	allItems := append(successItems, failureItems...)

	return &models.BatchNotification{
		UserID:    batchKey.UserID,
		EventType: batchKey.EventType,
		Channel:   batchKey.Channel,
		Title:     title,
		Message:   message,
		Count:     len(allItems),
		Items:     allItems,
		Status:    models.StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// createBatchTitleAndMessage creates title and message for batch notification
func (s *BatchService) createBatchTitleAndMessage(successItems, failureItems []models.BatchItem) (string, string) {
	successCount := len(successItems)
	failureCount := len(failureItems)

	if successCount > 0 && failureCount > 0 {
		// Mixed results
		title := "File Upload Results"
		message := fmt.Sprintf("📁 %d files uploaded successfully, %d failed", successCount, failureCount)
		return title, message
	} else if successCount > 0 {
		// All successful
		title := "Files Uploaded Successfully"
		message := fmt.Sprintf("✅ %d files uploaded successfully", successCount)
		return title, message
	} else if failureCount > 0 {
		// All failed
		title := "File Upload Failed"
		message := fmt.Sprintf("❌ %d files failed to upload", failureCount)
		return title, message
	}

	// Fallback
	return "Batch Notification", "No items to process"
}

// sendBatchNotification sends a batch notification
func (s *BatchService) sendBatchNotification(ctx context.Context, batch *models.BatchNotification) error {
	// Create notification request for batch
	req := &models.NotificationRequest{
		UserID:         batch.UserID,
		EventType:      batch.EventType,
		Channel:        batch.Channel,
		Title:          batch.Title,
		Message:        batch.Message,
		Priority:       models.PriorityNormal,
		Metadata:       make(map[string]interface{}),
		BypassBatching: true, // Don't batch the batch notification
	}

	// Add batch metadata
	req.Metadata["batch_id"] = batch.ID.Hex()
	req.Metadata["count"] = batch.Count
	req.Metadata["success_count"] = s.countSuccessItems(batch.Items)
	req.Metadata["failure_count"] = s.countFailureItems(batch.Items)

	// Send immediately (bypass batching)
	return s.sendImmediately(ctx, req)
}

// sendImmediately sends a notification immediately without batching
func (s *BatchService) sendImmediately(ctx context.Context, req *models.NotificationRequest) error {
	// This would integrate with the main notification service
	// For now, just log the notification
	s.logger.WithFields(logrus.Fields{
		"user_id":    req.UserID,
		"event_type": req.EventType,
		"title":      req.Title,
		"bypass":     req.BypassBatching,
	}).Info("Sending immediate notification")

	// TODO: Integrate with notification service to send immediately
	return nil
}

// shouldBypassBatching checks if an event type should bypass batching
func (s *BatchService) shouldBypassBatching(eventType models.EventType) bool {
	// Critical notifications should not be batched
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

// getBatchKey creates a Redis key for a batch
func (s *BatchService) getBatchKey(batchKey BatchKey) string {
	return fmt.Sprintf("%s:%s:%s:%s", s.config.RedisKeyPrefix, batchKey.UserID, batchKey.EventType, batchKey.Channel)
}

// parseBatchKey parses a Redis key to extract batch information
func (s *BatchService) parseBatchKey(key string) (BatchKey, error) {
	// Remove prefix
	key = key[len(s.config.RedisKeyPrefix):]

	// Split by colon
	parts := splitString(key, ":")
	if len(parts) != 3 {
		return BatchKey{}, fmt.Errorf("invalid batch key format: %s", key)
	}

	return BatchKey{
		UserID:    parts[0],
		EventType: models.EventType(parts[1]),
		Channel:   models.NotificationChannel(parts[2]),
	}, nil
}

// splitString splits a string by delimiter
func splitString(s, delimiter string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i:i+len(delimiter)] == delimiter {
			result = append(result, s[start:i])
			start = i + len(delimiter)
		}
	}
	result = append(result, s[start:])
	return result
}

// countSuccessItems counts successful items in a batch
func (s *BatchService) countSuccessItems(items []models.BatchItem) int {
	count := 0
	for _, item := range items {
		if item.Success {
			count++
		}
	}
	return count
}

// countFailureItems counts failed items in a batch
func (s *BatchService) countFailureItems(items []models.BatchItem) int {
	count := 0
	for _, item := range items {
		if !item.Success {
			count++
		}
	}
	return count
}

// getStringFromMetadata gets a string value from metadata
func (s *BatchService) getStringFromMetadata(metadata map[string]interface{}, key, defaultValue string) string {
	if value, ok := metadata[key].(string); ok {
		return value
	}
	return defaultValue
}

// getInt64FromMetadata gets an int64 value from metadata
func (s *BatchService) getInt64FromMetadata(metadata map[string]interface{}, key string, defaultValue int64) int64 {
	if value, ok := metadata[key].(int64); ok {
		return value
	}
	if value, ok := metadata[key].(float64); ok {
		return int64(value)
	}
	return defaultValue
}

// StartBatchProcessor starts the batch processor goroutine
func (s *BatchService) StartBatchProcessor(ctx context.Context) {
	ticker := time.NewTicker(s.config.FlushInterval)
	defer ticker.Stop()

	s.logger.Info("Batch processor started")

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Batch processor stopped")
			return
		case <-ticker.C:
			if err := s.ProcessBatches(ctx); err != nil {
				s.logger.WithError(err).Error("Failed to process batches")
			}
		}
	}
}

// GetBatchStats gets batch statistics
func (s *BatchService) GetBatchStats(ctx context.Context, userID string, startDate, endDate time.Time) (map[string]int64, error) {
	return s.batchRepo.GetBatchStats(ctx, userID, startDate, endDate)
}

// CleanupOldBatches cleans up old batch notifications
func (s *BatchService) CleanupOldBatches(ctx context.Context, olderThan time.Time) (int64, error) {
	return s.batchRepo.CleanupOldBatches(ctx, olderThan)
}

// GetBatchNotifications gets batch notifications for a user with pagination and filtering
func (s *BatchService) GetBatchNotifications(ctx context.Context, userID string, page, limit int, statusFilter *models.NotificationStatus) ([]*models.BatchNotification, int64, error) {
	return s.batchRepo.GetByUserID(ctx, userID, page, limit, statusFilter)
}
