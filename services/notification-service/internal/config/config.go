package config

import (
	"os"
	"strings"
)

type Config struct {
	ServicePort     string
	GRPCPort        string
	ServiceHost     string
	MongoURI        string
	MongoDatabase   string
	KafkaBrokers    []string
	KafkaGroupID    string
	KafkaTopic      string
	AuthServiceGRPC string
	Environment     string
	LogLevel        string
}

func Load() *Config {
	kafkaBrokers := getEnv("KAFKA_BROKERS", "localhost:9092")

	return &Config{
		ServicePort:     getEnv("NOTIFICATION_SERVICE_PORT", "8083"),
		GRPCPort:        getEnv("NOTIFICATION_GRPC_PORT", "50053"),
		ServiceHost:     getEnv("NOTIFICATION_SERVICE_HOST", "0.0.0.0"),
		MongoURI:        getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDatabase:   getEnv("MONGO_DATABASE", "file_sharing"),
		KafkaBrokers:    strings.Split(kafkaBrokers, ","),
		KafkaGroupID:    getEnv("KAFKA_GROUP_ID", "notification-service"),
		KafkaTopic:      getEnv("KAFKA_TOPIC", "file-events"),
		AuthServiceGRPC: getEnv("AUTH_SERVICE_GRPC", "localhost:50051"),
		Environment:     getEnv("ENVIRONMENT", "development"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
