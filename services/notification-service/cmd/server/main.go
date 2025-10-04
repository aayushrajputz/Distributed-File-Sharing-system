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
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/config"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/database"
	grpchandler "github.com/yourusername/distributed-file-sharing/services/notification-service/internal/grpc"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/handlers"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/kafka"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/metrics"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/models"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/repository"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/rest"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/services"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/websocket"
	notificationv1 "github.com/yourusername/distributed-file-sharing/services/notification-service/pkg/pb/notification/v1"
	"google.golang.org/grpc"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	if cfg.IsDevelopment() {
		logger.SetLevel(logrus.DebugLevel)
	}

	// Initialize metrics
	metricsInstance := metrics.NewMetrics()

	// Initialize MongoDB
	mongodb, err := database.NewMongoDB(cfg.GetMongoURI(), cfg.MongoDatabase, 10*time.Second)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongodb.Close(context.Background())

	// Initialize Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.GetRedisURI(),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	defer redisClient.Close()

	// Test Redis connection
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Initialize repositories
	notifRepo := repository.NewNotificationRepository(mongodb.Database)
	preferencesRepo := repository.NewPreferencesRepository(mongodb.Database)
	templateRepo := repository.NewTemplateRepository(mongodb.Database)
	batchRepo := repository.NewBatchRepository(mongodb.Database)
	dlqRepo := repository.NewDLQRepository(mongodb.Database)

	// Create indexes
	createIndexes(context.Background(), notifRepo, preferencesRepo, templateRepo, batchRepo, dlqRepo)

	// Initialize services
	preferenceSvc := services.NewPreferenceService(preferencesRepo, logger)
	templateSvc := services.NewTemplateService(templateRepo, logger)

	// Initialize batch service
	batchConfig := &services.BatchConfig{
		WindowDuration: cfg.BatchWindowDuration,
		MaxSize:        cfg.BatchMaxSize,
		FlushInterval:  cfg.BatchFlushInterval,
		RedisKeyPrefix: "notification_batch:",
	}
	batchSvc := services.NewBatchService(redisClient, batchRepo, notifRepo, preferenceSvc, templateSvc, batchConfig, logger)

	// Initialize DLQ service
	dlqConfig := &services.DLQConfig{
		MaxRetries:      cfg.DLQMaxRetries,
		RetryInterval:   cfg.DLQRetryInterval,
		CleanupInterval: cfg.DLQCleanupInterval,
		BatchSize:       100,
	}
	dlqSvc := services.NewDLQService(dlqRepo, notifRepo, preferenceSvc, templateSvc, dlqConfig, logger)

	// Initialize retry service
	retryConfig := &services.RetryConfig{
		MaxRetries:    cfg.MaxRetries,
		BaseDelay:     cfg.RetryBaseDelay,
		MaxDelay:      cfg.RetryMaxDelay,
		Multiplier:    cfg.RetryMultiplier,
		Jitter:        true,
		RetryInterval: cfg.RetryBaseDelay,
		BatchSize:     100,
	}
	retrySvc := services.NewRetryService(notifRepo, dlqSvc, retryConfig, logger)

	// Initialize notification service
	serviceConfig := &services.ServiceConfig{
		EnableBatching:   true,
		EnableRetry:      true,
		EnableDLQ:        true,
		DefaultChannel:   models.ChannelInApp,
		FallbackChannels: []models.NotificationChannel{models.ChannelEmail, models.ChannelSMS},
	}
	notifSvc := services.NewNotificationService(notifRepo, preferenceSvc, templateSvc, batchSvc, dlqSvc, retrySvc, serviceConfig, logger)

	// Initialize handlers
	emailHandler := handlers.NewEmailHandler(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPFromEmail, cfg.SMTPFromName, cfg.SMTPTLS, logger)
	smsHandler := handlers.NewMockSMSHandler(true, logger)   // Use mock for testing
	pushHandler := handlers.NewMockPushHandler(true, logger) // Use mock for testing
	inAppHandler := handlers.NewInAppHandler(true, logger)
	wsHandler := handlers.NewWebSocketHandler(true, logger)

	// Register handlers
	notifSvc.RegisterHandler(models.ChannelEmail, emailHandler)
	notifSvc.RegisterHandler(models.ChannelSMS, smsHandler)
	notifSvc.RegisterHandler(models.ChannelPush, pushHandler)
	notifSvc.RegisterHandler(models.ChannelInApp, inAppHandler)
	notifSvc.RegisterHandler(models.ChannelWebSocket, wsHandler)

	// Initialize WebSocket server
	wsServer := websocket.NewServer(wsHandler, logger)

	// Initialize StreamBroker for Kafka
	streamBroker := kafka.NewStreamBroker()

	// Initialize REST handlers
	restHandlers := rest.NewRestHandlers(notifSvc, preferenceSvc, templateSvc, batchSvc, dlqSvc, logger)

	// Initialize Kafka consumer
	consumer := kafka.NewConsumer(cfg.GetKafkaBrokers(), cfg.KafkaGroupID, cfg.FileEventsTopic, notifRepo, streamBroker, notifSvc)

	// Start background processes
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start Kafka consumer
	go func() {
		if err := consumer.Start(ctx); err != nil {
			logger.WithError(err).Error("Kafka consumer stopped")
		}
	}()

	// Start notification service background processes
	notifSvc.StartBackgroundProcesses(ctx)

	// Start WebSocket cleanup routine
	go wsServer.StartCleanupRoutine(ctx)

	// Create default templates
	if err := templateSvc.CreateDefaultTemplates(ctx); err != nil {
		logger.WithError(err).Warn("Failed to create default templates")
	}

	// Start servers
	go startRESTServer(cfg, restHandlers, logger)
	go startWebSocketServer(cfg, wsServer, logger)
	go startMetricsServer(cfg, metricsInstance, logger)
	go startGRPCServer(cfg, notifSvc, logger)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down servers...")
	cancel()

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Close WebSocket connections with timeout
	done := make(chan struct{})
	go func() {
		wsServer.CloseAllConnections()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("WebSocket connections closed")
	case <-shutdownCtx.Done():
		logger.Warn("WebSocket close timeout")
	}

	logger.Info("Servers stopped")
}

// createIndexes creates necessary database indexes
func createIndexes(ctx context.Context, repos ...interface{}) {
	// This would create indexes for all repositories
	// Implementation depends on the specific repository interface
}

// startRESTServer starts the REST API server
func startRESTServer(cfg *config.Config, handlers *rest.RestHandlers, logger *logrus.Logger) {
	// Set Gin mode
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create Gin router
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User-ID")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Setup routes
	handlers.SetupRoutes(router)

	// Start server
	addr := fmt.Sprintf("%s:%s", cfg.ServiceHost, cfg.ServicePort)
	logger.WithField("address", addr).Info("Starting REST server")

	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.WithError(err).Fatal("Failed to start REST server")
	}
}

// startWebSocketServer starts the WebSocket server
func startWebSocketServer(cfg *config.Config, wsServer *websocket.Server, logger *logrus.Logger) {
	// Create Gin router for WebSocket
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User-ID")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// WebSocket endpoint
	router.GET("/ws", wsServer.HandleWebSocket)

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":      "healthy",
			"service":     "websocket",
			"connections": wsServer.GetConnectionCount(),
		})
	})

	// Start server
	addr := fmt.Sprintf("%s:%s", cfg.ServiceHost, cfg.WebSocketPort)
	logger.WithField("address", addr).Info("Starting WebSocket server")

	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.WithError(err).Fatal("Failed to start WebSocket server")
	}
}

// startMetricsServer starts the metrics server
func startMetricsServer(cfg *config.Config, metricsInstance *metrics.Metrics, logger *logrus.Logger) {
	// Create metrics server
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Metrics endpoint
	router.GET("/metrics", func(c *gin.Context) {
		// This would expose Prometheus metrics
		c.String(http.StatusOK, "# Metrics endpoint\n# Implementation would go here\n")
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "metrics",
		})
	})

	// Start server
	addr := fmt.Sprintf("%s:%s", cfg.ServiceHost, cfg.MetricsPort)
	logger.WithField("address", addr).Info("Starting metrics server")

	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.WithError(err).Fatal("Failed to start metrics server")
	}
}

// startGRPCServer starts the gRPC server
func startGRPCServer(cfg *config.Config, notifSvc *services.NotificationService, logger *logrus.Logger) {
	// Create gRPC server
	grpcServer := grpchandler.NewNotificationGRPCServer(notifSvc, logger)

	// Create listener
	addr := fmt.Sprintf("%s:%s", cfg.ServiceHost, cfg.GRPCPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		logger.WithError(err).Fatal("Failed to listen on gRPC port")
	}

	// Create gRPC server
	s := grpc.NewServer()
	notificationv1.RegisterNotificationServiceServer(s, grpcServer)

	logger.WithField("address", addr).Info("Starting gRPC server")
	if err := s.Serve(lis); err != nil {
		logger.WithError(err).Fatal("Failed to start gRPC server")
	}
}
