package channels

import (
	"context"
	"fmt"
	"time"

	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
	"github.com/sirupsen/logrus"
)

// SMSChannel implements SMS notifications using Twilio
type SMSChannel struct {
	client          *twilio.RestClient
	fromPhoneNumber string
	enabled         bool
	logger          *logrus.Logger
}

// NewSMSChannel creates a new SMS notification channel
func NewSMSChannel(accountSID, authToken, fromPhoneNumber string, logger *logrus.Logger) *SMSChannel {
	enabled := accountSID != "" && authToken != "" && fromPhoneNumber != ""
	
	var client *twilio.RestClient
	if enabled {
		client = twilio.NewRestClientWithParams(twilio.ClientParams{
			Username: accountSID,
			Password: authToken,
		})
	}
	
	return &SMSChannel{
		client:          client,
		fromPhoneNumber: fromPhoneNumber,
		enabled:         enabled,
		logger:          logger,
	}
}

// Send sends an SMS notification
func (s *SMSChannel) Send(ctx context.Context, notification *NotificationRequest) (*DeliveryResult, error) {
	if !s.enabled {
		return &DeliveryResult{
			Channel:     "sms",
			Success:     false,
			ErrorMessage: "SMS channel is disabled",
			DeliveredAt: time.Now(),
		}, fmt.Errorf("SMS channel is disabled")
	}

	if err := s.Validate(notification); err != nil {
		return &DeliveryResult{
			Channel:     "sms",
			Success:     false,
			ErrorMessage: err.Error(),
			DeliveredAt: time.Now(),
		}, err
	}

	// Generate SMS content
	message := s.generateSMSContent(notification)

	// Send SMS
	params := &twilioApi.CreateMessageParams{}
	params.SetTo(notification.Phone)
	params.SetFrom(s.fromPhoneNumber)
	params.SetBody(message)

	// Add custom parameters
	params.SetStatusCallback("") // Optional: webhook for delivery status

	resp, err := s.client.Api.CreateMessage(params)
	
	result := &DeliveryResult{
		Channel:     "sms",
		DeliveredAt: time.Now(),
	}

	if err != nil {
		result.Success = false
		result.ErrorMessage = err.Error()
		s.logger.WithError(err).Error("Failed to send SMS notification")
		return result, err
	}

	if resp.Sid != nil {
		result.Success = true
		result.MessageID = *resp.Sid
		s.logger.WithFields(logrus.Fields{
			"user_id": notification.UserID,
			"phone":   notification.Phone,
			"type":    notification.Type,
			"sid":     *resp.Sid,
		}).Info("SMS notification sent successfully")
	} else {
		result.Success = false
		result.ErrorMessage = "Twilio returned empty SID"
		s.logger.Error("Twilio returned empty SID for SMS")
	}

	return result, nil
}

// GetName returns the channel name
func (s *SMSChannel) GetName() string {
	return "sms"
}

// IsEnabled checks if the channel is enabled
func (s *SMSChannel) IsEnabled() bool {
	return s.enabled
}

// Validate validates the notification request for SMS channel
func (s *SMSChannel) Validate(req *NotificationRequest) error {
	if req.Phone == "" {
		return fmt.Errorf("phone number is required for SMS notifications")
	}
	
	if req.Body == "" {
		return fmt.Errorf("body is required for SMS notifications")
	}
	
	// Validate phone number format (basic validation)
	if len(req.Phone) < 10 {
		return fmt.Errorf("invalid phone number format")
	}
	
	return nil
}

// generateSMSContent generates SMS content
func (s *SMSChannel) generateSMSContent(notification *NotificationRequest) string {
	// SMS has character limits, so we need to be concise
	content := fmt.Sprintf("%s: %s", notification.Title, notification.Body)
	
	// Truncate if too long (SMS limit is typically 160 characters)
	maxLength := 150 // Leave some room for link
	if len(content) > maxLength {
		content = content[:maxLength-3] + "..."
	}
	
	// Add link if provided and there's space
	if notification.Link != "" && len(content) < 120 {
		// Use a short URL service or just truncate the link
		shortLink := s.shortenLink(notification.Link)
		content += fmt.Sprintf(" %s", shortLink)
	}
	
	return content
}

// shortenLink creates a short version of the link for SMS
func (s *SMSChannel) shortenLink(link string) string {
	// In a real implementation, you would use a URL shortener service
	// For now, we'll just truncate it
	if len(link) > 20 {
		return link[:17] + "..."
	}
	return link
}

