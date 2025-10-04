package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/models"
	"github.com/sirupsen/logrus"
)

// InAppHandler handles in-app notifications
type InAppHandler struct {
	enabled bool
	logger  *logrus.Logger
}

// NewInAppHandler creates a new in-app notification handler
func NewInAppHandler(enabled bool, logger *logrus.Logger) *InAppHandler {
	return &InAppHandler{
		enabled: enabled,
		logger:  logger,
	}
}

// Send sends an in-app notification
func (h *InAppHandler) Send(ctx context.Context, req *models.NotificationRequest) (*models.NotificationResponse, error) {
	start := time.Now()
	
	// Validate request
	if err := h.Validate(req); err != nil {
		return &models.NotificationResponse{
			Status:   models.StatusFailed,
			Channel:  models.ChannelInApp,
			Error:    err.Error(),
			Duration: time.Since(start).Milliseconds(),
		}, err
	}

	// In-app notifications are typically stored in the database
	// This handler just simulates the processing
	// The actual storage happens in the notification service
	
	// Simulate processing time
	time.Sleep(50 * time.Millisecond)
	
	h.logger.WithFields(logrus.Fields{
		"user_id":    req.UserID,
		"event_type": req.EventType,
		"title":      req.Title,
		"priority":   req.Priority,
	}).Info("In-app notification processed")

	return &models.NotificationResponse{
		Status:   models.StatusSent,
		Channel:  models.ChannelInApp,
		Duration: time.Since(start).Milliseconds(),
	}, nil
}

// Validate validates the notification request
func (h *InAppHandler) Validate(req *models.NotificationRequest) error {
	if req.UserID == "" {
		return fmt.Errorf("user ID is required")
	}
	
	if req.Title == "" {
		return fmt.Errorf("title is required")
	}
	
	if req.Message == "" {
		return fmt.Errorf("message is required")
	}

	return nil
}

// GetName returns the handler name
func (h *InAppHandler) GetName() string {
	return "inapp"
}

// IsEnabled checks if the handler is enabled
func (h *InAppHandler) IsEnabled() bool {
	return h.enabled
}

// TestConnection tests the in-app handler (always succeeds)
func (h *InAppHandler) TestConnection(ctx context.Context) error {
	if !h.enabled {
		return fmt.Errorf("in-app handler is disabled")
	}
	
	// In-app handler doesn't require external connections
	return nil
}
