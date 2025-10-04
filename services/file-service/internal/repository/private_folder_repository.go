package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/models"
)

// PrivateFolderRepository handles private folder operations
type PrivateFolderRepository struct {
	collection *mongo.Collection
}

// NewPrivateFolderRepository creates a new private folder repository
func NewPrivateFolderRepository(db *mongo.Database) *PrivateFolderRepository {
	return &PrivateFolderRepository{
		collection: db.Collection("user_pins"),
	}
}

// CreateOrUpdatePIN creates or updates a user's PIN
func (r *PrivateFolderRepository) CreateOrUpdatePIN(ctx context.Context, userID, pinHash, salt string) error {
	filter := bson.M{"user_id": userID}
	update := bson.M{
		"$set": bson.M{
			"user_id":         userID,
			"pin_hash":        pinHash,
			"salt":            salt,
			"pin_length":      models.PINLength,
			"is_active":       true,
			"failed_attempts": 0,
			"locked_until":    nil,
			"updated_at":      time.Now(),
		},
		"$setOnInsert": bson.M{
			"created_at": time.Now(),
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

// GetPIN retrieves a user's PIN information
func (r *PrivateFolderRepository) GetPIN(ctx context.Context, userID string) (*models.UserPIN, error) {
	var pin models.UserPIN
	filter := bson.M{"user_id": userID, "is_active": true}
	err := r.collection.FindOne(ctx, filter).Decode(&pin)
	if err != nil {
		return nil, err
	}
	return &pin, nil
}

// UpdateFailedAttempts updates failed PIN attempts
func (r *PrivateFolderRepository) UpdateFailedAttempts(ctx context.Context, userID string, attempts int, lockedUntil *time.Time) error {
	filter := bson.M{"user_id": userID}
	update := bson.M{
		"$set": bson.M{
			"failed_attempts": attempts,
			"locked_until":    lockedUntil,
			"updated_at":      time.Now(),
		},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

// ResetFailedAttempts resets failed attempts after successful PIN validation
func (r *PrivateFolderRepository) ResetFailedAttempts(ctx context.Context, userID string) error {
	filter := bson.M{"user_id": userID}
	update := bson.M{
		"$set": bson.M{
			"failed_attempts": 0,
			"locked_until":    nil,
			"last_used_at":    time.Now(),
			"updated_at":      time.Now(),
		},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

// LogAccess logs private folder access
func (r *PrivateFolderRepository) LogAccess(ctx context.Context, log *models.PrivateFolderAccessLog) error {
	accessLogsCollection := r.collection.Database().Collection("private_folder_access_logs")
	_, err := accessLogsCollection.InsertOne(ctx, log)
	return err
}

// GetAccessLogs retrieves access logs for a user
func (r *PrivateFolderRepository) GetAccessLogs(ctx context.Context, userID string, limit int64) ([]models.PrivateFolderAccessLog, error) {
	accessLogsCollection := r.collection.Database().Collection("private_folder_access_logs")

	filter := bson.M{"user_id": userID}
	opts := options.Find().
		SetSort(bson.D{{"created_at", -1}}).
		SetLimit(limit)

	cursor, err := accessLogsCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var logs []models.PrivateFolderAccessLog
	if err = cursor.All(ctx, &logs); err != nil {
		return nil, err
	}

	return logs, nil
}

// AddFileToPrivateFolder adds a file to private folder
func (r *PrivateFolderRepository) AddFileToPrivateFolder(ctx context.Context, userID, fileID, originalFolderID string) error {
	privateFilesCollection := r.collection.Database().Collection("private_folder_files")

	fileMapping := models.PrivateFolderFile{
		ID:               primitive.NewObjectID(),
		UserID:           userID,
		FileID:           fileID,
		OriginalFolderID: originalFolderID,
		MovedAt:          time.Now(),
		IsPrivate:        true,
	}

	_, err := privateFilesCollection.InsertOne(ctx, fileMapping)
	return err
}

// RemoveFileFromPrivateFolder removes a file from private folder
func (r *PrivateFolderRepository) RemoveFileFromPrivateFolder(ctx context.Context, userID, fileID string) error {
	privateFilesCollection := r.collection.Database().Collection("private_folder_files")

	filter := bson.M{"user_id": userID, "file_id": fileID}
	_, err := privateFilesCollection.DeleteOne(ctx, filter)
	return err
}

// GetPrivateFiles retrieves all private files for a user
func (r *PrivateFolderRepository) GetPrivateFiles(ctx context.Context, userID string, limit, offset int64) ([]models.PrivateFileInfo, int64, error) {
	privateFilesCollection := r.collection.Database().Collection("private_folder_files")
	filesCollection := r.collection.Database().Collection("files")

	// Get private file IDs
	filter := bson.M{"user_id": userID, "is_private": true}
	opts := options.Find().
		SetSort(bson.D{{"moved_at", -1}}).
		SetSkip(offset).
		SetLimit(limit)

	cursor, err := privateFilesCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var privateFiles []models.PrivateFolderFile
	if err = cursor.All(ctx, &privateFiles); err != nil {
		return nil, 0, err
	}

	// Get file details
	var fileIDs []string
	for _, pf := range privateFiles {
		fileIDs = append(fileIDs, pf.FileID)
	}

	if len(fileIDs) == 0 {
		return []models.PrivateFileInfo{}, 0, nil
	}

	// Get file information
	fileFilter := bson.M{"_id": bson.M{"$in": fileIDs}}
	fileCursor, err := filesCollection.Find(ctx, fileFilter)
	if err != nil {
		return nil, 0, err
	}
	defer fileCursor.Close(ctx)

	var files []models.File
	if err = fileCursor.All(ctx, &files); err != nil {
		return nil, 0, err
	}

	// Create file info map
	fileMap := make(map[string]models.File)
	for _, file := range files {
		fileMap[file.ID.Hex()] = file
	}

	// Build response
	var result []models.PrivateFileInfo
	for _, pf := range privateFiles {
		if file, exists := fileMap[pf.FileID]; exists {
			result = append(result, models.PrivateFileInfo{
				FileID:         file.ID.Hex(),
				FileName:       file.Name,
				FileSize:       file.Size,
				ContentType:    file.MimeType,
				MovedAt:        pf.MovedAt,
				OriginalFolder: pf.OriginalFolderID,
			})
		}
	}

	// Get total count
	total, err := privateFilesCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	return result, total, nil
}

// IsFilePrivate checks if a file is in private folder
func (r *PrivateFolderRepository) IsFilePrivate(ctx context.Context, userID, fileID string) (bool, error) {
	privateFilesCollection := r.collection.Database().Collection("private_folder_files")

	filter := bson.M{"user_id": userID, "file_id": fileID, "is_private": true}
	count, err := privateFilesCollection.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetPINAttempts retrieves PIN attempt information for brute force prevention
func (r *PrivateFolderRepository) GetPINAttempts(ctx context.Context, userID, ipAddress string) (*models.PINAttempt, error) {
	attemptsCollection := r.collection.Database().Collection("pin_attempts")

	var attempt models.PINAttempt
	filter := bson.M{"user_id": userID, "ip_address": ipAddress}
	err := attemptsCollection.FindOne(ctx, filter).Decode(&attempt)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // No previous attempts
		}
		return nil, err
	}

	return &attempt, nil
}

// UpdatePINAttempts updates PIN attempt tracking
func (r *PrivateFolderRepository) UpdatePINAttempts(ctx context.Context, userID, ipAddress string, isBlocked bool, blockedUntil *time.Time) error {
	attemptsCollection := r.collection.Database().Collection("pin_attempts")

	filter := bson.M{"user_id": userID, "ip_address": ipAddress}
	update := bson.M{
		"$inc": bson.M{"attempt_count": 1},
		"$set": bson.M{
			"last_attempt_at": time.Now(),
			"is_blocked":      isBlocked,
			"blocked_until":   blockedUntil,
		},
		"$setOnInsert": bson.M{
			"first_attempt_at": time.Now(),
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := attemptsCollection.UpdateOne(ctx, filter, update, opts)
	return err
}

// ResetPINAttempts resets PIN attempts after successful validation
func (r *PrivateFolderRepository) ResetPINAttempts(ctx context.Context, userID, ipAddress string) error {
	attemptsCollection := r.collection.Database().Collection("pin_attempts")

	filter := bson.M{"user_id": userID, "ip_address": ipAddress}
	_, err := attemptsCollection.DeleteOne(ctx, filter)
	return err
}
