package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/config"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/database"
	grpcHandler "github.com/yourusername/distributed-file-sharing/services/notification-service/internal/grpc"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/kafka"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/repository"
	notificationv1 "github.com/yourusername/distributed-file-sharing/services/notification-service/pkg/pb/notification/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize MongoDB
	mongodb, err := database.NewMongoDB(cfg.MongoURI, cfg.MongoDatabase, 10*time.Second)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongodb.Close(context.Background())

	// Initialize repositories
	notifRepo := repository.NewNotificationRepository(mongodb.Database)

	// Initialize stream broker for real-time notifications
	streamBroker := kafka.NewStreamBroker()

	// Initialize Kafka consumer
	consumer := kafka.NewConsumer(cfg.KafkaBrokers, cfg.KafkaGroupID, cfg.KafkaTopic, notifRepo, streamBroker)

	// Start Kafka consumer in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := consumer.Start(ctx); err != nil {
			log.Printf("Kafka consumer stopped: %v", err)
		}
	}()

	// Initialize gRPC handler
	notificationHandler := grpcHandler.NewNotificationHandler(notifRepo, streamBroker)

	// Start gRPC server
	grpcServer := grpc.NewServer()
	notificationv1.RegisterNotificationServiceServer(grpcServer, notificationHandler)
	reflection.Register(grpcServer)

	grpcListener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", cfg.ServiceHost, cfg.GRPCPort))
	if err != nil {
		log.Fatalf("Failed to listen on gRPC port: %v", err)
	}

	go func() {
		log.Printf("Notification Service gRPC server listening on :%s", cfg.GRPCPort)
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	// Start gRPC Gateway (REST API)
	go func() {
		if err := startGRPCGateway(cfg); err != nil {
			log.Fatalf("Failed to start gRPC Gateway: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Notification Service...")

	// Graceful shutdown
	cancel() // Stop Kafka consumer
	consumer.Close()
	grpcServer.GracefulStop()
	log.Println("Notification Service stopped")
}

func startGRPCGateway(cfg *config.Config) error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Create gRPC-Gateway mux
	mux := runtime.NewServeMux()

	// Setup gRPC connection to local gRPC server
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	grpcEndpoint := fmt.Sprintf("localhost:%s", cfg.GRPCPort)

	// Register Notification Service handler
	err := notificationv1.RegisterNotificationServiceHandlerFromEndpoint(ctx, mux, grpcEndpoint, opts)
	if err != nil {
		return fmt.Errorf("failed to register notification service handler: %w", err)
	}

	// Create Gin router
	router := gin.Default()

	// CORS middleware
	router.Use(corsMiddleware())

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "notification-service",
			"version": "1.0.0",
		})
	})

	// Mount gRPC-Gateway
	router.Any("/api/*path", gin.WrapH(mux))

	// Start HTTP server
	httpAddr := fmt.Sprintf("%s:%s", cfg.ServiceHost, cfg.ServicePort)
	log.Printf("Notification Service REST API (gRPC-Gateway) listening on %s", httpAddr)

	server := &http.Server{
		Addr:         httpAddr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return server.ListenAndServe()
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
