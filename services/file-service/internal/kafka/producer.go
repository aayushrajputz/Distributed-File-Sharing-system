package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

type EventType string

const (
	EventFileUploaded EventType = "file.uploaded"
	EventFileShared   EventType = "file.shared"
	EventFileDeleted  EventType = "file.deleted"
)

type FileEvent struct {
	Type      EventType         `json:"type"`
	FileID    string            `json:"file_id"`
	FileName  string            `json:"file_name"`
	OwnerID   string            `json:"owner_id"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Timestamp string            `json:"timestamp"`
}

type Producer struct {
	writer     *kafka.Writer
	mu         sync.RWMutex
	closed     bool
	maxRetries int
	logger     *logrus.Logger
}

func NewProducer(brokers []string, topic string, maxRetries int, logger *logrus.Logger) *Producer {
	if logger == nil {
		logger = logrus.New()
	}

	return &Producer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			MaxAttempts:  3,
			BatchTimeout: 10 * time.Millisecond,
			WriteTimeout: 10 * time.Second,
			ReadTimeout:  10 * time.Second,
			RequiredAcks: kafka.RequireOne,
		},
		maxRetries: maxRetries,
		logger:     logger,
		closed:     false,
	}
}

func (p *Producer) PublishFileEvent(ctx context.Context, event FileEvent) error {
	// Check if producer is closed
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return fmt.Errorf("producer is closed")
	}
	p.mu.RUnlock()

	// Marshal event data
	data, err := json.Marshal(event)
	if err != nil {
		p.logger.WithError(err).Error("Failed to marshal Kafka event")
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Retry logic with exponential backoff
	var lastErr error
	for attempt := 0; attempt < p.maxRetries; attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Check if producer was closed during retries
		p.mu.RLock()
		if p.closed {
			p.mu.RUnlock()
			return fmt.Errorf("producer closed during publish")
		}
		p.mu.RUnlock()

		// Attempt to publish
		err = p.writer.WriteMessages(ctx, kafka.Message{
			Key:   []byte(event.FileID),
			Value: data,
			Time:  time.Now(),
		})

		if err == nil {
			p.logger.WithFields(logrus.Fields{
				"event_type": event.Type,
				"file_id":    event.FileID,
				"attempt":    attempt + 1,
			}).Debug("Successfully published Kafka event")
			return nil
		}

		lastErr = err
		p.logger.WithFields(logrus.Fields{
			"attempt":     attempt + 1,
			"max_retries": p.maxRetries,
			"error":       err.Error(),
			"event_type":  event.Type,
			"file_id":     event.FileID,
		}).Warn("Failed to publish Kafka message, retrying...")

		// Exponential backoff
		if attempt < p.maxRetries-1 {
			backoff := time.Duration(attempt+1) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}
	}

	p.logger.WithFields(logrus.Fields{
		"event_type":  event.Type,
		"file_id":     event.FileID,
		"max_retries": p.maxRetries,
		"error":       lastErr.Error(),
	}).Error("Failed to publish Kafka event after all retries")

	return fmt.Errorf("failed to publish event after %d retries: %w", p.maxRetries, lastErr)
}

func (p *Producer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true
	p.logger.Info("Closing Kafka producer")

	if err := p.writer.Close(); err != nil {
		p.logger.WithError(err).Error("Error closing Kafka writer")
		return err
	}

	p.logger.Info("Kafka producer closed successfully")
	return nil
}

// IsClosed returns whether the producer is closed
func (p *Producer) IsClosed() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.closed
}
