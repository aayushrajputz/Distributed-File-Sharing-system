package repository

import (
	"context"
	"errors"
	"time"

	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ErrFileNotFound  = errors.New("file not found")
	ErrShareNotFound = errors.New("file share not found")
)

type FileRepository struct {
	collection         *mongo.Collection
	shareCollection    *mongo.Collection
	favoriteCollection *mongo.Collection
}

func NewFileRepository(db *mongo.Database) *FileRepository {
	return &FileRepository{
		collection:         db.Collection("files"),
		shareCollection:    db.Collection("file_shares"),
		favoriteCollection: db.Collection("favorites"),
	}
}

// EnsureIndexes creates necessary database indexes for performance
func (r *FileRepository) EnsureIndexes(ctx context.Context) error {
	// Files collection indexes
	fileIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "owner_id", Value: 1},
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().SetName("owner_created_idx"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("status_idx"),
		},
		{
			Keys: bson.D{
				{Key: "owner_id", Value: 1},
				{Key: "status", Value: 1},
			},
			Options: options.Index().SetName("owner_status_idx"),
		},
		{
			Keys: bson.D{
				{Key: "owner_id", Value: 1},
				{Key: "content_hash", Value: 1},
			},
			Options: options.Index().SetName("owner_hash_idx").SetSparse(true),
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, fileIndexes)
	if err != nil {
		return err
	}

	// File shares indexes
	shareIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "file_id", Value: 1}},
			Options: options.Index().SetName("file_id_idx"),
		},
		{
			Keys:    bson.D{{Key: "shared_with_id", Value: 1}},
			Options: options.Index().SetName("shared_with_id_idx"),
		},
		{
			Keys:    bson.D{{Key: "shared_with_email", Value: 1}},
			Options: options.Index().SetName("shared_with_email_idx"),
		},
		{
			Keys: bson.D{
				{Key: "file_id", Value: 1},
				{Key: "shared_with_id", Value: 1},
			},
			Options: options.Index().SetName("file_user_share_idx").SetUnique(true),
		},
	}

	_, err = r.shareCollection.Indexes().CreateMany(ctx, shareIndexes)
	if err != nil {
		return err
	}

	// Favorites collection indexes
	favoriteIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}},
			Options: options.Index().SetName("user_id_idx"),
		},
		{
			Keys:    bson.D{{Key: "file_id", Value: 1}},
			Options: options.Index().SetName("file_id_idx"),
		},
		{
			Keys: bson.D{
				{Key: "user_id", Value: 1},
				{Key: "file_id", Value: 1},
			},
			Options: options.Index().SetName("user_file_idx").SetUnique(true),
		},
	}

	_, err = r.favoriteCollection.Indexes().CreateMany(ctx, favoriteIndexes)
	return err
}

func (r *FileRepository) Create(ctx context.Context, file *models.File) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	file.ID = primitive.NewObjectID()
	file.CreatedAt = time.Now()
	file.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, file)
	return err
}

func (r *FileRepository) FindByID(ctx context.Context, id string) (*models.File, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var file models.File
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&file)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrFileNotFound
		}
		return nil, err
	}
	return &file, nil
}

func (r *FileRepository) FindByOwner(ctx context.Context, ownerID string, page, limit int32) ([]*models.File, int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	skip := (page - 1) * limit

	cursor, err := r.collection.Find(
		ctx,
		bson.M{"owner_id": ownerID},
		options.Find().SetSkip(int64(skip)).SetLimit(int64(limit)).SetSort(bson.M{"created_at": -1}),
	)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var files []*models.File
	if err = cursor.All(ctx, &files); err != nil {
		return nil, 0, err
	}

	total, err := r.collection.CountDocuments(ctx, bson.M{"owner_id": ownerID})
	if err != nil {
		return nil, 0, err
	}

	return files, total, nil
}

func (r *FileRepository) Update(ctx context.Context, file *models.File) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	file.UpdatedAt = time.Now()

	filter := bson.M{"_id": file.ID}
	update := bson.M{
		"$set": bson.M{
			"name":         file.Name,
			"description":  file.Description,
			"checksum":     file.Checksum,
			"content_hash": file.ContentHash,
			"status":       file.Status,
			"metadata":     file.Metadata,
			"updated_at":   file.UpdatedAt,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return ErrFileNotFound
	}

	return nil
}

// Delete method removed - files are now permanently deleted directly
// Use PermanentDeleteDirect instead

func (r *FileRepository) FindByContentHash(ctx context.Context, ownerID, hash string) (*models.File, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var file models.File
	err := r.collection.FindOne(ctx, bson.M{
		"owner_id":     ownerID,
		"content_hash": hash,
	}).Decode(&file)

	if err == mongo.ErrNoDocuments {
		return nil, nil // Not a duplicate
	}

	if err != nil {
		return nil, err
	}

	return &file, nil
}

func (r *FileRepository) CreateShare(ctx context.Context, share *models.FileShare) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	share.ID = primitive.NewObjectID()
	share.CreatedAt = time.Now()

	_, err := r.shareCollection.InsertOne(ctx, share)
	return err
}

func (r *FileRepository) FindSharesByFileID(ctx context.Context, fileID string) ([]*models.FileShare, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cursor, err := r.shareCollection.Find(ctx, bson.M{"file_id": fileID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var shares []*models.FileShare
	if err = cursor.All(ctx, &shares); err != nil {
		return nil, err
	}

	return shares, nil
}

// FindSharedWithUser uses aggregation pipeline for efficient query
func (r *FileRepository) FindSharedWithUser(ctx context.Context, userID string, page, limit int32) ([]*models.File, int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	skip := (page - 1) * limit

	// Use aggregation pipeline for efficiency
	pipeline := mongo.Pipeline{
		// Match shares for user
		{{Key: "$match", Value: bson.M{"shared_with_id": userID}}},

		// Convert file_id string to ObjectID
		{{Key: "$addFields", Value: bson.M{
			"file_oid": bson.M{"$toObjectId": "$file_id"},
		}}},

		// Join with files collection
		{{Key: "$lookup", Value: bson.M{
			"from":         "files",
			"localField":   "file_oid",
			"foreignField": "_id",
			"as":           "file",
		}}},

		// Unwind file array
		{{Key: "$unwind", Value: "$file"}},

		// Sort by file creation date
		{{Key: "$sort", Value: bson.M{"file.created_at": -1}}},

		// Pagination
		{{Key: "$skip", Value: skip}},
		{{Key: "$limit", Value: limit}},

		// Project only file
		{{Key: "$replaceRoot", Value: bson.M{"newRoot": "$file"}}},
	}

	cursor, err := r.shareCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var files []*models.File
	if err = cursor.All(ctx, &files); err != nil {
		return nil, 0, err
	}

	// Count total shared files
	countPipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"shared_with_id": userID}}},
		{{Key: "$count", Value: "total"}},
	}

	countCursor, err := r.shareCollection.Aggregate(ctx, countPipeline)
	if err != nil {
		return files, 0, nil // Return files even if count fails
	}
	defer countCursor.Close(ctx)

	var countResult []bson.M
	if err = countCursor.All(ctx, &countResult); err != nil {
		return files, 0, nil
	}

	total := int64(0)
	if len(countResult) > 0 {
		if totalVal, ok := countResult[0]["total"].(int32); ok {
			total = int64(totalVal)
		} else if totalVal, ok := countResult[0]["total"].(int64); ok {
			total = totalVal
		}
	}

	return files, total, nil
}

func (r *FileRepository) DeleteShare(ctx context.Context, shareID string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(shareID)
	if err != nil {
		return err
	}

	result, err := r.shareCollection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return ErrShareNotFound
	}

	return nil
}

// CheckShareAccess checks if a user has access to a file via sharing
func (r *FileRepository) CheckShareAccess(ctx context.Context, fileID, userID string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	count, err := r.shareCollection.CountDocuments(ctx, bson.M{
		"file_id":        fileID,
		"shared_with_id": userID,
		"is_active":      true,
		"$or": []bson.M{
			{"expiry_time": nil},
			{"expiry_time": bson.M{"$gt": time.Now()}},
		},
	})

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// CheckShareAccessWithPermission checks if a user has access to a file with specific permission
func (r *FileRepository) CheckShareAccessWithPermission(ctx context.Context, fileID, userID string) (bool, models.Permission, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var share models.FileShare
	err := r.shareCollection.FindOne(ctx, bson.M{
		"file_id":        fileID,
		"shared_with_id": userID,
		"is_active":      true,
		"$or": []bson.M{
			{"expiry_time": nil},
			{"expiry_time": bson.M{"$gt": time.Now()}},
		},
	}).Decode(&share)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, "", nil
		}
		return false, "", err
	}

	return true, share.Permission, nil
}

// GetActiveShare gets the active share for a user and file
func (r *FileRepository) GetActiveShare(ctx context.Context, fileID, userID string) (*models.FileShare, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var share models.FileShare
	err := r.shareCollection.FindOne(ctx, bson.M{
		"file_id":        fileID,
		"shared_with_id": userID,
		"is_active":      true,
		"$or": []bson.M{
			{"expiry_time": nil},
			{"expiry_time": bson.M{"$gt": time.Now()}},
		},
	}).Decode(&share)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrShareNotFound
		}
		return nil, err
	}

	return &share, nil
}

// PermanentDelete permanently deletes a file from database (only files in trash)
func (r *FileRepository) PermanentDelete(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	result, err := r.collection.DeleteOne(ctx, bson.M{
		"_id": objectID,
	})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return ErrFileNotFound
	}

	return nil
}

// PermanentDeleteDirect permanently deletes a file directly from database (any status)
func (r *FileRepository) PermanentDeleteDirect(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	result, err := r.collection.DeleteOne(ctx, bson.M{
		"_id": objectID,
	})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return ErrFileNotFound
	}

	return nil
}

// IsErrFileNotFound checks if an error is ErrFileNotFound
func IsErrFileNotFound(err error) bool {
	return err == ErrFileNotFound
}

// AddToFavorites adds a file to user's favorites
func (r *FileRepository) AddToFavorites(ctx context.Context, userID, fileID string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	favorite := models.Favorite{
		ID:        primitive.NewObjectID(),
		UserID:    userID,
		FileID:    fileID,
		CreatedAt: time.Now(),
	}

	_, err := r.favoriteCollection.InsertOne(ctx, favorite)
	if err != nil {
		// Check if it's a duplicate key error (already favorited)
		if mongo.IsDuplicateKeyError(err) {
			return nil // Already favorited, not an error
		}
		return err
	}

	return nil
}

// RemoveFromFavorites removes a file from user's favorites
func (r *FileRepository) RemoveFromFavorites(ctx context.Context, userID, fileID string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := r.favoriteCollection.DeleteOne(ctx, bson.M{
		"user_id": userID,
		"file_id": fileID,
	})
	return err
}

// IsFavorite checks if a file is in user's favorites
func (r *FileRepository) IsFavorite(ctx context.Context, userID, fileID string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	count, err := r.favoriteCollection.CountDocuments(ctx, bson.M{
		"user_id": userID,
		"file_id": fileID,
	})
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// FindFavoritesByUser returns user's favorite files with pagination
func (r *FileRepository) FindFavoritesByUser(ctx context.Context, userID string, page, limit int32) ([]*models.File, int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	skip := int64((page - 1) * limit)

	// Aggregation pipeline to join favorites with files
	pipeline := []bson.M{
		// Match favorites for the user
		{"$match": bson.M{"user_id": userID}},
		
		// Convert file_id string to ObjectID for lookup
		{"$addFields": bson.M{
			"file_object_id": bson.M{"$toObjectId": "$file_id"},
		}},
		
		// Lookup file details
		{"$lookup": bson.M{
			"from":         "files",
			"localField":   "file_object_id",
			"foreignField": "_id",
			"as":           "file",
		}},
		
		// Unwind file array
		{"$unwind": "$file"},
		
		// Sort by favorite creation date (most recent first)
		{"$sort": bson.M{"created_at": -1}},
		
		// Pagination
		{"$skip": skip},
		{"$limit": int64(limit)},
		
		// Project only the file data
		{"$replaceRoot": bson.M{"newRoot": "$file"}},
	}

	cursor, err := r.favoriteCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var files []*models.File
	if err = cursor.All(ctx, &files); err != nil {
		return nil, 0, err
	}

	// Count total favorites
	total, err := r.favoriteCollection.CountDocuments(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, 0, err
	}

	return files, total, nil
}
