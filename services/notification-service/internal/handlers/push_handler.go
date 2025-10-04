package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/models"
)

// PushHandler handles push notifications
type PushHandler struct {
	fcmServerKey string
	fcmProjectID string
	apiURL       string
	httpClient   *http.Client
	logger       *logrus.Logger
}

// FCMRequest represents FCM notification request
type FCMRequest struct {
	To           string                 `json:"to"`
	Notification FCMNotification        `json:"notification"`
	Data         map[string]interface{} `json:"data,omitempty"`
	Priority     string                 `json:"priority,omitempty"`
}

// FCMNotification represents FCM notification payload
type FCMNotification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Icon  string `json:"icon,omitempty"`
	Sound string `json:"sound,omitempty"`
	Badge string `json:"badge,omitempty"`
}

// FCMResponse represents FCM response
type FCMResponse struct {
	Success int `json:"success"`
	Failure int `json:"failure"`
	Results []struct {
		MessageID string `json:"message_id,omitempty"`
		Error     string `json:"error,omitempty"`
	} `json:"results"`
}

// NewPushHandler creates a new push notification handler
func NewPushHandler(fcmServerKey, fcmProjectID string, logger *logrus.Logger) *PushHandler {
	return &PushHandler{
		fcmServerKey: fcmServerKey,
		fcmProjectID: fcmProjectID,
		apiURL:       "https://fcm.googleapis.com/fcm/send",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// Send sends a push notification
func (h *PushHandler) Send(ctx context.Context, req *models.NotificationRequest) (*models.NotificationResponse, error) {
	start := time.Now()

	// Validate request
	if err := h.Validate(req); err != nil {
		return &models.NotificationResponse{
			Status:   models.StatusFailed,
			Channel:  models.ChannelPush,
			Error:    err.Error(),
			Duration: time.Since(start).Milliseconds(),
		}, err
	}

	// Get user push token from metadata
	pushToken := h.getUserPushToken(req)
	if pushToken == "" {
		return &models.NotificationResponse{
			Status:   models.StatusFailed,
			Channel:  models.ChannelPush,
			Error:    "user push token not found",
			Duration: time.Since(start).Milliseconds(),
		}, fmt.Errorf("user push token not found")
	}

	// Create FCM request
	fcmReq := h.createFCMRequest(req, pushToken)

	// Send push notification
	err := h.sendFCM(ctx, fcmReq)

	response := &models.NotificationResponse{
		Channel:  models.ChannelPush,
		Duration: time.Since(start).Milliseconds(),
	}

	if err != nil {
		response.Status = models.StatusFailed
		response.Error = err.Error()
		h.logger.WithError(err).WithFields(logrus.Fields{
			"user_id": req.UserID,
			"channel": "push",
		}).Error("Failed to send push notification")
	} else {
		response.Status = models.StatusSent
		now := time.Now()
		response.SentAt = &now
		h.logger.WithFields(logrus.Fields{
			"user_id": req.UserID,
			"channel": "push",
		}).Info("Push notification sent successfully")
	}

	return response, err
}

// Validate validates the notification request
func (h *PushHandler) Validate(req *models.NotificationRequest) error {
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
func (h *PushHandler) GetName() string {
	return "push"
}

// IsEnabled checks if the handler is enabled
func (h *PushHandler) IsEnabled() bool {
	return h.fcmServerKey != "" && h.fcmProjectID != ""
}

// getUserPushToken gets the user's push token
func (h *PushHandler) getUserPushToken(req *models.NotificationRequest) string {
	// Try to get push token from metadata
	if token, ok := req.Metadata["push_token"].(string); ok && token != "" {
		return token
	}

	// Try to get push token from user preferences (would need to be passed in)
	// For now, return empty string
	return ""
}

// createFCMRequest creates the FCM request
func (h *PushHandler) createFCMRequest(req *models.NotificationRequest, pushToken string) *FCMRequest {
	fcmReq := &FCMRequest{
		To: pushToken,
		Notification: FCMNotification{
			Title: req.Title,
			Body:  req.Message,
			Sound: "default",
		},
		Data: map[string]interface{}{
			"event_type": string(req.EventType),
			"user_id":    req.UserID,
			"priority":   string(req.Priority),
		},
		Priority: "high",
	}

	// Add metadata to data payload
	for key, value := range req.Metadata {
		fcmReq.Data[key] = value
	}

	// Add link if provided
	if link, ok := req.Metadata["link"].(string); ok && link != "" {
		fcmReq.Data["link"] = link
	}

	return fcmReq
}

// sendFCM sends the push notification via FCM
func (h *PushHandler) sendFCM(ctx context.Context, fcmReq *FCMRequest) error {
	// Marshal request to JSON
	jsonData, err := json.Marshal(fcmReq)
	if err != nil {
		return fmt.Errorf("failed to marshal FCM request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", h.apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "key="+h.fcmServerKey)

	// Send request
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send FCM request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var fcmResp FCMResponse
	if err := json.NewDecoder(resp.Body).Decode(&fcmResp); err != nil {
		return fmt.Errorf("failed to parse FCM response: %w", err)
	}

	// Check if any notifications were sent successfully
	if fcmResp.Success == 0 {
		if len(fcmResp.Results) > 0 {
			return fmt.Errorf("FCM error: %s", fcmResp.Results[0].Error)
		}
		return fmt.Errorf("FCM request failed")
	}

	return nil
}

// TestConnection tests the FCM connection
func (h *PushHandler) TestConnection(ctx context.Context) error {
	if !h.IsEnabled() {
		return fmt.Errorf("push handler is not enabled")
	}

	// Test by sending a test notification to a test token
	testToken := "test_token"
	fcmReq := &FCMRequest{
		To: testToken,
		Notification: FCMNotification{
			Title: "Test",
			Body:  "This is a test notification",
		},
		Data: map[string]interface{}{
			"test": true,
		},
	}

	err := h.sendFCM(ctx, fcmReq)

	// We expect this to fail with invalid token, but we can check if it's a connection error
	if err != nil && !contains(err.Error(), "InvalidRegistration") {
		return fmt.Errorf("FCM connection test failed: %w", err)
	}

	return nil
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}

// MockPushHandler is a mock implementation for testing
type MockPushHandler struct {
	enabled bool
	logger  *logrus.Logger
}

// NewMockPushHandler creates a new mock push handler
func NewMockPushHandler(enabled bool, logger *logrus.Logger) *MockPushHandler {
	return &MockPushHandler{
		enabled: enabled,
		logger:  logger,
	}
}

// Send sends a mock push notification
func (h *MockPushHandler) Send(ctx context.Context, req *models.NotificationRequest) (*models.NotificationResponse, error) {
	start := time.Now()

	if !h.enabled {
		return &models.NotificationResponse{
			Status:   models.StatusFailed,
			Channel:  models.ChannelPush,
			Error:    "Push handler is disabled",
			Duration: time.Since(start).Milliseconds(),
		}, fmt.Errorf("push handler is disabled")
	}

	// Simulate processing time
	time.Sleep(100 * time.Millisecond)

	// Simulate occasional failures for testing
	if req.UserID == "test-fail" {
		return &models.NotificationResponse{
			Status:   models.StatusFailed,
			Channel:  models.ChannelPush,
			Error:    "mock push failure",
			Duration: time.Since(start).Milliseconds(),
		}, fmt.Errorf("mock push failure")
	}

	h.logger.WithFields(logrus.Fields{
		"user_id": req.UserID,
		"title":   req.Title,
		"message": req.Message,
	}).Info("Mock push notification sent")

	return &models.NotificationResponse{
		Status:   models.StatusSent,
		Channel:  models.ChannelPush,
		Duration: time.Since(start).Milliseconds(),
	}, nil
}

// Validate validates the notification request
func (h *MockPushHandler) Validate(req *models.NotificationRequest) error {
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
func (h *MockPushHandler) GetName() string {
	return "push"
}

// IsEnabled checks if the handler is enabled
func (h *MockPushHandler) IsEnabled() bool {
	return h.enabled
}

// TestConnection tests the push handler connection
func (h *MockPushHandler) TestConnection(ctx context.Context) error {
	if !h.enabled {
		return fmt.Errorf("push handler is disabled")
	}

	// Mock connection test - always succeeds
	h.logger.Info("Mock push connection test successful")
	return nil
}
