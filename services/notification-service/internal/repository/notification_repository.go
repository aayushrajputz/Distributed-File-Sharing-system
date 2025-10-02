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

func NewNotificationRepository(db *mongo.Database) *NotificationRepository {
	return &NotificationRepository{
		collection: db.Collection("notifications"),
	}
}

func (r *NotificationRepository) Create(ctx context.Context, notification *models.Notification) error {
	notification.ID = primitive.NewObjectID()
	notification.CreatedAt = time.Now()
	notification.IsRead = false

	_, err := r.collection.InsertOne(ctx, notification)
	return err
}

func (r *NotificationRepository) FindByID(ctx context.Context, id string) (*models.Notification, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var notification models.Notification
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&notification)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotificationNotFound
		}
		return nil, err
	}
	return &notification, nil
}

func (r *NotificationRepository) FindByUser(ctx context.Context, userID string, page, limit int32, unreadOnly bool) ([]*models.Notification, int64, int64, error) {
	skip := (page - 1) * limit

	filter := bson.M{"user_id": userID}
	if unreadOnly {
		filter["is_read"] = false
	}

	cursor, err := r.collection.Find(
		ctx,
		filter,
		options.Find().SetSkip(int64(skip)).SetLimit(int64(limit)).SetSort(bson.M{"created_at": -1}),
	)
	if err != nil {
		return nil, 0, 0, err
	}
	defer cursor.Close(ctx)

	var notifications []*models.Notification
	if err = cursor.All(ctx, &notifications); err != nil {
		return nil, 0, 0, err
	}

	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, 0, err
	}

	unreadCount, err := r.collection.CountDocuments(ctx, bson.M{"user_id": userID, "is_read": false})
	if err != nil {
		return nil, 0, 0, err
	}

	return notifications, total, unreadCount, nil
}

func (r *NotificationRepository) MarkAsRead(ctx context.Context, id, userID string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	filter := bson.M{"_id": objectID, "user_id": userID}
	update := bson.M{"$set": bson.M{"is_read": true}}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return ErrNotificationNotFound
	}

	return nil
}

func (r *NotificationRepository) MarkAllAsRead(ctx context.Context, userID string) (int64, error) {
	filter := bson.M{"user_id": userID, "is_read": false}
	update := bson.M{"$set": bson.M{"is_read": true}}

	result, err := r.collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, err
	}

	return result.ModifiedCount, nil
}

func (r *NotificationRepository) Delete(ctx context.Context, id, userID string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	filter := bson.M{"_id": objectID, "user_id": userID}
	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return ErrNotificationNotFound
	}

	return nil
}

func (r *NotificationRepository) GetUnreadCount(ctx context.Context, userID string) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{"user_id": userID, "is_read": false})
}
