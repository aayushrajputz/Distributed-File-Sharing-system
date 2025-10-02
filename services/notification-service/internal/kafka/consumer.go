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
)

type FileEvent struct {
	Type      string            `json:"type"`
	FileID    string            `json:"file_id"`
	FileName  string            `json:"file_name"`
	OwnerID   string            `json:"owner_id"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Timestamp string            `json:"timestamp"`
}

type Consumer struct {
	reader       *kafka.Reader
	notifRepo    *repository.NotificationRepository
	streamBroker *StreamBroker
}

func NewConsumer(brokers []string, groupID, topic string, notifRepo *repository.NotificationRepository, streamBroker *StreamBroker) *Consumer {
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

	log.Printf("Processing event: %s for file %s", event.Type, event.FileID)

	// Create notification based on event type
	notification, err := c.createNotificationFromEvent(event)
	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	// Save to MongoDB
	if err := c.notifRepo.Create(ctx, notification); err != nil {
		return fmt.Errorf("failed to save notification: %w", err)
	}

	log.Printf("Created notification %s for user %s", notification.ID.Hex(), notification.UserID)

	// Send to streaming subscribers
	c.streamBroker.Broadcast(notification)

	return nil
}

func (c *Consumer) createNotificationFromEvent(event FileEvent) (*models.Notification, error) {
	notification := &models.Notification{
		Type:     models.NotificationType(event.Type),
		Metadata: event.Metadata,
	}

	switch event.Type {
	case "file.uploaded":
		notification.UserID = event.OwnerID
		notification.Title = "File Uploaded"
		notification.Body = fmt.Sprintf("Your file '%s' has been uploaded successfully", event.FileName)
		notification.Link = fmt.Sprintf("/files/%s", event.FileID)

	case "file.shared":
		// Notification for the person receiving the share
		if sharedWith, ok := event.Metadata["shared_with"]; ok {
			// TODO: Look up user ID from email via Auth Service
			notification.UserID = sharedWith // Placeholder
			notification.Title = "File Shared With You"
			notification.Body = fmt.Sprintf("A file '%s' has been shared with you", event.FileName)
			notification.Link = fmt.Sprintf("/files/%s", event.FileID)
		} else {
			return nil, fmt.Errorf("shared_with metadata missing")
		}

	case "file.deleted":
		notification.UserID = event.OwnerID
		notification.Title = "File Deleted"
		notification.Body = fmt.Sprintf("Your file '%s' has been deleted", event.FileName)

	default:
		return nil, fmt.Errorf("unknown event type: %s", event.Type)
	}

	return notification, nil
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
