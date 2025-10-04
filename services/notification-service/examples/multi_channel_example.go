package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/channels"
	"github.com/sirupsen/logrus"
)

// Example demonstrating multi-channel notification system
func main() {
	// Setup logger
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// Create notification channels
	emailChannel := channels.NewEmailChannel(
		"your-sendgrid-api-key",
		"noreply@file-sharing.com",
		"File Sharing Platform",
		logger,
	)

	smsChannel := channels.NewSMSChannel(
		"your-twilio-account-sid",
		"your-twilio-auth-token",
		"+1234567890",
		logger,
	)

	pushChannel := channels.NewPushChannel(
		"your-fcm-server-key",
		"your-apns-key-id",
		"path/to/apns-key.p8",
		"com.yourcompany.filesharing",
		logger,
	)

	websocketChannel := channels.NewWebSocketChannel(logger)
	inAppChannel := channels.NewInAppChannel(logger)

	// Create channel manager
	config := &channels.ManagerConfig{
		EnableFallback:   true,
		FallbackChannel:  "email",
		RetryAttempts:    3,
		RetryDelay:       5 * time.Second,
		Timeout:          30 * time.Second,
	}

	manager := channels.NewChannelManager(config, logger)

	// Register channels
	manager.RegisterChannel(emailChannel)
	manager.RegisterChannel(smsChannel)
	manager.RegisterChannel(pushChannel)
	manager.RegisterChannel(websocketChannel)
	manager.RegisterChannel(inAppChannel)

	// Example 1: Send notification via multiple channels
	fmt.Println("=== Example 1: Multi-Channel Notification ===")
	notificationReq := &channels.NotificationRequest{
		UserID:    "user123",
		Type:      "file.uploaded",
		Title:     "File Upload Complete",
		Body:      "Your file 'document.pdf' has been uploaded successfully",
		Link:      "https://app.filesharing.com/files/abc123",
		Priority:  "normal",
		Email:     "user@example.com",
		Phone:     "+1234567890",
		PushToken: "fcm_token_here",
		Channels:  []string{"email", "sms", "push", "inapp"},
		Metadata: map[string]string{
			"file_id":   "abc123",
			"file_name": "document.pdf",
			"file_size": "1024",
		},
	}

	ctx := context.Background()
	results, err := manager.SendMultiChannel(ctx, notificationReq)
	if err != nil {
		log.Printf("Error sending multi-channel notification: %v", err)
	} else {
		fmt.Printf("Multi-channel notification sent successfully!\n")
		for _, result := range results {
			status := "SUCCESS"
			if !result.Success {
				status = "FAILED"
			}
			fmt.Printf("  %s: %s - %s\n", result.Channel, status, result.ErrorMessage)
		}
	}

	// Example 2: Send with fallback mechanism
	fmt.Println("\n=== Example 2: Fallback Notification ===")
	fallbackReq := &channels.NotificationRequest{
		UserID:    "user456",
		Type:      "file.shared",
		Title:     "File Shared With You",
		Body:      "John Doe shared 'presentation.pptx' with you",
		Link:      "https://app.filesharing.com/files/def456",
		Priority:  "high",
		Email:     "user456@example.com",
		Channels:  []string{"push", "websocket"}, // These might fail
	}

	results, err = manager.SendWithFallback(ctx, fallbackReq)
	if err != nil {
		log.Printf("Error sending fallback notification: %v", err)
	} else {
		fmt.Printf("Fallback notification sent!\n")
		for _, result := range results {
			status := "SUCCESS"
			if !result.Success {
				status = "FAILED"
			}
			fmt.Printf("  %s: %s - %s\n", result.Channel, status, result.ErrorMessage)
		}
	}

	// Example 3: User preferences
	fmt.Println("\n=== Example 3: User Preferences ===")
	preferences := &channels.UserPreferences{
		UserID: "user123",
		EnabledChannels: []string{"email", "inapp", "websocket"},
		ChannelSettings: map[string]bool{
			"email":     true,
			"sms":       false,
			"push":      false,
			"inapp":     true,
			"websocket": true,
		},
		TypeSettings: map[string]bool{
			"file.uploaded": true,
			"file.shared":   true,
			"file.deleted":  false,
			"system":        true,
		},
		Email:                 "user123@example.com",
		Phone:                 "+1234567890",
		PushToken:             "fcm_token_123",
		EmailNotifications:    true,
		SMSNotifications:      false,
		PushNotifications:     false,
		InAppNotifications:    true,
		WebSocketNotifications: true,
	}

	fmt.Printf("User preferences configured:\n")
	fmt.Printf("  Enabled channels: %v\n", preferences.EnabledChannels)
	fmt.Printf("  Email notifications: %v\n", preferences.EmailNotifications)
	fmt.Printf("  SMS notifications: %v\n", preferences.SMSNotifications)

	// Example 4: Channel status
	fmt.Println("\n=== Example 4: Channel Status ===")
	status := manager.GetChannelStatus()
	fmt.Printf("Channel status:\n")
	for channel, enabled := range status {
		fmt.Printf("  %s: %v\n", channel, enabled)
	}

	// Example 5: WebSocket connection simulation
	fmt.Println("\n=== Example 5: WebSocket Connection ===")
	// In a real application, this would be handled by HTTP server
	// Here we just demonstrate the concept
	connectedUsers := websocketChannel.GetConnectedUsers()
	fmt.Printf("Connected users: %v\n", connectedUsers)
	fmt.Printf("Connection count: %d\n", websocketChannel.GetConnectionCount())

	fmt.Println("\n=== Multi-Channel Notification System Demo Complete ===")
}

// Example of how to integrate with the notification service
func ExampleIntegration() {
	logger := logrus.New()
	
	// This would be called from your main service
	// when you want to send a notification
	
	// 1. Create notification request
	req := &channels.NotificationRequest{
		UserID:    "user789",
		Type:      "file.uploaded",
		Title:     "Upload Complete",
		Body:      "Your file has been uploaded successfully",
		Channels:  []string{"email", "inapp", "websocket"},
		Priority:  "normal",
		Email:     "user789@example.com",
	}

	// 2. Send via service (this would be injected)
	// results, err := notificationService.SendMultiChannelNotification(ctx, req)
	
	fmt.Printf("Would send notification: %+v\n", req)
}
