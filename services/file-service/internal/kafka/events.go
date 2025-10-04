package kafka

import (
	"time"

	"github.com/google/uuid"
)

// FileUploadedEvent represents a file upload event for Cassandra integration
type FileUploadedEvent struct {
	EventID     string    `json:"event_id"` // UUID for idempotency
	FileID      string    `json:"file_id"`
	UserID      string    `json:"user_id"`
	FileName    string    `json:"file_name"`
	FileSize    int64     `json:"file_size"`
	ContentType string    `json:"content_type"`
	Action      string    `json:"action"` // "upload"
	Status      string    `json:"status"` // "success"
	Timestamp   time.Time `json:"timestamp"`
	Metadata    string    `json:"metadata"` // JSON string
}

// FileDeletedEvent represents a file deletion event
type FileDeletedEvent struct {
	EventID   string    `json:"event_id"`
	FileID    string    `json:"file_id"`
	UserID    string    `json:"user_id"`
	FileName  string    `json:"file_name"`
	Action    string    `json:"action"` // "delete"
	Status    string    `json:"status"` // "success"
	Timestamp time.Time `json:"timestamp"`
	Metadata  string    `json:"metadata"`
}

// FileDownloadedEvent represents a file download event
type FileDownloadedEvent struct {
	EventID   string    `json:"event_id"`
	FileID    string    `json:"file_id"`
	UserID    string    `json:"user_id"`
	FileName  string    `json:"file_name"`
	Action    string    `json:"action"` // "download"
	Status    string    `json:"status"` // "success"
	Timestamp time.Time `json:"timestamp"`
	Metadata  string    `json:"metadata"`
}

// FileVersionedEvent represents a file version creation event
type FileVersionedEvent struct {
	EventID     string    `json:"event_id"`
	FileID      string    `json:"file_id"`
	UserID      string    `json:"user_id"`
	Version     int       `json:"version"`
	FileName    string    `json:"file_name"`
	FileSize    int64     `json:"file_size"`
	ContentType string    `json:"content_type"`
	StoragePath string    `json:"storage_path"`
	Checksum    string    `json:"checksum"`
	Action      string    `json:"action"` // "version_created"
	Status      string    `json:"status"` // "success"
	Timestamp   time.Time `json:"timestamp"`
	Metadata    string    `json:"metadata"`
}

// NewFileUploadedEvent creates a new file upload event
func NewFileUploadedEvent(fileID, userID, fileName, contentType string, fileSize int64, metadata string) *FileUploadedEvent {
	return &FileUploadedEvent{
		EventID:     uuid.New().String(),
		FileID:      fileID,
		UserID:      userID,
		FileName:    fileName,
		FileSize:    fileSize,
		ContentType: contentType,
		Action:      "upload",
		Status:      "success",
		Timestamp:   time.Now(),
		Metadata:    metadata,
	}
}

// NewFileDeletedEvent creates a new file deletion event
func NewFileDeletedEvent(fileID, userID, fileName, metadata string) *FileDeletedEvent {
	return &FileDeletedEvent{
		EventID:   uuid.New().String(),
		FileID:    fileID,
		UserID:    userID,
		FileName:  fileName,
		Action:    "delete",
		Status:    "success",
		Timestamp: time.Now(),
		Metadata:  metadata,
	}
}

// NewFileDownloadedEvent creates a new file download event
func NewFileDownloadedEvent(fileID, userID, fileName, metadata string) *FileDownloadedEvent {
	return &FileDownloadedEvent{
		EventID:   uuid.New().String(),
		FileID:    fileID,
		UserID:    userID,
		FileName:  fileName,
		Action:    "download",
		Status:    "success",
		Timestamp: time.Now(),
		Metadata:  metadata,
	}
}

// NewFileVersionedEvent creates a new file version event
func NewFileVersionedEvent(fileID, userID, fileName, contentType, storagePath, checksum, metadata string, fileSize int64, version int) *FileVersionedEvent {
	return &FileVersionedEvent{
		EventID:     uuid.New().String(),
		FileID:      fileID,
		UserID:      userID,
		Version:     version,
		FileName:    fileName,
		FileSize:    fileSize,
		ContentType: contentType,
		StoragePath: storagePath,
		Checksum:    checksum,
		Action:      "version_created",
		Status:      "success",
		Timestamp:   time.Now(),
		Metadata:    metadata,
	}
}
