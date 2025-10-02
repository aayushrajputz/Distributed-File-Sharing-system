package config

import (
	"fmt"
	"log"
	"os"
	"strings"
)

type Config struct {
	Port                    string
	Environment             string
	LogLevel                string
	JWTSecret               string
	AuthServiceGRPC         string
	FileServiceGRPC         string
	NotificationServiceGRPC string
	BillingServiceGRPC      string
	CORSAllowedOrigins      []string
	RateLimitEnabled        bool
	RateLimitRequests       int
	RateLimitDuration       int
}

func Load() *Config {
	cfg := &Config{
		Port:                    getEnv("GATEWAY_PORT", "8080"),
		Environment:             getEnv("ENVIRONMENT", "development"),
		LogLevel:                getEnv("LOG_LEVEL", "info"),
		JWTSecret:               getEnv("JWT_SECRET", "your-super-secret-key-change-in-production"),
		AuthServiceGRPC:         getEnv("AUTH_SERVICE_GRPC", "localhost:50051"),
		FileServiceGRPC:         getEnv("FILE_SERVICE_GRPC", "localhost:50052"),
		NotificationServiceGRPC: getEnv("NOTIFICATION_SERVICE_GRPC", "localhost:50053"),
		BillingServiceGRPC:      getEnv("BILLING_SERVICE_GRPC", "localhost:50054"),
		CORSAllowedOrigins:      getCORSOrigins(),
		RateLimitEnabled:        getEnv("RATE_LIMIT_ENABLED", "true") == "true",
		RateLimitRequests:       getEnvAsInt("RATE_LIMIT_REQUESTS", 100),
		RateLimitDuration:       getEnvAsInt("RATE_LIMIT_DURATION", 60),
	}

	log.Printf("Configuration loaded:")
	log.Printf("  Port: %s", cfg.Port)
	log.Printf("  Environment: %s", cfg.Environment)
	log.Printf("  Auth Service: %s", cfg.AuthServiceGRPC)
	log.Printf("  File Service: %s", cfg.FileServiceGRPC)
	log.Printf("  Notification Service: %s", cfg.NotificationServiceGRPC)
	log.Printf("  Billing Service: %s", cfg.BillingServiceGRPC)
	log.Printf("  CORS Origins: %v", cfg.CORSAllowedOrigins)

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	var value int
	if _, err := fmt.Sscanf(valueStr, "%d", &value); err != nil {
		return defaultValue
	}
	return value
}

func getCORSOrigins() []string {
	origins := getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:8080")
	return strings.Split(origins, ",")
}
