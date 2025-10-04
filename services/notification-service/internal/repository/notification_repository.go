package repository

import (
	"context"
	"errors"
	"time"

	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ErrNotificationNotFound = errors.New("notification not found")
)

type NotificationRepository struct {
	collection *mongo.Collection
}

func NewNotificationRepository(database *mongo.Database) *NotificationRepository {
	return &NotificationRepository{
		collection: database.Collection("notifications"),
	}
}

// Create creates a new notification
func (r *NotificationRepository) Create(ctx context.Context, notification *models.Notification) error {
	notification.CreatedAt = time.Now()
	notification.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, notification)
	return err
}

// GetByID gets a notification by ID
func (r *NotificationRepository) GetByID(ctx context.Context, id string) (*models.Notification, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var notification models.Notification
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&notification)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotificationNotFound
		}
		return nil, err
	}

	return &notification, nil
}

// GetByUserID gets notifications for a user with pagination
func (r *NotificationRepository) GetByUserID(ctx context.Context, userID string, page, limit int, status *models.NotificationStatus, eventType *models.EventType) ([]*models.Notification, int64, error) {
	filter := bson.M{"user_id": userID}

	if status != nil {
		filter["status"] = *status
	}

	if eventType != nil {
		filter["event_type"] = *eventType
	}

	// Get total count
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Calculate skip
	skip := (page - 1) * limit

	// Find notifications
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var notifications []*models.Notification
	if err = cursor.All(ctx, &notifications); err != nil {
		return nil, 0, err
	}

	return notifications, total, nil
}

// GetUnreadCount gets the count of unread notifications for a user
func (r *NotificationRepository) GetUnreadCount(ctx context.Context, userID string) (int64, error) {
	filter := bson.M{
		"user_id": userID,
		"$or": []bson.M{
			{"status": bson.M{"$ne": models.StatusRead}},
			{"read_at": bson.M{"$exists": false}},
		},
	}

	return r.collection.CountDocuments(ctx, filter)
}

// MarkAsRead marks a notification as read
func (r *NotificationRepository) MarkAsRead(ctx context.Context, id, userID string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	now := time.Now()
	filter := bson.M{
		"_id":     objectID,
		"user_id": userID,
	}
	update := bson.M{
		"$set": bson.M{
			"status":     models.StatusRead,
			"read_at":    now,
			"updated_at": now,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return ErrNotificationNotFound
	}

	return nil
}

// MarkAllAsRead marks all notifications as read for a user
func (r *NotificationRepository) MarkAllAsRead(ctx context.Context, userID string) (int64, error) {
	now := time.Now()
	filter := bson.M{
		"user_id": userID,
		"status":  bson.M{"$ne": models.StatusRead},
	}
	update := bson.M{
		"$set": bson.M{
			"status":     models.StatusRead,
			"read_at":    now,
			"updated_at": now,
		},
	}

	result, err := r.collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, err
	}

	return result.ModifiedCount, nil
}

// Delete deletes a notification
func (r *NotificationRepository) Delete(ctx context.Context, id, userID string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	filter := bson.M{
		"_id":     objectID,
		"user_id": userID,
	}

	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return ErrNotificationNotFound
	}

	return nil
}

// UpdateStatus updates the status of a notification
func (r *NotificationRepository) UpdateStatus(ctx context.Context, id string, status models.NotificationStatus, errorReason string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}

	if status == models.StatusSent {
		update["$set"].(bson.M)["sent_at"] = time.Now()
	}

	if errorReason != "" {
		update["$set"].(bson.M)["error_reason"] = errorReason
	}

	_, err = r.collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	return err
}

// AddDeliveryAttempt adds a delivery attempt to a notification
func (r *NotificationRepository) AddDeliveryAttempt(ctx context.Context, id string, attempt *models.DeliveryAttempt) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	update := bson.M{
		"$push": bson.M{
			"delivery_attempts": attempt,
		},
		"$set": bson.M{
			"updated_at": time.Now(),
		},
	}

	_, err = r.collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	return err
}

// UpdateRetryInfo updates retry information for a notification
func (r *NotificationRepository) UpdateRetryInfo(ctx context.Context, id string, retryCount int, nextRetryAt *time.Time, errorReason string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"retry_count":   retryCount,
			"last_retry_at": time.Now(),
			"updated_at":    time.Now(),
		},
	}

	if nextRetryAt != nil {
		update["$set"].(bson.M)["next_retry_at"] = nextRetryAt
	}

	if errorReason != "" {
		update["$set"].(bson.M)["error_reason"] = errorReason
	}

	_, err = r.collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	return err
}

// GetPendingRetries gets notifications that are ready for retry
func (r *NotificationRepository) GetPendingRetries(ctx context.Context, limit int) ([]*models.Notification, error) {
	filter := bson.M{
		"status":        models.StatusFailed,
		"retry_count":   bson.M{"$lt": 3},
		"next_retry_at": bson.M{"$lte": time.Now()},
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "next_retry_at", Value: 1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var notifications []*models.Notification
	if err = cursor.All(ctx, &notifications); err != nil {
		return nil, err
	}

	return notifications, nil
}

// GetFailedNotifications gets notifications that have failed all retries
func (r *NotificationRepository) GetFailedNotifications(ctx context.Context, limit int) ([]*models.Notification, error) {
	filter := bson.M{
		"status":      models.StatusFailed,
		"retry_count": bson.M{"$gte": 3},
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "updated_at", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var notifications []*models.Notification
	if err = cursor.All(ctx, &notifications); err != nil {
		return nil, err
	}

	return notifications, nil
}

// GetNotificationsByStatus gets notifications by status
func (r *NotificationRepository) GetNotificationsByStatus(ctx context.Context, status models.NotificationStatus, limit int) ([]*models.Notification, error) {
	filter := bson.M{"status": status}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var notifications []*models.Notification
	if err = cursor.All(ctx, &notifications); err != nil {
		return nil, err
	}

	return notifications, nil
}

// GetNotificationsByChannel gets notifications by channel
func (r *NotificationRepository) GetNotificationsByChannel(ctx context.Context, channel models.NotificationChannel, limit int) ([]*models.Notification, error) {
	filter := bson.M{"channel": channel}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var notifications []*models.Notification
	if err = cursor.All(ctx, &notifications); err != nil {
		return nil, err
	}

	return notifications, nil
}

// GetNotificationStats gets notification statistics
func (r *NotificationRepository) GetNotificationStats(ctx context.Context, userID string, startDate, endDate time.Time) (map[string]int64, error) {
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"user_id": userID,
				"created_at": bson.M{
					"$gte": startDate,
					"$lte": endDate,
				},
			},
		},
		{
			"$group": bson.M{
				"_id":   "$status",
				"count": bson.M{"$sum": 1},
			},
		},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	stats := make(map[string]int64)
	for cursor.Next(ctx) {
		var result struct {
			ID    string `bson:"_id"`
			Count int64  `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		stats[result.ID] = result.Count
	}

	return stats, nil
}

// CreateIndexes creates necessary indexes
func (r *NotificationRepository) CreateIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "status", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}, {Key: "next_retry_at", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "channel", Value: 1}, {Key: "created_at", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "event_type", Value: 1}, {Key: "created_at", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "created_at", Value: -1}},
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// GetByIDAndUserID gets a notification by ID and user ID
func (r *NotificationRepository) GetByIDAndUserID(ctx context.Context, notificationID, userID string) (*models.Notification, error) {
	objID, err := primitive.ObjectIDFromHex(notificationID)
	if err != nil {
		return nil, err
	}

	filter := bson.M{
		"_id":     objID,
		"user_id": userID,
	}

	var notification models.Notification
	err = r.collection.FindOne(ctx, filter).Decode(&notification)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotificationNotFound
		}
		return nil, err
	}

	return &notification, nil
}

// DeleteByIDAndUserID deletes a notification by ID and user ID
func (r *NotificationRepository) DeleteByIDAndUserID(ctx context.Context, notificationID, userID string) error {
	objID, err := primitive.ObjectIDFromHex(notificationID)
	if err != nil {
		return err
	}

	filter := bson.M{
		"_id":     objID,
		"user_id": userID,
	}

	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return ErrNotificationNotFound
	}

	return nil
}
