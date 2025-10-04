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
	ErrDLQEntryNotFound = errors.New("DLQ entry not found")
)

type DLQRepository struct {
	collection *mongo.Collection
}

func NewDLQRepository(database *mongo.Database) *DLQRepository {
	return &DLQRepository{
		collection: database.Collection("notification_dlq"),
	}
}

// Create creates a new DLQ entry
func (r *DLQRepository) Create(ctx context.Context, entry *models.DeadLetterQueueEntry) error {
	entry.CreatedAt = time.Now()
	
	_, err := r.collection.InsertOne(ctx, entry)
	return err
}

// GetByID gets a DLQ entry by ID
func (r *DLQRepository) GetByID(ctx context.Context, id string) (*models.DeadLetterQueueEntry, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var entry models.DeadLetterQueueEntry
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&entry)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrDLQEntryNotFound
		}
		return nil, err
	}

	return &entry, nil
}

// GetByNotificationID gets a DLQ entry by notification ID
func (r *DLQRepository) GetByNotificationID(ctx context.Context, notificationID primitive.ObjectID) (*models.DeadLetterQueueEntry, error) {
	var entry models.DeadLetterQueueEntry
	err := r.collection.FindOne(ctx, bson.M{"notification_id": notificationID}).Decode(&entry)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrDLQEntryNotFound
		}
		return nil, err
	}

	return &entry, nil
}

// GetAll gets all DLQ entries with pagination
func (r *DLQRepository) GetAll(ctx context.Context, page, limit int, processed *bool) ([]*models.DeadLetterQueueEntry, int64, error) {
	filter := bson.M{}
	
	if processed != nil {
		filter["is_processed"] = *processed
	}

	// Get total count
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Calculate skip
	skip := (page - 1) * limit

	// Find entries
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var entries []*models.DeadLetterQueueEntry
	if err = cursor.All(ctx, &entries); err != nil {
		return nil, 0, err
	}

	return entries, total, nil
}

// GetReadyForRetry gets DLQ entries that are ready for retry
func (r *DLQRepository) GetReadyForRetry(ctx context.Context, limit int) ([]*models.DeadLetterQueueEntry, error) {
	filter := bson.M{
		"is_processed": false,
		"$or": []bson.M{
			{"next_retry_at": bson.M{"$lte": time.Now()}},
			{"next_retry_at": bson.M{"$exists": false}},
		},
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "next_retry_at", Value: 1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var entries []*models.DeadLetterQueueEntry
	if err = cursor.All(ctx, &entries); err != nil {
		return nil, err
	}

	return entries, nil
}

// UpdateRetryInfo updates retry information for a DLQ entry
func (r *DLQRepository) UpdateRetryInfo(ctx context.Context, id string, retryAttempt *models.RetryAttempt, nextRetryAt *time.Time) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	update := bson.M{
		"$push": bson.M{
			"retry_history": retryAttempt,
		},
		"$set": bson.M{
			"last_retry_at": time.Now(),
		},
	}

	if nextRetryAt != nil {
		update["$set"].(bson.M)["next_retry_at"] = nextRetryAt
	}

	_, err = r.collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	return err
}

// MarkAsProcessed marks a DLQ entry as processed
func (r *DLQRepository) MarkAsProcessed(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"is_processed": true,
			"processed_at": time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return ErrDLQEntryNotFound
	}

	return nil
}

// Delete deletes a DLQ entry
func (r *DLQRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return ErrDLQEntryNotFound
	}

	return nil
}

// GetStats gets DLQ statistics
func (r *DLQRepository) GetStats(ctx context.Context) (map[string]int64, error) {
	pipeline := []bson.M{
		{
			"$group": bson.M{
				"_id": "$is_processed",
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
			ID    bool  `bson:"_id"`
			Count int64 `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		
		key := "processed"
		if !result.ID {
			key = "pending"
		}
		stats[key] = result.Count
	}

	return stats, nil
}

// GetStatsByEventType gets DLQ statistics by event type
func (r *DLQRepository) GetStatsByEventType(ctx context.Context) (map[string]int64, error) {
	pipeline := []bson.M{
		{
			"$group": bson.M{
				"_id": "$event_type",
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

// CleanupOldEntries removes old processed entries
func (r *DLQRepository) CleanupOldEntries(ctx context.Context, olderThan time.Time) (int64, error) {
	filter := bson.M{
		"is_processed": true,
		"processed_at": bson.M{"$lt": olderThan},
	}

	result, err := r.collection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, err
	}

	return result.DeletedCount, nil
}

// GetFailedEntries gets entries that have exceeded max retries
func (r *DLQRepository) GetFailedEntries(ctx context.Context, limit int) ([]*models.DeadLetterQueueEntry, error) {
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"is_processed": false,
			},
		},
		{
			"$addFields": bson.M{
				"retry_count": bson.M{"$size": "$retry_history"},
			},
		},
		{
			"$match": bson.M{
				"retry_count": bson.M{"$gte": "$max_retries"},
			},
		},
		{
			"$sort": bson.M{"created_at": -1},
		},
		{
			"$limit": limit,
		},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var entries []*models.DeadLetterQueueEntry
	if err = cursor.All(ctx, &entries); err != nil {
		return nil, err
	}

	return entries, nil
}

// CreateIndexes creates necessary indexes
func (r *DLQRepository) CreateIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "notification_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "user_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "event_type", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "is_processed", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "next_retry_at", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "created_at", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "processed_at", Value: 1}},
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
}
