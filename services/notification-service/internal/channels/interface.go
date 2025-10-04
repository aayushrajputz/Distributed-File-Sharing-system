package channels

import (
	"context"
	"time"
)

// NotificationChannel defines the interface for all notification channels
type NotificationChannel interface {
	// Send sends a notification through this channel
	Send(ctx context.Context, notification *NotificationRequest) (*DeliveryResult, error)

	// GetName returns the channel name
	GetName() string

	// IsEnabled checks if the channel is enabled
	IsEnabled() bool

	// Validate validates the notification request for this channel
	Validate(req *NotificationRequest) error
}

// NotificationRequest represents a notification to be sent
type NotificationRequest struct {
	UserID    string            `json:"user_id"`
	Type      string            `json:"type"`
	Title     string            `json:"title"`
	Body      string            `json:"body"`
	Link      string            `json:"link,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Priority  string            `json:"priority"`
	Email     string            `json:"email,omitempty"`
	Phone     string            `json:"phone,omitempty"`
	PushToken string            `json:"push_token,omitempty"`
	Channels  []string          `json:"channels"`
}

// DeliveryResult represents the result of a notification delivery
type DeliveryResult struct {
	Channel      string    `json:"channel"`
	Success      bool      `json:"success"`
	ErrorMessage string    `json:"error_message,omitempty"`
	DeliveredAt  time.Time `json:"delivered_at"`
	MessageID    string    `json:"message_id,omitempty"`
}

// UserPreferences represents user notification preferences
type UserPreferences struct {
	UserID                 string          `bson:"user_id" json:"user_id"`
	EnabledChannels        []string        `bson:"enabled_channels" json:"enabled_channels"`
	ChannelSettings        map[string]bool `bson:"channel_settings" json:"channel_settings"`
	TypeSettings           map[string]bool `bson:"type_settings" json:"type_settings"`
	Email                  string          `bson:"email" json:"email"`
	Phone                  string          `bson:"phone" json:"phone"`
	PushToken              string          `bson:"push_token" json:"push_token"`
	EmailNotifications     bool            `bson:"email_notifications" json:"email_notifications"`
	SMSNotifications       bool            `bson:"sms_notifications" json:"sms_notifications"`
	PushNotifications      bool            `bson:"push_notifications" json:"push_notifications"`
	InAppNotifications     bool            `bson:"in_app_notifications" json:"in_app_notifications"`
	WebSocketNotifications bool            `bson:"websocket_notifications" json:"websocket_notifications"`
	CreatedAt              time.Time       `bson:"created_at" json:"created_at"`
	UpdatedAt              time.Time       `bson:"updated_at" json:"updated_at"`
}

// ChannelManager manages all notification channels
type ChannelManager interface {
	// SendMultiChannel sends notification through multiple channels
	SendMultiChannel(ctx context.Context, req *NotificationRequest) ([]*DeliveryResult, error)

	// SendWithFallback sends notification with fallback mechanism
	SendWithFallback(ctx context.Context, req *NotificationRequest) ([]*DeliveryResult, error)

	// GetChannel returns a specific channel by name
	GetChannel(name string) (NotificationChannel, error)

	// RegisterChannel registers a new notification channel
	RegisterChannel(channel NotificationChannel)

	// GetEnabledChannels returns all enabled channels
	GetEnabledChannels() []NotificationChannel

	// GetChannelStatus returns the status of all channels
	GetChannelStatus() map[string]bool

	// ValidateChannels validates that all specified channels are available
	ValidateChannels(channelNames []string) error
}
