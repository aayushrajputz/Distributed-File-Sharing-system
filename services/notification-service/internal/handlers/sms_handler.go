package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/models"
)

// SMSHandler handles SMS notifications
type SMSHandler struct {
	accountSID  string
	authToken   string
	phoneNumber string
	apiURL      string
	httpClient  *http.Client
	logger      *logrus.Logger
}

// TwilioResponse represents Twilio API response
type TwilioResponse struct {
	SID          string `json:"sid"`
	Status       string `json:"status"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// NewSMSHandler creates a new SMS handler
func NewSMSHandler(accountSID, authToken, phoneNumber string, logger *logrus.Logger) *SMSHandler {
	return &SMSHandler{
		accountSID:  accountSID,
		authToken:   authToken,
		phoneNumber: phoneNumber,
		apiURL:      "https://api.twilio.com/2010-04-01",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// Send sends an SMS notification
func (h *SMSHandler) Send(ctx context.Context, req *models.NotificationRequest) (*models.NotificationResponse, error) {
	start := time.Now()

	// Validate request
	if err := h.Validate(req); err != nil {
		return &models.NotificationResponse{
			Status:   models.StatusFailed,
			Channel:  models.ChannelSMS,
			Error:    err.Error(),
			Duration: time.Since(start).Milliseconds(),
		}, err
	}

	// Get user phone number from metadata
	phoneNumber := h.getUserPhoneNumber(req)
	if phoneNumber == "" {
		return &models.NotificationResponse{
			Status:   models.StatusFailed,
			Channel:  models.ChannelSMS,
			Error:    "user phone number not found",
			Duration: time.Since(start).Milliseconds(),
		}, fmt.Errorf("user phone number not found")
	}

	// Create SMS message
	message := h.createSMSMessage(req)

	// Send SMS
	err := h.sendSMS(ctx, phoneNumber, message)

	response := &models.NotificationResponse{
		Channel:  models.ChannelSMS,
		Duration: time.Since(start).Milliseconds(),
	}

	if err != nil {
		response.Status = models.StatusFailed
		response.Error = err.Error()
		h.logger.WithError(err).WithFields(logrus.Fields{
			"user_id": req.UserID,
			"channel": "sms",
		}).Error("Failed to send SMS notification")
	} else {
		response.Status = models.StatusSent
		now := time.Now()
		response.SentAt = &now
		h.logger.WithFields(logrus.Fields{
			"user_id":      req.UserID,
			"channel":      "sms",
			"phone_number": phoneNumber,
		}).Info("SMS notification sent successfully")
	}

	return response, err
}

// Validate validates the notification request
func (h *SMSHandler) Validate(req *models.NotificationRequest) error {
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
func (h *SMSHandler) GetName() string {
	return "sms"
}

// IsEnabled checks if the handler is enabled
func (h *SMSHandler) IsEnabled() bool {
	return h.accountSID != "" && h.authToken != "" && h.phoneNumber != ""
}

// getUserPhoneNumber gets the user's phone number
func (h *SMSHandler) getUserPhoneNumber(req *models.NotificationRequest) string {
	// Try to get phone number from metadata
	if phone, ok := req.Metadata["phone_number"].(string); ok && phone != "" {
		return phone
	}

	// Try to get phone number from user preferences (would need to be passed in)
	// For now, return empty string
	return ""
}

// createSMSMessage creates the SMS message
func (h *SMSHandler) createSMSMessage(req *models.NotificationRequest) string {
	// SMS has character limits, so we need to be concise
	message := fmt.Sprintf("%s: %s", req.Title, req.Message)

	// Truncate if too long (SMS limit is typically 160 characters)
	maxLength := 150 // Leave some room for link
	if len(message) > maxLength {
		message = message[:maxLength-3] + "..."
	}

	// Add link if provided and there's space
	if link, ok := req.Metadata["link"].(string); ok && link != "" && len(message) < 120 {
		shortLink := h.shortenLink(link)
		message += fmt.Sprintf(" %s", shortLink)
	}

	return message
}

// shortenLink creates a short version of the link for SMS
func (h *SMSHandler) shortenLink(link string) string {
	// In a real implementation, you would use a URL shortener service
	// For now, we'll just truncate it
	if len(link) > 20 {
		return link[:17] + "..."
	}
	return link
}

// sendSMS sends the SMS using Twilio API
func (h *SMSHandler) sendSMS(ctx context.Context, toPhoneNumber, message string) error {
	// Create request URL
	url := fmt.Sprintf("%s/Accounts/%s/Messages.json", h.apiURL, h.accountSID)

	// Create form data
	data := map[string]string{
		"From": h.phoneNumber,
		"To":   toPhoneNumber,
		"Body": message,
	}

	// Convert to form data
	formData := make([]string, 0, len(data))
	for key, value := range data {
		formData = append(formData, fmt.Sprintf("%s=%s", key, value))
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(strings.Join(formData, "&")))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(h.accountSID, h.authToken)

	// Send request
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send SMS: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusCreated {
		var twilioResp TwilioResponse
		if err := json.NewDecoder(resp.Body).Decode(&twilioResp); err != nil {
			return fmt.Errorf("SMS failed with status %d", resp.StatusCode)
		}
		return fmt.Errorf("SMS failed: %s (code: %s)", twilioResp.ErrorMessage, twilioResp.ErrorCode)
	}

	// Parse response
	var twilioResp TwilioResponse
	if err := json.NewDecoder(resp.Body).Decode(&twilioResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if twilioResp.Status != "queued" && twilioResp.Status != "sent" {
		return fmt.Errorf("SMS not queued: %s", twilioResp.Status)
	}

	return nil
}

// TestConnection tests the Twilio connection
func (h *SMSHandler) TestConnection(ctx context.Context) error {
	if !h.IsEnabled() {
		return fmt.Errorf("SMS handler is not enabled")
	}

	// Test by trying to get account info
	url := fmt.Sprintf("%s/Accounts/%s.json", h.apiURL, h.accountSID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(h.accountSID, h.authToken)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to test connection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("connection test failed with status %d", resp.StatusCode)
	}

	return nil
}

// MockSMSHandler is a mock implementation for testing
type MockSMSHandler struct {
	enabled bool
	logger  *logrus.Logger
}

// NewMockSMSHandler creates a new mock SMS handler
func NewMockSMSHandler(enabled bool, logger *logrus.Logger) *MockSMSHandler {
	return &MockSMSHandler{
		enabled: enabled,
		logger:  logger,
	}
}

// Send sends a mock SMS notification
func (h *MockSMSHandler) Send(ctx context.Context, req *models.NotificationRequest) (*models.NotificationResponse, error) {
	start := time.Now()

	if !h.enabled {
		return &models.NotificationResponse{
			Status:   models.StatusFailed,
			Channel:  models.ChannelSMS,
			Error:    "SMS handler is disabled",
			Duration: time.Since(start).Milliseconds(),
		}, fmt.Errorf("SMS handler is disabled")
	}

	// Simulate processing time
	time.Sleep(100 * time.Millisecond)

	// Simulate occasional failures for testing
	if req.UserID == "test-fail" {
		return &models.NotificationResponse{
			Status:   models.StatusFailed,
			Channel:  models.ChannelSMS,
			Error:    "mock SMS failure",
			Duration: time.Since(start).Milliseconds(),
		}, fmt.Errorf("mock SMS failure")
	}

	h.logger.WithFields(logrus.Fields{
		"user_id": req.UserID,
		"title":   req.Title,
		"message": req.Message,
	}).Info("Mock SMS notification sent")

	return &models.NotificationResponse{
		Status:   models.StatusSent,
		Channel:  models.ChannelSMS,
		Duration: time.Since(start).Milliseconds(),
	}, nil
}

// Validate validates the notification request
func (h *MockSMSHandler) Validate(req *models.NotificationRequest) error {
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
func (h *MockSMSHandler) GetName() string {
	return "sms"
}

// IsEnabled checks if the handler is enabled
func (h *MockSMSHandler) IsEnabled() bool {
	return h.enabled
}

// TestConnection tests the SMS handler connection
func (h *MockSMSHandler) TestConnection(ctx context.Context) error {
	if !h.enabled {
		return fmt.Errorf("SMS handler is disabled")
	}

	// Mock connection test - always succeeds
	h.logger.Info("Mock SMS connection test successful")
	return nil
}
