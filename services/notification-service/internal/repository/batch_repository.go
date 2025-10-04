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
	ErrBatchNotificationNotFound = errors.New("batch notification not found")
)

type BatchRepository struct {
	collection *mongo.Collection
}

func NewBatchRepository(database *mongo.Database) *BatchRepository {
	return &BatchRepository{
		collection: database.Collection("batch_notifications"),
	}
}

// Create creates a new batch notification
func (r *BatchRepository) Create(ctx context.Context, batch *models.BatchNotification) error {
	batch.CreatedAt = time.Now()
	batch.UpdatedAt = time.Now()
	
	_, err := r.collection.InsertOne(ctx, batch)
	return err
}

// GetByID gets a batch notification by ID
func (r *BatchRepository) GetByID(ctx context.Context, id string) (*models.BatchNotification, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var batch models.BatchNotification
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&batch)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrBatchNotificationNotFound
		}
		return nil, err
	}

	return &batch, nil
}

// GetByUserID gets batch notifications for a user with pagination
func (r *BatchRepository) GetByUserID(ctx context.Context, userID string, page, limit int, status *models.NotificationStatus) ([]*models.BatchNotification, int64, error) {
	filter := bson.M{"user_id": userID}
	
	if status != nil {
		filter["status"] = *status
	}

	// Get total count
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Calculate skip
	skip := (page - 1) * limit

	// Find batch notifications
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var batches []*models.BatchNotification
	if err = cursor.All(ctx, &batches); err != nil {
		return nil, 0, err
	}

	return batches, total, nil
}

// UpdateStatus updates the status of a batch notification
func (r *BatchRepository) UpdateStatus(ctx context.Context, id string, status models.NotificationStatus) error {
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

	_, err = r.collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	return err
}

// GetPendingBatches gets batch notifications that are pending
func (r *BatchRepository) GetPendingBatches(ctx context.Context, limit int) ([]*models.BatchNotification, error) {
	filter := bson.M{
		"status": models.StatusPending,
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: 1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var batches []*models.BatchNotification
	if err = cursor.All(ctx, &batches); err != nil {
		return nil, err
	}

	return batches, nil
}

// GetBatchesByTimeRange gets batch notifications within a time range
func (r *BatchRepository) GetBatchesByTimeRange(ctx context.Context, startTime, endTime time.Time) ([]*models.BatchNotification, error) {
	filter := bson.M{
		"created_at": bson.M{
			"$gte": startTime,
			"$lte": endTime,
		},
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var batches []*models.BatchNotification
	if err = cursor.All(ctx, &batches); err != nil {
		return nil, err
	}

	return batches, nil
}

// GetBatchStats gets batch notification statistics
func (r *BatchRepository) GetBatchStats(ctx context.Context, userID string, startDate, endDate time.Time) (map[string]int64, error) {
	filter := bson.M{
		"user_id": userID,
		"created_at": bson.M{
			"$gte": startDate,
			"$lte": endDate,
		},
	}

	pipeline := []bson.M{
		{
			"$match": filter,
		},
		{
			"$group": bson.M{
				"_id": "$status",
				"count": bson.M{"$sum": 1},
				"total_items": bson.M{"$sum": "$count"},
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
			ID         string `bson:"_id"`
			Count      int64  `bson:"count"`
			TotalItems int64  `bson:"total_items"`
		}
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		stats[result.ID] = result.Count
		stats[result.ID+"_items"] = result.TotalItems
	}

	return stats, nil
}

// GetBatchStatsByChannel gets batch notification statistics by channel
func (r *BatchRepository) GetBatchStatsByChannel(ctx context.Context, startDate, endDate time.Time) (map[string]int64, error) {
	filter := bson.M{
		"created_at": bson.M{
			"$gte": startDate,
			"$lte": endDate,
		},
	}

	pipeline := []bson.M{
		{
			"$match": filter,
		},
		{
			"$group": bson.M{
				"_id": "$channel",
				"count": bson.M{"$sum": 1},
				"total_items": bson.M{"$sum": "$count"},
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
			ID         string `bson:"_id"`
			Count      int64  `bson:"count"`
			TotalItems int64  `bson:"total_items"`
		}
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		stats[result.ID] = result.Count
		stats[result.ID+"_items"] = result.TotalItems
	}

	return stats, nil
}

// Delete deletes a batch notification
func (r *BatchRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return ErrBatchNotificationNotFound
	}

	return nil
}

// CleanupOldBatches removes old batch notifications
func (r *BatchRepository) CleanupOldBatches(ctx context.Context, olderThan time.Time) (int64, error) {
	filter := bson.M{
		"created_at": bson.M{"$lt": olderThan},
		"status": bson.M{"$in": []models.NotificationStatus{models.StatusSent, models.StatusFailed}},
	}

	result, err := r.collection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, err
	}

	return result.DeletedCount, nil
}

// CreateIndexes creates necessary indexes
func (r *BatchRepository) CreateIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "status", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}, {Key: "created_at", Value: 1}},
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
