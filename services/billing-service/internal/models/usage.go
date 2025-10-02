package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Usage represents a user's storage usage
type Usage struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    primitive.ObjectID `bson:"userId" json:"userId"`
	UsedBytes int64              `bson:"usedBytes" json:"usedBytes"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt" json:"updatedAt"`
}

// GetUsedGB returns the used storage in GB
func (u *Usage) GetUsedGB() float64 {
	return float64(u.UsedBytes) / (1024 * 1024 * 1024)
}

// GetPercentUsed calculates the percentage of quota used
func (u *Usage) GetPercentUsed(quotaBytes int64) float64 {
	if quotaBytes == 0 {
		return 0
	}
	return (float64(u.UsedBytes) / float64(quotaBytes)) * 100
}

// CanUpload checks if a file of given size can be uploaded
func (u *Usage) CanUpload(fileSize int64, quotaBytes int64) bool {
	return (u.UsedBytes + fileSize) <= quotaBytes
}

// GetAvailableBytes returns the available storage in bytes
func (u *Usage) GetAvailableBytes(quotaBytes int64) int64 {
	available := quotaBytes - u.UsedBytes
	if available < 0 {
		return 0
	}
	return available
}
