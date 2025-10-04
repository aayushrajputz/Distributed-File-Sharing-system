package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// NotificationStatus represents the status of a notification
type NotificationStatus string

const (
	StatusPending NotificationStatus = "pending"
	StatusSent    NotificationStatus = "sent"
	StatusFailed  NotificationStatus = "failed"
	StatusRead    NotificationStatus = "read"
)

// NotificationChannel represents the delivery channel
type NotificationChannel string

const (
	ChannelEmail   NotificationChannel = "email"
	ChannelSMS     NotificationChannel = "sms"
	ChannelPush    NotificationChannel = "push"
	ChannelInApp   NotificationChannel = "inapp"
	ChannelWebSocket NotificationChannel = "websocket"
)

// EventType represents the type of event that triggered the notification
type EventType string

const (
	EventTypeFileUploaded     EventType = "file.uploaded"
	EventTypeFileUploadFailed EventType = "file.upload.failed"
	EventTypeFileDeleted      EventType = "file.deleted"
	EventTypeFileShared       EventType = "file.shared"
	EventTypeQuotaWarning80   EventType = "quota.warning.80"
	EventTypeQuotaWarning90   EventType = "quota.warning.90"
	EventTypeQuotaExceeded    EventType = "quota.exceeded"
	EventTypeSecurityAlert    EventType = "security.alert"
	EventTypeSystemMaintenance EventType = "system.maintenance"
)

// Priority represents notification priority
type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityNormal   Priority = "normal"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

// Notification represents a notification in the system
type Notification struct {
	ID           primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	UserID       string               `bson:"user_id" json:"user_id"`
	EventType    EventType            `bson:"event_type" json:"event_type"`
	Channel      NotificationChannel  `bson:"channel" json:"channel"`
	Title        string               `bson:"title" json:"title"`
	Message      string               `bson:"message" json:"message"`
	Status       NotificationStatus   `bson:"status" json:"status"`
	Priority     Priority             `bson:"priority" json:"priority"`
	TemplateID   string               `bson:"template_id,omitempty" json:"template_id,omitempty"`
	Metadata     map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
	SentAt       *time.Time           `bson:"sent_at,omitempty" json:"sent_at,omitempty"`
	ReadAt       *time.Time           `bson:"read_at,omitempty" json:"read_at,omitempty"`
	CreatedAt    time.Time            `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time            `bson:"updated_at" json:"updated_at"`
	
	// Retry information
	RetryCount   int                  `bson:"retry_count" json:"retry_count"`
	LastRetryAt  *time.Time           `bson:"last_retry_at,omitempty" json:"last_retry_at,omitempty"`
	NextRetryAt  *time.Time           `bson:"next_retry_at,omitempty" json:"next_retry_at,omitempty"`
	ErrorReason  string               `bson:"error_reason,omitempty" json:"error_reason,omitempty"`
	
	// Delivery tracking
	DeliveryAttempts []DeliveryAttempt `bson:"delivery_attempts,omitempty" json:"delivery_attempts,omitempty"`
}

// DeliveryAttempt represents a single delivery attempt
type DeliveryAttempt struct {
	AttemptedAt time.Time `bson:"attempted_at" json:"attempted_at"`
	Success     bool      `bson:"success" json:"success"`
	ErrorReason string    `bson:"error_reason,omitempty" json:"error_reason,omitempty"`
	Duration    int64     `bson:"duration_ms" json:"duration_ms"` // Duration in milliseconds
}

// NotificationTemplate represents a notification template
type NotificationTemplate struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TemplateID      string             `bson:"template_id" json:"template_id"`
	EventType       EventType          `bson:"event_type" json:"event_type"`
	Channel         NotificationChannel `bson:"channel" json:"channel"`
	SubjectTemplate string             `bson:"subject_template" json:"subject_template"`
	BodyTemplate    string             `bson:"body_template" json:"body_template"`
	IsActive        bool               `bson:"is_active" json:"is_active"`
	CreatedAt       time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time          `bson:"updated_at" json:"updated_at"`
}

// UserNotificationPreferences represents user notification preferences
type UserNotificationPreferences struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID            string             `bson:"user_id" json:"user_id"`
	EmailEnabled      bool               `bson:"email_enabled" json:"email_enabled"`
	SMSEnabled        bool               `bson:"sms_enabled" json:"sms_enabled"`
	PushEnabled       bool               `bson:"push_enabled" json:"push_enabled"`
	InAppEnabled      bool               `bson:"in_app_enabled" json:"in_app_enabled"`
	WebSocketEnabled  bool               `bson:"websocket_enabled" json:"websocket_enabled"`
	
	// Contact information
	Email             string             `bson:"email,omitempty" json:"email,omitempty"`
	PhoneNumber       string             `bson:"phone_number,omitempty" json:"phone_number,omitempty"`
	PushToken         string             `bson:"push_token,omitempty" json:"push_token,omitempty"`
	
	// Quiet hours (24-hour format)
	QuietHoursStart   string             `bson:"quiet_hours_start,omitempty" json:"quiet_hours_start,omitempty"` // "22:00"
	QuietHoursEnd     string             `bson:"quiet_hours_end,omitempty" json:"quiet_hours_end,omitempty"`     // "08:00"
	QuietHoursEnabled bool               `bson:"quiet_hours_enabled" json:"quiet_hours_enabled"`
	
	// Event subscriptions
	EventSubscriptions []EventType       `bson:"event_subscriptions" json:"event_subscriptions"`
	
	// Channel priorities for fallback
	ChannelPriorities map[EventType][]NotificationChannel `bson:"channel_priorities,omitempty" json:"channel_priorities,omitempty"`
	
	CreatedAt         time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt         time.Time          `bson:"updated_at" json:"updated_at"`
}

// DeadLetterQueueEntry represents a failed notification in the DLQ
type DeadLetterQueueEntry struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	OriginalEvent   map[string]interface{} `bson:"original_event" json:"original_event"`
	NotificationID  primitive.ObjectID `bson:"notification_id" json:"notification_id"`
	UserID          string             `bson:"user_id" json:"user_id"`
	EventType       EventType          `bson:"event_type" json:"event_type"`
	FailureReason   string             `bson:"failure_reason" json:"failure_reason"`
	RetryHistory    []RetryAttempt     `bson:"retry_history" json:"retry_history"`
	MaxRetries      int                `bson:"max_retries" json:"max_retries"`
	CreatedAt       time.Time          `bson:"created_at" json:"created_at"`
	LastRetryAt     *time.Time         `bson:"last_retry_at,omitempty" json:"last_retry_at,omitempty"`
	NextRetryAt     *time.Time         `bson:"next_retry_at,omitempty" json:"next_retry_at,omitempty"`
	IsProcessed     bool               `bson:"is_processed" json:"is_processed"`
	ProcessedAt     *time.Time         `bson:"processed_at,omitempty" json:"processed_at,omitempty"`
}

// RetryAttempt represents a retry attempt
type RetryAttempt struct {
	AttemptedAt time.Time `bson:"attempted_at" json:"attempted_at"`
	Success     bool      `bson:"success" json:"success"`
	ErrorReason string    `bson:"error_reason,omitempty" json:"error_reason,omitempty"`
	Duration    int64     `bson:"duration_ms" json:"duration_ms"`
}

// BatchNotification represents a batched notification
type BatchNotification struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID       string             `bson:"user_id" json:"user_id"`
	EventType    EventType          `bson:"event_type" json:"event_type"`
	Channel      NotificationChannel `bson:"channel" json:"channel"`
	Title        string             `bson:"title" json:"title"`
	Message      string             `bson:"message" json:"message"`
	Count        int                `bson:"count" json:"count"`
	Items        []BatchItem        `bson:"items" json:"items"`
	Status       NotificationStatus `bson:"status" json:"status"`
	SentAt       *time.Time         `bson:"sent_at,omitempty" json:"sent_at,omitempty"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
}

// BatchItem represents an item in a batched notification
type BatchItem struct {
	FileName    string                 `bson:"file_name" json:"file_name"`
	FileSize    int64                  `bson:"file_size" json:"file_size"`
	Success     bool                   `bson:"success" json:"success"`
	ErrorReason string                 `bson:"error_reason,omitempty" json:"error_reason,omitempty"`
	Metadata    map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
	Timestamp   time.Time              `bson:"timestamp" json:"timestamp"`
}

// KafkaFileEvent represents the Kafka event from file service
type KafkaFileEvent struct {
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

// TemplateData represents data available in notification templates
type TemplateData struct {
	UserName     string                 `json:"user_name"`
	FileName     string                 `json:"file_name"`
	FileSize     int64                  `json:"file_size"`
	FileSizeFormatted string            `json:"file_size_formatted"`
	Timestamp    time.Time              `json:"timestamp"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	Count        int                    `json:"count,omitempty"`
	Items        []BatchItem            `json:"items,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// NotificationRequest represents a request to send a notification
type NotificationRequest struct {
	UserID       string                 `json:"user_id"`
	EventType    EventType              `json:"event_type"`
	Channel      NotificationChannel    `json:"channel"`
	Title        string                 `json:"title"`
	Message      string                 `json:"message"`
	Priority     Priority               `json:"priority"`
	TemplateID   string                 `json:"template_id,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	BypassBatching bool                 `json:"bypass_batching,omitempty"`
	BypassQuietHours bool               `json:"bypass_quiet_hours,omitempty"`
}

// NotificationResponse represents the response after sending a notification
type NotificationResponse struct {
	ID        string                 `json:"id"`
	Status    NotificationStatus     `json:"status"`
	Channel   NotificationChannel    `json:"channel"`
	SentAt    *time.Time             `json:"sent_at,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Duration  int64                  `json:"duration_ms"`
}

// GetDefaultPreferences returns default user preferences
func GetDefaultPreferences(userID string) *UserNotificationPreferences {
	return &UserNotificationPreferences{
		UserID:            userID,
		EmailEnabled:      true,
		SMSEnabled:        false,
		PushEnabled:       false,
		InAppEnabled:      true,
		WebSocketEnabled:  true,
		QuietHoursEnabled: false,
		EventSubscriptions: []EventType{
			EventTypeFileUploaded,
			EventTypeFileUploadFailed,
			EventTypeFileDeleted,
			EventTypeFileShared,
			EventTypeQuotaWarning80,
			EventTypeQuotaWarning90,
			EventTypeQuotaExceeded,
			EventTypeSecurityAlert,
		},
		ChannelPriorities: map[EventType][]NotificationChannel{
			EventTypeFileUploaded:     {ChannelInApp, ChannelWebSocket, ChannelEmail},
			EventTypeFileUploadFailed: {ChannelInApp, ChannelWebSocket, ChannelEmail},
			EventTypeFileDeleted:      {ChannelInApp, ChannelWebSocket},
			EventTypeFileShared:       {ChannelInApp, ChannelWebSocket, ChannelEmail},
			EventTypeQuotaWarning80:   {ChannelInApp, ChannelWebSocket, ChannelEmail},
			EventTypeQuotaWarning90:   {ChannelInApp, ChannelWebSocket, ChannelEmail},
			EventTypeQuotaExceeded:    {ChannelInApp, ChannelWebSocket, ChannelEmail, ChannelSMS},
			EventTypeSecurityAlert:    {ChannelInApp, ChannelWebSocket, ChannelEmail, ChannelSMS, ChannelPush},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}