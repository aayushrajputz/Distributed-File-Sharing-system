package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type NotificationType string

const (
	NotificationTypeFileUploaded NotificationType = "file.uploaded"
	NotificationTypeFileShared   NotificationType = "file.shared"
	NotificationTypeFileDeleted  NotificationType = "file.deleted"
	NotificationTypeShareRevoked NotificationType = "share.revoked"
	NotificationTypeSystem       NotificationType = "system"
)

type Notification struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    string             `bson:"user_id" json:"user_id"`
	Type      NotificationType   `bson:"type" json:"type"`
	Title     string             `bson:"title" json:"title"`
	Body      string             `bson:"body" json:"body"` // Changed from Message to Body
	Link      string             `bson:"link,omitempty" json:"link,omitempty"`
	IsRead    bool               `bson:"is_read" json:"is_read"` // Changed from Read to IsRead
	Metadata  map[string]string  `bson:"metadata,omitempty" json:"metadata,omitempty"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}
