package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserPIN represents a user's PIN for private folder access
type UserPIN struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID         string             `bson:"user_id" json:"user_id"`
	PINHash        string             `bson:"pin_hash" json:"-"`
	Salt           string             `bson:"salt" json:"-"`
	CreatedAt      time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt      time.Time          `bson:"updated_at" json:"updated_at"`
	LastUsedAt     *time.Time         `bson:"last_used_at" json:"last_used_at"`
	IsActive       bool               `bson:"is_active" json:"is_active"`
	FailedAttempts int                `bson:"failed_attempts" json:"failed_attempts"`
	LockedUntil    *time.Time         `bson:"locked_until" json:"locked_until"`
	PINLength      int                `bson:"pin_length" json:"pin_length"`
	ExpiresAt      *time.Time         `bson:"expires_at" json:"expires_at"`
}

// PrivateFolderAccessLog represents access logs for private folder
type PrivateFolderAccessLog struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID        string             `bson:"user_id" json:"user_id"`
	FileID        string             `bson:"file_id" json:"file_id"`
	Action        string             `bson:"action" json:"action"` // PIN_VERIFIED, PIN_FAILED, FOLDER_ACCESSED, etc.
	IPAddress     string             `bson:"ip_address" json:"ip_address"`
	UserAgent     string             `bson:"user_agent" json:"user_agent"`
	Success       bool               `bson:"success" json:"success"`
	FailureReason string             `bson:"failure_reason" json:"failure_reason"`
	CreatedAt     time.Time          `bson:"created_at" json:"created_at"`
}

// PrivateFolderFile represents a file in the private folder
type PrivateFolderFile struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID           string             `bson:"user_id" json:"user_id"`
	FileID           string             `bson:"file_id" json:"file_id"`
	OriginalFolderID string             `bson:"original_folder_id" json:"original_folder_id"`
	MovedAt          time.Time          `bson:"moved_at" json:"moved_at"`
	IsPrivate        bool               `bson:"is_private" json:"is_private"`
}

// PINAttempt represents PIN attempt tracking for brute force prevention
type PINAttempt struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID         string             `bson:"user_id" json:"user_id"`
	IPAddress      string             `bson:"ip_address" json:"ip_address"`
	AttemptCount   int                `bson:"attempt_count" json:"attempt_count"`
	FirstAttemptAt time.Time          `bson:"first_attempt_at" json:"first_attempt_at"`
	LastAttemptAt  time.Time          `bson:"last_attempt_at" json:"last_attempt_at"`
	IsBlocked      bool               `bson:"is_blocked" json:"is_blocked"`
	BlockedUntil   *time.Time         `bson:"blocked_until" json:"blocked_until"`
}

// PINValidationRequest represents the request to validate a PIN
type PINValidationRequest struct {
	UserID    string `json:"user_id" validate:"required"`
	PIN       string `json:"pin" validate:"required,min=4,max=8"`
	IPAddress string `json:"ip_address"`
	UserAgent string `json:"user_agent"`
}

// PINValidationResponse represents the response for PIN validation
type PINValidationResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	AttemptsLeft int    `json:"attempts_left,omitempty"`
	LockedUntil  string `json:"locked_until,omitempty"`
}

// MakePrivateRequest represents the request to make a file private
type MakePrivateRequest struct {
	UserID string `json:"user_id" validate:"required"`
	FileID string `json:"file_id" validate:"required"`
	PIN    string `json:"pin" validate:"required"`
}

// MakePrivateResponse represents the response for making a file private
type MakePrivateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	FileID  string `json:"file_id,omitempty"`
}

// PrivateFolderListResponse represents the list of private files
type PrivateFolderListResponse struct {
	Files []PrivateFileInfo `json:"files"`
	Total int64             `json:"total"`
}

// PrivateFileInfo represents file information in private folder
type PrivateFileInfo struct {
	FileID         string    `json:"file_id"`
	FileName       string    `json:"file_name"`
	FileSize       int64     `json:"file_size"`
	ContentType    string    `json:"content_type"`
	MovedAt        time.Time `json:"moved_at"`
	OriginalFolder string    `json:"original_folder,omitempty"`
}

// Constants for PIN validation
const (
	MaxPINAttempts     = 5
	PINLockoutDuration = 15 * time.Minute
	PINLength          = 4
	MaxPINLength       = 8
)

// PIN Actions
const (
	ActionPINVerified          = "PIN_VERIFIED"
	ActionPINFailed            = "PIN_FAILED"
	ActionFolderAccessed       = "FOLDER_ACCESSED"
	ActionFileMovedToPrivate   = "FILE_MOVED_TO_PRIVATE"
	ActionFileMovedFromPrivate = "FILE_MOVED_FROM_PRIVATE"
)

