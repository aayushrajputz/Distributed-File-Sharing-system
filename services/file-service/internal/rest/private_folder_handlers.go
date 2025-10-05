package rest

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/models"
	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/service"
)

// PrivateFolderHandlers handles private folder REST endpoints
type PrivateFolderHandlers struct {
	service *service.PrivateFolderService
	logger  *logrus.Logger
}

// NewPrivateFolderHandlers creates new private folder handlers
func NewPrivateFolderHandlers(service *service.PrivateFolderService, logger *logrus.Logger) *PrivateFolderHandlers {
	return &PrivateFolderHandlers{
		service: service,
		logger:  logger,
	}
}

// SetPIN sets or updates a user's PIN
// POST /api/v1/private-folder/set-pin
func (h *PrivateFolderHandlers) SetPIN(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id" binding:"required"`
		PIN    string `json:"pin" binding:"required,min=4,max=8"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.service.SetPIN(c.Request.Context(), req.UserID, req.PIN)
	if err != nil {
		h.logger.WithError(err).Error("Failed to set PIN")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "PIN set successfully",
	})
}

// ValidatePIN validates a user's PIN
// POST /api/v1/private-folder/validate-pin
func (h *PrivateFolderHandlers) ValidatePIN(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id" binding:"required"`
		PIN    string `json:"pin" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pinReq := &models.PINValidationRequest{
		UserID:    req.UserID,
		PIN:       req.PIN,
		IPAddress: c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
	}

	resp, err := h.service.ValidatePIN(c.Request.Context(), pinReq)
	if err != nil {
		h.logger.WithError(err).Error("Failed to validate PIN")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Internal server error",
		})
		return
	}

	statusCode := http.StatusOK
	if !resp.Success {
		statusCode = http.StatusUnauthorized
	}

	c.JSON(statusCode, gin.H{
		"success":       resp.Success,
		"message":       resp.Message,
		"attempts_left": resp.AttemptsLeft,
		"locked_until":  resp.LockedUntil,
	})
}

// MakeFilePrivate moves a file to private folder
// POST /api/v1/private-folder/make-private
func (h *PrivateFolderHandlers) MakeFilePrivate(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id" binding:"required"`
		FileID string `json:"file_id" binding:"required"`
		PIN    string `json:"pin" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	makePrivateReq := &models.MakePrivateRequest{
		UserID: req.UserID,
		FileID: req.FileID,
		PIN:    req.PIN,
	}

	resp, err := h.service.MakeFilePrivate(c.Request.Context(), makePrivateReq)
	if err != nil {
		h.logger.WithError(err).Error("Failed to make file private")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Internal server error",
		})
		return
	}

	statusCode := http.StatusOK
	if !resp.Success {
		statusCode = http.StatusBadRequest
	}

	c.JSON(statusCode, gin.H{
		"success": resp.Success,
		"message": resp.Message,
		"file_id": resp.FileID,
	})
}

// RemoveFileFromPrivate removes a file from private folder
// POST /api/v1/private-folder/remove-from-private
func (h *PrivateFolderHandlers) RemoveFileFromPrivate(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id" binding:"required"`
		FileID string `json:"file_id" binding:"required"`
		PIN    string `json:"pin" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.RemoveFileFromPrivate(c.Request.Context(), req.UserID, req.FileID, req.PIN)
	if err != nil {
		h.logger.WithError(err).Error("Failed to remove file from private folder")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Internal server error",
		})
		return
	}

	statusCode := http.StatusOK
	if !resp.Success {
		statusCode = http.StatusBadRequest
	}

	c.JSON(statusCode, gin.H{
		"success": resp.Success,
		"message": resp.Message,
		"file_id": resp.FileID,
	})
}

// GetPrivateFiles retrieves all private files for a user
// GET /api/v1/private-folder/files?user_id=xxx&limit=10&offset=0
func (h *PrivateFolderHandlers) GetPrivateFiles(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit parameter"})
		return
	}

	offset, err := strconv.ParseInt(offsetStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offset parameter"})
		return
	}

	resp, err := h.service.GetPrivateFiles(c.Request.Context(), userID, limit, offset)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get private files")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Private files retrieved successfully",
		"files":   resp.Files,
		"total":   resp.Total,
	})
}

// GetAccessLogs retrieves access logs for a user
// GET /api/v1/private-folder/access-logs?user_id=xxx&limit=50
func (h *PrivateFolderHandlers) GetAccessLogs(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit parameter"})
		return
	}

	logs, err := h.service.GetAccessLogs(c.Request.Context(), userID, limit)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get access logs")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "Access logs retrieved successfully",
		"access_logs": logs,
	})
}

// CheckFileAccess checks if user can access a private file
// GET /api/v1/private-folder/check-access?user_id=xxx&file_id=xxx
func (h *PrivateFolderHandlers) CheckFileAccess(c *gin.Context) {
	userID := c.Query("user_id")
	fileID := c.Query("file_id")

	if userID == "" || fileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id and file_id are required"})
		return
	}

	hasAccess, err := h.service.CheckFileAccess(c.Request.Context(), userID, fileID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to check file access")
		c.JSON(http.StatusInternalServerError, gin.H{
			"has_access": false,
			"message":    "Internal server error",
		})
		return
	}

	message := "Access granted"
	if !hasAccess {
		message = "PIN required for private file access"
	}

	c.JSON(http.StatusOK, gin.H{
		"has_access": hasAccess,
		"message":    message,
	})
}

// RegisterRoutes registers all private folder routes
func (h *PrivateFolderHandlers) RegisterRoutes(router *gin.RouterGroup) {
	privateFolder := router.Group("/private-folder")
	{
		privateFolder.POST("/set-pin", h.SetPIN)
		privateFolder.POST("/validate-pin", h.ValidatePIN)
		privateFolder.POST("/make-private", h.MakeFilePrivate)
		privateFolder.POST("/remove-from-private", h.RemoveFileFromPrivate)
		privateFolder.GET("/files", h.GetPrivateFiles)
		privateFolder.GET("/access-logs", h.GetAccessLogs)
		privateFolder.GET("/check-access", h.CheckFileAccess)
	}
}


