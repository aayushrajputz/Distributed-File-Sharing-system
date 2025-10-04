package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

// Service configuration constants
const (
	DefaultMaxFileSize           = 5 * 1024 * 1024 * 1024 // 5GB
	DefaultMinFileSize           = 1                      // 1 byte
	DefaultPresignedURLExpiry    = 15 * time.Minute
	DefaultUploadRetries         = 3
	DefaultPageSize              = 20
	DefaultMaxPageSize           = 100
	DefaultOperationTimeout      = 30 * time.Second
	DefaultQueryTimeout          = 5 * time.Second
	DefaultShutdownTimeout       = 10 * time.Second
	DefaultUploadRatePerMinute   = 10
	DefaultUploadRateBurst       = 10
	DefaultCircuitBreakerMaxReq  = 3
	DefaultCircuitBreakerTimeout = 30 * time.Second
	// Redis defaults
	DefaultRedisCacheTTL     = 5 * time.Minute
	DefaultRedisMaxRetries   = 3
	DefaultRedisPoolSize     = 10
	DefaultRedisMinIdleConns = 5
)

type Config struct {
	ServicePort           string
	GRPCPort              string
	ServiceHost           string
	MongoURI              string
	MongoDatabase         string
	StorageType           string
	MinioEndpoint         string
	MinioExternalEndpoint string
	MinioAccessKey        string
	MinioSecretKey        string
	MinioBucket           string
	MinioUseSSL           bool
	KafkaBrokers          []string
	AuthServiceGRPC       string
	BillingServiceGRPC    string
	JWTSecret             string
	Environment           string
	LogLevel              string
	MaxFileSize           int64
	MinFileSize           int64
	PresignedURLExpiry    time.Duration
	UploadRetries         int
	DefaultPageSize       int32
	MaxPageSize           int32
	OperationTimeout      time.Duration
	QueryTimeout          time.Duration
	ShutdownTimeout       time.Duration
	UploadRatePerMinute   int
	UploadRateBurst       int
	CircuitBreakerMaxReq  uint32
	CircuitBreakerTimeout time.Duration
	AllowedMimeTypes      map[string]bool
	// Redis Configuration
	RedisEnabled      bool
	RedisAddr         string
	RedisPassword     string
	RedisDB           int
	RedisCacheTTL     time.Duration
	RedisMaxRetries   int
	RedisPoolSize     int
	RedisMinIdleConns int
	FrontendURL       string
	// Cassandra Configuration
	CassandraHosts       []string
	CassandraPort        int
	CassandraKeyspace    string
	CassandraUsername    string
	CassandraPassword    string
	CassandraConsistency string
	CassandraTimeout     time.Duration
	CassandraNumConns    int
	CassandraEnableTLS   bool
}

func Load() (*Config, error) {
	// Validate required credentials
	minioAccessKey := getEnv("MINIO_ACCESS_KEY", "")
	minioSecretKey := getEnv("MINIO_SECRET_KEY", "")

	if minioAccessKey == "" || minioSecretKey == "" {
		return nil, errors.New("MINIO_ACCESS_KEY and MINIO_SECRET_KEY are required environment variables")
	}

	mongoURI := getEnv("MONGO_URI", "")
	if mongoURI == "" {
		return nil, errors.New("MONGO_URI is required environment variable")
	}

	kafkaBrokers := getEnv("KAFKA_BROKERS", "")
	if kafkaBrokers == "" {
		return nil, errors.New("KAFKA_BROKERS is required environment variable")
	}

	// Parse optional configuration with defaults
	maxFileSize := getEnvInt64("MAX_FILE_SIZE", DefaultMaxFileSize)
	minFileSize := getEnvInt64("MIN_FILE_SIZE", DefaultMinFileSize)
	presignedURLExpiry := getEnvDuration("PRESIGNED_URL_EXPIRY", DefaultPresignedURLExpiry)
	uploadRetries := getEnvInt("UPLOAD_RETRIES", DefaultUploadRetries)
	operationTimeout := getEnvDuration("OPERATION_TIMEOUT", DefaultOperationTimeout)
	queryTimeout := getEnvDuration("QUERY_TIMEOUT", DefaultQueryTimeout)
	shutdownTimeout := getEnvDuration("SHUTDOWN_TIMEOUT", DefaultShutdownTimeout)

	return &Config{
		ServicePort:           getEnv("FILE_SERVICE_PORT", "8082"),
		GRPCPort:              getEnv("FILE_GRPC_PORT", "50052"),
		ServiceHost:           getEnv("FILE_SERVICE_HOST", "0.0.0.0"),
		MongoURI:              mongoURI,
		MongoDatabase:         getEnv("MONGO_DATABASE", "file_sharing"),
		StorageType:           getEnv("STORAGE_TYPE", "minio"),
		MinioEndpoint:         getEnv("MINIO_ENDPOINT", "minio:9000"),
		MinioExternalEndpoint: getEnv("MINIO_EXTERNAL_ENDPOINT", "localhost:9000"),
		MinioAccessKey:        minioAccessKey,
		MinioSecretKey:        minioSecretKey,
		MinioBucket:           getEnv("MINIO_BUCKET", "file-sharing"),
		MinioUseSSL:           getEnv("MINIO_USE_SSL", "false") == "true",
		KafkaBrokers:          strings.Split(kafkaBrokers, ","),
		AuthServiceGRPC:       getEnv("AUTH_SERVICE_GRPC", "localhost:50051"),
		BillingServiceGRPC:    getEnv("BILLING_SERVICE_GRPC", ""),
		JWTSecret:             getEnv("JWT_SECRET", "your-super-secret-key-change-in-production"),
		Environment:           getEnv("ENVIRONMENT", "development"),
		LogLevel:              getEnv("LOG_LEVEL", "info"),
		MaxFileSize:           maxFileSize,
		MinFileSize:           minFileSize,
		PresignedURLExpiry:    presignedURLExpiry,
		UploadRetries:         uploadRetries,
		DefaultPageSize:       int32(getEnvInt("DEFAULT_PAGE_SIZE", int(DefaultPageSize))),
		MaxPageSize:           int32(getEnvInt("MAX_PAGE_SIZE", int(DefaultMaxPageSize))),
		OperationTimeout:      operationTimeout,
		QueryTimeout:          queryTimeout,
		ShutdownTimeout:       shutdownTimeout,
		UploadRatePerMinute:   getEnvInt("UPLOAD_RATE_PER_MINUTE", DefaultUploadRatePerMinute),
		UploadRateBurst:       getEnvInt("UPLOAD_RATE_BURST", DefaultUploadRateBurst),
		CircuitBreakerMaxReq:  uint32(getEnvInt("CIRCUIT_BREAKER_MAX_REQ", int(DefaultCircuitBreakerMaxReq))),
		CircuitBreakerTimeout: getEnvDuration("CIRCUIT_BREAKER_TIMEOUT", DefaultCircuitBreakerTimeout),
		AllowedMimeTypes:      getAllowedMimeTypes(),
		// Redis Configuration
		RedisEnabled:      getEnv("REDIS_ENABLED", "true") == "true",
		RedisAddr:         getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:     getEnv("REDIS_PASSWORD", ""),
		RedisDB:           getEnvInt("REDIS_DB", 0),
		RedisCacheTTL:     getEnvDuration("REDIS_CACHE_TTL", DefaultRedisCacheTTL),
		RedisMaxRetries:   getEnvInt("REDIS_MAX_RETRIES", DefaultRedisMaxRetries),
		RedisPoolSize:     getEnvInt("REDIS_POOL_SIZE", DefaultRedisPoolSize),
		RedisMinIdleConns: getEnvInt("REDIS_MIN_IDLE_CONNS", DefaultRedisMinIdleConns),
		FrontendURL:       getEnv("FRONTEND_URL", "http://localhost:3000"),
		// Cassandra Configuration
		CassandraHosts:       strings.Split(getEnv("CASSANDRA_HOSTS", "localhost"), ","),
		CassandraPort:        getEnvInt("CASSANDRA_PORT", 9042),
		CassandraKeyspace:    getEnv("CASSANDRA_KEYSPACE", "file_service"),
		CassandraUsername:    getEnv("CASSANDRA_USER", ""),
		CassandraPassword:    getEnv("CASSANDRA_PASSWORD", ""),
		CassandraConsistency: getEnv("CASSANDRA_CONSISTENCY", "LOCAL_QUORUM"),
		CassandraTimeout:     getEnvDuration("CASSANDRA_TIMEOUT", 10*time.Second),
		CassandraNumConns:    getEnvInt("CASSANDRA_NUM_CONNS", 2),
		CassandraEnableTLS:   getEnv("CASSANDRA_TLS_ENABLED", "false") == "true",
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getAllowedMimeTypes() map[string]bool {
	// Default allowed MIME types (whitelist)
	defaults := map[string]bool{
		// Images
		"image/jpeg":    true,
		"image/jpg":     true,
		"image/png":     true,
		"image/gif":     true,
		"image/webp":    true,
		"image/svg+xml": true,
		// Documents
		"application/pdf":    true,
		"application/msword": true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
		"application/vnd.ms-excel": true,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         true,
		"application/vnd.ms-powerpoint":                                             true,
		"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
		// Text
		"text/plain":    true,
		"text/csv":      true,
		"text/html":     true,
		"text/markdown": true,
		// Archives
		"application/zip":             true,
		"application/x-rar":           true,
		"application/x-tar":           true,
		"application/gzip":            true,
		"application/x-7z-compressed": true,
		// Code
		"application/json":       true,
		"application/xml":        true,
		"application/javascript": true,
		// Media
		"video/mp4":       true,
		"video/mpeg":      true,
		"video/quicktime": true,
		"audio/mpeg":      true,
		"audio/wav":       true,
		"audio/ogg":       true,
	}

	// Allow override via environment variable (comma-separated list)
	if customTypes := os.Getenv("ALLOWED_MIME_TYPES"); customTypes != "" {
		result := make(map[string]bool)
		for _, mimeType := range strings.Split(customTypes, ",") {
			result[strings.TrimSpace(mimeType)] = true
		}
		return result
	}

	return defaults
}
