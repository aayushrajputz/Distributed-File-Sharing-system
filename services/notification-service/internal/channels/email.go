package channels

import (
	"context"
	"fmt"
	"time"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/sirupsen/logrus"
)

// EmailChannel implements email notifications using SendGrid
type EmailChannel struct {
	apiKey    string
	fromEmail string
	fromName  string
	enabled   bool
	logger    *logrus.Logger
}

// NewEmailChannel creates a new email notification channel
func NewEmailChannel(apiKey, fromEmail, fromName string, logger *logrus.Logger) *EmailChannel {
	enabled := apiKey != ""

	return &EmailChannel{
		apiKey:    apiKey,
		fromEmail: fromEmail,
		fromName:  fromName,
		enabled:   enabled,
		logger:    logger,
	}
}

// Send sends an email notification
func (e *EmailChannel) Send(ctx context.Context, notification *NotificationRequest) (*DeliveryResult, error) {
	if !e.enabled {
		return &DeliveryResult{
			Channel:      "email",
			Success:      false,
			ErrorMessage: "email channel is disabled",
			DeliveredAt:  time.Now(),
		}, fmt.Errorf("email channel is disabled")
	}

	if err := e.Validate(notification); err != nil {
		return &DeliveryResult{
			Channel:      "email",
			Success:      false,
			ErrorMessage: err.Error(),
			DeliveredAt:  time.Now(),
		}, err
	}

	// Create email message
	from := mail.NewEmail(e.fromName, e.fromEmail)
	to := mail.NewEmail("", notification.Email)

	subject := notification.Title
	plainTextContent := e.generatePlainTextContent(notification)
	htmlContent := e.generateHTMLContent(notification)

	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)

	// Add custom headers
	message.SetHeader("X-Notification-Type", notification.Type)
	message.SetHeader("X-Notification-Priority", notification.Priority)

	// Add metadata as custom headers
	for key, value := range notification.Metadata {
		message.SetHeader(fmt.Sprintf("X-Notification-%s", key), value)
	}

	// Send email
	client := sendgrid.NewSendClient(e.apiKey)
	response, err := client.Send(message)

	result := &DeliveryResult{
		Channel:     "email",
		DeliveredAt: time.Now(),
	}

	if err != nil {
		result.Success = false
		result.ErrorMessage = err.Error()
		e.logger.WithError(err).Error("Failed to send email notification")
		return result, err
	}

	if response.StatusCode >= 200 && response.StatusCode < 300 {
		result.Success = true
		if messageIDs, ok := response.Headers["X-Message-Id"]; ok && len(messageIDs) > 0 {
			result.MessageID = messageIDs[0]
		}
		e.logger.WithFields(logrus.Fields{
			"user_id": notification.UserID,
			"email":   notification.Email,
			"type":    notification.Type,
		}).Info("Email notification sent successfully")
	} else {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("SendGrid returned status code: %d", response.StatusCode)
		e.logger.WithFields(logrus.Fields{
			"status_code": response.StatusCode,
			"user_id":     notification.UserID,
		}).Error("SendGrid returned error status")
	}

	return result, nil
}

// GetName returns the channel name
func (e *EmailChannel) GetName() string {
	return "email"
}

// IsEnabled checks if the channel is enabled
func (e *EmailChannel) IsEnabled() bool {
	return e.enabled
}

// Validate validates the notification request for email channel
func (e *EmailChannel) Validate(req *NotificationRequest) error {
	if req.Email == "" {
		return fmt.Errorf("email address is required for email notifications")
	}

	if req.Title == "" {
		return fmt.Errorf("title is required for email notifications")
	}

	if req.Body == "" {
		return fmt.Errorf("body is required for email notifications")
	}

	return nil
}

// generatePlainTextContent generates plain text email content
func (e *EmailChannel) generatePlainTextContent(notification *NotificationRequest) string {
	content := fmt.Sprintf("%s\n\n%s", notification.Title, notification.Body)

	if notification.Link != "" {
		content += fmt.Sprintf("\n\nView: %s", notification.Link)
	}

	content += "\n\n---\nFile Sharing Platform"
	return content
}

// generateHTMLContent generates HTML email content
func (e *EmailChannel) generateHTMLContent(notification *NotificationRequest) string {
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #f8f9fa; padding: 20px; border-radius: 8px; margin-bottom: 20px; }
        .content { padding: 20px 0; }
        .button { display: inline-block; padding: 12px 24px; background-color: #007bff; color: white; text-decoration: none; border-radius: 4px; margin: 20px 0; }
        .footer { margin-top: 30px; padding-top: 20px; border-top: 1px solid #eee; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>%s</h1>
        </div>
        <div class="content">
            <p>%s</p>
            %s
        </div>
        <div class="footer">
            <p>This is an automated message from File Sharing Platform.</p>
        </div>
    </div>
</body>
</html>`,
		notification.Title,
		notification.Title,
		notification.Body,
		e.generateLinkButton(notification.Link),
	)

	return html
}

// generateLinkButton generates a link button if link is provided
func (e *EmailChannel) generateLinkButton(link string) string {
	if link == "" {
		return ""
	}

	return fmt.Sprintf(`<a href="%s" class="button">View Details</a>`, link)
}
