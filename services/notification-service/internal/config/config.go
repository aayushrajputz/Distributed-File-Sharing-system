package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration for the notification service
type Config struct {
	// Service configuration
	ServicePort     string
	GRPCPort        string
	WebSocketPort   string
	MetricsPort     string
	ServiceHost     string
	Environment     string
	LogLevel        string

	// Database configuration
	MongoURI        string
	MongoDatabase   string
	RedisURI        string
	RedisPassword   string
	RedisDB         int

	// Kafka configuration
	KafkaBrokers    []string
	KafkaGroupID    string
	FileEventsTopic string
	DLQTopic        string

	// SMTP configuration
	SMTPHost        string
	SMTPPort        int
	SMTPUsername    string
	SMTPPassword    string
	SMTPFromEmail   string
	SMTPFromName    string
	SMTPTLS         bool

	// Twilio configuration
	TwilioAccountSID string
	TwilioAuthToken  string
	TwilioPhoneNumber string

	// FCM configuration
	FCMServerKey    string
	FCMProjectID    string

	// WebPush configuration
	WebPushVAPIDPublicKey  string
	WebPushVAPIDPrivateKey string
	WebPushSubject         string

	// Batch configuration
	BatchWindowDuration time.Duration
	BatchMaxSize        int
	BatchFlushInterval  time.Duration

	// Retry configuration
	MaxRetries         int
	RetryBaseDelay     time.Duration
	RetryMaxDelay      time.Duration
	RetryMultiplier    float64

	// DLQ configuration
	DLQMaxRetries      int
	DLQRetryInterval   time.Duration
	DLQCleanupInterval time.Duration

	// Circuit breaker configuration
	CircuitBreakerMaxRequests uint32
	CircuitBreakerInterval    time.Duration
	CircuitBreakerTimeout     time.Duration

	// Health check configuration
	HealthCheckInterval time.Duration
	HealthCheckTimeout  time.Duration

	// WebSocket configuration
	WebSocketReadBufferSize  int
	WebSocketWriteBufferSize int
	WebSocketPingPeriod      time.Duration
	WebSocketPongWait        time.Duration
	WebSocketWriteWait       time.Duration

	// Template configuration
	DefaultTemplatePath string
	TemplateCacheSize   int
	TemplateCacheTTL    time.Duration
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		// Service configuration
		ServicePort:     getEnv("NOTIFICATION_SERVICE_PORT", "8084"),
		GRPCPort:        getEnv("NOTIFICATION_GRPC_PORT", "50054"),
		WebSocketPort:   getEnv("NOTIFICATION_WEBSOCKET_PORT", "8085"),
		MetricsPort:     getEnv("NOTIFICATION_METRICS_PORT", "9094"),
		ServiceHost:     getEnv("NOTIFICATION_SERVICE_HOST", "0.0.0.0"),
		Environment:     getEnv("ENVIRONMENT", "development"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),

		// Database configuration
		MongoURI:        getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDatabase:   getEnv("MONGO_DATABASE", "file_sharing"),
		RedisURI:        getEnv("REDIS_URI", "localhost:6379"),
		RedisPassword:   getEnv("REDIS_PASSWORD", ""),
		RedisDB:         getEnvAsInt("REDIS_DB", 0),

		// Kafka configuration
		KafkaBrokers:    strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ","),
		KafkaGroupID:    getEnv("KAFKA_GROUP_ID", "notification-service"),
		FileEventsTopic: getEnv("KAFKA_FILE_EVENTS_TOPIC", "file-events"),
		DLQTopic:        getEnv("KAFKA_DLQ_TOPIC", "notification-dlq"),

		// SMTP configuration
		SMTPHost:        getEnv("SMTP_HOST", "localhost"),
		SMTPPort:        getEnvAsInt("SMTP_PORT", 587),
		SMTPUsername:    getEnv("SMTP_USERNAME", ""),
		SMTPPassword:    getEnv("SMTP_PASSWORD", ""),
		SMTPFromEmail:   getEnv("SMTP_FROM_EMAIL", "noreply@file-sharing.com"),
		SMTPFromName:    getEnv("SMTP_FROM_NAME", "File Sharing Platform"),
		SMTPTLS:         getEnvAsBool("SMTP_TLS", true),

		// Twilio configuration
		TwilioAccountSID:   getEnv("TWILIO_ACCOUNT_SID", ""),
		TwilioAuthToken:    getEnv("TWILIO_AUTH_TOKEN", ""),
		TwilioPhoneNumber:  getEnv("TWILIO_PHONE_NUMBER", ""),

		// FCM configuration
		FCMServerKey:     getEnv("FCM_SERVER_KEY", ""),
		FCMProjectID:     getEnv("FCM_PROJECT_ID", ""),

		// WebPush configuration
		WebPushVAPIDPublicKey:  getEnv("WEBPUSH_VAPID_PUBLIC_KEY", ""),
		WebPushVAPIDPrivateKey: getEnv("WEBPUSH_VAPID_PRIVATE_KEY", ""),
		WebPushSubject:         getEnv("WEBPUSH_SUBJECT", "mailto:admin@file-sharing.com"),

		// Batch configuration
		BatchWindowDuration: getEnvAsDuration("BATCH_WINDOW_DURATION", "5m"),
		BatchMaxSize:        getEnvAsInt("BATCH_MAX_SIZE", 100),
		BatchFlushInterval:  getEnvAsDuration("BATCH_FLUSH_INTERVAL", "1m"),

		// Retry configuration
		MaxRetries:      getEnvAsInt("MAX_RETRIES", 3),
		RetryBaseDelay:  getEnvAsDuration("RETRY_BASE_DELAY", "1s"),
		RetryMaxDelay:   getEnvAsDuration("RETRY_MAX_DELAY", "5m"),
		RetryMultiplier: getEnvAsFloat("RETRY_MULTIPLIER", 2.0),

		// DLQ configuration
		DLQMaxRetries:      getEnvAsInt("DLQ_MAX_RETRIES", 3),
		DLQRetryInterval:   getEnvAsDuration("DLQ_RETRY_INTERVAL", "1h"),
		DLQCleanupInterval: getEnvAsDuration("DLQ_CLEANUP_INTERVAL", "24h"),

		// Circuit breaker configuration
		CircuitBreakerMaxRequests: uint32(getEnvAsInt("CIRCUIT_BREAKER_MAX_REQUESTS", 10)),
		CircuitBreakerInterval:    getEnvAsDuration("CIRCUIT_BREAKER_INTERVAL", "10s"),
		CircuitBreakerTimeout:     getEnvAsDuration("CIRCUIT_BREAKER_TIMEOUT", "30s"),

		// Health check configuration
		HealthCheckInterval: getEnvAsDuration("HEALTH_CHECK_INTERVAL", "30s"),
		HealthCheckTimeout:  getEnvAsDuration("HEALTH_CHECK_TIMEOUT", "5s"),

		// WebSocket configuration
		WebSocketReadBufferSize:  getEnvAsInt("WEBSOCKET_READ_BUFFER_SIZE", 1024),
		WebSocketWriteBufferSize: getEnvAsInt("WEBSOCKET_WRITE_BUFFER_SIZE", 1024),
		WebSocketPingPeriod:      getEnvAsDuration("WEBSOCKET_PING_PERIOD", "54s"),
		WebSocketPongWait:        getEnvAsDuration("WEBSOCKET_PONG_WAIT", "60s"),
		WebSocketWriteWait:       getEnvAsDuration("WEBSOCKET_WRITE_WAIT", "10s"),

		// Template configuration
		DefaultTemplatePath: getEnv("DEFAULT_TEMPLATE_PATH", "./templates"),
		TemplateCacheSize:   getEnvAsInt("TEMPLATE_CACHE_SIZE", 1000),
		TemplateCacheTTL:    getEnvAsDuration("TEMPLATE_CACHE_TTL", "1h"),
	}
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvAsFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue string) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	duration, _ := time.ParseDuration(defaultValue)
	return duration
}

// IsProduction returns true if the environment is production
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// IsDevelopment returns true if the environment is development
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// GetMongoURI returns the MongoDB URI
func (c *Config) GetMongoURI() string {
	return c.MongoURI
}

// GetRedisURI returns the Redis URI
func (c *Config) GetRedisURI() string {
	return c.RedisURI
}

// GetKafkaBrokers returns the Kafka brokers
func (c *Config) GetKafkaBrokers() []string {
	return c.KafkaBrokers
}

// GetFileEventsTopic returns the file events topic
func (c *Config) GetFileEventsTopic() string {
	return c.FileEventsTopic
}

// GetDLQTopic returns the DLQ topic
func (c *Config) GetDLQTopic() string {
	return c.DLQTopic
}

// GetSMTPConfig returns SMTP configuration
func (c *Config) GetSMTPConfig() (host string, port int, username, password, fromEmail, fromName string, tls bool) {
	return c.SMTPHost, c.SMTPPort, c.SMTPUsername, c.SMTPPassword, c.SMTPFromEmail, c.SMTPFromName, c.SMTPTLS
}

// GetTwilioConfig returns Twilio configuration
func (c *Config) GetTwilioConfig() (accountSID, authToken, phoneNumber string) {
	return c.TwilioAccountSID, c.TwilioAuthToken, c.TwilioPhoneNumber
}

// GetFCMConfig returns FCM configuration
func (c *Config) GetFCMConfig() (serverKey, projectID string) {
	return c.FCMServerKey, c.FCMProjectID
}

// GetWebPushConfig returns WebPush configuration
func (c *Config) GetWebPushConfig() (vapidPublicKey, vapidPrivateKey, subject string) {
	return c.WebPushVAPIDPublicKey, c.WebPushVAPIDPrivateKey, c.WebPushSubject
}

// GetBatchConfig returns batch configuration
func (c *Config) GetBatchConfig() (windowDuration time.Duration, maxSize int, flushInterval time.Duration) {
	return c.BatchWindowDuration, c.BatchMaxSize, c.BatchFlushInterval
}

// GetRetryConfig returns retry configuration
func (c *Config) GetRetryConfig() (maxRetries int, baseDelay, maxDelay time.Duration, multiplier float64) {
	return c.MaxRetries, c.RetryBaseDelay, c.RetryMaxDelay, c.RetryMultiplier
}

// GetDLQConfig returns DLQ configuration
func (c *Config) GetDLQConfig() (maxRetries int, retryInterval, cleanupInterval time.Duration) {
	return c.DLQMaxRetries, c.DLQRetryInterval, c.DLQCleanupInterval
}

// GetCircuitBreakerConfig returns circuit breaker configuration
func (c *Config) GetCircuitBreakerConfig() (maxRequests uint32, interval, timeout time.Duration) {
	return c.CircuitBreakerMaxRequests, c.CircuitBreakerInterval, c.CircuitBreakerTimeout
}

// GetWebSocketConfig returns WebSocket configuration
func (c *Config) GetWebSocketConfig() (readBufferSize, writeBufferSize int, pingPeriod, pongWait, writeWait time.Duration) {
	return c.WebSocketReadBufferSize, c.WebSocketWriteBufferSize, c.WebSocketPingPeriod, c.WebSocketPongWait, c.WebSocketWriteWait
}