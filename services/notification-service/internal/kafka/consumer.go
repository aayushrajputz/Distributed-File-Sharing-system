package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/models"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/repository"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/services"
)

type FileEvent struct {
	Type        string                 `json:"type"`
	UserID      string                 `json:"user_id"`
	FileID      string                 `json:"file_id"`
	FileName    string                 `json:"file_name"`
	FileSize    int64                  `json:"file_size"`
	Success     bool                   `json:"success"`
	ErrorReason string                 `json:"error_reason,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
}

type Consumer struct {
	reader       *kafka.Reader
	notifRepo    *repository.NotificationRepository
	streamBroker *StreamBroker
	notifSvc     *services.NotificationService
}

func NewConsumer(brokers []string, groupID, topic string, notifRepo *repository.NotificationRepository, streamBroker *StreamBroker, notifSvc *services.NotificationService) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		GroupID:        groupID,
		Topic:          topic,
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		CommitInterval: time.Second,
		StartOffset:    kafka.LastOffset,
	})

	return &Consumer{
		reader:       reader,
		notifRepo:    notifRepo,
		streamBroker: streamBroker,
		notifSvc:     notifSvc,
	}
}

func (c *Consumer) Start(ctx context.Context) error {
	log.Println("Starting Kafka consumer...")

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping Kafka consumer...")
			return c.reader.Close()
		default:
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				log.Printf("Error reading message: %v", err)
				continue
			}

			if err := c.processMessage(ctx, msg); err != nil {
				log.Printf("Error processing message: %v", err)
			}
		}
	}
}

func (c *Consumer) processMessage(ctx context.Context, msg kafka.Message) error {
	var event FileEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	log.Printf("Processing event: %s for file %s (user: %s)", event.Type, event.FileID, event.UserID)

	// Convert to KafkaFileEvent format
	kafkaEvent := &models.KafkaFileEvent{
		Type:        event.Type,
		UserID:      event.UserID,
		FileID:      event.FileID,
		FileName:    event.FileName,
		FileSize:    event.FileSize,
		Success:     event.Success,
		ErrorReason: event.ErrorReason,
		Metadata:    event.Metadata,
		Timestamp:   event.Timestamp,
	}

	// Process through notification service
	if err := c.notifSvc.ProcessKafkaEvent(ctx, kafkaEvent); err != nil {
		log.Printf("Failed to process Kafka event: %v", err)
		return err
	}

	log.Printf("Successfully processed event: %s for user %s", event.Type, event.UserID)
	return nil
}

// createNotificationFromEvent is deprecated - use ProcessKafkaEvent instead
func (c *Consumer) createNotificationFromEvent(event FileEvent) (*models.Notification, error) {
	// This method is kept for backward compatibility but should not be used
	// Use ProcessKafkaEvent in the notification service instead
	return nil, fmt.Errorf("deprecated method - use ProcessKafkaEvent instead")
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
