package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"

	// billingv1 "github.com/yourusername/distributed-file-sharing/services/api-gateway/pkg/pb/billing/v1"
	"github.com/yourusername/distributed-file-sharing/services/api-gateway/internal/config"
	"github.com/yourusername/distributed-file-sharing/services/api-gateway/internal/middleware"
	authv1 "github.com/yourusername/distributed-file-sharing/services/api-gateway/pkg/pb/auth/v1"
	filev1 "github.com/yourusername/distributed-file-sharing/services/api-gateway/pkg/pb/file/v1"
	notificationv1 "github.com/yourusername/distributed-file-sharing/services/api-gateway/pkg/pb/notification/v1"
)

// FileResponse represents a file in the API response with properly formatted timestamps
type FileResponse struct {
	FileId      string `json:"file_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Size        int64  `json:"size"`
	MimeType    string `json:"mime_type"`
	OwnerId     string `json:"owner_id"`
	StoragePath string `json:"storage_path"`
	Checksum    string `json:"checksum,omitempty"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// convertProtoFileToResponse converts a protobuf file to FileResponse with RFC3339 timestamps
func convertProtoFileToResponse(protoFile *filev1.File) FileResponse {
	// Convert timestamps to RFC3339 format
	var createdAt, updatedAt string

	if protoFile.CreatedAt != nil {
		createdAt = protoFile.CreatedAt.AsTime().Format(time.RFC3339)
	} else {
		createdAt = time.Now().Format(time.RFC3339)
	}

	if protoFile.UpdatedAt != nil {
		updatedAt = protoFile.UpdatedAt.AsTime().Format(time.RFC3339)
	} else {
		updatedAt = time.Now().Format(time.RFC3339)
	}

	// Convert status to string
	status := protoFile.Status.String()
	if status == "" {
		status = "unknown"
	}

	return FileResponse{
		FileId:      protoFile.FileId,
		Name:        protoFile.Name,
		Description: protoFile.Description,
		Size:        protoFile.Size,
		MimeType:    protoFile.MimeType,
		OwnerId:     protoFile.OwnerId,
		StoragePath: protoFile.StoragePath,
		Checksum:    protoFile.Checksum,
		Status:      status,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}

// proxyToBillingService proxies requests to the billing service
func proxyToBillingService(c *gin.Context, cfg *config.Config) {
	// Get the path after /api/v1/billing
	path := c.Param("path")

	// Build the target URL - billing service runs on port 8084
	billingHost := "billing-service:8084"
	if cfg.Environment == "development" {
		billingHost = "localhost:8084"
	}
	targetURL := fmt.Sprintf("http://%s/api/v1/billing%s", billingHost, path)

	// Add query parameters
	if c.Request.URL.RawQuery != "" {
		targetURL += "?" + c.Request.URL.RawQuery
	}

	log.Printf("Proxying billing request to: %s", targetURL)

	// Create a new request
	req, err := http.NewRequest(c.Request.Method, targetURL, c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	// Copy headers
	for key, values := range c.Request.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Make the request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to reach billing service: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to reach billing service"})
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			c.Writer.Header().Add(key, value)
		}
	}

	// Copy status code
	c.Writer.WriteHeader(resp.StatusCode)

	// Copy body
	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			c.Writer.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
}

// proxyToFileService proxies requests to the file service
func proxyToFileService(c *gin.Context, cfg *config.Config) {
	// Get the path after /api/v1/files/private-folder
	path := c.Param("path")

	// Build the target URL - file service runs on port 8082
	fileHost := "file-service:8082"
	if cfg.Environment == "development" {
		fileHost = "localhost:8082"
	}
	targetURL := fmt.Sprintf("http://%s/api/v1/private-folder%s", fileHost, path)

	// Add query parameters
	if c.Request.URL.RawQuery != "" {
		targetURL += "?" + c.Request.URL.RawQuery
	}

	log.Printf("Proxying file service request to: %s", targetURL)

	// Create a new request
	req, err := http.NewRequest(c.Request.Method, targetURL, c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	// Copy headers
	for key, values := range c.Request.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Make the request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to reach file service: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to reach file service"})
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			c.Writer.Header().Add(key, value)
		}
	}

	// Copy status code
	c.Writer.WriteHeader(resp.StatusCode)

	// Copy body
	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			c.Writer.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
}

// handleListFiles handles the ListFiles API endpoint with proper query parameter parsing
func handleListFiles(c *gin.Context, cfg *config.Config) {
	// Extract query parameters
	pageStr := c.Query("page")
	limitStr := c.Query("limit")

	// Set default values
	page := 1
	limit := 20

	// Parse page parameter
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	// Parse limit parameter
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// Get user_id from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id not found in context"})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user_id in context"})
		return
	}

	// Create gRPC request
	req := &filev1.ListFilesRequest{
		UserId: userIDStr,
		Page:   int32(page),
		Limit:  int32(limit),
	}

	// Create gRPC connection to file service
	conn, err := grpc.Dial(cfg.FileServiceGRPC, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to file service"})
		return
	}
	defer conn.Close()

	// Create gRPC client
	client := filev1.NewFileServiceClient(conn)

	// Create context with metadata
	ctx := context.Background()
	md := metadata.New(nil)
	md.Set("user_id", userIDStr)
	ctx = metadata.NewOutgoingContext(ctx, md)

	// Call gRPC service
	resp, err := client.ListFiles(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("gRPC call failed: %s", err.Error())})
		return
	}

	// Convert protobuf files to response format with proper timestamps
	files := make([]FileResponse, len(resp.Files))
	for i, protoFile := range resp.Files {
		files[i] = convertProtoFileToResponse(protoFile)
	}

	// Return response with properly formatted timestamps
	c.JSON(http.StatusOK, gin.H{
		"files": files,
		"page":  resp.Page,
		"limit": resp.Limit,
		"total": resp.Total,
	})
}

func main() {
	// Load configuration
	cfg := config.Load()

	// Create gRPC-Gateway mux with custom metadata annotator
	gwmux := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(customMatcher),
		runtime.WithErrorHandler(customErrorHandler),
		runtime.WithMetadata(metadataAnnotator),
	)

	// Create gRPC dial options with timeout
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),                   // Block until connection is established
		grpc.WithTimeout(30 * time.Second), // Timeout after 30 seconds
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(math.MaxInt32)),
		grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(math.MaxInt32)),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             3 * time.Second,
			PermitWithoutStream: true,
		}),
	}

	// Use background context for gRPC connections to keep them alive
	ctx := context.Background()

	// Register Auth Service with retry logic
	log.Printf("Connecting to Auth Service at %s", cfg.AuthServiceGRPC)
	var authErr error
	for i := 0; i < 3; i++ {
		authErr = authv1.RegisterAuthServiceHandlerFromEndpoint(ctx, gwmux, cfg.AuthServiceGRPC, opts)
		if authErr == nil {
			log.Printf("Successfully connected to Auth Service")
			break
		}
		log.Printf("Failed to connect to Auth Service (attempt %d/3): %v", i+1, authErr)
		time.Sleep(2 * time.Second)
	}
	if authErr != nil {
		log.Printf("Warning: Could not connect to Auth Service after 3 attempts: %v", authErr)
	}

	// Register File Service with retry logic
	log.Printf("Connecting to File Service at %s", cfg.FileServiceGRPC)
	var fileErr error
	for i := 0; i < 3; i++ {
		fileErr = filev1.RegisterFileServiceHandlerFromEndpoint(ctx, gwmux, cfg.FileServiceGRPC, opts)
		if fileErr == nil {
			log.Printf("Successfully connected to File Service")
			break
		}
		log.Printf("Failed to connect to File Service (attempt %d/3): %v", i+1, fileErr)
		time.Sleep(2 * time.Second)
	}
	if fileErr != nil {
		log.Printf("Warning: Could not connect to File Service after 3 attempts: %v", fileErr)
	}

	// Register Notification Service with retry logic
	log.Printf("Connecting to Notification Service at %s", cfg.NotificationServiceGRPC)
	var notifErr error
	for i := 0; i < 3; i++ {
		notifErr = notificationv1.RegisterNotificationServiceHandlerFromEndpoint(ctx, gwmux, cfg.NotificationServiceGRPC, opts)
		if notifErr == nil {
			log.Printf("Successfully connected to Notification Service")
			break
		}
		log.Printf("Failed to connect to Notification Service (attempt %d/3): %v", i+1, notifErr)
		time.Sleep(2 * time.Second)
	}
	if notifErr != nil {
		log.Printf("Warning: Could not connect to Notification Service after 3 attempts: %v", notifErr)
	}

	// Register Billing Service (temporarily disabled for Docker build)
	// log.Printf("Connecting to Billing Service at %s", cfg.BillingServiceGRPC)
	// if err := billingv1.RegisterBillingServiceHandlerFromEndpoint(ctx, gwmux, cfg.BillingServiceGRPC, opts); err != nil {
	// 	log.Fatalf("Failed to register billing service: %v", err)
	// }

	// Create Gin router
	router := gin.Default()

	// Setup CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // Allow all origins for development
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"},
		AllowHeaders:     []string{"*"}, // Allow all headers
		ExposeHeaders:    []string{"Content-Length", "Access-Control-Allow-Origin", "Access-Control-Allow-Credentials"},
		AllowCredentials: false, // Set to false when using wildcard origins
		MaxAge:           12 * time.Hour,
	}))

	// Health check endpoint
	router.GET("/health", healthCheckHandler)
	router.GET("/", rootHandler)

	// API versioning
	router.GET("/api/versions", versionsHandler)

	// Create a custom handler that extracts user_id from Gin context
	fileServiceHandler := func(c *gin.Context) {
		// Store the Gin context in the request context so metadataAnnotator can access it
		ctx := context.WithValue(c.Request.Context(), "gin_context", c)
		c.Request = c.Request.WithContext(ctx)
		gwmux.ServeHTTP(c.Writer, c.Request)
	}

	// Apply auth middleware to file service endpoints
	fileServiceGroup := router.Group("/api")
	fileServiceGroup.Use(middleware.AuthMiddleware())

	// Custom handler for ListFiles to handle query parameters properly
	// Handle both /v1/files and /v1/files/ routes
	fileServiceGroup.GET("/v1/files", func(c *gin.Context) {
		handleListFiles(c, cfg)
	})

	fileServiceGroup.GET("/v1/files/", func(c *gin.Context) {
		handleListFiles(c, cfg)
	})

	// Handle storage usage route (must come before :id route)
	fileServiceGroup.GET("/v1/files/storage/usage", func(c *gin.Context) {
		// Mock response for storage usage
		c.JSON(http.StatusOK, gin.H{
			"used":       0,
			"total":      1073741824, // 1GB
			"percentage": 0,
		})
	})

	// Handle other file service routes
	fileServiceGroup.Any("/v1/files/upload", fileServiceHandler)
	fileServiceGroup.Any("/v1/files/shared", fileServiceHandler)
	fileServiceGroup.Any("/v1/files/favorites", fileServiceHandler)
	fileServiceGroup.Any("/v1/files/trash", fileServiceHandler)
	fileServiceGroup.Any("/v1/files/:id/complete", fileServiceHandler)
	fileServiceGroup.Any("/v1/files/:id/download", fileServiceHandler)
	fileServiceGroup.Any("/v1/files/:id/share", fileServiceHandler)
	fileServiceGroup.Any("/v1/files/:id/favorite", fileServiceHandler)
	fileServiceGroup.Any("/v1/files/:id/restore", fileServiceHandler)
	fileServiceGroup.Any("/v1/files/:id/permanent", fileServiceHandler)
	fileServiceGroup.Any("/v1/files/:id", fileServiceHandler)

	// Private folder routes (proxy directly to file service)
	fileServiceGroup.Any("/v1/private-folder/*path", fileServiceHandler)

	// Mount other services without auth middleware
	router.Any("/api/v1/auth/*path", gin.WrapF(gwmux.ServeHTTP))

	// Proxy notification service requests directly to notification service REST API
	// This bypasses gRPC and uses the notification service's REST endpoints
	notificationServiceURL := os.Getenv("NOTIFICATION_SERVICE_REST_URL")
	if notificationServiceURL == "" {
		notificationServiceURL = "http://notification-service:8084" // Default Docker service name
	}

	router.Any("/api/v1/notifications/*path", func(c *gin.Context) {
		// Extract user ID from JWT token
		userID := ""
		if authHeader := c.GetHeader("Authorization"); authHeader != "" {
			log.Printf("API Gateway - Authorization header found: %s", authHeader[:50]+"...")
			// Extract user ID from JWT token
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			token, err := jwt.ParseWithClaims(tokenString, &jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
				return []byte(os.Getenv("JWT_SECRET")), nil
			})
			if err == nil && token.Valid {
				if claims, ok := token.Claims.(*jwt.MapClaims); ok {
					if uid, ok := (*claims)["user_id"].(string); ok {
						userID = uid
						log.Printf("API Gateway - User ID extracted from token: %s", userID)
					}
				}
			}
		}

		// Also check query parameter for user_id (for unread-count endpoint)
		if queryUserID := c.Query("user_id"); queryUserID != "" {
			userID = queryUserID
			log.Printf("API Gateway - User ID extracted from query param: %s", userID)
		}

		// Build target URL
		path := c.Param("path")
		targetURL := fmt.Sprintf("%s/api/v1/notifications%s", notificationServiceURL, path)
		if c.Request.URL.RawQuery != "" {
			targetURL += "?" + c.Request.URL.RawQuery
		}

		log.Printf("API Gateway - Proxying notification request to: %s", targetURL)

		// Create proxy request
		proxyReq, err := http.NewRequest(c.Request.Method, targetURL, c.Request.Body)
		if err != nil {
			log.Printf("API Gateway - Failed to create proxy request: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to proxy request"})
			return
		}

		// Copy headers
		for key, values := range c.Request.Header {
			for _, value := range values {
				proxyReq.Header.Add(key, value)
			}
		}

		// Add X-User-ID header for notification service
		if userID != "" {
			proxyReq.Header.Set("X-User-ID", userID)
			log.Printf("API Gateway - Added X-User-ID header: %s", userID)
		}

		// Send request
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(proxyReq)
		if err != nil {
			log.Printf("API Gateway - Failed to proxy request: %v", err)
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Notification service unavailable"})
			return
		}
		defer resp.Body.Close()

		// Copy response headers
		for key, values := range resp.Header {
			for _, value := range values {
				c.Header(key, value)
			}
		}

		// Copy response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("API Gateway - Failed to read response body: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
			return
		}

		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
	})

	// Mount billing service - proxy directly to billing service
	// All billing endpoints go through the same proxy
	// Auth middleware will be applied selectively based on the path
	router.Any("/api/v1/billing/*path", func(c *gin.Context) {
		// Skip auth for public endpoints
		path := c.Param("path")
		if path == "/plans" {
			// Public endpoint - no auth required
			proxyToBillingService(c, cfg)
			return
		}

		// All other endpoints require auth
		middleware.AuthMiddleware()(c)
		if c.IsAborted() {
			return
		}
		proxyToBillingService(c, cfg)
	})

	// Mount file service private folder endpoints - proxy directly to file service
	router.Any("/api/v1/files/private-folder/*path", func(c *gin.Context) {
		// All private folder endpoints require auth
		middleware.AuthMiddleware()(c)
		if c.IsAborted() {
			return
		}
		proxyToFileService(c, cfg)
	})

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      router,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("API Gateway starting on port %s", cfg.Port)
		log.Printf("Environment: %s", cfg.Environment)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down API Gateway...")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("API Gateway stopped")
}

// customMatcher matches all headers including Authorization
func customMatcher(key string) (string, bool) {
	switch key {
	case "Authorization":
		return key, true
	default:
		return runtime.DefaultHeaderMatcher(key)
	}
}

// customErrorHandler handles gRPC errors
func customErrorHandler(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
	runtime.DefaultHTTPErrorHandler(ctx, mux, marshaler, w, r, err)
}

// metadataAnnotator extracts user_id from Gin context and adds it to gRPC metadata
func metadataAnnotator(ctx context.Context, r *http.Request) metadata.MD {
	md := metadata.New(nil)

	// Extract user_id from Gin context if available
	if ginCtx, ok := ctx.Value("gin_context").(*gin.Context); ok {
		if userID, exists := ginCtx.Get("user_id"); exists {
			if userIDStr, ok := userID.(string); ok {
				md.Set("user_id", userIDStr)
				fmt.Printf("API Gateway - User ID extracted from Gin context: %s\n", userIDStr)
			}
		}
	}

	// Extract Authorization header and add to metadata
	if auth := r.Header.Get("Authorization"); auth != "" {
		md.Set("authorization", auth)
		fmt.Printf("API Gateway - Authorization header found: %s\n", auth[:int(math.Min(50, float64(len(auth))))])
	}

	// Fallback: Extract user_id from query parameters for file service
	if userID := r.URL.Query().Get("user_id"); userID != "" {
		md.Set("user_id", userID)
		fmt.Printf("API Gateway - User ID extracted from query param: %s\n", userID)
	}

	// Debug: Print all query parameters
	fmt.Printf("API Gateway - Query parameters: %v\n", r.URL.Query())
	fmt.Printf("API Gateway - Request URL: %s\n", r.URL.String())

	return md
}

// healthCheckHandler returns service health status
func healthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "api-gateway",
		"time":    time.Now().UTC().Format(time.RFC3339),
	})
}

// rootHandler returns API information
func rootHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"service": "Distributed File-Sharing Platform API Gateway",
		"version": "1.0.0",
		"docs":    "/api/v1",
		"health":  "/health",
	})
}

// versionsHandler returns supported API versions
func versionsHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"versions": []string{"v1"},
		"current":  "v1",
		"endpoints": map[string]string{
			"auth":          "/api/v1/auth",
			"files":         "/api/v1/files",
			"notifications": "/api/v1/notifications",
		},
	})
}
