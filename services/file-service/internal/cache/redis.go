package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/models"
)

var (
	ErrCacheMiss     = errors.New("cache miss")
	ErrCacheDisabled = errors.New("cache disabled")
)

// Cache keys prefixes
const (
	FileMetadataPrefix = "file:metadata:"
	PresignedURLPrefix = "file:presigned:"
	UserFilesPrefix    = "user:files:"
	SharedFilesPrefix  = "user:shared:"
)

type RedisCache struct {
	client  *redis.Client
	enabled bool
	ttl     time.Duration
	logger  *logrus.Logger
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(addr, password string, db int, ttl time.Duration, maxRetries, poolSize, minIdleConns int, logger *logrus.Logger, enabled bool) (*RedisCache, error) {
	if !enabled {
		logger.Info("Redis cache is disabled")
		return &RedisCache{
			enabled: false,
			logger:  logger,
			ttl:     ttl,
		}, nil
	}

	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		MaxRetries:   maxRetries,
		PoolSize:     poolSize,
		MinIdleConns: minIdleConns,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		logger.WithError(err).Error("Failed to connect to Redis")
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.WithFields(logrus.Fields{
		"addr": addr,
		"db":   db,
		"ttl":  ttl,
	}).Info("Redis cache connected successfully")

	return &RedisCache{
		client:  client,
		enabled: true,
		ttl:     ttl,
		logger:  logger,
	}, nil
}

// IsEnabled returns whether cache is enabled
func (c *RedisCache) IsEnabled() bool {
	return c.enabled
}

// Close closes the Redis connection
func (c *RedisCache) Close() error {
	if !c.enabled || c.client == nil {
		return nil
	}

	c.logger.Info("Closing Redis connection")
	return c.client.Close()
}

// GetFileMetadata retrieves cached file metadata
func (c *RedisCache) GetFileMetadata(ctx context.Context, fileID string) (*models.File, error) {
	if !c.enabled {
		return nil, ErrCacheDisabled
	}

	key := FileMetadataPrefix + fileID

	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			c.logger.WithField("file_id", fileID).Debug("Cache miss for file metadata")
			return nil, ErrCacheMiss
		}
		c.logger.WithError(err).WithField("file_id", fileID).Error("Failed to get file metadata from cache")
		return nil, err
	}

	var file models.File
	if err := json.Unmarshal(data, &file); err != nil {
		c.logger.WithError(err).WithField("file_id", fileID).Error("Failed to unmarshal cached file metadata")
		// Delete corrupted cache entry
		c.client.Del(ctx, key)
		return nil, err
	}

	c.logger.WithField("file_id", fileID).Debug("Cache hit for file metadata")
	return &file, nil
}

// SetFileMetadata caches file metadata
func (c *RedisCache) SetFileMetadata(ctx context.Context, file *models.File) error {
	if !c.enabled {
		return ErrCacheDisabled
	}

	key := FileMetadataPrefix + file.ID.Hex()

	data, err := json.Marshal(file)
	if err != nil {
		c.logger.WithError(err).WithField("file_id", file.ID.Hex()).Error("Failed to marshal file metadata")
		return err
	}

	if err := c.client.Set(ctx, key, data, c.ttl).Err(); err != nil {
		c.logger.WithError(err).WithField("file_id", file.ID.Hex()).Error("Failed to cache file metadata")
		return err
	}

	c.logger.WithField("file_id", file.ID.Hex()).Debug("Cached file metadata")
	return nil
}

// InvalidateFileMetadata removes file metadata from cache
func (c *RedisCache) InvalidateFileMetadata(ctx context.Context, fileID string) error {
	if !c.enabled {
		return ErrCacheDisabled
	}

	key := FileMetadataPrefix + fileID

	if err := c.client.Del(ctx, key).Err(); err != nil {
		c.logger.WithError(err).WithField("file_id", fileID).Error("Failed to invalidate file metadata cache")
		return err
	}

	c.logger.WithField("file_id", fileID).Debug("Invalidated file metadata cache")
	return nil
}

// PresignedURLData stores presigned URL expiry information
type PresignedURLData struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
	FileID    string    `json:"file_id"`
	UserID    string    `json:"user_id"`
}

// SetPresignedURLData caches presigned URL data
func (c *RedisCache) SetPresignedURLData(ctx context.Context, fileID string, data *PresignedURLData) error {
	if !c.enabled {
		return ErrCacheDisabled
	}

	key := PresignedURLPrefix + fileID

	jsonData, err := json.Marshal(data)
	if err != nil {
		c.logger.WithError(err).WithField("file_id", fileID).Error("Failed to marshal presigned URL data")
		return err
	}

	// Calculate TTL based on URL expiry
	ttl := time.Until(data.ExpiresAt)
	if ttl < 0 {
		ttl = 1 * time.Minute // Minimum TTL
	}

	if err := c.client.Set(ctx, key, jsonData, ttl).Err(); err != nil {
		c.logger.WithError(err).WithField("file_id", fileID).Error("Failed to cache presigned URL data")
		return err
	}

	c.logger.WithFields(logrus.Fields{
		"file_id":    fileID,
		"expires_at": data.ExpiresAt,
		"ttl":        ttl,
	}).Debug("Cached presigned URL data")
	return nil
}

// GetPresignedURLData retrieves cached presigned URL data
func (c *RedisCache) GetPresignedURLData(ctx context.Context, fileID string) (*PresignedURLData, error) {
	if !c.enabled {
		return nil, ErrCacheDisabled
	}

	key := PresignedURLPrefix + fileID

	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			c.logger.WithField("file_id", fileID).Debug("Cache miss for presigned URL data")
			return nil, ErrCacheMiss
		}
		c.logger.WithError(err).WithField("file_id", fileID).Error("Failed to get presigned URL data from cache")
		return nil, err
	}

	var urlData PresignedURLData
	if err := json.Unmarshal(data, &urlData); err != nil {
		c.logger.WithError(err).WithField("file_id", fileID).Error("Failed to unmarshal cached presigned URL data")
		c.client.Del(ctx, key)
		return nil, err
	}

	// Check if URL is still valid
	if time.Now().After(urlData.ExpiresAt) {
		c.logger.WithField("file_id", fileID).Debug("Presigned URL expired")
		c.client.Del(ctx, key)
		return nil, ErrCacheMiss
	}

	c.logger.WithField("file_id", fileID).Debug("Cache hit for presigned URL data")
	return &urlData, nil
}

// InvalidatePresignedURLData removes presigned URL data from cache
func (c *RedisCache) InvalidatePresignedURLData(ctx context.Context, fileID string) error {
	if !c.enabled {
		return ErrCacheDisabled
	}

	key := PresignedURLPrefix + fileID

	if err := c.client.Del(ctx, key).Err(); err != nil {
		c.logger.WithError(err).WithField("file_id", fileID).Error("Failed to invalidate presigned URL cache")
		return err
	}

	c.logger.WithField("file_id", fileID).Debug("Invalidated presigned URL cache")
	return nil
}

// InvalidateUserFilesList invalidates user's file list cache
func (c *RedisCache) InvalidateUserFilesList(ctx context.Context, userID string) error {
	if !c.enabled {
		return ErrCacheDisabled
	}

	// Delete all keys matching the pattern
	pattern := UserFilesPrefix + userID + ":*"

	iter := c.client.Scan(ctx, 0, pattern, 100).Iterator()
	deletedCount := 0

	for iter.Next(ctx) {
		if err := c.client.Del(ctx, iter.Val()).Err(); err != nil {
			c.logger.WithError(err).WithField("key", iter.Val()).Error("Failed to delete cache key")
		} else {
			deletedCount++
		}
	}

	if err := iter.Err(); err != nil {
		c.logger.WithError(err).WithField("user_id", userID).Error("Failed to scan user files cache")
		return err
	}

	c.logger.WithFields(logrus.Fields{
		"user_id": userID,
		"count":   deletedCount,
	}).Debug("Invalidated user files list cache")

	return nil
}

// InvalidateSharedFilesList invalidates shared files list cache
func (c *RedisCache) InvalidateSharedFilesList(ctx context.Context, userID string) error {
	if !c.enabled {
		return ErrCacheDisabled
	}

	pattern := SharedFilesPrefix + userID + ":*"

	iter := c.client.Scan(ctx, 0, pattern, 100).Iterator()
	deletedCount := 0

	for iter.Next(ctx) {
		if err := c.client.Del(ctx, iter.Val()).Err(); err != nil {
			c.logger.WithError(err).WithField("key", iter.Val()).Error("Failed to delete cache key")
		} else {
			deletedCount++
		}
	}

	if err := iter.Err(); err != nil {
		c.logger.WithError(err).WithField("user_id", userID).Error("Failed to scan shared files cache")
		return err
	}

	c.logger.WithFields(logrus.Fields{
		"user_id": userID,
		"count":   deletedCount,
	}).Debug("Invalidated shared files list cache")

	return nil
}

// InvalidateAllFileCache invalidates all cache entries related to a file
func (c *RedisCache) InvalidateAllFileCache(ctx context.Context, fileID, ownerID string) error {
	if !c.enabled {
		return ErrCacheDisabled
	}

	// Invalidate file metadata
	if err := c.InvalidateFileMetadata(ctx, fileID); err != nil {
		c.logger.WithError(err).Error("Failed to invalidate file metadata")
	}

	// Invalidate presigned URL data
	if err := c.InvalidatePresignedURLData(ctx, fileID); err != nil {
		c.logger.WithError(err).Error("Failed to invalidate presigned URL data")
	}

	// Invalidate owner's file list
	if err := c.InvalidateUserFilesList(ctx, ownerID); err != nil {
		c.logger.WithError(err).Error("Failed to invalidate user files list")
	}

	c.logger.WithFields(logrus.Fields{
		"file_id":  fileID,
		"owner_id": ownerID,
	}).Info("Invalidated all file-related cache")

	return nil
}

// GetStats returns cache statistics
func (c *RedisCache) GetStats(ctx context.Context) (map[string]interface{}, error) {
	if !c.enabled {
		return map[string]interface{}{
			"enabled": false,
		}, nil
	}

	stats := c.client.PoolStats()

	return map[string]interface{}{
		"enabled":     true,
		"hits":        stats.Hits,
		"misses":      stats.Misses,
		"timeouts":    stats.Timeouts,
		"total_conns": stats.TotalConns,
		"idle_conns":  stats.IdleConns,
		"stale_conns": stats.StaleConns,
	}, nil
}

// HealthCheck checks if Redis is healthy
func (c *RedisCache) HealthCheck(ctx context.Context) error {
	if !c.enabled {
		return nil
	}

	return c.client.Ping(ctx).Err()
}
