package repository

import (
	"context"
	"errors"
	"fmt"
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

// CheckDownloadPermission checks if a user has permission to download a file
// Returns true if user is the owner OR file is shared with user with any permission level
func (r *FileRepository) CheckDownloadPermission(ctx context.Context, fileID, userID string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Convert fileID to ObjectID
	objectID, err := primitive.ObjectIDFromHex(fileID)
	if err != nil {
		return false, err
	}

	// Check if user is the owner
	var file models.File
	err = r.collection.FindOne(ctx, bson.M{
		"_id":      objectID,
		"owner_id": userID,
	}).Decode(&file)

	if err == nil {
		// User is the owner
		return true, nil
	}

	if err != mongo.ErrNoDocuments {
		// Database error
		return false, err
	}

	// Check if file is shared with user
	var share models.FileShare
	err = r.shareCollection.FindOne(ctx, bson.M{
		"file_id":        fileID,
		"shared_with_id": userID,
		"is_active":      true,
	}).Decode(&share)

	if err == mongo.ErrNoDocuments {
		// File not shared with user
		return false, nil
	}

	if err != nil {
		// Database error
		return false, err
	}

	// Check if share has expired
	if share.ExpiryTime != nil && share.ExpiryTime.Before(time.Now()) {
		return false, nil
	}

	// User has access through sharing
	return true, nil
}

// UpdateFilePrivacy updates the privacy settings of a file
func (r *FileRepository) UpdateFilePrivacy(ctx context.Context, fileID, ownerID string, isPrivate bool, sharedWith []string) error {
	objectID, err := primitive.ObjectIDFromHex(fileID)
	if err != nil {
		return fmt.Errorf("invalid file ID: %w", err)
	}

	// Verify ownership
	var file models.File
	err = r.collection.FindOne(ctx, bson.M{
		"_id":      objectID,
		"owner_id": ownerID,
	}).Decode(&file)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("file not found or you don't have permission")
		}
		return fmt.Errorf("failed to verify ownership: %w", err)
	}

	// Update privacy settings
	update := bson.M{
		"$set": bson.M{
			"is_private":  isPrivate,
			"shared_with": sharedWith,
			"updated_at":  time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	if err != nil {
		return fmt.Errorf("failed to update privacy settings: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("file not found")
	}

	return nil
}

// ManagePrivateAccess adds or removes users from a private file's access list
func (r *FileRepository) ManagePrivateAccess(ctx context.Context, fileID, ownerID string, userIDs []string, action string) error {
	objectID, err := primitive.ObjectIDFromHex(fileID)
	if err != nil {
		return fmt.Errorf("invalid file ID: %w", err)
	}

	// Verify ownership
	var file models.File
	err = r.collection.FindOne(ctx, bson.M{
		"_id":      objectID,
		"owner_id": ownerID,
	}).Decode(&file)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("file not found or you don't have permission")
		}
		return fmt.Errorf("failed to verify ownership: %w", err)
	}

	// Prepare update based on action
	var update bson.M
	if action == "add" {
		update = bson.M{
			"$addToSet": bson.M{
				"shared_with": bson.M{"$each": userIDs},
			},
			"$set": bson.M{
				"updated_at": time.Now(),
			},
		}
	} else if action == "remove" {
		update = bson.M{
			"$pull": bson.M{
				"shared_with": bson.M{"$in": userIDs},
			},
			"$set": bson.M{
				"updated_at": time.Now(),
			},
		}
	} else {
		return fmt.Errorf("invalid action: must be 'add' or 'remove'")
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	if err != nil {
		return fmt.Errorf("failed to manage private access: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("file not found")
	}

	return nil
}

// ListPrivateFiles returns private files that the user has access to (owner or in shared_with list)
func (r *FileRepository) ListPrivateFiles(ctx context.Context, userID string, page, limit int) ([]*models.File, int64, error) {
	skip := (page - 1) * limit

	// Query for private files where user is owner OR user is in shared_with list
	filter := bson.M{
		"is_private": true,
		"deleted_at": bson.M{"$exists": false},
		"$or": []bson.M{
			{"owner_id": userID},
			{"shared_with": userID},
		},
	}

	// Get total count
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count private files: %w", err)
	}

	// Get files with pagination
	cursor, err := r.collection.Find(ctx, filter, &options.FindOptions{
		Skip:  &[]int64{int64(skip)}[0],
		Limit: &[]int64{int64(limit)}[0],
		Sort:  bson.M{"created_at": -1},
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list private files: %w", err)
	}
	defer cursor.Close(ctx)

	var files []*models.File
	if err := cursor.All(ctx, &files); err != nil {
		return nil, 0, fmt.Errorf("failed to decode private files: %w", err)
	}

	return files, total, nil
}

// CheckPublicShareAccess checks if a file has active public shares (link-only shares)
func (r *FileRepository) CheckPublicShareAccess(ctx context.Context, fileID string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Check if there are any active public shares for this file
	// Public shares are those with empty shared_with_email (link-only shares)
	filter := bson.M{
		"file_id":        fileID,
		"shared_with_id": "", // Empty for public shares
		"is_active":      true,
		"$or": []bson.M{
			{"expiry_time": bson.M{"$exists": false}},  // No expiry
			{"expiry_time": bson.M{"$gt": time.Now()}}, // Not expired
		},
	}

	count, err := r.shareCollection.CountDocuments(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("failed to check public share access: %w", err)
	}

	return count > 0, nil
}

// GetPublicShare gets the public share details for a file
func (r *FileRepository) GetPublicShare(ctx context.Context, fileID string) (*models.FileShare, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Find the active public share for this file
	filter := bson.M{
		"file_id":        fileID,
		"shared_with_id": "", // Empty for public shares
		"is_active":      true,
		"$or": []bson.M{
			{"expiry_time": bson.M{"$exists": false}},  // No expiry
			{"expiry_time": bson.M{"$gt": time.Now()}}, // Not expired
		},
	}

	var share models.FileShare
	err := r.shareCollection.FindOne(ctx, filter).Decode(&share)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("no active public share found for file")
		}
		return nil, fmt.Errorf("failed to get public share: %w", err)
	}

	return &share, nil
}
