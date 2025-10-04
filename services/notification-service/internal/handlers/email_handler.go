package handlers

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/models"
	"github.com/sirupsen/logrus"
)

// EmailHandler handles email notifications
type EmailHandler struct {
	host     string
	port     int
	username string
	password string
	fromEmail string
	fromName  string
	tls      bool
	logger   *logrus.Logger
}

// NewEmailHandler creates a new email handler
func NewEmailHandler(host string, port int, username, password, fromEmail, fromName string, tls bool, logger *logrus.Logger) *EmailHandler {
	return &EmailHandler{
		host:      host,
		port:      port,
		username:  username,
		password:  password,
		fromEmail: fromEmail,
		fromName:  fromName,
		tls:       tls,
		logger:    logger,
	}
}

// Send sends an email notification
func (h *EmailHandler) Send(ctx context.Context, req *models.NotificationRequest) (*models.NotificationResponse, error) {
	start := time.Now()
	
	// Validate request
	if err := h.Validate(req); err != nil {
		return &models.NotificationResponse{
			Status:   models.StatusFailed,
			Channel:  models.ChannelEmail,
			Error:    err.Error(),
			Duration: time.Since(start).Milliseconds(),
		}, err
	}

	// Get user email from metadata or use default
	email := h.getUserEmail(req)
	if email == "" {
		return &models.NotificationResponse{
			Status:   models.StatusFailed,
			Channel:  models.ChannelEmail,
			Error:    "user email not found",
			Duration: time.Since(start).Milliseconds(),
		}, fmt.Errorf("user email not found")
	}

	// Create email message
	message := h.createEmailMessage(req, email)
	
	// Send email
	err := h.sendEmail(ctx, email, message)
	
	response := &models.NotificationResponse{
		Channel:  models.ChannelEmail,
		Duration: time.Since(start).Milliseconds(),
	}

	if err != nil {
		response.Status = models.StatusFailed
		response.Error = err.Error()
		h.logger.WithError(err).WithFields(logrus.Fields{
			"user_id": req.UserID,
			"channel": "email",
		}).Error("Failed to send email notification")
	} else {
		response.Status = models.StatusSent
		now := time.Now()
		response.SentAt = &now
		h.logger.WithFields(logrus.Fields{
			"user_id": req.UserID,
			"channel": "email",
			"email":   email,
		}).Info("Email notification sent successfully")
	}

	return response, err
}

// Validate validates the notification request
func (h *EmailHandler) Validate(req *models.NotificationRequest) error {
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
func (h *EmailHandler) GetName() string {
	return "email"
}

// IsEnabled checks if the handler is enabled
func (h *EmailHandler) IsEnabled() bool {
	return h.host != "" && h.username != "" && h.password != ""
}

// getUserEmail gets the user's email address
func (h *EmailHandler) getUserEmail(req *models.NotificationRequest) string {
	// Try to get email from metadata first
	if email, ok := req.Metadata["email"].(string); ok && email != "" {
		return email
	}
	
	// Try to get email from user preferences (would need to be passed in)
	// For now, return empty string
	return ""
}

// createEmailMessage creates the email message
func (h *EmailHandler) createEmailMessage(req *models.NotificationRequest, toEmail string) []byte {
	// Create headers
	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("%s <%s>", h.fromName, h.fromEmail)
	headers["To"] = toEmail
	headers["Subject"] = req.Title
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"
	
	// Add custom headers
	headers["X-Notification-Type"] = string(req.EventType)
	headers["X-Notification-Priority"] = string(req.Priority)
	headers["X-User-ID"] = req.UserID

	// Create message body
	body := h.createEmailBody(req)
	
	// Combine headers and body
	var message strings.Builder
	for key, value := range headers {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}
	message.WriteString("\r\n")
	message.WriteString(body)
	
	return []byte(message.String())
}

// createEmailBody creates the email body
func (h *EmailHandler) createEmailBody(req *models.NotificationRequest) string {
	// Create HTML email body
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s</title>
    <style>
        body { 
            font-family: Arial, sans-serif; 
            line-height: 1.6; 
            color: #333; 
            max-width: 600px; 
            margin: 0 auto; 
            padding: 20px; 
        }
        .header { 
            background-color: #f8f9fa; 
            padding: 20px; 
            border-radius: 8px; 
            margin-bottom: 20px; 
            text-align: center;
        }
        .content { 
            padding: 20px 0; 
        }
        .button { 
            display: inline-block; 
            padding: 12px 24px; 
            background-color: #007bff; 
            color: white; 
            text-decoration: none; 
            border-radius: 4px; 
            margin: 20px 0; 
        }
        .footer { 
            margin-top: 30px; 
            padding-top: 20px; 
            border-top: 1px solid #eee; 
            font-size: 12px; 
            color: #666; 
            text-align: center;
        }
        .priority-high { 
            border-left: 4px solid #dc3545; 
            padding-left: 16px; 
        }
        .priority-critical { 
            border-left: 4px solid #dc3545; 
            padding-left: 16px; 
            background-color: #f8d7da; 
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>%s</h1>
    </div>
    <div class="content %s">
        <p>%s</p>
        %s
    </div>
    <div class="footer">
        <p>This is an automated message from File Sharing Platform.</p>
        <p>If you no longer wish to receive these notifications, please update your preferences.</p>
    </div>
</body>
</html>`,
		req.Title,
		req.Title,
		h.getPriorityClass(req.Priority),
		h.formatMessage(req.Message),
		h.createActionButton(req),
	)
	
	return html
}

// getPriorityClass returns the CSS class for priority
func (h *EmailHandler) getPriorityClass(priority models.Priority) string {
	switch priority {
	case models.PriorityHigh:
		return "priority-high"
	case models.PriorityCritical:
		return "priority-critical"
	default:
		return ""
	}
}

// formatMessage formats the message for HTML
func (h *EmailHandler) formatMessage(message string) string {
	// Convert line breaks to HTML
	message = strings.ReplaceAll(message, "\n", "<br>")
	return message
}

// createActionButton creates an action button if there's a link
func (h *EmailHandler) createActionButton(req *models.NotificationRequest) string {
	if link, ok := req.Metadata["link"].(string); ok && link != "" {
		return fmt.Sprintf(`<a href="%s" class="button">View Details</a>`, link)
	}
	return ""
}

// sendEmail sends the email using SMTP
func (h *EmailHandler) sendEmail(ctx context.Context, toEmail string, message []byte) error {
	// Create SMTP address
	addr := fmt.Sprintf("%s:%d", h.host, h.port)
	
	// Create authentication
	auth := smtp.PlainAuth("", h.username, h.password, h.host)
	
	// Send email
	if h.tls {
		// Use TLS
		return h.sendEmailTLS(ctx, addr, auth, h.fromEmail, []string{toEmail}, message)
	} else {
		// Use plain SMTP
		return smtp.SendMail(addr, auth, h.fromEmail, []string{toEmail}, message)
	}
}

// sendEmailTLS sends email with TLS
func (h *EmailHandler) sendEmailTLS(ctx context.Context, addr string, auth smtp.Auth, from string, to []string, message []byte) error {
	// This is a simplified implementation
	// In production, you would use a proper SMTP client with TLS support
	return smtp.SendMail(addr, auth, from, to, message)
}

// TestConnection tests the SMTP connection
func (h *EmailHandler) TestConnection(ctx context.Context) error {
	if !h.IsEnabled() {
		return fmt.Errorf("email handler is not enabled")
	}

	// Create a test message
	testMessage := []byte("To: test@example.com\r\nSubject: Test\r\n\r\nThis is a test message.")
	
	// Try to send to a test address (this will fail but we can check the connection)
	addr := fmt.Sprintf("%s:%d", h.host, h.port)
	auth := smtp.PlainAuth("", h.username, h.password, h.host)
	
	err := smtp.SendMail(addr, auth, h.fromEmail, []string{"test@example.com"}, testMessage)
	
	// We expect this to fail, but we can check if it's a connection error
	if err != nil && !strings.Contains(err.Error(), "test@example.com") {
		return fmt.Errorf("SMTP connection failed: %w", err)
	}
	
	return nil
}
