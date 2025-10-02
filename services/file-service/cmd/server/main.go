package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/sirupsen/logrus"
	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/cache"
	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/config"
	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/database"
	grpcHandler "github.com/yourusername/distributed-file-sharing/services/file-service/internal/grpc"
	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/jwt"
	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/kafka"
	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/logger"
	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/repository"
	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/storage"
	filev1 "github.com/yourusername/distributed-file-sharing/services/file-service/pkg/pb/file/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
)

// BillingClient handles communication with the billing service
type BillingClient struct {
	baseURL string
	client  *http.Client
}

// NewBillingClient creates a new billing client
func NewBillingClient(baseURL string) *BillingClient {
	return &BillingClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

// UpdateUsage updates storage usage in billing service
func (bc *BillingClient) UpdateUsage(ctx context.Context, userID string, usedBytes int64, fileCount int64, operation string) error {
	// For now, we'll just log the update
	// In a real implementation, this would make an HTTP call to the billing service
	fmt.Printf("Billing usage update (mock): user_id=%s, used_bytes=%d, file_count=%d, operation=%s\n",
		userID, usedBytes, fileCount, operation)
	return nil
}

// CheckQuota checks if user can upload a file
func (bc *BillingClient) CheckQuota(ctx context.Context, userID string, fileSizeBytes int64) (bool, string, int64, error) {
	// For now, we'll just return true (allow upload)
	// In a real implementation, this would make an HTTP call to the billing service
	fmt.Printf("Billing quota check (mock): user_id=%s, file_size_bytes=%d\n", userID, fileSizeBytes)
	return true, "Upload allowed", 1000000000, nil // 1GB available
}

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	log := logger.NewLogger(cfg.LogLevel)
	log.WithFields(logrus.Fields{
		"environment": cfg.Environment,
		"log_level":   cfg.LogLevel,
	}).Info("Starting File Service")

	// Initialize MongoDB
	log.Info("Connecting to MongoDB...")
	mongodb, err := database.NewMongoDB(cfg.MongoURI, cfg.MongoDatabase, 10*time.Second)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer func() {
		log.Info("Closing MongoDB connection...")
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
	if err := storageRepo.EnsureIndexes(context.Background()); err != nil {
		log.Fatalf("Failed to create storage indexes: %v", err)
	}
	log.Info("MongoDB indexes created successfully")

	// Initialize Redis cache
	var redisCache *cache.RedisCache
	if cfg.RedisEnabled {
		log.WithFields(logrus.Fields{
			"addr": cfg.RedisAddr,
			"db":   cfg.RedisDB,
			"ttl":  cfg.RedisCacheTTL,
		}).Info("Connecting to Redis...")

		redisCache, err = cache.NewRedisCache(
			cfg.RedisAddr,
			cfg.RedisPassword,
			cfg.RedisDB,
			cfg.RedisCacheTTL,
			cfg.RedisMaxRetries,
			cfg.RedisPoolSize,
			cfg.RedisMinIdleConns,
			log,
			cfg.RedisEnabled,
		)
		if err != nil {
			log.Fatalf("Failed to connect to Redis: %v", err)
		}
		defer func() {
			log.Info("Closing Redis connection...")
			if err := redisCache.Close(); err != nil {
				log.Errorf("Error closing Redis: %v", err)
			}
		}()
		log.Info("Redis connected successfully")
	} else {
		log.Warn("Redis caching is disabled")
		redisCache, _ = cache.NewRedisCache("", "", 0, 0, 0, 0, 0, log, false)
	}

	// Initialize MinIO storage
	log.Info("Connecting to MinIO...")
	minioStorage, err := storage.NewMinioStorage(
		cfg.MinioEndpoint,
		cfg.MinioExternalEndpoint,
		cfg.MinioAccessKey,
		cfg.MinioSecretKey,
		cfg.MinioBucket,
		cfg.MinioUseSSL,
	)
	if err != nil {
		log.Fatalf("Failed to initialize MinIO storage: %v", err)
	}
	log.Info("MinIO connected successfully")

	// Initialize Kafka producer
	log.WithFields(logrus.Fields{
		"brokers": cfg.KafkaBrokers,
		"retries": cfg.UploadRetries,
	}).Info("Connecting to Kafka...")

	producer := kafka.NewProducer(
		cfg.KafkaBrokers,
		"file-events",
		cfg.UploadRetries,
		log,
	)
	defer func() {
		log.Info("Closing Kafka producer...")
		if err := producer.Close(); err != nil {
			log.Errorf("Error closing Kafka producer: %v", err)
		}
	}()
	log.Info("Kafka producer initialized successfully")

	// Initialize billing client (using HTTP for now)
	var billingClient *BillingClient
	if cfg.BillingServiceGRPC != "" {
		billingClient = NewBillingClient(cfg.BillingServiceGRPC)
		log.Info("Initialized billing client")
	} else {
		log.Warn("Billing service address not configured, continuing without billing integration")
		billingClient = nil
	}

	// Initialize gRPC handler with Redis cache
	fileHandler := grpcHandler.NewFileHandler(
		fileRepo,
		storageRepo,
		minioStorage,
		producer,
		cfg,
		log,
		redisCache,
		billingClient,
	)

	// Start gRPC server
	grpcServer := grpc.NewServer()
	filev1.RegisterFileServiceServer(grpcServer, fileHandler)
	reflection.Register(grpcServer)

	grpcListener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", cfg.ServiceHost, cfg.GRPCPort))
	if err != nil {
		log.Fatalf("Failed to listen on gRPC port: %v", err)
	}

	// Start gRPC server in goroutine
	go func() {
		log.WithField("port", cfg.GRPCPort).Info("File Service gRPC server starting...")
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	// Start gRPC Gateway (REST API) in goroutine
	httpServer := &http.Server{}
	go func() {
		if err := startGRPCGateway(cfg, log, redisCache, httpServer, fileHandler, storageRepo); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start gRPC Gateway: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down File Service...")

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	// Shutdown HTTP server
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Errorf("Error shutting down HTTP server: %v", err)
	}

	// Stop gRPC server gracefully
	grpcServer.GracefulStop()

	log.Info("File Service stopped successfully")
}

func startGRPCGateway(cfg *config.Config, log *logrus.Logger, redisCache *cache.RedisCache, httpServer *http.Server, fileHandler *grpcHandler.FileHandler, storageRepo *repository.StorageRepository) error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Create JWT validator
	jwtValidator := jwt.NewJWTValidator(cfg.JWTSecret)

	// Create gRPC-Gateway mux with custom metadata annotator
	mux := runtime.NewServeMux(
		runtime.WithMetadata(func(ctx context.Context, req *http.Request) metadata.MD {
			md := metadata.New(nil)

			// Extract Authorization header and validate JWT token
			if auth := req.Header.Get("Authorization"); auth != "" {
				md.Set("authorization", auth)
				fmt.Printf("File Service - Authorization header found: %s\n", auth[:int(math.Min(50, float64(len(auth))))])

				// Validate JWT token and extract user_id
				if userID, err := jwtValidator.ExtractUserID(auth); err == nil {
					md.Set("user_id", userID)
					fmt.Printf("File Service - User ID extracted from JWT: %s\n", userID)
				} else {
					fmt.Printf("File Service - JWT validation failed: %v\n", err)
				}
			} else {
				fmt.Printf("File Service - No Authorization header found\n")
			}

			// Fallback: Extract user_id from query parameters (for backward compatibility)
			if userID := req.URL.Query().Get("user_id"); userID != "" {
				md.Set("user_id", userID)
				fmt.Printf("File Service - User ID extracted from query param: %s\n", userID)
			}

			// Fallback: Extract user_id from request body for POST requests (for backward compatibility)
			if req.Method == "POST" && req.Body != nil {
				// Read the request body
				body, err := io.ReadAll(req.Body)
				if err == nil {
					// Parse JSON to extract user_id
					var jsonData map[string]interface{}
					if err := json.Unmarshal(body, &jsonData); err == nil {
						if userID, ok := jsonData["user_id"].(string); ok && userID != "" {
							md.Set("user_id", userID)
							fmt.Printf("File Service - User ID extracted from body: %s\n", userID)
						}
					}
					// Restore the request body for further processing
					req.Body = io.NopCloser(strings.NewReader(string(body)))
				}
			}

			// Debug: Print all query parameters
			fmt.Printf("File Service - Query parameters: %v\n", req.URL.Query())
			fmt.Printf("File Service - Request URL: %s\n", req.URL.String())

			return md
		}),
	)

	// Setup gRPC connection to local gRPC server
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	grpcEndpoint := fmt.Sprintf("localhost:%s", cfg.GRPCPort)

	// Register File Service handler
	err := filev1.RegisterFileServiceHandlerFromEndpoint(ctx, mux, grpcEndpoint, opts)
	if err != nil {
		return fmt.Errorf("failed to register file service handler: %w", err)
	}

	// Add custom handler for ListFiles to handle query parameters properly
	mux.HandlePath("GET", "/api/v1/files", func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		// Extract query parameters
		pageStr := r.URL.Query().Get("page")
		limitStr := r.URL.Query().Get("limit")

		// Set default values
		page := int32(1)
		limit := int32(20)

		// Parse page parameter
		if pageStr != "" {
			if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
				page = int32(p)
			}
		}

		// Parse limit parameter
		if limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
				limit = int32(l)
			}
		}

		// Extract user_id from Authorization header
		userID := ""
		if auth := r.Header.Get("Authorization"); auth != "" {
			// Create JWT validator
			jwtValidator := jwt.NewJWTValidator(cfg.JWTSecret)
			if extractedUserID, err := jwtValidator.ExtractUserID(auth); err == nil {
				userID = extractedUserID
			}
		}

		// If no user_id found, return error
		if userID == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"user_id not found in token"}`))
			return
		}

		// Create gRPC request
		req := &filev1.ListFilesRequest{
			UserId: userID,
			Page:   page,
			Limit:  limit,
		}

		// Create gRPC connection to local server
		conn, err := grpc.Dial(grpcEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"failed to connect to gRPC server"}`))
			return
		}
		defer conn.Close()

		// Create gRPC client
		client := filev1.NewFileServiceClient(conn)

		// Create context with metadata
		ctx := context.Background()
		md := metadata.New(nil)
		md.Set("user_id", userID)
		ctx = metadata.NewOutgoingContext(ctx, md)

		// Call gRPC service
		resp, err := client.ListFiles(ctx, req)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf(`{"error":"gRPC call failed: %s"}`, err.Error())))
			return
		}

		// Convert response to JSON
		response := map[string]interface{}{
			"files": resp.Files,
			"page":  resp.Page,
			"limit": resp.Limit,
			"total": resp.Total,
		}

		jsonResp, err := json.Marshal(response)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"failed to marshal response"}`))
			return
		}

		// Return response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResp)
	})

	// Create Gin router
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.Default()

	// CORS middleware
	router.Use(corsMiddleware())

	// Health check endpoint with Redis check
	router.GET("/health", func(c *gin.Context) {
		health := gin.H{
			"status":  "healthy",
			"service": "file-service",
			"version": "1.1.0",
			"time":    time.Now().Format(time.RFC3339),
		}

		// Check Redis health
		if redisCache != nil && redisCache.IsEnabled() {
			if err := redisCache.HealthCheck(c.Request.Context()); err != nil {
				health["redis_status"] = "unhealthy"
				health["redis_error"] = err.Error()
				c.JSON(http.StatusServiceUnavailable, health)
				return
			}
			health["redis_status"] = "healthy"
		} else {
			health["redis_status"] = "disabled"
		}

		c.JSON(http.StatusOK, health)
	})

	// Direct storage usage endpoint (bypass gRPC-Gateway)
	router.GET("/v1/files/storage/usage", func(c *gin.Context) {
		// Get user ID from JWT token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		jwtValidator := jwt.NewJWTValidator(cfg.JWTSecret)
		userID, err := jwtValidator.ExtractUserID(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		// Get storage stats
		stats, err := storageRepo.GetOrCreate(c.Request.Context(), userID)
		if err != nil {
			log.WithError(err).Error("Failed to get storage stats")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get storage usage"})
			return
		}

		// Return storage usage
		c.JSON(http.StatusOK, gin.H{
			"used_bytes":       stats.UsedBytes,
			"quota_bytes":      stats.QuotaBytes,
			"file_count":       stats.FileCount,
			"used_gb":          stats.GetUsedGB(),
			"quota_gb":         stats.GetQuotaGB(),
			"usage_percentage": stats.GetUsagePercentage(),
		})
	})

	// Cache stats endpoint (for monitoring)
	router.GET("/cache/stats", func(c *gin.Context) {
		if redisCache == nil || !redisCache.IsEnabled() {
			c.JSON(http.StatusOK, gin.H{
				"enabled": false,
				"message": "Redis cache is disabled",
			})
			return
		}

		stats, err := redisCache.GetStats(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, stats)
	})

	// Delete file endpoint (soft delete - move to trash)
	router.DELETE("/v1/files/:id", func(c *gin.Context) {
		// Get user ID from JWT token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		jwtValidator := jwt.NewJWTValidator(cfg.JWTSecret)
		userID, err := jwtValidator.ExtractUserID(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		fileID := c.Param("id")

		// Add user_id to context metadata
		ctx := metadata.NewOutgoingContext(c.Request.Context(), metadata.Pairs("user_id", userID))

		req := &filev1.DeleteFileRequest{
			FileId: fileID,
		}
		resp, err := fileHandler.DeleteFile(ctx, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, resp)
	})

	// Mount gRPC-Gateway
	router.Any("/api/*path", gin.WrapH(mux))

	// Start HTTP server
	httpAddr := fmt.Sprintf("%s:%s", cfg.ServiceHost, cfg.ServicePort)
	log.WithField("address", httpAddr).Info("File Service REST API (gRPC-Gateway) starting...")

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
