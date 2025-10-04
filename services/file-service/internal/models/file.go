package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type FileStatus string

const (
	FileStatusUploading  FileStatus = "uploading"
	FileStatusAvailable  FileStatus = "available"
	FileStatusProcessing FileStatus = "processing"
	FileStatusError      FileStatus = "error"
)

type Permission string

const (
	PermissionRead  Permission = "read"
	PermissionWrite Permission = "write"
	PermissionAdmin Permission = "admin"
)

type File struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string             `bson:"name" json:"name"`
	Description string             `bson:"description,omitempty" json:"description,omitempty"`
	Size        int64              `bson:"size" json:"size"`
	MimeType    string             `bson:"mime_type" json:"mime_type"`
	OwnerID     string             `bson:"owner_id" json:"owner_id"`
	StoragePath string             `bson:"storage_path" json:"storage_path"`
	Checksum    string             `bson:"checksum,omitempty" json:"checksum,omitempty"`
	ContentHash string             `bson:"content_hash,omitempty" json:"content_hash,omitempty"` // For deduplication
	Status      FileStatus         `bson:"status" json:"status"`
	Metadata    map[string]string  `bson:"metadata,omitempty" json:"metadata,omitempty"`
	IsPrivate   bool               `bson:"is_private" json:"is_private"`                       // Privacy flag - true for private files
	SharedWith  []string           `bson:"shared_with,omitempty" json:"shared_with,omitempty"` // User IDs with explicit private access
	DeletedAt   *time.Time         `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`   // Timestamp when file was moved to trash
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

type FileShare struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	FileID          string             `bson:"file_id" json:"file_id"`
	OwnerID         string             `bson:"owner_id" json:"owner_id"`
	SharedWithID    string             `bson:"shared_with_id" json:"shared_with_id"`
	SharedWithEmail string             `bson:"shared_with_email" json:"shared_with_email"`
	Permission      Permission         `bson:"permission" json:"permission"`
	ExpiryTime      *time.Time         `bson:"expiry_time,omitempty" json:"expiry_time,omitempty"`
	ShareLink       string             `bson:"share_link,omitempty" json:"share_link,omitempty"`
	IsActive        bool               `bson:"is_active" json:"is_active"`
	CreatedAt       time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time          `bson:"updated_at" json:"updated_at"`
}
