package kafka

import (
	"sync"

	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/models"
)

// StreamBroker manages gRPC streaming connections and broadcasts notifications
type StreamBroker struct {
	mu          sync.RWMutex
	subscribers map[string][]chan *models.Notification
}

func NewStreamBroker() *StreamBroker {
	return &StreamBroker{
		subscribers: make(map[string][]chan *models.Notification),
	}
}

// Subscribe creates a new channel for user to receive notifications
func (sb *StreamBroker) Subscribe(userID string) chan *models.Notification {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	ch := make(chan *models.Notification, 10)
	sb.subscribers[userID] = append(sb.subscribers[userID], ch)

	return ch
}

// Unsubscribe removes a channel from subscribers
func (sb *StreamBroker) Unsubscribe(userID string, ch chan *models.Notification) {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	channels := sb.subscribers[userID]
	for i, sub := range channels {
		if sub == ch {
			close(ch)
			sb.subscribers[userID] = append(channels[:i], channels[i+1:]...)
			break
		}
	}

	// Clean up if no more subscribers for this user
	if len(sb.subscribers[userID]) == 0 {
		delete(sb.subscribers, userID)
	}
}

// Broadcast sends notification to all subscribers of a user
func (sb *StreamBroker) Broadcast(notification *models.Notification) {
	sb.mu.RLock()
	defer sb.mu.RUnlock()

	channels := sb.subscribers[notification.UserID]
	for _, ch := range channels {
		select {
		case ch <- notification:
		default:
			// Channel buffer full, skip
		}
	}
}
