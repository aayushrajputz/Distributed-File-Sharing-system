package repository

import (
	"context"
	"errors"
	"time"

	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/channels"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ErrUserPreferencesNotFound = errors.New("user preferences not found")
)

type UserPreferencesRepository struct {
	collection *mongo.Collection
}

func NewUserPreferencesRepository(collection *mongo.Collection) *UserPreferencesRepository {
	return &UserPreferencesRepository{
		collection: collection,
	}
}

// Create creates user preferences
func (r *UserPreferencesRepository) Create(ctx context.Context, preferences *channels.UserPreferences) error {
	preferences.CreatedAt = time.Now()
	preferences.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, preferences)
	return err
}

// GetByUserID gets user preferences by user ID
func (r *UserPreferencesRepository) GetByUserID(ctx context.Context, userID string) (*channels.UserPreferences, error) {
	var preferences channels.UserPreferences

	err := r.collection.FindOne(ctx, bson.M{"user_id": userID}).Decode(&preferences)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrUserPreferencesNotFound
		}
		return nil, err
	}

	return &preferences, nil
}

// Update updates user preferences
func (r *UserPreferencesRepository) Update(ctx context.Context, userID string, preferences *channels.UserPreferences) error {
	preferences.UpdatedAt = time.Now()

	filter := bson.M{"user_id": userID}
	update := bson.M{"$set": preferences}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return ErrUserPreferencesNotFound
	}

	return nil
}

// Upsert creates or updates user preferences
func (r *UserPreferencesRepository) Upsert(ctx context.Context, preferences *channels.UserPreferences) error {
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
func (r *UserPreferencesRepository) Delete(ctx context.Context, userID string) error {
	filter := bson.M{"user_id": userID}

	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return ErrUserPreferencesNotFound
	}

	return nil
}

// GetDefaultPreferences returns default user preferences
func (r *UserPreferencesRepository) GetDefaultPreferences(userID string) *channels.UserPreferences {
	return &channels.UserPreferences{
		UserID:          userID,
		EnabledChannels: []string{"inapp", "email"},
		ChannelSettings: map[string]bool{
			"inapp":     true,
			"email":     true,
			"sms":       false,
			"push":      false,
			"websocket": true,
		},
		TypeSettings: map[string]bool{
			"file.uploaded": true,
			"file.shared":   true,
			"file.deleted":  true,
			"share.revoked": true,
			"system":        true,
		},
		EmailNotifications:     true,
		SMSNotifications:       false,
		PushNotifications:      false,
		InAppNotifications:     true,
		WebSocketNotifications: true,
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
	}
}
