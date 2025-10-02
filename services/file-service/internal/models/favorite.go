package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Favorite represents a user's favorite file
type Favorite struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    string             `bson:"user_id" json:"user_id"`
	FileID    string             `bson:"file_id" json:"file_id"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}
