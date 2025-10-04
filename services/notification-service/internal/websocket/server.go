package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/handlers"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/models"
)

// Server handles WebSocket connections and real-time notifications
type Server struct {
	upgrader    websocket.Upgrader
	connections map[string]*Connection
	mu          sync.RWMutex
	handler     *handlers.WebSocketHandler
	logger      *logrus.Logger
}

// Connection represents a WebSocket connection
type Connection struct {
	UserID   string
	Conn     *websocket.Conn
	Send     chan []byte
	LastPing time.Time
	IsActive bool
	mu       sync.Mutex
}

// Message represents a WebSocket message
type Message struct {
	Type      string                 `json:"type"`
	Data      interface{}            `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NotificationMessage represents a notification sent via WebSocket
type NotificationMessage struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	EventType string                 `json:"event_type"`
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	Priority  string                 `json:"priority"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// NewServer creates a new WebSocket server
func NewServer(handler *handlers.WebSocketHandler, logger *logrus.Logger) *Server {
	return &Server{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// In production, implement proper origin checking
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		connections: make(map[string]*Connection),
		handler:     handler,
		logger:      logger,
	}
}

// HandleWebSocket handles WebSocket connections
func (s *Server) HandleWebSocket(c *gin.Context) {
	// Extract user ID from query parameters or headers
	userID := c.Query("user_id")
	if userID == "" {
		userID = c.GetHeader("X-User-ID")
	}

	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	// Upgrade connection to WebSocket
	conn, err := s.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		s.logger.WithError(err).Error("Failed to upgrade WebSocket connection")
		return
	}

	// Create connection object
	connection := &Connection{
		UserID:   userID,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		LastPing: time.Now(),
		IsActive: true,
	}

	// Register connection
	s.registerConnection(userID, connection)

	// Start goroutines for handling the connection
	go s.handleConnection(connection)
	go s.writePump(connection)

	s.logger.WithField("user_id", userID).Info("WebSocket connection established")
}

// handleConnection handles incoming messages from a WebSocket connection
func (s *Server) handleConnection(conn *Connection) {
	defer func() {
		s.unregisterConnection(conn.UserID)
		conn.Conn.Close()
	}()

	// Set read deadline
	conn.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.Conn.SetPongHandler(func(string) error {
		conn.mu.Lock()
		conn.LastPing = time.Now()
		conn.mu.Unlock()
		conn.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := conn.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger.WithError(err).WithField("user_id", conn.UserID).Error("WebSocket error")
			}
			break
		}
	}
}

// writePump handles outgoing messages to a WebSocket connection
func (s *Server) writePump(conn *Connection) {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		conn.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-conn.Send:
			conn.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				conn.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := conn.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				s.logger.WithError(err).WithField("user_id", conn.UserID).Error("Failed to write WebSocket message")
				return
			}

		case <-ticker.C:
			conn.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				s.logger.WithError(err).WithField("user_id", conn.UserID).Error("Failed to send ping")
				return
			}
		}
	}
}

// registerConnection registers a new WebSocket connection
func (s *Server) registerConnection(userID string, conn *Connection) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Close existing connection if any
	if existingConn, exists := s.connections[userID]; exists {
		existingConn.IsActive = false
		close(existingConn.Send)
		existingConn.Conn.Close()
	}

	s.connections[userID] = conn
	s.logger.WithField("user_id", userID).Debug("WebSocket connection registered")
}

// unregisterConnection unregisters a WebSocket connection
func (s *Server) unregisterConnection(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if conn, exists := s.connections[userID]; exists {
		conn.IsActive = false
		close(conn.Send)
		delete(s.connections, userID)
		s.logger.WithField("user_id", userID).Debug("WebSocket connection unregistered")
	}
}

// SendNotification sends a notification to a specific user
func (s *Server) SendNotification(userID string, notification *models.Notification) error {
	s.mu.RLock()
	conn, exists := s.connections[userID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("user %s not connected", userID)
	}

	// Create notification message
	notifMsg := NotificationMessage{
		ID:        notification.ID.Hex(),
		UserID:    notification.UserID,
		EventType: string(notification.EventType),
		Title:     notification.Title,
		Message:   notification.Message,
		Priority:  string(notification.Priority),
		Metadata:  notification.Metadata,
		Timestamp: notification.CreatedAt,
	}

	// Create WebSocket message
	message := Message{
		Type:      "notification",
		Data:      notifMsg,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"channel": "websocket",
		},
	}

	// Marshal message
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Send message
	select {
	case conn.Send <- messageBytes:
		return nil
	default:
		return fmt.Errorf("connection send channel is full")
	}
}

// BroadcastNotification broadcasts a notification to all connected users
func (s *Server) BroadcastNotification(notification *models.Notification) {
	s.mu.RLock()
	connections := make(map[string]*Connection)
	for userID, conn := range s.connections {
		connections[userID] = conn
	}
	s.mu.RUnlock()

	// Create notification message
	notifMsg := NotificationMessage{
		ID:        notification.ID.Hex(),
		UserID:    notification.UserID,
		EventType: string(notification.EventType),
		Title:     notification.Title,
		Message:   notification.Message,
		Priority:  string(notification.Priority),
		Metadata:  notification.Metadata,
		Timestamp: notification.CreatedAt,
	}

	// Create WebSocket message
	message := Message{
		Type:      "notification",
		Data:      notifMsg,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"channel": "websocket",
		},
	}

	// Marshal message
	messageBytes, err := json.Marshal(message)
	if err != nil {
		s.logger.WithError(err).Error("Failed to marshal broadcast message")
		return
	}

	// Send to all connections
	for userID, conn := range connections {
		select {
		case conn.Send <- messageBytes:
			s.logger.WithField("user_id", userID).Debug("Broadcast message sent")
		default:
			s.logger.WithField("user_id", userID).Warn("Failed to send broadcast message, channel full")
		}
	}
}

// SendSystemMessage sends a system message to a specific user
func (s *Server) SendSystemMessage(userID string, messageType string, data interface{}) error {
	s.mu.RLock()
	conn, exists := s.connections[userID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("user %s not connected", userID)
	}

	// Create system message
	message := Message{
		Type:      messageType,
		Data:      data,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"channel": "websocket",
			"system":  true,
		},
	}

	// Marshal message
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal system message: %w", err)
	}

	// Send message
	select {
	case conn.Send <- messageBytes:
		return nil
	default:
		return fmt.Errorf("connection send channel is full")
	}
}

// BroadcastSystemMessage broadcasts a system message to all connected users
func (s *Server) BroadcastSystemMessage(messageType string, data interface{}) {
	s.mu.RLock()
	connections := make(map[string]*Connection)
	for userID, conn := range s.connections {
		connections[userID] = conn
	}
	s.mu.RUnlock()

	// Create system message
	message := Message{
		Type:      messageType,
		Data:      data,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"channel": "websocket",
			"system":  true,
		},
	}

	// Marshal message
	messageBytes, err := json.Marshal(message)
	if err != nil {
		s.logger.WithError(err).Error("Failed to marshal broadcast system message")
		return
	}

	// Send to all connections
	for userID, conn := range connections {
		select {
		case conn.Send <- messageBytes:
			s.logger.WithField("user_id", userID).Debug("Broadcast system message sent")
		default:
			s.logger.WithField("user_id", userID).Warn("Failed to send broadcast system message, channel full")
		}
	}
}

// GetConnectionCount returns the number of active connections
func (s *Server) GetConnectionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.connections)
}

// GetConnectedUsers returns a list of connected user IDs
func (s *Server) GetConnectedUsers() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]string, 0, len(s.connections))
	for userID := range s.connections {
		users = append(users, userID)
	}
	return users
}

// IsUserConnected checks if a user is connected
func (s *Server) IsUserConnected(userID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	conn, exists := s.connections[userID]
	return exists && conn.IsActive
}

// CloseConnection closes a specific user's connection
func (s *Server) CloseConnection(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if conn, exists := s.connections[userID]; exists {
		conn.IsActive = false
		close(conn.Send)
		conn.Conn.Close()
		delete(s.connections, userID)
		s.logger.WithField("user_id", userID).Info("WebSocket connection closed")
	}
}

// CloseAllConnections closes all WebSocket connections
func (s *Server) CloseAllConnections() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for userID, conn := range s.connections {
		conn.IsActive = false
		close(conn.Send)
		conn.Conn.Close()
		s.logger.WithField("user_id", userID).Debug("WebSocket connection closed")
	}

	s.connections = make(map[string]*Connection)
	s.logger.Info("All WebSocket connections closed")
}

// GetConnectionStats returns connection statistics
func (s *Server) GetConnectionStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := map[string]interface{}{
		"total_connections": len(s.connections),
		"connected_users":   make([]string, 0, len(s.connections)),
	}

	users := make([]string, 0, len(s.connections))
	for userID := range s.connections {
		users = append(users, userID)
	}
	stats["connected_users"] = users

	return stats
}

// StartCleanupRoutine starts a routine to clean up inactive connections
func (s *Server) StartCleanupRoutine(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	s.logger.Info("WebSocket cleanup routine started")

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("WebSocket cleanup routine stopped")
			return
		case <-ticker.C:
			s.cleanupInactiveConnections()
		}
	}
}

// cleanupInactiveConnections removes inactive connections
func (s *Server) cleanupInactiveConnections() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	inactiveThreshold := 5 * time.Minute

	for userID, conn := range s.connections {
		conn.mu.Lock()
		lastPing := conn.LastPing
		conn.mu.Unlock()

		if now.Sub(lastPing) > inactiveThreshold {
			conn.IsActive = false
			close(conn.Send)
			conn.Conn.Close()
			delete(s.connections, userID)
			s.logger.WithField("user_id", userID).Info("Removed inactive WebSocket connection")
		}
	}
}
