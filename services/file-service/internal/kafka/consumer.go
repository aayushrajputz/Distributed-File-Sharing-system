package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/cassandra"
	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/metrics"
)

// Consumer handles Kafka message consumption for Cassandra integration
type Consumer struct {
	reader        *kafka.Reader
	cassandraRepo *cassandra.Repository
	logger        *logrus.Logger
	mu            sync.RWMutex
	closed        bool
	maxRetries    int
	retryDelay    time.Duration
}

// NewConsumer creates a new Kafka consumer for file events
func NewConsumer(brokers []string, topic, groupID string, cassandraRepo *cassandra.Repository, maxRetries int, logger *logrus.Logger) *Consumer {
	if logger == nil {
		logger = logrus.New()
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		CommitInterval: time.Second,
		StartOffset:    kafka.LastOffset,
	})

	return &Consumer{
		reader:        reader,
		cassandraRepo: cassandraRepo,
		logger:        logger,
		closed:        false,
		maxRetries:    maxRetries,
		retryDelay:    time.Second,
	}
}

// Start begins consuming messages from Kafka
func (c *Consumer) Start(ctx context.Context) error {
	c.logger.Info("Starting Kafka consumer for file events")

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Context cancelled, stopping consumer")
			return ctx.Err()
		default:
			// Check if consumer is closed
			c.mu.RLock()
			if c.closed {
				c.mu.RUnlock()
				return fmt.Errorf("consumer is closed")
			}
			c.mu.RUnlock()

			// Read message with timeout
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				c.logger.WithError(err).Error("Failed to read message from Kafka")
				continue
			}

			// Process message asynchronously
			go c.processMessage(ctx, msg)
		}
	}
}

// processMessage processes a single Kafka message
func (c *Consumer) processMessage(ctx context.Context, msg kafka.Message) {
	c.logger.WithFields(logrus.Fields{
		"topic":     msg.Topic,
		"partition": msg.Partition,
		"offset":    msg.Offset,
		"key":       string(msg.Key),
	}).Debug("Processing Kafka message")

	// Parse message based on key or content
	var err error
	switch string(msg.Key) {
	case "file.uploaded":
		err = c.handleFileUploadedEvent(ctx, msg.Value)
	case "file.deleted":
		err = c.handleFileDeletedEvent(ctx, msg.Value)
	case "file.downloaded":
		err = c.handleFileDownloadedEvent(ctx, msg.Value)
	case "file.versioned":
		err = c.handleFileVersionedEvent(ctx, msg.Value)
	default:
		// Try to auto-detect event type from content
		err = c.handleGenericFileEvent(ctx, msg.Value)
	}

	if err != nil {
		c.logger.WithError(err).WithFields(logrus.Fields{
			"topic":     msg.Topic,
			"partition": msg.Partition,
			"offset":    msg.Offset,
			"key":       string(msg.Key),
		}).Error("Failed to process Kafka message")
		// TODO: Send to DLQ after max retries
		return
	}

	c.logger.WithFields(logrus.Fields{
		"topic":     msg.Topic,
		"partition": msg.Partition,
		"offset":    msg.Offset,
	}).Debug("Successfully processed Kafka message")
}

// handleFileUploadedEvent handles file upload events
func (c *Consumer) handleFileUploadedEvent(ctx context.Context, data []byte) error {
	start := time.Now()
	defer func() {
		metrics.RecordCassandraQueryDuration("handle_file_uploaded_event", time.Since(start).Seconds())
	}()

	var event FileUploadedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		metrics.RecordCassandraEventProcessed("file_uploaded", "unmarshal_error")
		return fmt.Errorf("failed to unmarshal file upload event: %w", err)
	}

	// Convert to Cassandra event
	cassandraEvent := &cassandra.FileEvent{
		UserID:   event.UserID,
		EventTS:  event.Timestamp,
		EventID:  uuid.MustParse(event.EventID),
		FileID:   uuid.MustParse(event.FileID),
		Action:   event.Action,
		Status:   event.Status,
		FileName: event.FileName,
		FileSize: event.FileSize,
		Metadata: event.Metadata,
	}

	// Check for idempotency
	exists, err := c.cassandraRepo.CheckEventExists(ctx, event.UserID, event.Timestamp, cassandraEvent.EventID)
	if err != nil {
		metrics.RecordCassandraEventProcessed("file_uploaded", "check_exists_error")
		return fmt.Errorf("failed to check event existence: %w", err)
	}

	if exists {
		metrics.RecordCassandraEventProcessed("file_uploaded", "duplicate_skipped")
		c.logger.WithFields(logrus.Fields{
			"event_id": event.EventID,
			"user_id":  event.UserID,
		}).Debug("Event already exists, skipping")
		return nil
	}

	// Log event to Cassandra
	if err := c.cassandraRepo.LogFileEvent(ctx, cassandraEvent); err != nil {
		metrics.RecordCassandraEventProcessed("file_uploaded", "cassandra_error")
		return err
	}

	metrics.RecordCassandraEventProcessed("file_uploaded", "success")
	return nil
}

// handleFileDeletedEvent handles file deletion events
func (c *Consumer) handleFileDeletedEvent(ctx context.Context, data []byte) error {
	var event FileDeletedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return fmt.Errorf("failed to unmarshal file deletion event: %w", err)
	}

	// Convert to Cassandra event
	cassandraEvent := &cassandra.FileEvent{
		UserID:   event.UserID,
		EventTS:  event.Timestamp,
		EventID:  uuid.MustParse(event.EventID),
		FileID:   uuid.MustParse(event.FileID),
		Action:   event.Action,
		Status:   event.Status,
		FileName: event.FileName,
		FileSize: 0, // Deleted files have no size
		Metadata: event.Metadata,
	}

	// Check for idempotency
	exists, err := c.cassandraRepo.CheckEventExists(ctx, event.UserID, event.Timestamp, cassandraEvent.EventID)
	if err != nil {
		return fmt.Errorf("failed to check event existence: %w", err)
	}

	if exists {
		c.logger.WithFields(logrus.Fields{
			"event_id": event.EventID,
			"user_id":  event.UserID,
		}).Debug("Event already exists, skipping")
		return nil
	}

	// Log event to Cassandra
	return c.cassandraRepo.LogFileEvent(ctx, cassandraEvent)
}

// handleFileDownloadedEvent handles file download events
func (c *Consumer) handleFileDownloadedEvent(ctx context.Context, data []byte) error {
	var event FileDownloadedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return fmt.Errorf("failed to unmarshal file download event: %w", err)
	}

	// Convert to Cassandra event
	cassandraEvent := &cassandra.FileEvent{
		UserID:   event.UserID,
		EventTS:  event.Timestamp,
		EventID:  uuid.MustParse(event.EventID),
		FileID:   uuid.MustParse(event.FileID),
		Action:   event.Action,
		Status:   event.Status,
		FileName: event.FileName,
		FileSize: 0, // Downloads don't change file size
		Metadata: event.Metadata,
	}

	// Check for idempotency
	exists, err := c.cassandraRepo.CheckEventExists(ctx, event.UserID, event.Timestamp, cassandraEvent.EventID)
	if err != nil {
		return fmt.Errorf("failed to check event existence: %w", err)
	}

	if exists {
		c.logger.WithFields(logrus.Fields{
			"event_id": event.EventID,
			"user_id":  event.UserID,
		}).Debug("Event already exists, skipping")
		return nil
	}

	// Log event to Cassandra
	return c.cassandraRepo.LogFileEvent(ctx, cassandraEvent)
}

// handleFileVersionedEvent handles file version events
func (c *Consumer) handleFileVersionedEvent(ctx context.Context, data []byte) error {
	var event FileVersionedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return fmt.Errorf("failed to unmarshal file version event: %w", err)
	}

	// Convert to Cassandra version
	cassandraVersion := &cassandra.FileVersion{
		FileID:      uuid.MustParse(event.FileID),
		Version:     event.Version,
		FileName:    event.FileName,
		FileSize:    event.FileSize,
		ContentType: event.ContentType,
		StoragePath: event.StoragePath,
		Checksum:    event.Checksum,
		UploadedAt:  event.Timestamp,
		UploadedBy:  event.UserID,
		Metadata:    event.Metadata,
	}

	// Add version to Cassandra
	return c.cassandraRepo.AddFileVersion(ctx, cassandraVersion)
}

// handleGenericFileEvent attempts to auto-detect and handle file events
func (c *Consumer) handleGenericFileEvent(ctx context.Context, data []byte) error {
	// Try to parse as a generic event structure
	var genericEvent struct {
		EventID   string    `json:"event_id"`
		FileID    string    `json:"file_id"`
		UserID    string    `json:"user_id"`
		Action    string    `json:"action"`
		Status    string    `json:"status"`
		FileName  string    `json:"file_name"`
		FileSize  int64     `json:"file_size"`
		Timestamp time.Time `json:"timestamp"`
		Metadata  string    `json:"metadata"`
	}

	if err := json.Unmarshal(data, &genericEvent); err != nil {
		return fmt.Errorf("failed to unmarshal generic file event: %w", err)
	}

	// Convert to Cassandra event
	// Handle both UUID and ObjectID formats
	var eventID, fileID uuid.UUID
	var err error
	
	// Parse EventID (should be UUID)
	eventID, err = uuid.Parse(genericEvent.EventID)
	if err != nil {
		// If it's not a valid UUID, generate a new one
		eventID = uuid.New()
		c.logger.WithFields(logrus.Fields{
			"original_event_id": genericEvent.EventID,
			"new_event_id":      eventID.String(),
		}).Warn("Invalid EventID format, generated new UUID")
	}
	
	// Parse FileID (could be ObjectID or UUID)
	fileID, err = uuid.Parse(genericEvent.FileID)
	if err != nil {
		// If it's an ObjectID (24 chars), convert to UUID by padding
		if len(genericEvent.FileID) == 24 {
			// Convert ObjectID to UUID by creating a deterministic UUID
			// This is a simple approach - in production you might want a more sophisticated mapping
			fileID = uuid.NewSHA1(uuid.NameSpaceOID, []byte(genericEvent.FileID))
		} else {
			// Generate a new UUID for other invalid formats
			fileID = uuid.New()
		}
		c.logger.WithFields(logrus.Fields{
			"original_file_id": genericEvent.FileID,
			"new_file_id":      fileID.String(),
		}).Warn("Invalid FileID format, generated new UUID")
	}

	cassandraEvent := &cassandra.FileEvent{
		UserID:   genericEvent.UserID,
		EventTS:  genericEvent.Timestamp,
		EventID:  eventID,
		FileID:   fileID,
		Action:   genericEvent.Action,
		Status:   genericEvent.Status,
		FileName: genericEvent.FileName,
		FileSize: genericEvent.FileSize,
		Metadata: genericEvent.Metadata,
	}

	// Check for idempotency
	exists, err := c.cassandraRepo.CheckEventExists(ctx, genericEvent.UserID, genericEvent.Timestamp, cassandraEvent.EventID)
	if err != nil {
		return fmt.Errorf("failed to check event existence: %w", err)
	}

	if exists {
		c.logger.WithFields(logrus.Fields{
			"event_id": genericEvent.EventID,
			"user_id":  genericEvent.UserID,
		}).Debug("Event already exists, skipping")
		return nil
	}

	// Log event to Cassandra
	return c.cassandraRepo.LogFileEvent(ctx, cassandraEvent)
}

// Close gracefully closes the consumer
func (c *Consumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	c.logger.Info("Closing Kafka consumer")

	if err := c.reader.Close(); err != nil {
		c.logger.WithError(err).Error("Error closing Kafka reader")
		return err
	}

	c.logger.Info("Kafka consumer closed successfully")
	return nil
}

// IsClosed returns whether the consumer is closed
func (c *Consumer) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closed
}
