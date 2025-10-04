package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// PushChannel implements push notifications for FCM and APNS
type PushChannel struct {
	fcmServerKey string
	apnsKeyID    string
	apnsKeyPath  string
	apnsBundleID string
	enabled      bool
	logger       *logrus.Logger
	httpClient   *http.Client
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

// NewPushChannel creates a new push notification channel
func NewPushChannel(fcmServerKey, apnsKeyID, apnsKeyPath, apnsBundleID string, logger *logrus.Logger) *PushChannel {
	enabled := fcmServerKey != "" || (apnsKeyID != "" && apnsKeyPath != "")

	return &PushChannel{
		fcmServerKey: fcmServerKey,
		apnsKeyID:    apnsKeyID,
		apnsKeyPath:  apnsKeyPath,
		apnsBundleID: apnsBundleID,
		enabled:      enabled,
		logger:       logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Send sends a push notification
func (p *PushChannel) Send(ctx context.Context, notification *NotificationRequest) (*DeliveryResult, error) {
	if !p.enabled {
		return &DeliveryResult{
			Channel:      "push",
			Success:      false,
			ErrorMessage: "push channel is disabled",
			DeliveredAt:  time.Now(),
		}, fmt.Errorf("push channel is disabled")
	}

	if err := p.Validate(notification); err != nil {
		return &DeliveryResult{
			Channel:      "push",
			Success:      false,
			ErrorMessage: err.Error(),
			DeliveredAt:  time.Now(),
		}, err
	}

	// Try FCM first, then APNS as fallback
	if p.fcmServerKey != "" {
		return p.sendFCM(ctx, notification)
	} else if p.apnsKeyID != "" && p.apnsKeyPath != "" {
		return p.sendAPNS(ctx, notification)
	}

	return &DeliveryResult{
		Channel:      "push",
		Success:      false,
		ErrorMessage: "no push service configured",
		DeliveredAt:  time.Now(),
	}, fmt.Errorf("no push service configured")
}

// sendFCM sends notification via Firebase Cloud Messaging
func (p *PushChannel) sendFCM(ctx context.Context, notification *NotificationRequest) (*DeliveryResult, error) {
	fcmReq := FCMRequest{
		To: notification.PushToken,
		Notification: FCMNotification{
			Title: notification.Title,
			Body:  notification.Body,
			Sound: "default",
		},
		Data: map[string]interface{}{
			"type":    notification.Type,
			"link":    notification.Link,
			"user_id": notification.UserID,
		},
		Priority: "high",
	}

	// Add metadata to data payload
	for key, value := range notification.Metadata {
		fcmReq.Data[key] = value
	}

	jsonData, err := json.Marshal(fcmReq)
	if err != nil {
		return &DeliveryResult{
			Channel:      "push",
			Success:      false,
			ErrorMessage: err.Error(),
			DeliveredAt:  time.Now(),
		}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://fcm.googleapis.com/fcm/send",
		strings.NewReader(string(jsonData)))
	if err != nil {
		return &DeliveryResult{
			Channel:      "push",
			Success:      false,
			ErrorMessage: err.Error(),
			DeliveredAt:  time.Now(),
		}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "key="+p.fcmServerKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return &DeliveryResult{
			Channel:      "push",
			Success:      false,
			ErrorMessage: err.Error(),
			DeliveredAt:  time.Now(),
		}, err
	}
	defer resp.Body.Close()

	var fcmResp FCMResponse
	if err := json.NewDecoder(resp.Body).Decode(&fcmResp); err != nil {
		return &DeliveryResult{
			Channel:      "push",
			Success:      false,
			ErrorMessage: err.Error(),
			DeliveredAt:  time.Now(),
		}, err
	}

	result := &DeliveryResult{
		Channel:     "push",
		DeliveredAt: time.Now(),
	}

	if fcmResp.Success > 0 && len(fcmResp.Results) > 0 {
		result.Success = true
		result.MessageID = fcmResp.Results[0].MessageID
		p.logger.WithFields(logrus.Fields{
			"user_id": notification.UserID,
			"type":    notification.Type,
			"fcm_id":  fcmResp.Results[0].MessageID,
		}).Info("FCM push notification sent successfully")
	} else {
		result.Success = false
		if len(fcmResp.Results) > 0 {
			result.ErrorMessage = fcmResp.Results[0].Error
		} else {
			result.ErrorMessage = "FCM returned no results"
		}
		p.logger.WithFields(logrus.Fields{
			"user_id": notification.UserID,
			"error":   result.ErrorMessage,
		}).Error("FCM push notification failed")
	}

	return result, nil
}

// sendAPNS sends notification via Apple Push Notification Service
func (p *PushChannel) sendAPNS(_ context.Context, notification *NotificationRequest) (*DeliveryResult, error) {
	// APNS implementation would go here
	// This is a simplified version - in production you'd use a proper APNS library
	// like github.com/sideshow/apns2

	result := &DeliveryResult{
		Channel:      "push",
		Success:      false,
		ErrorMessage: "APNS implementation not completed",
		DeliveredAt:  time.Now(),
	}

	p.logger.WithFields(logrus.Fields{
		"user_id": notification.UserID,
		"type":    notification.Type,
	}).Warn("APNS push notification not implemented")

	return result, fmt.Errorf("APNS implementation not completed")
}

// GetName returns the channel name
func (p *PushChannel) GetName() string {
	return "push"
}

// IsEnabled checks if the channel is enabled
func (p *PushChannel) IsEnabled() bool {
	return p.enabled
}

// Validate validates the notification request for push channel
func (p *PushChannel) Validate(req *NotificationRequest) error {
	if req.PushToken == "" {
		return fmt.Errorf("push token is required for push notifications")
	}

	if req.Title == "" {
		return fmt.Errorf("title is required for push notifications")
	}

	if req.Body == "" {
		return fmt.Errorf("body is required for push notifications")
	}

	return nil
}
