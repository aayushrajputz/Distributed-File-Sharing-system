package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/cache"
	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/cassandra"
	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/config"
	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/database"
	grpchandler "github.com/yourusername/distributed-file-sharing/services/file-service/internal/grpc"
	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/jwt"
	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/kafka"
	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/logger"
	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/models"
	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/repository"
	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/rest"
	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/service"
	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/storage"
	filev1 "github.com/yourusername/distributed-file-sharing/services/file-service/pkg/pb/file/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	// Initialize logger
	log := logger.NewLogger(cfg.LogLevel)

	// Connect to MongoDB
	mongodb, err := database.NewMongoDB(cfg.MongoURI, cfg.MongoDatabase, cfg.OperationTimeout)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer func() {
		if err := mongodb.Close(context.Background()); err != nil {
			log.Errorf("Error closing MongoDB: %v", err)
		}
	}()
	log.Info("MongoDB connected successfully")

	// Initialize repositories
	fileRepo := repository.NewFileRepository(mongodb.Database)
	storageRepo := repository.NewStorageRepository(mongodb.Database)

	// Ensure MongoDB indexes
	log.Info("Creating MongoDB indexes...")
	if err := fileRepo.EnsureIndexes(context.Background()); err != nil {
		log.Fatalf("Failed to create MongoDB indexes: %v", err)
	}
	log.Info("MongoDB indexes created successfully")

	// Initialize Redis cache
	var redisCache *cache.RedisCache
	if cfg.RedisEnabled {
		redisCache, err = cache.NewRedisCache(
			cfg.RedisAddr,
			cfg.RedisPassword,
			cfg.RedisDB,
			cfg.RedisCacheTTL,
			cfg.RedisMaxRetries,
			cfg.RedisPoolSize,
			cfg.RedisMinIdleConns,
			log,
			true,
		)
		if err != nil {
			log.Fatalf("Failed to connect to Redis: %v", err)
		}
		log.Info("Redis connected successfully")
	} else {
		log.Warn("Redis caching is disabled")
		redisCache, _ = cache.NewRedisCache("", "", 0, 0, 0, 0, 0, log, false)
	}

	// Initialize Cassandra
	var cassandraRepo *cassandra.Repository
	if len(cfg.CassandraHosts) > 0 {
		cassandraConfig := &cassandra.Config{
			Hosts:       cfg.CassandraHosts,
			Port:        cfg.CassandraPort,
			Keyspace:    cfg.CassandraKeyspace,
			Username:    cfg.CassandraUsername,
			Password:    cfg.CassandraPassword,
			Consistency: cfg.CassandraConsistency,
			Timeout:     cfg.CassandraTimeout,
			NumConns:    cfg.CassandraNumConns,
			EnableTLS:   cfg.CassandraEnableTLS,
		}

		cassandraClient, err := cassandra.NewClient(cassandraConfig, log)
		if err != nil {
			log.Fatalf("Failed to connect to Cassandra: %v", err)
		}

		cassandraRepo = cassandra.NewRepository(cassandraClient, log)
		log.Info("Cassandra connected successfully")
	} else {
		log.Warn("Cassandra is disabled")
	}

	// Initialize Kafka producer (simplified for now)
	log.WithFields(logrus.Fields{
		"brokers": cfg.KafkaBrokers,
		"retries": cfg.UploadRetries,
	}).Info("Initializing Kafka producer...")

	producer := kafka.NewProducer(cfg.KafkaBrokers, "file-events", cfg.UploadRetries, log)
	defer producer.Close()
	log.Info("Kafka producer initialized successfully")

	// Kafka consumer is disabled for now
	log.Info("Kafka consumer is disabled for this simplified version")

	// Initialize MinIO storage
	minioStorage, err := storage.NewMinioStorage(cfg.MinioEndpoint, cfg.MinioExternalEndpoint, cfg.MinioAccessKey, cfg.MinioSecretKey, cfg.MinioBucket, cfg.MinioUseSSL)
	if err != nil {
		log.Fatalf("Failed to initialize MinIO storage: %v", err)
	}

	// Initialize private folder repository
	privateFolderRepo := repository.NewPrivateFolderRepository(mongodb.Database)

	// Initialize private folder service
	privateFolderService := service.NewPrivateFolderService(privateFolderRepo, fileRepo, storageRepo)

	// Initialize gRPC handlers
	fileHandler := grpchandler.NewFileHandler(fileRepo, storageRepo, minioStorage, producer, cfg, log, redisCache, nil)

	// Start gRPC server
	grpcServer := grpc.NewServer()
	filev1.RegisterFileServiceServer(grpcServer, fileHandler)

	// Enable reflection for debugging
	reflection.Register(grpcServer)

	// Start gRPC server in goroutine
	go func() {
		lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
		if err != nil {
			log.Fatalf("Failed to listen on gRPC port %s: %v", cfg.GRPCPort, err)
		}
		log.Infof("gRPC server starting on port %s", cfg.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	// Start gRPC Gateway (REST API) in goroutine
	httpServer := &http.Server{}
	go func() {
		if err := startGRPCGateway(cfg, log, redisCache, httpServer, fileHandler, storageRepo, cassandraRepo, fileRepo, minioStorage, privateFolderService); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start gRPC Gateway: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down File Service...")

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown gRPC server
	log.Info("Shutting down gRPC server...")
	grpcServer.GracefulStop()

	// Shutdown HTTP server
	log.Info("Shutting down HTTP server...")
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Errorf("Error shutting down HTTP server: %v", err)
	}

	log.Info("File Service stopped successfully")
}

func startGRPCGateway(cfg *config.Config, log *logrus.Logger, redisCache *cache.RedisCache, httpServer *http.Server, fileHandler interface{}, storageRepo *repository.StorageRepository, cassandraRepo *cassandra.Repository, fileRepo *repository.FileRepository, minioStorage interface{}, privateFolderService *service.PrivateFolderService) error {
	// Create Gin router for REST API
	router := gin.Default()

	// CORS middleware
	router.Use(corsMiddleware())

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "file-service",
			"version": "1.0.0",
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// Storage usage endpoint
	router.GET("/api/v1/files/storage/usage", func(c *gin.Context) {
		// Get user ID from JWT token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		token := authHeader
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		jwtValidator := jwt.NewJWTValidator(cfg.JWTSecret)
		userID, err := jwtValidator.ExtractUserID(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		// Get storage stats by calculating from actual files
		stats, err := storageRepo.CalculateUsageFromFiles(c.Request.Context(), userID, fileRepo)
		if err != nil {
			log.WithError(err).Error("Failed to calculate storage stats")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get storage usage"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"used_bytes":  stats.UsedBytes,
			"quota_bytes": stats.QuotaBytes,
			"file_count":  stats.FileCount,
		})
	})

	// Test endpoint for debugging
	router.GET("/api/v1/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Test endpoint working",
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// Test private folder service
	router.GET("/api/v1/test-private", func(c *gin.Context) {
		// Test if private folder service is initialized
		if privateFolderService == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Private folder service is nil",
			})
			return
		}

		// Test basic functionality
		err := privateFolderService.SetPIN(c.Request.Context(), "test-user", "1234")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Private folder service working",
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// Test PIN validation endpoint
	router.POST("/api/v1/test-validate", func(c *gin.Context) {
		// Log the raw request body
		body, _ := c.GetRawData()
		log.WithField("raw_body", string(body)).Info("Raw request body")

		var req struct {
			UserID string `json:"user_id" binding:"required"`
			PIN    string `json:"pin" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			log.WithError(err).Error("JSON binding error")
			c.JSON(http.StatusBadRequest, gin.H{
				"error":    "JSON binding error: " + err.Error(),
				"raw_body": string(body),
			})
			return
		}

		log.WithFields(logrus.Fields{
			"user_id": req.UserID,
			"pin":     req.PIN,
		}).Info("Parsed request")

		// Test PIN validation
		pinReq := &models.PINValidationRequest{
			UserID:    req.UserID,
			PIN:       req.PIN,
			IPAddress: c.ClientIP(),
			UserAgent: c.GetHeader("User-Agent"),
		}

		resp, err := privateFolderService.ValidatePIN(c.Request.Context(), pinReq)
		if err != nil {
			log.WithError(err).Error("Service error")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Service error: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success":       resp.Success,
			"message":       resp.Message,
			"attempts_left": resp.AttemptsLeft,
			"locked_until":  resp.LockedUntil,
		})
	})

	// Private folder routes
	apiV1 := router.Group("/api/v1")
	privateFolderHandlers := rest.NewPrivateFolderHandlers(privateFolderService, log)
	privateFolderHandlers.RegisterRoutes(apiV1)

	// Start HTTP server
	httpAddr := fmt.Sprintf("%s:%s", cfg.ServiceHost, cfg.ServicePort)
	log.WithField("address", httpAddr).Info("File Service REST API starting...")

	httpServer.Addr = httpAddr
	httpServer.Handler = router
	httpServer.ReadTimeout = 15 * time.Second
	httpServer.WriteTimeout = 15 * time.Second
	httpServer.IdleTimeout = 60 * time.Second

	return httpServer.ListenAndServe()
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
