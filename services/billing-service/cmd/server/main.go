package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/yourusername/distributed-file-sharing-platform/services/billing-service/internal/config"
	"github.com/yourusername/distributed-file-sharing-platform/services/billing-service/internal/database"
	grpcHandler "github.com/yourusername/distributed-file-sharing-platform/services/billing-service/internal/grpc"
	"github.com/yourusername/distributed-file-sharing-platform/services/billing-service/internal/payment"
	"github.com/yourusername/distributed-file-sharing-platform/services/billing-service/internal/repository"
	"github.com/yourusername/distributed-file-sharing-platform/services/billing-service/internal/service"
	billingv1 "github.com/yourusername/distributed-file-sharing-platform/services/billing-service/pkg/pb/billing/v1"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Setup logger
	log := logrus.New()
	log.SetLevel(logrus.InfoLevel)

	// Connect to MongoDB
	db, err := database.NewMongoDB(cfg.MongoURI, cfg.MongoDatabase)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer db.Close()

	// Initialize repositories
	planRepo := repository.NewPlanRepository(db.Database)
	subscriptionRepo := repository.NewSubscriptionRepository(db.Database)
	usageRepo := repository.NewUsageRepository(db.Database)

	// Initialize payment services
	stripeService := payment.NewStripeService(
		cfg.StripeSecretKey,
		cfg.StripeWebhookSecret,
		"http://localhost:3000/billing/success", // TODO: Make configurable
		"http://localhost:3000/billing/cancel",
	)

	razorpayService := payment.NewRazorpayService(
		cfg.RazorpayKeyID,
		cfg.RazorpayKeySecret,
		cfg.RazorpayWebhookSecret,
	)

	// Initialize service layer
	billingService := service.NewBillingService(planRepo, subscriptionRepo, usageRepo, stripeService, razorpayService)

	// Initialize gRPC handler
	grpcHandler := grpcHandler.NewBillingHandler(billingService)

	// Start gRPC server
	go startGRPCServer(cfg, grpcHandler, log)

	// Start HTTP server
	startHTTPServer(cfg, log)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down servers...")
}

func startGRPCServer(cfg *config.Config, handler *grpcHandler.BillingHandler, log *logrus.Logger) {
	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", cfg.GRPCPort, err)
	}

	grpcServer := grpc.NewServer()
	billingv1.RegisterBillingServiceServer(grpcServer, handler)

	log.Infof("gRPC server starting on port %s", cfg.GRPCPort)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}

func startHTTPServer(cfg *config.Config, log *logrus.Logger) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service":   "billing-service",
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
		})
	})

	// Basic API endpoints
	api := r.Group("/api/v1/billing")
	{
		api.GET("/plans", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"plans": []gin.H{
					{
						"id":              "free",
						"name":            "Free",
						"quota_bytes":     5368709120, // 5GB
						"price_per_month": 0,
						"description":     "Free plan with basic features",
						"features":        []string{"5GB Storage", "Basic Support"},
						"is_popular":      false,
						"created_at":      time.Now().Format(time.RFC3339),
						"updated_at":      time.Now().Format(time.RFC3339),
					},
					{
						"id":              "pro",
						"name":            "Pro",
						"quota_bytes":     107374182400, // 100GB
						"price_per_month": 9.99,
						"description":     "Pro plan with advanced features",
						"features":        []string{"100GB Storage", "Priority Support", "Advanced Analytics"},
						"is_popular":      true,
						"created_at":      time.Now().Format(time.RFC3339),
						"updated_at":      time.Now().Format(time.RFC3339),
					},
					{
						"id":              "enterprise",
						"name":            "Enterprise",
						"quota_bytes":     1099511627776, // 1TB
						"price_per_month": 29.99,
						"description":     "Enterprise plan with all features",
						"features":        []string{"1TB Storage", "24/7 Support", "Advanced Analytics", "API Access"},
						"is_popular":      false,
						"created_at":      time.Now().Format(time.RFC3339),
						"updated_at":      time.Now().Format(time.RFC3339),
					},
				},
			})
		})

		// Get user subscription
		api.GET("/subscription", func(c *gin.Context) {
			userID := c.Query("user_id")
			if userID == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
				return
			}

			// For now, return a mock subscription (Pro plan)
			c.JSON(http.StatusOK, gin.H{
				"subscription": gin.H{
					"id":             "sub_123",
					"user_id":        userID,
					"plan_id":        "pro",
					"status":         "active",
					"payment_status": "paid",
					"start_date":     time.Now().AddDate(0, -1, 0).Format(time.RFC3339),
					"end_date":       time.Now().AddDate(0, 11, 0).Format(time.RFC3339),
					"payment_method": "stripe",
					"created_at":     time.Now().AddDate(0, -1, 0).Format(time.RFC3339),
					"updated_at":     time.Now().Format(time.RFC3339),
				},
				"has_active_subscription": true,
			})
		})

		// Get storage usage
		api.GET("/usage", func(c *gin.Context) {
			userID := c.Query("user_id")
			if userID == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
				return
			}

			// Mock usage data - in real implementation, this would call the file service
			c.JSON(http.StatusOK, gin.H{
				"usage": gin.H{
					"user_id":           userID,
					"plan_name":         "Pro",
					"quota_bytes":       107374182400, // 100GB
					"used_bytes":        0,            // This should be calculated from actual files
					"quota_gb":          100,
					"used_gb":           0,
					"percent_used":      0,
					"upgrade_available": true,
					"quota_exceeded":    false,
				},
			})
		})

		// Create subscription
		api.POST("/subscribe", func(c *gin.Context) {
			var req struct {
				UserID        string `json:"user_id"`
				PlanID        string `json:"plan_id"`
				PaymentMethod string `json:"payment_method"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			// Mock subscription creation
			c.JSON(http.StatusOK, gin.H{
				"subscription": gin.H{
					"id":             "sub_" + fmt.Sprintf("%d", time.Now().Unix()),
					"user_id":        req.UserID,
					"plan_id":        req.PlanID,
					"status":         "pending",
					"payment_status": "pending",
					"start_date":     time.Now().Format(time.RFC3339),
					"end_date":       time.Now().AddDate(1, 0, 0).Format(time.RFC3339),
					"payment_method": req.PaymentMethod,
					"created_at":     time.Now().Format(time.RFC3339),
					"updated_at":     time.Now().Format(time.RFC3339),
				},
				"payment_url":   "https://checkout.stripe.com/pay/mock_session",
				"client_secret": "pi_mock_client_secret",
				"session_id":    "cs_mock_session_id",
			})
		})

		// Cancel subscription
		api.POST("/subscription/cancel", func(c *gin.Context) {
			var req struct {
				UserID         string `json:"user_id"`
				SubscriptionID string `json:"subscription_id"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "Subscription cancelled successfully",
			})
		})
	}

	log.Infof("HTTP server starting on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}
