package services

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/models"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/repository"
)

// TemplateService handles notification templates
type TemplateService struct {
	templateRepo *repository.TemplateRepository
	cache        map[string]*models.NotificationTemplate
	logger       *logrus.Logger
}

// NewTemplateService creates a new template service
func NewTemplateService(templateRepo *repository.TemplateRepository, logger *logrus.Logger) *TemplateService {
	return &TemplateService{
		templateRepo: templateRepo,
		cache:        make(map[string]*models.NotificationTemplate),
		logger:       logger,
	}
}

// RenderNotification renders a notification using templates
func (s *TemplateService) RenderNotification(ctx context.Context, req *models.NotificationRequest, templateData *models.TemplateData) (*models.NotificationRequest, error) {
	// Get template for event type and channel
	tmpl, err := s.getTemplate(ctx, req.EventType, req.Channel)
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"event_type": req.EventType,
			"channel":    req.Channel,
		}).Warn("Template not found, using default formatting")

		// Use default formatting if template not found
		return s.applyDefaultFormatting(req, templateData), nil
	}

	// Render subject template
	subject, err := s.renderTemplate(tmpl.SubjectTemplate, templateData)
	if err != nil {
		s.logger.WithError(err).Error("Failed to render subject template")
		return req, fmt.Errorf("failed to render subject template: %w", err)
	}

	// Render body template
	body, err := s.renderTemplate(tmpl.BodyTemplate, templateData)
	if err != nil {
		s.logger.WithError(err).Error("Failed to render body template")
		return req, fmt.Errorf("failed to render body template: %w", err)
	}

	// Update request with rendered content
	req.Title = subject
	req.Message = body
	req.TemplateID = tmpl.TemplateID

	return req, nil
}

// getTemplate gets a template for the given event type and channel
func (s *TemplateService) getTemplate(ctx context.Context, eventType models.EventType, channel models.NotificationChannel) (*models.NotificationTemplate, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s_%s", eventType, channel)
	if tmpl, exists := s.cache[cacheKey]; exists {
		return tmpl, nil
	}

	// Get from database
	tmpl, err := s.templateRepo.GetByEventTypeAndChannel(ctx, eventType, channel)
	if err != nil {
		return nil, err
	}

	// Cache the template
	s.cache[cacheKey] = tmpl

	return tmpl, nil
}

// renderTemplate renders a template with the given data
func (s *TemplateService) renderTemplate(templateStr string, data *models.TemplateData) (string, error) {
	tmpl, err := template.New("notification").Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// applyDefaultFormatting applies default formatting when no template is found
func (s *TemplateService) applyDefaultFormatting(req *models.NotificationRequest, data *models.TemplateData) *models.NotificationRequest {
	// Create a copy of the request
	formattedReq := *req

	// Apply default formatting based on event type
	switch req.EventType {
	case models.EventTypeFileUploaded:
		formattedReq.Title = "File Upload Complete"
		formattedReq.Message = fmt.Sprintf("Your file '%s' has been uploaded successfully", data.FileName)
	case models.EventTypeFileUploadFailed:
		formattedReq.Title = "File Upload Failed"
		formattedReq.Message = fmt.Sprintf("Failed to upload file '%s': %s", data.FileName, data.ErrorMessage)
	case models.EventTypeFileDeleted:
		formattedReq.Title = "File Deleted"
		formattedReq.Message = fmt.Sprintf("Your file '%s' has been deleted", data.FileName)
	case models.EventTypeFileShared:
		formattedReq.Title = "File Shared"
		formattedReq.Message = fmt.Sprintf("A file '%s' has been shared with you", data.FileName)
	case models.EventTypeQuotaWarning80:
		formattedReq.Title = "Storage Quota Warning"
		formattedReq.Message = "You have used 80% of your storage quota"
	case models.EventTypeQuotaWarning90:
		formattedReq.Title = "Storage Quota Warning"
		formattedReq.Message = "You have used 90% of your storage quota"
	case models.EventTypeQuotaExceeded:
		formattedReq.Title = "Storage Quota Exceeded"
		formattedReq.Message = "You have exceeded your storage quota"
	case models.EventTypeSecurityAlert:
		formattedReq.Title = "Security Alert"
		formattedReq.Message = "A security alert has been triggered"
	case models.EventTypeSystemMaintenance:
		formattedReq.Title = "System Maintenance"
		formattedReq.Message = "System maintenance is scheduled"
	default:
		// Use the original title and message
	}

	return &formattedReq
}

// CreateDefaultTemplates creates default templates for all event types and channels
func (s *TemplateService) CreateDefaultTemplates(ctx context.Context) error {
	templates := s.getDefaultTemplates()

	for _, tmpl := range templates {
		// Check if template already exists
		_, err := s.templateRepo.GetByTemplateID(ctx, tmpl.TemplateID)
		if err == nil {
			// Template already exists, skip
			continue
		}

		// Create template
		if err := s.templateRepo.Create(ctx, tmpl); err != nil {
			s.logger.WithError(err).WithField("template_id", tmpl.TemplateID).Error("Failed to create default template")
			return err
		}

		s.logger.WithField("template_id", tmpl.TemplateID).Info("Created default template")
	}

	return nil
}

// getDefaultTemplates returns default templates
func (s *TemplateService) getDefaultTemplates() []*models.NotificationTemplate {
	now := time.Now()

	return []*models.NotificationTemplate{
		// File Upload Success - Email
		{
			TemplateID:      "file_uploaded_email",
			EventType:       models.EventTypeFileUploaded,
			Channel:         models.ChannelEmail,
			SubjectTemplate: "✅ File Upload Complete: {{.FileName}}",
			BodyTemplate:    "Hello {{.UserName}},\n\nYour file '{{.FileName}}' ({{.FileSizeFormatted}}) has been uploaded successfully.\n\nUploaded at: {{.Timestamp.Format \"2006-01-02 15:04:05\"}}\n\nBest regards,\nFile Sharing Platform",
			IsActive:        true,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		// File Upload Success - SMS
		{
			TemplateID:      "file_uploaded_sms",
			EventType:       models.EventTypeFileUploaded,
			Channel:         models.ChannelSMS,
			SubjectTemplate: "File Upload Complete",
			BodyTemplate:    "✅ {{.FileName}} uploaded successfully ({{.FileSizeFormatted}})",
			IsActive:        true,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		// File Upload Success - Push
		{
			TemplateID:      "file_uploaded_push",
			EventType:       models.EventTypeFileUploaded,
			Channel:         models.ChannelPush,
			SubjectTemplate: "File Upload Complete",
			BodyTemplate:    "{{.FileName}} uploaded successfully",
			IsActive:        true,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		// File Upload Success - In-App
		{
			TemplateID:      "file_uploaded_inapp",
			EventType:       models.EventTypeFileUploaded,
			Channel:         models.ChannelInApp,
			SubjectTemplate: "File Upload Complete",
			BodyTemplate:    "{{.FileName}} ({{.FileSizeFormatted}}) uploaded successfully",
			IsActive:        true,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		// File Upload Failed - Email
		{
			TemplateID:      "file_upload_failed_email",
			EventType:       models.EventTypeFileUploadFailed,
			Channel:         models.ChannelEmail,
			SubjectTemplate: "❌ File Upload Failed: {{.FileName}}",
			BodyTemplate:    "Hello {{.UserName}},\n\nUnfortunately, your file '{{.FileName}}' could not be uploaded.\n\nError: {{.ErrorMessage}}\n\nPlease try again or contact support if the issue persists.\n\nBest regards,\nFile Sharing Platform",
			IsActive:        true,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		// File Upload Failed - SMS
		{
			TemplateID:      "file_upload_failed_sms",
			EventType:       models.EventTypeFileUploadFailed,
			Channel:         models.ChannelSMS,
			SubjectTemplate: "File Upload Failed",
			BodyTemplate:    "❌ {{.FileName}} upload failed: {{.ErrorMessage}}",
			IsActive:        true,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		// File Deleted - Email
		{
			TemplateID:      "file_deleted_email",
			EventType:       models.EventTypeFileDeleted,
			Channel:         models.ChannelEmail,
			SubjectTemplate: "🗑️ File Deleted: {{.FileName}}",
			BodyTemplate:    "Hello {{.UserName}},\n\nYour file '{{.FileName}}' has been deleted.\n\nDeleted at: {{.Timestamp.Format \"2006-01-02 15:04:05\"}}\n\nBest regards,\nFile Sharing Platform",
			IsActive:        true,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		// File Shared - Email
		{
			TemplateID:      "file_shared_email",
			EventType:       models.EventTypeFileShared,
			Channel:         models.ChannelEmail,
			SubjectTemplate: "📁 File Shared: {{.FileName}}",
			BodyTemplate:    "Hello {{.UserName}},\n\nA file '{{.FileName}}' has been shared with you.\n\nShared at: {{.Timestamp.Format \"2006-01-02 15:04:05\"}}\n\nBest regards,\nFile Sharing Platform",
			IsActive:        true,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		// Quota Warning 80% - Email
		{
			TemplateID:      "quota_warning_80_email",
			EventType:       models.EventTypeQuotaWarning80,
			Channel:         models.ChannelEmail,
			SubjectTemplate: "⚠️ Storage Quota Warning (80%)",
			BodyTemplate:    "Hello {{.UserName}},\n\nYou have used 80% of your storage quota.\n\nCurrent usage: {{.FileSizeFormatted}}\n\nConsider upgrading your plan or deleting unused files.\n\nBest regards,\nFile Sharing Platform",
			IsActive:        true,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		// Quota Warning 90% - Email
		{
			TemplateID:      "quota_warning_90_email",
			EventType:       models.EventTypeQuotaWarning90,
			Channel:         models.ChannelEmail,
			SubjectTemplate: "⚠️ Storage Quota Warning (90%)",
			BodyTemplate:    "Hello {{.UserName}},\n\nYou have used 90% of your storage quota.\n\nCurrent usage: {{.FileSizeFormatted}}\n\nPlease upgrade your plan or delete unused files immediately.\n\nBest regards,\nFile Sharing Platform",
			IsActive:        true,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		// Quota Exceeded - Email
		{
			TemplateID:      "quota_exceeded_email",
			EventType:       models.EventTypeQuotaExceeded,
			Channel:         models.ChannelEmail,
			SubjectTemplate: "🚨 Storage Quota Exceeded",
			BodyTemplate:    "Hello {{.UserName}},\n\nYou have exceeded your storage quota.\n\nCurrent usage: {{.FileSizeFormatted}}\n\nPlease upgrade your plan immediately to continue using the service.\n\nBest regards,\nFile Sharing Platform",
			IsActive:        true,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		// Security Alert - Email
		{
			TemplateID:      "security_alert_email",
			EventType:       models.EventTypeSecurityAlert,
			Channel:         models.ChannelEmail,
			SubjectTemplate: "🚨 Security Alert",
			BodyTemplate:    "Hello {{.UserName}},\n\nA security alert has been triggered for your account.\n\nPlease review your account activity and contact support if you notice any suspicious activity.\n\nBest regards,\nFile Sharing Platform",
			IsActive:        true,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		// System Maintenance - Email
		{
			TemplateID:      "system_maintenance_email",
			EventType:       models.EventTypeSystemMaintenance,
			Channel:         models.ChannelEmail,
			SubjectTemplate: "🔧 System Maintenance Scheduled",
			BodyTemplate:    "Hello {{.UserName}},\n\nSystem maintenance is scheduled for {{.Timestamp.Format \"2006-01-02 15:04:05\"}}.\n\nDuring this time, the service may be temporarily unavailable.\n\nWe apologize for any inconvenience.\n\nBest regards,\nFile Sharing Platform",
			IsActive:        true,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
	}
}

// FormatFileSize formats file size in human-readable format
func (s *TemplateService) FormatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// CreateTemplateData creates template data from notification request and additional data
func (s *TemplateService) CreateTemplateData(req *models.NotificationRequest, additionalData map[string]interface{}) *models.TemplateData {
	data := &models.TemplateData{
		UserName:     s.getStringFromMetadata(req.Metadata, "user_name", "User"),
		FileName:     s.getStringFromMetadata(req.Metadata, "file_name", ""),
		FileSize:     s.getInt64FromMetadata(req.Metadata, "file_size", 0),
		Timestamp:    time.Now(),
		ErrorMessage: s.getStringFromMetadata(req.Metadata, "error_message", ""),
		Metadata:     req.Metadata,
	}

	// Format file size
	if data.FileSize > 0 {
		data.FileSizeFormatted = s.FormatFileSize(data.FileSize)
	}

	// Add additional data
	for key, value := range additionalData {
		switch v := value.(type) {
		case string:
			if key == "user_name" {
				data.UserName = v
			} else if key == "file_name" {
				data.FileName = v
			} else if key == "error_message" {
				data.ErrorMessage = v
			}
		case int64:
			if key == "file_size" {
				data.FileSize = v
				data.FileSizeFormatted = s.FormatFileSize(v)
			}
		case time.Time:
			if key == "timestamp" {
				data.Timestamp = v
			}
		}
	}

	return data
}

// getStringFromMetadata gets a string value from metadata
func (s *TemplateService) getStringFromMetadata(metadata map[string]interface{}, key, defaultValue string) string {
	if value, ok := metadata[key].(string); ok {
		return value
	}
	return defaultValue
}

// getInt64FromMetadata gets an int64 value from metadata
func (s *TemplateService) getInt64FromMetadata(metadata map[string]interface{}, key string, defaultValue int64) int64 {
	if value, ok := metadata[key].(int64); ok {
		return value
	}
	if value, ok := metadata[key].(float64); ok {
		return int64(value)
	}
	return defaultValue
}

// ClearCache clears the template cache
func (s *TemplateService) ClearCache() {
	s.cache = make(map[string]*models.NotificationTemplate)
	s.logger.Info("Template cache cleared")
}

// GetCacheSize returns the size of the template cache
func (s *TemplateService) GetCacheSize() int {
	return len(s.cache)
}

// GetTemplates gets templates with pagination and filtering
func (s *TemplateService) GetTemplates(ctx context.Context, page, limit int, eventTypeFilter *models.EventType, channelFilter *models.NotificationChannel) ([]*models.NotificationTemplate, int64, error) {
	return s.templateRepo.GetAll(ctx, page, limit, eventTypeFilter, channelFilter)
}

// CreateTemplate creates a new template
func (s *TemplateService) CreateTemplate(ctx context.Context, template *models.NotificationTemplate) error {
	template.CreatedAt = time.Now()
	template.UpdatedAt = time.Now()

	if err := s.templateRepo.Create(ctx, template); err != nil {
		return fmt.Errorf("failed to create template: %w", err)
	}

	// Clear cache to ensure new template is loaded
	s.ClearCache()

	s.logger.WithField("template_id", template.TemplateID).Info("Template created successfully")
	return nil
}

// UpdateTemplate updates an existing template
func (s *TemplateService) UpdateTemplate(ctx context.Context, templateID string, template *models.NotificationTemplate) error {
	// Get existing template
	existing, err := s.templateRepo.GetByTemplateID(ctx, templateID)
	if err != nil {
		return fmt.Errorf("template not found: %w", err)
	}

	// Update fields
	existing.SubjectTemplate = template.SubjectTemplate
	existing.BodyTemplate = template.BodyTemplate
	existing.IsActive = template.IsActive
	existing.UpdatedAt = time.Now()

	if err := s.templateRepo.Update(ctx, templateID, existing); err != nil {
		return fmt.Errorf("failed to update template: %w", err)
	}

	// Clear cache to ensure updated template is loaded
	s.ClearCache()

	s.logger.WithField("template_id", templateID).Info("Template updated successfully")
	return nil
}

// DeleteTemplate deletes a template
func (s *TemplateService) DeleteTemplate(ctx context.Context, templateID string) error {
	// Get existing template
	existing, err := s.templateRepo.GetByTemplateID(ctx, templateID)
	if err != nil {
		return fmt.Errorf("template not found: %w", err)
	}

	if err := s.templateRepo.Delete(ctx, existing.ID.Hex()); err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	// Clear cache to ensure deleted template is removed
	s.ClearCache()

	s.logger.WithField("template_id", templateID).Info("Template deleted successfully")
	return nil
}
