package channels

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// InAppChannel implements in-app notifications (stored in database)
type InAppChannel struct {
	enabled bool
	logger  *logrus.Logger
}

// NewInAppChannel creates a new in-app notification channel
func NewInAppChannel(logger *logrus.Logger) *InAppChannel {
	return &InAppChannel{
		enabled: true, // Always enabled as it's the primary storage
		logger:  logger,
	}
}

// Send sends an in-app notification (stores in database)
func (i *InAppChannel) Send(ctx context.Context, notification *NotificationRequest) (*DeliveryResult, error) {
	if !i.enabled {
		return &DeliveryResult{
			Channel:     "inapp",
			Success:     false,
			ErrorMessage: "in-app channel is disabled",
			DeliveredAt: time.Now(),
		}, fmt.Errorf("in-app channel is disabled")
	}

	if err := i.Validate(notification); err != nil {
		return &DeliveryResult{
			Channel:     "inapp",
			Success:     false,
			ErrorMessage: err.Error(),
			DeliveredAt: time.Now(),
		}, err
	}

	// In-app notifications are typically stored in the database
	// This is handled by the main notification service
	// Here we just simulate success as the actual storage happens elsewhere
	
	i.logger.WithFields(logrus.Fields{
		"user_id": notification.UserID,
		"type":    notification.Type,
		"title":   notification.Title,
	}).Info("In-app notification processed")

	return &DeliveryResult{
		Channel:     "inapp",
		Success:     true,
		DeliveredAt: time.Now(),
		MessageID:   fmt.Sprintf("inapp_%d", time.Now().UnixNano()),
	}, nil
}

// GetName returns the channel name
func (i *InAppChannel) GetName() string {
	return "inapp"
}

// IsEnabled checks if the channel is enabled
func (i *InAppChannel) IsEnabled() bool {
	return i.enabled
}

// Validate validates the notification request for in-app channel
func (i *InAppChannel) Validate(req *NotificationRequest) error {
	if req.UserID == "" {
		return fmt.Errorf("user ID is required for in-app notifications")
	}
	
	if req.Title == "" {
		return fmt.Errorf("title is required for in-app notifications")
	}
	
	if req.Body == "" {
		return fmt.Errorf("body is required for in-app notifications")
	}
	
	return nil
}

