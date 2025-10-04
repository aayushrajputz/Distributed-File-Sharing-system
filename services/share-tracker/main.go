package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// FileEvent represents a Kafka file event
type FileEvent struct {
	Type      string            `json:"type"`
	FileID    string            `json:"file_id"`
	FileName  string            `json:"file_name"`
	OwnerID   string            `json:"owner_id"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Timestamp string            `json:"timestamp"`
}

// ShareEvent represents a file sharing event to be logged
type ShareEvent struct {
	FileName     string `json:"file_name"`
	FileID       string `json:"file_id"`
	OriginalPath string `json:"original_path"`
	SharedWith   string `json:"shared_with"`
	SharedBy     string `json:"shared_by"`
	Permission   string `json:"permission"`
	Timestamp    string `json:"timestamp"`
	ShareID      string `json:"share_id"`
}

// ShareLog represents the JSON log structure
type ShareLog struct {
	SharingEvents []ShareEvent `json:"sharing_events"`
	Metadata      LogMetadata  `json:"metadata"`
	mu            sync.Mutex   `json:"-"`
}

// LogMetadata contains metadata about the log file
type LogMetadata struct {
	CreatedAt   string `json:"created_at"`
	LastUpdated string `json:"last_updated"`
	TotalEvents int    `json:"total_events"`
	Description string `json:"description"`
}

// Config holds the service configuration
type Config struct {
	KafkaBrokers []string
	KafkaTopic   string
	LogFilePath  string
	GroupID      string
}

var log = logrus.New()

func main() {
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	log.SetLevel(logrus.InfoLevel)

	log.Info("Starting Share Tracker Service...")

	// Configuration from environment variables or defaults
	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		kafkaBrokers = "kafka:9092"
	}

	kafkaTopic := os.Getenv("KAFKA_TOPIC")
	if kafkaTopic == "" {
		kafkaTopic = "file-events"
	}

	groupID := os.Getenv("KAFKA_GROUP_ID")
	if groupID == "" {
		groupID = "share-tracker-group"
	}

	logFilePath := os.Getenv("LOG_FILE_PATH")
	if logFilePath == "" {
		logFilePath = "/app/SharedFiles/shared_files.json"
	}

	config := Config{
		KafkaBrokers: []string{kafkaBrokers},
		KafkaTopic:   kafkaTopic,
		LogFilePath:  logFilePath,
		GroupID:      groupID,
	}

	// Initialize share log
	shareLog, err := loadShareLog(config.LogFilePath)
	if err != nil {
		log.WithError(err).Fatal("Failed to load share log")
	}

	// Create Kafka reader
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        config.KafkaBrokers,
		Topic:          config.KafkaTopic,
		GroupID:        config.GroupID,
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		CommitInterval: time.Second,
		StartOffset:    kafka.LastOffset,
	})
	defer reader.Close()

	log.WithFields(logrus.Fields{
		"brokers": config.KafkaBrokers,
		"topic":   config.KafkaTopic,
		"group":   config.GroupID,
	}).Info("Connected to Kafka")

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Info("Shutdown signal received, stopping...")
		cancel()
	}()

	// Start consuming messages
	log.Info("Share Tracker is ready and listening for file sharing events...")
	log.Info("Waiting for file sharing events from Kafka...")

	lastStatusLog := time.Now()
	messageCount := 0

	for {
		select {
		case <-ctx.Done():
			log.WithField("messages_processed", messageCount).Info("Shutting down gracefully...")
			return
		default:
			// Read message with timeout
			readCtx, readCancel := context.WithTimeout(ctx, 30*time.Second)
			msg, err := reader.ReadMessage(readCtx)
			readCancel()

			if err != nil {
				if err == context.DeadlineExceeded || err == context.Canceled {
					// Log status every 5 minutes to show service is alive
					if time.Since(lastStatusLog) > 5*time.Minute {
						log.WithFields(logrus.Fields{
							"status":             "active",
							"messages_processed": messageCount,
						}).Info("Service is healthy and waiting for events...")
						lastStatusLog = time.Now()
					}
					continue
				}
				// Only log real errors (not timeouts)
				log.WithError(err).Error("Kafka connection error, retrying...")
				time.Sleep(5 * time.Second)
				continue
			}

			// Process message
			if err := processMessage(msg, shareLog, config.LogFilePath); err != nil {
				log.WithError(err).Error("Failed to process message")
			} else {
				messageCount++
			}
		}
	}
}

func processMessage(msg kafka.Message, shareLog *ShareLog, logFilePath string) error {
	// Parse Kafka message
	var event FileEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	// Only process file.shared events
	if event.Type != "file.shared" {
		return nil
	}

	log.WithFields(logrus.Fields{
		"event_type": event.Type,
		"file_id":    event.FileID,
		"file_name":  event.FileName,
	}).Info("Processing file sharing event")

	// Extract metadata
	sharedWith := event.Metadata["shared_with"]
	permission := event.Metadata["permission"]
	if sharedWith == "" {
		sharedWith = "link-only"
	}
	if permission == "" {
		permission = "READ"
	}

	// Create share event
	shareEvent := ShareEvent{
		FileName:     event.FileName,
		FileID:       event.FileID,
		OriginalPath: fmt.Sprintf("minio://files/%s", event.FileID),
		SharedWith:   sharedWith,
		SharedBy:     event.OwnerID,
		Permission:   permission,
		Timestamp:    event.Timestamp,
		ShareID:      fmt.Sprintf("share_%s_%d", event.FileID, time.Now().Unix()),
	}

	// Add to log
	if err := shareLog.addEvent(shareEvent, logFilePath); err != nil {
		return fmt.Errorf("failed to add event to log: %w", err)
	}

	// Output confirmation
	confirmation := map[string]interface{}{
		"status":      "success",
		"file_name":   shareEvent.FileName,
		"shared_with": shareEvent.SharedWith,
		"share_id":    shareEvent.ShareID,
		"timestamp":   shareEvent.Timestamp,
	}

	confirmJSON, _ := json.MarshalIndent(confirmation, "", "  ")
	fmt.Println(string(confirmJSON))

	log.WithFields(logrus.Fields{
		"file_name":   shareEvent.FileName,
		"shared_with": shareEvent.SharedWith,
		"share_id":    shareEvent.ShareID,
	}).Info("File sharing event logged successfully")

	return nil
}

func loadShareLog(filePath string) (*ShareLog, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Create new log
		shareLog := &ShareLog{
			SharingEvents: []ShareEvent{},
			Metadata: LogMetadata{
				CreatedAt:   time.Now().Format(time.RFC3339),
				LastUpdated: time.Now().Format(time.RFC3339),
				TotalEvents: 0,
				Description: "Log of all file sharing events in the distributed file-sharing platform",
			},
		}

		// Save initial log
		if err := saveShareLog(shareLog, filePath); err != nil {
			return nil, err
		}

		return shareLog, nil
	}

	// Load existing log
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read log file: %w", err)
	}

	var shareLog ShareLog
	if err := json.Unmarshal(data, &shareLog); err != nil {
		return nil, fmt.Errorf("failed to unmarshal log: %w", err)
	}

	return &shareLog, nil
}

func saveShareLog(shareLog *ShareLog, filePath string) error {
	data, err := json.MarshalIndent(shareLog, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal log: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write log file: %w", err)
	}

	return nil
}

func (sl *ShareLog) addEvent(event ShareEvent, filePath string) error {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	// Add event to list
	sl.SharingEvents = append(sl.SharingEvents, event)

	// Update metadata
	sl.Metadata.LastUpdated = time.Now().Format(time.RFC3339)
	sl.Metadata.TotalEvents = len(sl.SharingEvents)

	// Save to file
	return saveShareLog(sl, filePath)
}
