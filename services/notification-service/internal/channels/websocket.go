package channels

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// WebSocketChannel implements real-time notifications via WebSocket
type WebSocketChannel struct {
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
	Type      string                 `json:"type"`
	Title     string                 `json:"title"`
	Body      string                 `json:"body"`
	Link      string                 `json:"link,omitempty"`
	Priority  string                 `json:"priority"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// NewWebSocketChannel creates a new WebSocket notification channel
func NewWebSocketChannel(logger *logrus.Logger) *WebSocketChannel {
	return &WebSocketChannel{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// In production, implement proper origin checking
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		connections: make(map[string]*websocket.Conn),
		enabled:     true,
		logger:      logger,
	}
}

// Send sends a WebSocket notification
func (w *WebSocketChannel) Send(ctx context.Context, notification *NotificationRequest) (*DeliveryResult, error) {
	if !w.enabled {
		return &DeliveryResult{
			Channel:     "websocket",
			Success:     false,
			ErrorMessage: "WebSocket channel is disabled",
			DeliveredAt: time.Now(),
		}, fmt.Errorf("WebSocket channel is disabled")
	}

	if err := w.Validate(notification); err != nil {
		return &DeliveryResult{
			Channel:     "websocket",
			Success:     false,
			ErrorMessage: err.Error(),
			DeliveredAt: time.Now(),
		}, err
	}

	// Check if user has an active WebSocket connection
	w.mu.RLock()
	conn, exists := w.connections[notification.UserID]
	w.mu.RUnlock()

	if !exists {
		return &DeliveryResult{
			Channel:     "websocket",
			Success:     false,
			ErrorMessage: "user not connected via WebSocket",
			DeliveredAt: time.Now(),
		}, fmt.Errorf("user %s not connected via WebSocket", notification.UserID)
	}

	// Create WebSocket notification
	wsNotification := WebSocketNotification{
		ID:        fmt.Sprintf("notif_%d", time.Now().UnixNano()),
		UserID:    notification.UserID,
		Type:      notification.Type,
		Title:     notification.Title,
		Body:      notification.Body,
		Link:      notification.Link,
		Priority:  notification.Priority,
		Metadata:  make(map[string]interface{}),
		Timestamp: time.Now(),
	}

	// Convert metadata
	for key, value := range notification.Metadata {
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
	if err := w.sendMessage(conn, message); err != nil {
		// Remove disconnected connection
		w.removeConnection(notification.UserID)
		
		return &DeliveryResult{
			Channel:     "websocket",
			Success:     false,
			ErrorMessage: err.Error(),
			DeliveredAt: time.Now(),
		}, err
	}

	w.logger.WithFields(logrus.Fields{
		"user_id": notification.UserID,
		"type":    notification.Type,
	}).Info("WebSocket notification sent successfully")

	return &DeliveryResult{
		Channel:     "websocket",
		Success:     true,
		DeliveredAt: time.Now(),
		MessageID:   wsNotification.ID,
	}, nil
}

// sendMessage sends a message through WebSocket connection
func (w *WebSocketChannel) sendMessage(conn *websocket.Conn, message WebSocketMessage) error {
	// Set write deadline
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	
	// Send message as JSON
	return conn.WriteJSON(message)
}

// AddConnection adds a WebSocket connection for a user
func (w *WebSocketChannel) AddConnection(userID string, conn *websocket.Conn) {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	// Close existing connection if any
	if existingConn, exists := w.connections[userID]; exists {
		existingConn.Close()
	}
	
	w.connections[userID] = conn
	w.logger.WithField("user_id", userID).Info("WebSocket connection added")
}

// RemoveConnection removes a WebSocket connection for a user
func (w *WebSocketChannel) RemoveConnection(userID string) {
	w.removeConnection(userID)
}

// removeConnection removes a WebSocket connection (internal method)
func (w *WebSocketChannel) removeConnection(userID string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	if conn, exists := w.connections[userID]; exists {
		conn.Close()
		delete(w.connections, userID)
		w.logger.WithField("user_id", userID).Info("WebSocket connection removed")
	}
}

// GetConnectionCount returns the number of active connections
func (w *WebSocketChannel) GetConnectionCount() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.connections)
}

// GetConnectedUsers returns a list of connected user IDs
func (w *WebSocketChannel) GetConnectedUsers() []string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	
	users := make([]string, 0, len(w.connections))
	for userID := range w.connections {
		users = append(users, userID)
	}
	return users
}

// Broadcast sends a message to all connected users
func (w *WebSocketChannel) Broadcast(message WebSocketMessage) {
	w.mu.RLock()
	connections := make(map[string]*websocket.Conn)
	for userID, conn := range w.connections {
		connections[userID] = conn
	}
	w.mu.RUnlock()

	for userID, conn := range connections {
		if err := w.sendMessage(conn, message); err != nil {
			w.logger.WithFields(logrus.Fields{
				"user_id": userID,
				"error":   err,
			}).Error("Failed to broadcast message")
			w.removeConnection(userID)
		}
	}
}

// GetName returns the channel name
func (w *WebSocketChannel) GetName() string {
	return "websocket"
}

// IsEnabled checks if the channel is enabled
func (w *WebSocketChannel) IsEnabled() bool {
	return w.enabled
}

// Validate validates the notification request for WebSocket channel
func (w *WebSocketChannel) Validate(req *NotificationRequest) error {
	if req.UserID == "" {
		return fmt.Errorf("user ID is required for WebSocket notifications")
	}
	
	if req.Title == "" {
		return fmt.Errorf("title is required for WebSocket notifications")
	}
	
	if req.Body == "" {
		return fmt.Errorf("body is required for WebSocket notifications")
	}
	
	return nil
}

// HandleWebSocket handles WebSocket connections
func (w *WebSocketChannel) HandleWebSocket(writer http.ResponseWriter, request *http.Request) {
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
	conn, err := w.upgrader.Upgrade(writer, request, nil)
	if err != nil {
		w.logger.WithError(err).Error("Failed to upgrade WebSocket connection")
		return
	}
	defer conn.Close()

	// Add connection
	w.AddConnection(userID, conn)

	// Handle incoming messages (ping/pong, etc.)
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				w.logger.WithError(err).Error("WebSocket error")
			}
			break
		}
	}

	// Remove connection when done
	w.RemoveConnection(userID)
}

