package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StorageStats represents storage usage statistics for a user
type StorageStats struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID     string             `bson:"user_id" json:"user_id"`
	UsedBytes  int64              `bson:"used_bytes" json:"used_bytes"`
	QuotaBytes int64              `bson:"quota_bytes" json:"quota_bytes"`
	FileCount  int64              `bson:"file_count" json:"file_count"`
	CreatedAt  time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt  time.Time          `bson:"updated_at" json:"updated_at"`
}

// GetUsedGB returns used storage in GB
func (s *StorageStats) GetUsedGB() float64 {
	return float64(s.UsedBytes) / (1024 * 1024 * 1024)
}

// GetQuotaGB returns quota storage in GB
func (s *StorageStats) GetQuotaGB() float64 {
	return float64(s.QuotaBytes) / (1024 * 1024 * 1024)
}

// GetUsagePercentage returns usage percentage
func (s *StorageStats) GetUsagePercentage() float64 {
	if s.QuotaBytes == 0 {
		return 0
	}
	return float64(s.UsedBytes) / float64(s.QuotaBytes) * 100
}

// GetAvailableBytes returns available storage in bytes
func (s *StorageStats) GetAvailableBytes() int64 {
	return s.QuotaBytes - s.UsedBytes
}

// GetAvailableGB returns available storage in GB
func (s *StorageStats) GetAvailableGB() float64 {
	return float64(s.GetAvailableBytes()) / (1024 * 1024 * 1024)
}
