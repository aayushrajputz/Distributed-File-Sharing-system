package cassandra

import (
	"context"
	"fmt"
	"time"

	"github.com/gocql/gocql"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/metrics"
)

// FileEvent represents a file operation event
type FileEvent struct {
	UserID   string    `json:"user_id"`
	EventTS  time.Time `json:"event_ts"`
	EventID  uuid.UUID `json:"event_id"`
	FileID   uuid.UUID `json:"file_id"`
	Action   string    `json:"action"` // 'upload', 'delete', 'restore', 'permanent_delete', 'download'
	Status   string    `json:"status"` // 'success', 'failed', 'pending'
	FileName string    `json:"file_name"`
	FileSize int64     `json:"file_size"`
	Metadata string    `json:"metadata"` // JSON string for extensibility
}

// FileVersion represents a file version
type FileVersion struct {
	FileID      uuid.UUID `json:"file_id"`
	Version     int       `json:"version"`
	FileName    string    `json:"file_name"`
	FileSize    int64     `json:"file_size"`
	ContentType string    `json:"content_type"`
	StoragePath string    `json:"storage_path"`
	Checksum    string    `json:"checksum"`
	UploadedAt  time.Time `json:"uploaded_at"`
	UploadedBy  string    `json:"uploaded_by"` // user_id
	Metadata    string    `json:"metadata"`    // JSON string
}

// Repository handles Cassandra operations for file events and versions
type Repository struct {
	client Client
	logger *logrus.Logger
}

// NewRepository creates a new Cassandra repository
func NewRepository(client Client, logger *logrus.Logger) *Repository {
	return &Repository{
		client: client,
		logger: logger,
	}
}

// LogFileEvent logs a file operation event to Cassandra
func (r *Repository) LogFileEvent(ctx context.Context, event *FileEvent) error {
	start := time.Now()
	defer func() {
		metrics.RecordCassandraQueryDuration("log_file_event", time.Since(start).Seconds())
	}()

	query := r.client.GetSession().Query(`
		INSERT INTO file_events (user_id, event_ts, event_id, file_id, action, status, file_name, file_size, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, event.UserID, event.EventTS, event.EventID, event.FileID, event.Action, event.Status, event.FileName, event.FileSize, event.Metadata)

	if err := query.WithContext(ctx).Exec(); err != nil {
		metrics.RecordCassandraWrite("log_file_event", "error")
		metrics.RecordCassandraError("log_file_event", "execution_failed")
		r.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":  event.UserID,
			"file_id":  event.FileID,
			"action":   event.Action,
			"event_id": event.EventID,
		}).Error("Failed to log file event to Cassandra")
		return fmt.Errorf("failed to log file event: %w", err)
	}

	metrics.RecordCassandraWrite("log_file_event", "success")
	r.logger.WithFields(logrus.Fields{
		"user_id":  event.UserID,
		"file_id":  event.FileID,
		"action":   event.Action,
		"event_id": event.EventID,
	}).Debug("File event logged to Cassandra")

	return nil
}

// AddFileVersion adds a new file version to Cassandra
func (r *Repository) AddFileVersion(ctx context.Context, version *FileVersion) error {
	start := time.Now()
	defer func() {
		metrics.RecordCassandraQueryDuration("add_file_version", time.Since(start).Seconds())
	}()

	query := r.client.GetSession().Query(`
		INSERT INTO file_versions (file_id, version, file_name, file_size, content_type, storage_path, checksum, uploaded_at, uploaded_by, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, version.FileID, version.Version, version.FileName, version.FileSize, version.ContentType, version.StoragePath, version.Checksum, version.UploadedAt, version.UploadedBy, version.Metadata)

	if err := query.WithContext(ctx).Exec(); err != nil {
		metrics.RecordCassandraWrite("add_file_version", "error")
		metrics.RecordCassandraError("add_file_version", "execution_failed")
		r.logger.WithError(err).WithFields(logrus.Fields{
			"file_id":     version.FileID,
			"version":     version.Version,
			"uploaded_by": version.UploadedBy,
		}).Error("Failed to add file version to Cassandra")
		return fmt.Errorf("failed to add file version: %w", err)
	}

	metrics.RecordCassandraWrite("add_file_version", "success")
	r.logger.WithFields(logrus.Fields{
		"file_id":     version.FileID,
		"version":     version.Version,
		"uploaded_by": version.UploadedBy,
	}).Debug("File version added to Cassandra")

	return nil
}

// GetFileEvents retrieves file events for a user within a time range
func (r *Repository) GetFileEvents(ctx context.Context, userID string, fromTS, toTS time.Time, limit int) ([]*FileEvent, error) {
	start := time.Now()
	defer func() {
		metrics.RecordCassandraQueryDuration("get_file_events", time.Since(start).Seconds())
	}()

	var events []*FileEvent

	query := r.client.GetSession().Query(`
		SELECT user_id, event_ts, event_id, file_id, action, status, file_name, file_size, metadata
		FROM file_events
		WHERE user_id = ? AND event_ts >= ? AND event_ts <= ?
		ORDER BY event_ts DESC
		LIMIT ?
	`, userID, fromTS, toTS, limit)

	iter := query.WithContext(ctx).Iter()
	defer iter.Close()

	for {
		event := &FileEvent{}
		if !iter.Scan(&event.UserID, &event.EventTS, &event.EventID, &event.FileID, &event.Action, &event.Status, &event.FileName, &event.FileSize, &event.Metadata) {
			break
		}
		events = append(events, event)
	}

	if err := iter.Close(); err != nil {
		metrics.RecordCassandraError("get_file_events", "iteration_failed")
		r.logger.WithError(err).WithField("user_id", userID).Error("Failed to retrieve file events from Cassandra")
		return nil, fmt.Errorf("failed to retrieve file events: %w", err)
	}

	metrics.RecordCassandraWrite("get_file_events", "success")
	r.logger.WithFields(logrus.Fields{
		"user_id": userID,
		"count":   len(events),
	}).Debug("Retrieved file events from Cassandra")

	return events, nil
}

// GetFileVersions retrieves all versions for a file
func (r *Repository) GetFileVersions(ctx context.Context, fileID uuid.UUID, limit int) ([]*FileVersion, error) {
	start := time.Now()
	defer func() {
		metrics.RecordCassandraQueryDuration("get_file_versions", time.Since(start).Seconds())
	}()

	var versions []*FileVersion

	query := r.client.GetSession().Query(`
		SELECT file_id, version, file_name, file_size, content_type, storage_path, checksum, uploaded_at, uploaded_by, metadata
		FROM file_versions
		WHERE file_id = ?
		ORDER BY version DESC
		LIMIT ?
	`, fileID, limit)

	iter := query.WithContext(ctx).Iter()
	defer iter.Close()

	for {
		version := &FileVersion{}
		if !iter.Scan(&version.FileID, &version.Version, &version.FileName, &version.FileSize, &version.ContentType, &version.StoragePath, &version.Checksum, &version.UploadedAt, &version.UploadedBy, &version.Metadata) {
			break
		}
		versions = append(versions, version)
	}

	if err := iter.Close(); err != nil {
		metrics.RecordCassandraError("get_file_versions", "iteration_failed")
		r.logger.WithError(err).WithField("file_id", fileID).Error("Failed to retrieve file versions from Cassandra")
		return nil, fmt.Errorf("failed to retrieve file versions: %w", err)
	}

	metrics.RecordCassandraWrite("get_file_versions", "success")
	r.logger.WithFields(logrus.Fields{
		"file_id": fileID,
		"count":   len(versions),
	}).Debug("Retrieved file versions from Cassandra")

	return versions, nil
}

// CheckEventExists checks if an event with the given ID already exists (for idempotency)
func (r *Repository) CheckEventExists(ctx context.Context, userID string, eventTS time.Time, eventID uuid.UUID) (bool, error) {
	var existingID uuid.UUID

	query := r.client.GetSession().Query(`
		SELECT event_id FROM file_events 
		WHERE user_id = ? AND event_ts = ? AND event_id = ?
		LIMIT 1
	`, userID, eventTS, eventID)

	if err := query.WithContext(ctx).Scan(&existingID); err != nil {
		if err == gocql.ErrNotFound {
			return false, nil
		}
		r.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":  userID,
			"event_id": eventID,
		}).Error("Failed to check if event exists in Cassandra")
		return false, fmt.Errorf("failed to check event existence: %w", err)
	}

	return true, nil
}

// GetLatestFileVersion gets the latest version number for a file
func (r *Repository) GetLatestFileVersion(ctx context.Context, fileID uuid.UUID) (int, error) {
	var version int

	query := r.client.GetSession().Query(`
		SELECT version FROM file_versions 
		WHERE file_id = ? 
		ORDER BY version DESC 
		LIMIT 1
	`, fileID)

	if err := query.WithContext(ctx).Scan(&version); err != nil {
		if err == gocql.ErrNotFound {
			return 0, nil // No versions exist yet
		}
		r.logger.WithError(err).WithField("file_id", fileID).Error("Failed to get latest file version from Cassandra")
		return 0, fmt.Errorf("failed to get latest file version: %w", err)
	}

	return version, nil
}

// HealthCheck verifies Cassandra connectivity
func (r *Repository) HealthCheck(ctx context.Context) error {
	return r.client.HealthCheck(ctx)
}
