package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
)

type Config struct {
	// Server
	Port     string
	GRPCPort string

	// MongoDB
	MongoURI      string
	MongoDatabase string

	// Stripe
	StripeSecretKey      string
	StripePublishableKey string
	StripeWebhookSecret  string

	// Razorpay (optional)
	RazorpayKeyID     string
	RazorpayKeySecret string

	// Service URLs
	FileServiceGRPC string

	// Environment
	Environment string
	LogLevel    string
}

func Load() *Config {
	cfg := &Config{
		Port:                 getEnv("BILLING_SERVICE_PORT", "8084"),
		GRPCPort:             getEnv("BILLING_GRPC_PORT", "50054"),
		MongoURI:             getEnv("MONGO_URI", "mongodb://mongodb:27017"),
		MongoDatabase:        getEnv("MONGO_DATABASE", "file_sharing"),
		StripeSecretKey:      getEnv("STRIPE_SECRET_KEY", ""),
		StripePublishableKey: getEnv("STRIPE_PUBLISHABLE_KEY", ""),
		StripeWebhookSecret:  getEnv("STRIPE_WEBHOOK_SECRET", ""),
		RazorpayKeyID:        getEnv("RAZORPAY_KEY_ID", ""),
		RazorpayKeySecret:    getEnv("RAZORPAY_KEY_SECRET", ""),
		FileServiceGRPC:      getEnv("FILE_SERVICE_GRPC", "file-service:50052"),
		Environment:          getEnv("ENVIRONMENT", "development"),
		LogLevel:             getEnv("LOG_LEVEL", "info"),
	}

	log.Println("Billing Service Configuration:")
	log.Printf("  Port: %s", cfg.Port)
	log.Printf("  gRPC Port: %s", cfg.GRPCPort)
	log.Printf("  MongoDB URI: %s", cfg.MongoURI)
	log.Printf("  MongoDB Database: %s", cfg.MongoDatabase)
	log.Printf("  Environment: %s", cfg.Environment)
	log.Printf("  Stripe Configured: %v", cfg.StripeSecretKey != "")

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
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseBool(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func (c *Config) Validate() error {
	if c.MongoURI == "" {
		return fmt.Errorf("MONGO_URI is required")
	}
	if c.StripeSecretKey == "" && c.Environment == "production" {
		return fmt.Errorf("STRIPE_SECRET_KEY is required in production")
	}
	return nil
}

