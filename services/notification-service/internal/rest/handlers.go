package rest

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/models"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/services"
)

// RestHandlers handles REST API endpoints
type RestHandlers struct {
	notifSvc      *services.NotificationService
	preferenceSvc *services.PreferenceService
	templateSvc   *services.TemplateService
	batchSvc      *services.BatchService
	dlqSvc        *services.DLQService
	logger        *logrus.Logger
}

// NewRestHandlers creates new REST handlers
func NewRestHandlers(
	notifSvc *services.NotificationService,
	preferenceSvc *services.PreferenceService,
	templateSvc *services.TemplateService,
	batchSvc *services.BatchService,
	dlqSvc *services.DLQService,
	logger *logrus.Logger,
) *RestHandlers {
	return &RestHandlers{
		notifSvc:      notifSvc,
		preferenceSvc: preferenceSvc,
		templateSvc:   templateSvc,
		batchSvc:      batchSvc,
		dlqSvc:        dlqSvc,
		logger:        logger,
	}
}

// HealthCheck handles health check endpoint
func (h *RestHandlers) HealthCheck(c *gin.Context) {
	health, err := h.notifSvc.GetServiceHealth(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, health)
}

// GetNotifications handles GET /v1/notifications
func (h *RestHandlers) GetNotifications(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-User-ID header is required"})
		return
	}

	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	status := c.Query("status")
	eventType := c.Query("event_type")

	// Parse status filter
	var statusFilter *models.NotificationStatus
	if status != "" {
		s := models.NotificationStatus(status)
		statusFilter = &s
	}

	// Parse event type filter
	var eventTypeFilter *models.EventType
	if eventType != "" {
		e := models.EventType(eventType)
		eventTypeFilter = &e
	}

	// Get notifications
	notifications, total, err := h.notifSvc.GetNotifications(c.Request.Context(), userID, page, limit, statusFilter, eventTypeFilter)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get notifications")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get notifications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"notifications": notifications,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetNotification handles GET /v1/notifications/:id
func (h *RestHandlers) GetNotification(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-User-ID header is required"})
		return
	}

	notificationID := c.Param("id")
	notification, err := h.notifSvc.GetNotification(c.Request.Context(), notificationID, userID)
	if err != nil {
		if err.Error() == "notification not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
			return
		}
		h.logger.WithError(err).Error("Failed to get notification")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get notification"})
		return
	}

	c.JSON(http.StatusOK, notification)
}

// MarkAsRead handles PUT /v1/notifications/:id/read
func (h *RestHandlers) MarkAsRead(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-User-ID header is required"})
		return
	}

	notificationID := c.Param("id")
	err := h.notifSvc.MarkAsRead(c.Request.Context(), notificationID, userID)
	if err != nil {
		if err.Error() == "notification not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
			return
		}
		h.logger.WithError(err).Error("Failed to mark notification as read")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark notification as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notification marked as read"})
}

// MarkAllAsRead handles PUT /v1/notifications/read-all
func (h *RestHandlers) MarkAllAsRead(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-User-ID header is required"})
		return
	}

	count, err := h.notifSvc.MarkAllAsRead(c.Request.Context(), userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to mark all notifications as read")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark all notifications as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "All notifications marked as read",
		"count":   count,
	})
}

// DeleteNotification handles DELETE /v1/notifications/:id
func (h *RestHandlers) DeleteNotification(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-User-ID header is required"})
		return
	}

	notificationID := c.Param("id")
	err := h.notifSvc.DeleteNotification(c.Request.Context(), notificationID, userID)
	if err != nil {
		if err.Error() == "notification not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
			return
		}
		h.logger.WithError(err).Error("Failed to delete notification")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete notification"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notification deleted"})
}

// GetUnreadCount handles GET /v1/notifications/unread/count
func (h *RestHandlers) GetUnreadCount(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-User-ID header is required"})
		return
	}

	count, err := h.notifSvc.GetUnreadCount(c.Request.Context(), userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get unread count")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get unread count"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

// GetUserPreferences handles GET /v1/preferences
func (h *RestHandlers) GetUserPreferences(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-User-ID header is required"})
		return
	}

	preferences, err := h.preferenceSvc.GetUserPreferences(c.Request.Context(), userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get user preferences")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user preferences"})
		return
	}

	c.JSON(http.StatusOK, preferences)
}

// UpdateUserPreferences handles PUT /v1/preferences
func (h *RestHandlers) UpdateUserPreferences(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-User-ID header is required"})
		return
	}

	var preferences models.UserNotificationPreferences
	if err := c.ShouldBindJSON(&preferences); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	err := h.preferenceSvc.UpdateUserPreferences(c.Request.Context(), userID, &preferences)
	if err != nil {
		h.logger.WithError(err).Error("Failed to update user preferences")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user preferences"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Preferences updated successfully"})
}

// SendTestNotification handles POST /v1/preferences/test
func (h *RestHandlers) SendTestNotification(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-User-ID header is required"})
		return
	}

	var req struct {
		Channel string `json:"channel"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Create test notification request
	testReq := &models.NotificationRequest{
		UserID:           userID,
		EventType:        models.EventTypeSystemMaintenance,
		Channel:          models.NotificationChannel(req.Channel),
		Title:            "Test Notification",
		Message:          "This is a test notification to verify your settings",
		Priority:         models.PriorityNormal,
		BypassBatching:   true,
		BypassQuietHours: true,
	}

	response, err := h.notifSvc.SendNotification(c.Request.Context(), testReq)
	if err != nil {
		h.logger.WithError(err).Error("Failed to send test notification")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send test notification"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Test notification sent",
		"response": response,
	})
}

// GetTemplates handles GET /v1/templates
func (h *RestHandlers) GetTemplates(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	eventType := c.Query("event_type")
	channel := c.Query("channel")

	// Parse filters
	var eventTypeFilter *models.EventType
	if eventType != "" {
		e := models.EventType(eventType)
		eventTypeFilter = &e
	}

	var channelFilter *models.NotificationChannel
	if channel != "" {
		ch := models.NotificationChannel(channel)
		channelFilter = &ch
	}

	templates, total, err := h.templateSvc.GetTemplates(c.Request.Context(), page, limit, eventTypeFilter, channelFilter)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get templates")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get templates"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"templates": templates,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// CreateTemplate handles POST /v1/templates
func (h *RestHandlers) CreateTemplate(c *gin.Context) {
	var template models.NotificationTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	err := h.templateSvc.CreateTemplate(c.Request.Context(), &template)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create template")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create template"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Template created successfully"})
}

// UpdateTemplate handles PUT /v1/templates/:id
func (h *RestHandlers) UpdateTemplate(c *gin.Context) {
	templateID := c.Param("id")

	var template models.NotificationTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	err := h.templateSvc.UpdateTemplate(c.Request.Context(), templateID, &template)
	if err != nil {
		if err.Error() == "template not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
			return
		}
		h.logger.WithError(err).Error("Failed to update template")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update template"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Template updated successfully"})
}

// DeleteTemplate handles DELETE /v1/templates/:id
func (h *RestHandlers) DeleteTemplate(c *gin.Context) {
	templateID := c.Param("id")

	err := h.templateSvc.DeleteTemplate(c.Request.Context(), templateID)
	if err != nil {
		if err.Error() == "template not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
			return
		}
		h.logger.WithError(err).Error("Failed to delete template")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete template"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Template deleted successfully"})
}

// GetBatchNotifications handles GET /v1/batch
func (h *RestHandlers) GetBatchNotifications(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-User-ID header is required"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	status := c.Query("status")

	var statusFilter *models.NotificationStatus
	if status != "" {
		s := models.NotificationStatus(status)
		statusFilter = &s
	}

	batches, total, err := h.batchSvc.GetBatchNotifications(c.Request.Context(), userID, page, limit, statusFilter)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get batch notifications")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get batch notifications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"batches": batches,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetDLQEntries handles GET /v1/dlq
func (h *RestHandlers) GetDLQEntries(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	processed := c.Query("processed")

	var processedFilter *bool
	if processed != "" {
		p := processed == "true"
		processedFilter = &p
	}

	entries, total, err := h.dlqSvc.GetDLQEntries(c.Request.Context(), page, limit, processedFilter)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get DLQ entries")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get DLQ entries"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"entries": entries,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// RetryDLQEntry handles POST /v1/dlq/:id/retry
func (h *RestHandlers) RetryDLQEntry(c *gin.Context) {
	dlqID := c.Param("id")

	err := h.dlqSvc.RetryDLQEntry(c.Request.Context(), dlqID)
	if err != nil {
		if err.Error() == "DLQ entry not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "DLQ entry not found"})
			return
		}
		h.logger.WithError(err).Error("Failed to retry DLQ entry")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retry DLQ entry"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "DLQ entry retried successfully"})
}

// DeleteDLQEntry handles DELETE /v1/dlq/:id
func (h *RestHandlers) DeleteDLQEntry(c *gin.Context) {
	dlqID := c.Param("id")

	err := h.dlqSvc.DeleteDLQEntry(c.Request.Context(), dlqID)
	if err != nil {
		if err.Error() == "DLQ entry not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "DLQ entry not found"})
			return
		}
		h.logger.WithError(err).Error("Failed to delete DLQ entry")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete DLQ entry"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "DLQ entry deleted successfully"})
}

// GetStats handles GET /v1/stats
func (h *RestHandlers) GetStats(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-User-ID header is required"})
		return
	}

	// Get date range
	startDateStr := c.DefaultQuery("start_date", time.Now().AddDate(0, 0, -30).Format("2006-01-02"))
	endDateStr := c.DefaultQuery("end_date", time.Now().Format("2006-01-02"))

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_date format"})
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_date format"})
		return
	}

	// Get notification stats
	notifStats, err := h.notifSvc.GetNotificationStats(c.Request.Context(), userID, startDate, endDate)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get notification stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get notification stats"})
		return
	}

	// Get batch stats
	batchStats, err := h.batchSvc.GetBatchStats(c.Request.Context(), userID, startDate, endDate)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get batch stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get batch stats"})
		return
	}

	// Get DLQ stats
	dlqStats, err := h.dlqSvc.GetDLQStats(c.Request.Context())
	if err != nil {
		h.logger.WithError(err).Error("Failed to get DLQ stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get DLQ stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"notifications": notifStats,
		"batches":       batchStats,
		"dlq":           dlqStats,
		"period": gin.H{
			"start_date": startDate,
			"end_date":   endDate,
		},
	})
}

// SetupRoutes sets up all REST API routes
func (h *RestHandlers) SetupRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1")
	{
		// Health check
		v1.GET("/health", h.HealthCheck)

		// Notifications
		notifications := v1.Group("/notifications")
		{
			notifications.GET("", h.GetNotifications)
			notifications.GET("/:id", h.GetNotification)
			notifications.PUT("/:id/read", h.MarkAsRead)
			notifications.PUT("/read-all", h.MarkAllAsRead)
			notifications.DELETE("/:id", h.DeleteNotification)
			notifications.GET("/unread/count", h.GetUnreadCount)
		}

		// User preferences
		preferences := v1.Group("/preferences")
		{
			preferences.GET("", h.GetUserPreferences)
			preferences.PUT("", h.UpdateUserPreferences)
			preferences.POST("/test", h.SendTestNotification)
		}

		// Templates
		templates := v1.Group("/templates")
		{
			templates.GET("", h.GetTemplates)
			templates.POST("", h.CreateTemplate)
			templates.PUT("/:id", h.UpdateTemplate)
			templates.DELETE("/:id", h.DeleteTemplate)
		}

		// Batch notifications
		batches := v1.Group("/batch")
		{
			batches.GET("", h.GetBatchNotifications)
		}

		// Dead Letter Queue
		dlq := v1.Group("/dlq")
		{
			dlq.GET("", h.GetDLQEntries)
			dlq.POST("/:id/retry", h.RetryDLQEntry)
			dlq.DELETE("/:id", h.DeleteDLQEntry)
		}

		// Statistics
		v1.GET("/stats", h.GetStats)
	}
}
