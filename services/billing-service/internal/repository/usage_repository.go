package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/yourusername/distributed-file-sharing-platform/services/billing-service/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UsageRepository struct {
	collection *mongo.Collection
}

func NewUsageRepository(db *mongo.Database) *UsageRepository {
	return &UsageRepository{
		collection: db.Collection("usage"),
	}
}

// Create creates a new usage record
func (r *UsageRepository) Create(ctx context.Context, usage *models.Usage) error {
	usage.CreatedAt = time.Now()
	usage.UpdatedAt = time.Now()

	result, err := r.collection.InsertOne(ctx, usage)
	if err != nil {
		return fmt.Errorf("failed to create usage: %w", err)
	}

	usage.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// FindByUserID finds usage by user ID
func (r *UsageRepository) FindByUserID(ctx context.Context, userID primitive.ObjectID) (*models.Usage, error) {
	var usage models.Usage
	err := r.collection.FindOne(ctx, bson.M{"userId": userID}).Decode(&usage)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // No usage record found
		}
		return nil, fmt.Errorf("failed to find usage: %w", err)
	}

	return &usage, nil
}

// Update updates usage
func (r *UsageRepository) Update(ctx context.Context, usage *models.Usage) error {
	usage.UpdatedAt = time.Now()

	filter := bson.M{"_id": usage.ID}
	update := bson.M{"$set": usage}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update usage: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("usage not found")
	}

	return nil
}

// Upsert creates or updates usage for a user
func (r *UsageRepository) Upsert(ctx context.Context, usage *models.Usage) error {
	usage.UpdatedAt = time.Now()

	filter := bson.M{"userId": usage.UserID}
	update := bson.M{
		"$set": bson.M{
			"usedBytes": usage.UsedBytes,
			"updatedAt": usage.UpdatedAt,
		},
		"$setOnInsert": bson.M{
			"_id":       primitive.NewObjectID(),
			"createdAt": time.Now(),
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

// EnsureIndexes creates necessary indexes
func (r *UsageRepository) EnsureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "userId", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// FindOrCreate finds or creates a usage record for a user
func (r *UsageRepository) FindOrCreate(ctx context.Context, userID primitive.ObjectID) (*models.Usage, error) {
	var usage models.Usage
	err := r.collection.FindOne(ctx, bson.M{"userId": userID}).Decode(&usage)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Create new usage record
			usage = models.Usage{
				UserID:    userID,
				UsedBytes: 0,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			result, err := r.collection.InsertOne(ctx, usage)
			if err != nil {
				return nil, fmt.Errorf("failed to create usage record: %w", err)
			}
			usage.ID = result.InsertedID.(primitive.ObjectID)
			return &usage, nil
		}
		return nil, fmt.Errorf("failed to find usage: %w", err)
	}
	return &usage, nil
}

// IncrementUsage increments the usage by the given amount
func (r *UsageRepository) IncrementUsage(ctx context.Context, userID primitive.ObjectID, bytes int64) error {
	_, err := r.collection.UpdateOne(ctx,
		bson.M{"userId": userID},
		bson.M{
			"$inc": bson.M{"usedBytes": bytes},
			"$set": bson.M{"updatedAt": time.Now()},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to increment usage: %w", err)
	}
	return nil
}

// DecrementUsage decrements the usage by the given amount
func (r *UsageRepository) DecrementUsage(ctx context.Context, userID primitive.ObjectID, bytes int64) error {
	_, err := r.collection.UpdateOne(ctx,
		bson.M{"userId": userID},
		bson.M{
			"$inc": bson.M{"usedBytes": -bytes},
			"$set": bson.M{"updatedAt": time.Now()},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to decrement usage: %w", err)
	}
	return nil
}