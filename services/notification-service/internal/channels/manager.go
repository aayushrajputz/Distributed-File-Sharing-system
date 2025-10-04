package channels

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ChannelManager manages all notification channels
type channelManager struct {
	channels map[string]NotificationChannel
	mu       sync.RWMutex
	logger   *logrus.Logger
	config   *ManagerConfig
}

// ManagerConfig contains configuration for the channel manager
type ManagerConfig struct {
	EnableFallback   bool
	FallbackChannel  string
	RetryAttempts    int
	RetryDelay       time.Duration
	Timeout          time.Duration
}

// NewChannelManager creates a new channel manager
func NewChannelManager(config *ManagerConfig, logger *logrus.Logger) ChannelManager {
	return &channelManager{
		channels: make(map[string]NotificationChannel),
		logger:   logger,
		config:   config,
	}
}

// RegisterChannel registers a new notification channel
func (cm *channelManager) RegisterChannel(channel NotificationChannel) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	cm.channels[channel.GetName()] = channel
	cm.logger.WithField("channel", channel.GetName()).Info("Notification channel registered")
}

// GetChannel returns a specific channel by name
func (cm *channelManager) GetChannel(name string) (NotificationChannel, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	channel, exists := cm.channels[name]
	if !exists {
		return nil, fmt.Errorf("channel %s not found", name)
	}
	
	return channel, nil
}

// GetEnabledChannels returns all enabled channels
func (cm *channelManager) GetEnabledChannels() []NotificationChannel {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	var enabled []NotificationChannel
	for _, channel := range cm.channels {
		if channel.IsEnabled() {
			enabled = append(enabled, channel)
		}
	}
	
	return enabled
}

// SendMultiChannel sends notification through multiple channels
func (cm *channelManager) SendMultiChannel(ctx context.Context, req *NotificationRequest) ([]*DeliveryResult, error) {
	if len(req.Channels) == 0 {
		return nil, fmt.Errorf("no channels specified")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, cm.config.Timeout)
	defer cancel()

	var results []*DeliveryResult
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Send to each specified channel concurrently
	for _, channelName := range req.Channels {
		wg.Add(1)
		go func(ch string) {
			defer wg.Done()
			
			channel, err := cm.GetChannel(ch)
			if err != nil {
				cm.logger.WithError(err).WithField("channel", ch).Error("Channel not found")
				return
			}

			result, err := cm.sendWithRetry(ctx, channel, req)
			if err != nil {
				cm.logger.WithError(err).WithField("channel", ch).Error("Failed to send notification")
			}

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(channelName)
	}

	wg.Wait()

	// Check if any channel succeeded
	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}
	}

	if successCount == 0 {
		return results, fmt.Errorf("all notification channels failed")
	}

	cm.logger.WithFields(logrus.Fields{
		"user_id":       req.UserID,
		"channels":      req.Channels,
		"success_count": successCount,
		"total_count":   len(results),
	}).Info("Multi-channel notification sent")

	return results, nil
}

// SendWithFallback sends notification with fallback mechanism
func (cm *channelManager) SendWithFallback(ctx context.Context, req *NotificationRequest) ([]*DeliveryResult, error) {
	if !cm.config.EnableFallback {
		return cm.SendMultiChannel(ctx, req)
	}

	// Try primary channels first
	results, err := cm.SendMultiChannel(ctx, req)
	if err == nil {
		// Check if any primary channel succeeded
		successCount := 0
		for _, result := range results {
			if result.Success {
				successCount++
			}
		}
		
		if successCount > 0 {
			return results, nil
		}
	}

	// If all primary channels failed, try fallback
	cm.logger.WithFields(logrus.Fields{
		"user_id":         req.UserID,
		"fallback_channel": cm.config.FallbackChannel,
	}).Warn("Primary channels failed, attempting fallback")

	fallbackChannel, err := cm.GetChannel(cm.config.FallbackChannel)
	if err != nil {
		return results, fmt.Errorf("fallback channel %s not available: %w", cm.config.FallbackChannel, err)
	}

	fallbackResult, err := cm.sendWithRetry(ctx, fallbackChannel, req)
	if err != nil {
		return results, fmt.Errorf("fallback channel also failed: %w", err)
	}

	results = append(results, fallbackResult)
	
	cm.logger.WithFields(logrus.Fields{
		"user_id":         req.UserID,
		"fallback_channel": cm.config.FallbackChannel,
		"success":         fallbackResult.Success,
	}).Info("Fallback notification sent")

	return results, nil
}

// sendWithRetry sends notification with retry logic
func (cm *channelManager) sendWithRetry(ctx context.Context, channel NotificationChannel, req *NotificationRequest) (*DeliveryResult, error) {
	var lastErr error
	
	for attempt := 0; attempt <= cm.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return &DeliveryResult{
					Channel:     channel.GetName(),
					Success:     false,
					ErrorMessage: "context cancelled",
					DeliveredAt: time.Now(),
				}, ctx.Err()
			case <-time.After(cm.config.RetryDelay):
				// Retry delay
			}
		}

		result, err := channel.Send(ctx, req)
		if err == nil && result.Success {
			return result, nil
		}
		
		lastErr = err
		cm.logger.WithFields(logrus.Fields{
			"channel": channel.GetName(),
			"attempt": attempt + 1,
			"error":   err,
		}).Warn("Channel send attempt failed")
	}

	return &DeliveryResult{
		Channel:     channel.GetName(),
		Success:     false,
		ErrorMessage: fmt.Sprintf("failed after %d attempts: %v", cm.config.RetryAttempts+1, lastErr),
		DeliveredAt: time.Now(),
	}, lastErr
}

// GetChannelStatus returns the status of all channels
func (cm *channelManager) GetChannelStatus() map[string]bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	status := make(map[string]bool)
	for name, channel := range cm.channels {
		status[name] = channel.IsEnabled()
	}
	
	return status
}

// GetChannelNames returns all registered channel names
func (cm *channelManager) GetChannelNames() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	var names []string
	for name := range cm.channels {
		names = append(names, name)
	}
	
	return names
}

// ValidateChannels validates that all specified channels are available and enabled
func (cm *channelManager) ValidateChannels(channelNames []string) error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	var missingChannels []string
	var disabledChannels []string
	
	for _, name := range channelNames {
		channel, exists := cm.channels[name]
		if !exists {
			missingChannels = append(missingChannels, name)
			continue
		}
		
		if !channel.IsEnabled() {
			disabledChannels = append(disabledChannels, name)
		}
	}
	
	var errors []string
	if len(missingChannels) > 0 {
		errors = append(errors, fmt.Sprintf("missing channels: %s", strings.Join(missingChannels, ", ")))
	}
	if len(disabledChannels) > 0 {
		errors = append(errors, fmt.Sprintf("disabled channels: %s", strings.Join(disabledChannels, ", ")))
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("channel validation failed: %s", strings.Join(errors, "; "))
	}
	
	return nil
}

