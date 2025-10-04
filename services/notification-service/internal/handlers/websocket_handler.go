package handlers

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/models"
)

// WebSocketHandler handles WebSocket notifications
type WebSocketHandler struct {
	upgrader    websocket.Upgrader
	connections map[string]*websocket.Conn
	mu          sync.RWMutex
	enabled     bool
	logger      *logrus.Logger
}

// WebSocketMessage represents a WebSocket message
type WebSocketMessage struct {
	Type      string                 `json:"type"`
	Data      interface{}            `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// WebSocketNotification represents a notification sent via WebSocket
type WebSocketNotification struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	EventType string                 `json:"event_type"`
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	Priority  string                 `json:"priority"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// NewWebSocketHandler creates a new WebSocket notification handler
func NewWebSocketHandler(enabled bool, logger *logrus.Logger) *WebSocketHandler {
	return &WebSocketHandler{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// In production, implement proper origin checking
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		connections: make(map[string]*websocket.Conn),
		enabled:     enabled,
		logger:      logger,
	}
}

// Send sends a WebSocket notification
func (h *WebSocketHandler) Send(ctx context.Context, req *models.NotificationRequest) (*models.NotificationResponse, error) {
	start := time.Now()

	// Validate request
	if err := h.Validate(req); err != nil {
		return &models.NotificationResponse{
			Status:   models.StatusFailed,
			Channel:  models.ChannelWebSocket,
			Error:    err.Error(),
			Duration: time.Since(start).Milliseconds(),
		}, err
	}

	// Check if user has an active WebSocket connection
	h.mu.RLock()
	conn, exists := h.connections[req.UserID]
	h.mu.RUnlock()

	if !exists {
		return &models.NotificationResponse{
			Status:   models.StatusFailed,
			Channel:  models.ChannelWebSocket,
			Error:    "user not connected via WebSocket",
			Duration: time.Since(start).Milliseconds(),
		}, fmt.Errorf("user %s not connected via WebSocket", req.UserID)
	}

	// Create WebSocket notification
	wsNotification := WebSocketNotification{
		ID:        fmt.Sprintf("notif_%d", time.Now().UnixNano()),
		UserID:    req.UserID,
		EventType: string(req.EventType),
		Title:     req.Title,
		Message:   req.Message,
		Priority:  string(req.Priority),
		Metadata:  make(map[string]interface{}),
		Timestamp: time.Now(),
	}

	// Convert metadata
	for key, value := range req.Metadata {
		wsNotification.Metadata[key] = value
	}

	// Create WebSocket message
	message := WebSocketMessage{
		Type:      "notification",
		Data:      wsNotification,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"channel": "websocket",
		},
	}

	// Send message
	if err := h.sendMessage(conn, message); err != nil {
		// Remove disconnected connection
		h.removeConnection(req.UserID)

		return &models.NotificationResponse{
			Status:   models.StatusFailed,
			Channel:  models.ChannelWebSocket,
			Error:    err.Error(),
			Duration: time.Since(start).Milliseconds(),
		}, err
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":    req.UserID,
		"event_type": req.EventType,
		"title":      req.Title,
	}).Info("WebSocket notification sent successfully")

	return &models.NotificationResponse{
		Status:   models.StatusSent,
		Channel:  models.ChannelWebSocket,
		Duration: time.Since(start).Milliseconds(),
	}, nil
}

// sendMessage sends a message through WebSocket connection
func (h *WebSocketHandler) sendMessage(conn *websocket.Conn, message WebSocketMessage) error {
	// Set write deadline
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

	// Send message as JSON
	return conn.WriteJSON(message)
}

// AddConnection adds a WebSocket connection for a user
func (h *WebSocketHandler) AddConnection(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Close existing connection if any
	if existingConn, exists := h.connections[userID]; exists {
		existingConn.Close()
	}

	h.connections[userID] = conn
	h.logger.WithField("user_id", userID).Info("WebSocket connection added")
}

// RemoveConnection removes a WebSocket connection for a user
func (h *WebSocketHandler) RemoveConnection(userID string) {
	h.removeConnection(userID)
}

// removeConnection removes a WebSocket connection (internal method)
func (h *WebSocketHandler) removeConnection(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if conn, exists := h.connections[userID]; exists {
		conn.Close()
		delete(h.connections, userID)
		h.logger.WithField("user_id", userID).Info("WebSocket connection removed")
	}
}

// GetConnectionCount returns the number of active connections
func (h *WebSocketHandler) GetConnectionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.connections)
}

// GetConnectedUsers returns a list of connected user IDs
func (h *WebSocketHandler) GetConnectedUsers() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	users := make([]string, 0, len(h.connections))
	for userID := range h.connections {
		users = append(users, userID)
	}
	return users
}

// Broadcast sends a message to all connected users
func (h *WebSocketHandler) Broadcast(message WebSocketMessage) {
	h.mu.RLock()
	connections := make(map[string]*websocket.Conn)
	for userID, conn := range h.connections {
		connections[userID] = conn
	}
	h.mu.RUnlock()

	for userID, conn := range connections {
		if err := h.sendMessage(conn, message); err != nil {
			h.logger.WithFields(logrus.Fields{
				"user_id": userID,
				"error":   err,
			}).Error("Failed to broadcast message")
			h.removeConnection(userID)
		}
	}
}

// HandleWebSocket handles WebSocket connections
func (h *WebSocketHandler) HandleWebSocket(writer http.ResponseWriter, request *http.Request) {
	// Extract user ID from query parameters or headers
	userID := request.URL.Query().Get("user_id")
	if userID == "" {
		userID = request.Header.Get("X-User-ID")
	}

	if userID == "" {
		http.Error(writer, "user_id is required", http.StatusBadRequest)
		return
	}

	// Upgrade connection to WebSocket
	conn, err := h.upgrader.Upgrade(writer, request, nil)
	if err != nil {
		h.logger.WithError(err).Error("Failed to upgrade WebSocket connection")
		return
	}
	defer conn.Close()

	// Add connection
	h.AddConnection(userID, conn)

	// Handle incoming messages (ping/pong, etc.)
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.logger.WithError(err).Error("WebSocket error")
			}
			break
		}
	}

	// Remove connection when done
	h.RemoveConnection(userID)
}

// Validate validates the notification request
func (h *WebSocketHandler) Validate(req *models.NotificationRequest) error {
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
func (h *WebSocketHandler) GetName() string {
	return "websocket"
}

// IsEnabled checks if the handler is enabled
func (h *WebSocketHandler) IsEnabled() bool {
	return h.enabled
}

// TestConnection tests the WebSocket handler
func (h *WebSocketHandler) TestConnection(ctx context.Context) error {
	if !h.enabled {
		return fmt.Errorf("WebSocket handler is disabled")
	}

	// WebSocket handler doesn't require external connections
	// Just check if it's enabled
	return nil
}
