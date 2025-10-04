package repository

import (
	"context"
	"errors"
	"time"

	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ErrPreferencesNotFound = errors.New("user preferences not found")
)

type PreferencesRepository struct {
	collection *mongo.Collection
}

func NewPreferencesRepository(database *mongo.Database) *PreferencesRepository {
	return &PreferencesRepository{
		collection: database.Collection("user_notification_preferences"),
	}
}

// Create creates user notification preferences
func (r *PreferencesRepository) Create(ctx context.Context, preferences *models.UserNotificationPreferences) error {
	preferences.CreatedAt = time.Now()
	preferences.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, preferences)
	return err
}

// GetByUserID gets user preferences by user ID
func (r *PreferencesRepository) GetByUserID(ctx context.Context, userID string) (*models.UserNotificationPreferences, error) {
	var preferences models.UserNotificationPreferences

	err := r.collection.FindOne(ctx, bson.M{"user_id": userID}).Decode(&preferences)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrPreferencesNotFound
		}
		return nil, err
	}

	return &preferences, nil
}

// Update updates user preferences
func (r *PreferencesRepository) Update(ctx context.Context, userID string, preferences *models.UserNotificationPreferences) error {
	preferences.UpdatedAt = time.Now()

	filter := bson.M{"user_id": userID}
	update := bson.M{"$set": preferences}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return ErrPreferencesNotFound
	}

	return nil
}

// Upsert creates or updates user preferences
func (r *PreferencesRepository) Upsert(ctx context.Context, preferences *models.UserNotificationPreferences) error {
	preferences.UpdatedAt = time.Now()

	filter := bson.M{"user_id": preferences.UserID}
	update := bson.M{
		"$set": preferences,
		"$setOnInsert": bson.M{
			"created_at": time.Now(),
		},
	}

	opts := options.Update().SetUpsert(true)

	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

// Delete deletes user preferences
func (r *PreferencesRepository) Delete(ctx context.Context, userID string) error {
	filter := bson.M{"user_id": userID}

	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return ErrPreferencesNotFound
	}

	return nil
}

// GetDefaultPreferences returns default user preferences
func (r *PreferencesRepository) GetDefaultPreferences(userID string) *models.UserNotificationPreferences {
	return models.GetDefaultPreferences(userID)
}

// GetUsersByEventType gets users who are subscribed to a specific event type
func (r *PreferencesRepository) GetUsersByEventType(ctx context.Context, eventType models.EventType) ([]string, error) {
	filter := bson.M{
		"event_subscriptions": eventType,
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var preferences []models.UserNotificationPreferences
	if err = cursor.All(ctx, &preferences); err != nil {
		return nil, err
	}

	var userIDs []string
	for _, pref := range preferences {
		userIDs = append(userIDs, pref.UserID)
	}

	return userIDs, nil
}

// GetUsersByChannel gets users who have a specific channel enabled
func (r *PreferencesRepository) GetUsersByChannel(ctx context.Context, channel models.NotificationChannel) ([]string, error) {
	var filter bson.M

	switch channel {
	case models.ChannelEmail:
		filter = bson.M{"email_enabled": true}
	case models.ChannelSMS:
		filter = bson.M{"sms_enabled": true}
	case models.ChannelPush:
		filter = bson.M{"push_enabled": true}
	case models.ChannelInApp:
		filter = bson.M{"in_app_enabled": true}
	case models.ChannelWebSocket:
		filter = bson.M{"websocket_enabled": true}
	default:
		return []string{}, nil
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var preferences []models.UserNotificationPreferences
	if err = cursor.All(ctx, &preferences); err != nil {
		return nil, err
	}

	var userIDs []string
	for _, pref := range preferences {
		userIDs = append(userIDs, pref.UserID)
	}

	return userIDs, nil
}

// GetUsersInQuietHours gets users who are currently in quiet hours
func (r *PreferencesRepository) GetUsersInQuietHours(ctx context.Context) ([]string, error) {
	now := time.Now()
	currentTime := now.Format("15:04")

	filter := bson.M{
		"quiet_hours_enabled": true,
		"$or": []bson.M{
			// Case 1: Quiet hours don't cross midnight (e.g., 22:00 to 08:00)
			{
				"$and": []bson.M{
					{"quiet_hours_start": bson.M{"$lte": currentTime}},
					{"quiet_hours_end": bson.M{"$gte": currentTime}},
				},
			},
			// Case 2: Quiet hours cross midnight (e.g., 22:00 to 08:00)
			{
				"$and": []bson.M{
					{"quiet_hours_start": bson.M{"$gte": currentTime}},
					{"quiet_hours_end": bson.M{"$lte": currentTime}},
				},
			},
		},
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var preferences []models.UserNotificationPreferences
	if err = cursor.All(ctx, &preferences); err != nil {
		return nil, err
	}

	var userIDs []string
	for _, pref := range preferences {
		userIDs = append(userIDs, pref.UserID)
	}

	return userIDs, nil
}

// GetChannelPriorities gets channel priorities for a user and event type
func (r *PreferencesRepository) GetChannelPriorities(ctx context.Context, userID string, eventType models.EventType) ([]models.NotificationChannel, error) {
	var preferences models.UserNotificationPreferences

	err := r.collection.FindOne(ctx, bson.M{"user_id": userID}).Decode(&preferences)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// Return default priorities
			defaultPrefs := models.GetDefaultPreferences(userID)
			if priorities, exists := defaultPrefs.ChannelPriorities[eventType]; exists {
				return priorities, nil
			}
			return []models.NotificationChannel{}, nil
		}
		return nil, err
	}

	if priorities, exists := preferences.ChannelPriorities[eventType]; exists {
		return priorities, nil
	}

	// Return default priorities if not found
	defaultPrefs := models.GetDefaultPreferences(userID)
	if priorities, exists := defaultPrefs.ChannelPriorities[eventType]; exists {
		return priorities, nil
	}

	return []models.NotificationChannel{}, nil
}

// IsChannelEnabled checks if a channel is enabled for a user
func (r *PreferencesRepository) IsChannelEnabled(ctx context.Context, userID string, channel models.NotificationChannel) (bool, error) {
	var preferences models.UserNotificationPreferences

	err := r.collection.FindOne(ctx, bson.M{"user_id": userID}).Decode(&preferences)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// Return default preferences
			defaultPrefs := models.GetDefaultPreferences(userID)
			switch channel {
			case models.ChannelEmail:
				return defaultPrefs.EmailEnabled, nil
			case models.ChannelSMS:
				return defaultPrefs.SMSEnabled, nil
			case models.ChannelPush:
				return defaultPrefs.PushEnabled, nil
			case models.ChannelInApp:
				return defaultPrefs.InAppEnabled, nil
			case models.ChannelWebSocket:
				return defaultPrefs.WebSocketEnabled, nil
			}
			return false, nil
		}
		return false, err
	}

	switch channel {
	case models.ChannelEmail:
		return preferences.EmailEnabled, nil
	case models.ChannelSMS:
		return preferences.SMSEnabled, nil
	case models.ChannelPush:
		return preferences.PushEnabled, nil
	case models.ChannelInApp:
		return preferences.InAppEnabled, nil
	case models.ChannelWebSocket:
		return preferences.WebSocketEnabled, nil
	}

	return false, nil
}

// IsEventSubscribed checks if a user is subscribed to an event type
func (r *PreferencesRepository) IsEventSubscribed(ctx context.Context, userID string, eventType models.EventType) (bool, error) {
	var preferences models.UserNotificationPreferences

	err := r.collection.FindOne(ctx, bson.M{"user_id": userID}).Decode(&preferences)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// Return default preferences
			defaultPrefs := models.GetDefaultPreferences(userID)
			for _, subscribedEvent := range defaultPrefs.EventSubscriptions {
				if subscribedEvent == eventType {
					return true, nil
				}
			}
			return false, nil
		}
		return false, err
	}

	for _, subscribedEvent := range preferences.EventSubscriptions {
		if subscribedEvent == eventType {
			return true, nil
		}
	}

	return false, nil
}

// CreateIndexes creates necessary indexes
func (r *PreferencesRepository) CreateIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "event_subscriptions", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "email_enabled", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "sms_enabled", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "push_enabled", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "in_app_enabled", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "websocket_enabled", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "quiet_hours_enabled", Value: 1}},
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
}
