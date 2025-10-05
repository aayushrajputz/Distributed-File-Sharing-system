package repository

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ErrStorageStatsNotFound = errors.New("storage stats not found")
)

type StorageRepository struct {
	collection *mongo.Collection
}

func NewStorageRepository(db *mongo.Database) *StorageRepository {
	return &StorageRepository{
		collection: db.Collection("storage_stats"),
	}
}

// EnsureIndexes creates necessary database indexes for storage stats
func (r *StorageRepository) EnsureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}},
			Options: options.Index().SetName("user_id_idx").SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "user_id", Value: 1},
				{Key: "updated_at", Value: -1},
			},
			Options: options.Index().SetName("user_updated_idx"),
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// GetOrCreate gets storage stats for a user, creating if not exists
func (r *StorageRepository) GetOrCreate(ctx context.Context, userID string) (*models.StorageStats, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var stats models.StorageStats
	err := r.collection.FindOne(ctx, bson.M{"user_id": userID}).Decode(&stats)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Create default storage stats
			stats = models.StorageStats{
				UserID:     userID,
				UsedBytes:  0,
				QuotaBytes: 100 * 1024 * 1024 * 1024, // 100GB default quota
				FileCount:  0,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}
			stats.ID = primitive.NewObjectID()

			_, err = r.collection.InsertOne(ctx, &stats)
			if err != nil {
				return nil, err
			}
			return &stats, nil
		}
		return nil, err
	}

	return &stats, nil
}

// UpdateUsage updates storage usage for a user
func (r *StorageRepository) UpdateUsage(ctx context.Context, userID string, usedBytes, fileCount int64) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"user_id": userID}
	update := bson.M{
		"$set": bson.M{
			"used_bytes": usedBytes,
			"file_count": fileCount,
			"updated_at": time.Now(),
		},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

// AddUsage adds to storage usage for a user
func (r *StorageRepository) AddUsage(ctx context.Context, userID string, additionalBytes int64) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"user_id": userID}
	update := bson.M{
		"$inc": bson.M{
			"used_bytes": additionalBytes,
			"file_count": 1,
		},
		"$set": bson.M{
			"updated_at": time.Now(),
		},
		"$setOnInsert": bson.M{
			"user_id":     userID,
			"quota_bytes": 100 * 1024 * 1024 * 1024, // 100GB default quota
			"created_at":  time.Now(),
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

// RemoveUsage removes from storage usage for a user
func (r *StorageRepository) RemoveUsage(ctx context.Context, userID string, removedBytes int64) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"user_id": userID}
	update := bson.M{
		"$inc": bson.M{
			"used_bytes": -removedBytes,
			"file_count": -1,
		},
		"$set": bson.M{
			"updated_at": time.Now(),
		},
		"$setOnInsert": bson.M{
			"user_id":     userID,
			"quota_bytes": 100 * 1024 * 1024 * 1024, // 100GB default quota
			"created_at":  time.Now(),
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

// SetQuota sets storage quota for a user
func (r *StorageRepository) SetQuota(ctx context.Context, userID string, quotaBytes int64) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"user_id": userID}
	update := bson.M{
		"$set": bson.M{
			"quota_bytes": quotaBytes,
			"updated_at":  time.Now(),
		},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

// CalculateUsageFromFiles calculates storage usage from actual files in the database
func (r *StorageRepository) CalculateUsageFromFiles(ctx context.Context, userID string, fileRepo *FileRepository) (*models.StorageStats, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Get all files for the user
	files, _, err := fileRepo.FindByOwner(ctx, userID, 1, 10000) // Get up to 10k files
	if err != nil {
		log.Printf("Error finding files for user %s: %v", userID, err)
		return nil, err
	}

	log.Printf("Found %d files for user %s", len(files), userID)

	var totalBytes int64
	var fileCount int64

	for _, file := range files {
		log.Printf("File: %s, Size: %d, Status: %s", file.Name, file.Size, file.Status)
		if file.Status == models.FileStatusAvailable {
			totalBytes += file.Size
			fileCount++
		}
	}

	log.Printf("Calculated storage for user %s: %d bytes, %d files", userID, totalBytes, fileCount)

	// Get or create storage stats
	stats, err := r.GetOrCreate(ctx, userID)
	if err != nil {
		log.Printf("Error getting/creating storage stats for user %s: %v", userID, err)
		return nil, err
	}

	// Update with calculated values
	stats.UsedBytes = totalBytes
	stats.FileCount = fileCount
	stats.UpdatedAt = time.Now()

	// Save updated stats
	filter := bson.M{"user_id": userID}
	update := bson.M{
		"$set": bson.M{
			"used_bytes": stats.UsedBytes,
			"file_count": stats.FileCount,
			"updated_at": stats.UpdatedAt,
		},
	}

	_, err = r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// RecalculateAllUsage recalculates storage usage for all users
func (r *StorageRepository) RecalculateAllUsage(ctx context.Context, fileRepo *FileRepository) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Get all unique user IDs from files
	pipeline := []bson.M{
		{"$group": bson.M{"_id": "$owner_id"}},
	}

	cursor, err := fileRepo.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	var userIDs []string
	for cursor.Next(ctx) {
		var result struct {
			ID string `bson:"_id"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}
		userIDs = append(userIDs, result.ID)
	}

	// Recalculate usage for each user
	for _, userID := range userIDs {
		_, err := r.CalculateUsageFromFiles(ctx, userID, fileRepo)
		if err != nil {
			// Log error but continue with other users
			continue
		}
	}

	return nil
}
