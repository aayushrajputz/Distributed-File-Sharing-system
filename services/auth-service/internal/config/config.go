package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	ServicePort      string
	GRPCPort         string
	ServiceHost      string
	MongoURI         string
	MongoDatabase    string
	MongoTimeout     time.Duration
	JWTSecret        string
	JWTExpiry        int64
	JWTRefreshExpiry int64
	Environment      string
	LogLevel         string
}

func Load() *Config {
	jwtExpiry, _ := strconv.ParseInt(getEnv("JWT_EXPIRY", "3600"), 10, 64)
	jwtRefreshExpiry, _ := strconv.ParseInt(getEnv("JWT_REFRESH_EXPIRY", "604800"), 10, 64)

	return &Config{
		ServicePort:      getEnv("AUTH_SERVICE_PORT", "8081"),
		GRPCPort:         getEnv("AUTH_GRPC_PORT", "50051"),
		ServiceHost:      getEnv("AUTH_SERVICE_HOST", "0.0.0.0"),
		MongoURI:         getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDatabase:    getEnv("MONGO_DATABASE", "file_sharing"),
		MongoTimeout:     10 * time.Second,
		JWTSecret:        getEnv("JWT_SECRET", "your-super-secret-key-change-in-production"),
		JWTExpiry:        jwtExpiry,
		JWTRefreshExpiry: jwtRefreshExpiry,
		Environment:      getEnv("ENVIRONMENT", "development"),
		LogLevel:         getEnv("LOG_LEVEL", "info"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
