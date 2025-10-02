package service

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/yourusername/distributed-file-sharing-platform/services/billing-service/internal/models"
	"github.com/yourusername/distributed-file-sharing-platform/services/billing-service/internal/payment"
	"github.com/yourusername/distributed-file-sharing-platform/services/billing-service/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BillingService struct {
	planRepo         *repository.PlanRepository
	subscriptionRepo *repository.SubscriptionRepository
	usageRepo        *repository.UsageRepository
	stripeService    *payment.StripeService
}

func NewBillingService(
	planRepo *repository.PlanRepository,
	subscriptionRepo *repository.SubscriptionRepository,
	usageRepo *repository.UsageRepository,
	stripeService *payment.StripeService,
) *BillingService {
	return &BillingService{
		planRepo:         planRepo,
		subscriptionRepo: subscriptionRepo,
		usageRepo:        usageRepo,
		stripeService:    stripeService,
	}
}

// ListPlans returns all available plans
func (s *BillingService) ListPlans(ctx context.Context) ([]models.Plan, error) {
	plans, err := s.planRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list plans: %w", err)
	}
	return plans, nil
}

// GetPlan returns a specific plan by ID
func (s *BillingService) GetPlan(ctx context.Context, planID string) (*models.Plan, error) {
	id, err := primitive.ObjectIDFromHex(planID)
	if err != nil {
		return nil, fmt.Errorf("invalid plan ID: %w", err)
	}

	plan, err := s.planRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}

	return plan, nil
}

// GetUserSubscription returns the user's current subscription
func (s *BillingService) GetUserSubscription(ctx context.Context, userID string) (*models.Subscription, *models.Plan, error) {
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid user ID: %w", err)
	}

	subscription, err := s.subscriptionRepo.FindActiveByUserID(ctx, uid)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	// If no active subscription, return Free plan
	if subscription == nil {
		freePlan, err := s.planRepo.FindByName(ctx, models.PlanFree)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get free plan: %w", err)
		}
		return nil, freePlan, nil
	}

	// Get plan details
	plan, err := s.planRepo.FindByID(ctx, subscription.PlanID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get plan: %w", err)
	}

	return subscription, plan, nil
}

// CreateSubscription creates a new subscription and payment session
func (s *BillingService) CreateSubscription(ctx context.Context, userID, planID, paymentMethod string) (*models.Subscription, string, string, error) {
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, "", "", fmt.Errorf("invalid user ID: %w", err)
	}

	pid, err := primitive.ObjectIDFromHex(planID)
	if err != nil {
		return nil, "", "", fmt.Errorf("invalid plan ID: %w", err)
	}

	// Get plan details
	plan, err := s.planRepo.FindByID(ctx, pid)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to get plan: %w", err)
	}

	// Check if user already has an active subscription
	existingSub, err := s.subscriptionRepo.FindActiveByUserID(ctx, uid)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to check existing subscription: %w", err)
	}

	if existingSub != nil {
		return nil, "", "", fmt.Errorf("user already has an active subscription")
	}

	// Create subscription record
	subscription := &models.Subscription{
		UserID:        uid,
		PlanID:        pid,
		Status:        models.SubscriptionStatusPending,
		PaymentStatus: models.PaymentStatusPending,
		StartDate:     time.Now(),
		EndDate:       time.Now().AddDate(0, 1, 0), // 1 month from now
		PaymentMethod: paymentMethod,
	}

	if err := s.subscriptionRepo.Create(ctx, subscription); err != nil {
		return nil, "", "", fmt.Errorf("failed to create subscription: %w", err)
	}

	// Create payment session based on payment method
	var paymentURL, sessionID string

	switch paymentMethod {
	case "stripe":
		session, err := s.stripeService.CreateCheckoutSession(plan, userID, subscription.ID.Hex())
		if err != nil {
			return nil, "", "", fmt.Errorf("failed to create Stripe session: %w", err)
		}
		paymentURL = session.URL
		sessionID = session.ID

		// Update subscription with session ID
		subscription.SessionID = sessionID
		if err := s.subscriptionRepo.Update(ctx, subscription); err != nil {
			logrus.WithError(err).Error("Failed to update subscription with session ID")
		}

	case "razorpay":
		// TODO: Implement Razorpay integration
		return nil, "", "", fmt.Errorf("Razorpay not implemented yet")

	default:
		return nil, "", "", fmt.Errorf("unsupported payment method: %s", paymentMethod)
	}

	logrus.WithFields(logrus.Fields{
		"user_id":         userID,
		"subscription_id": subscription.ID.Hex(),
		"plan":            plan.Name,
		"payment_method":  paymentMethod,
	}).Info("Subscription created")

	return subscription, paymentURL, sessionID, nil
}

// CancelSubscription cancels a user's subscription
func (s *BillingService) CancelSubscription(ctx context.Context, userID, subscriptionID string) error {
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	sid, err := primitive.ObjectIDFromHex(subscriptionID)
	if err != nil {
		return fmt.Errorf("invalid subscription ID: %w", err)
	}

	// Get subscription
	subscription, err := s.subscriptionRepo.FindByID(ctx, sid)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// Verify ownership
	if subscription.UserID != uid {
		return fmt.Errorf("subscription does not belong to user")
	}

	// Cancel subscription
	if err := s.subscriptionRepo.Cancel(ctx, sid); err != nil {
		return fmt.Errorf("failed to cancel subscription: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"user_id":         userID,
		"subscription_id": subscriptionID,
	}).Info("Subscription cancelled")

	return nil
}

// GetUsage returns the user's storage usage
func (s *BillingService) GetUsage(ctx context.Context, userID string) (*UsageInfo, error) {
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Get user's plan
	_, plan, err := s.GetUserSubscription(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user subscription: %w", err)
	}

	// Get usage
	usage, err := s.usageRepo.FindOrCreate(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage: %w", err)
	}

	usageInfo := &UsageInfo{
		UserID:           userID,
		PlanName:         plan.Name,
		QuotaBytes:       plan.QuotaBytes,
		UsedBytes:        usage.UsedBytes,
		QuotaGB:          plan.GetQuotaGB(),
		UsedGB:           usage.GetUsedGB(),
		PercentUsed:      usage.GetPercentUsed(plan.QuotaBytes),
		UpgradeAvailable: plan.Name != models.PlanEnterprise,
		QuotaExceeded:    usage.UsedBytes >= plan.QuotaBytes,
	}

	return usageInfo, nil
}

// UsageInfo represents storage usage information
type UsageInfo struct {
	UserID           string
	PlanName         string
	QuotaBytes       int64
	UsedBytes        int64
	QuotaGB          float64
	UsedGB           float64
	PercentUsed      float64
	UpgradeAvailable bool
	QuotaExceeded    bool
}

// CheckQuota checks if a user can upload a file of given size
func (s *BillingService) CheckQuota(ctx context.Context, userID string, fileSizeBytes int64) (bool, string, int64, error) {
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return false, "Invalid user ID", 0, fmt.Errorf("invalid user ID: %w", err)
	}

	// Get user's plan
	_, plan, err := s.GetUserSubscription(ctx, userID)
	if err != nil {
		return false, "Failed to get subscription", 0, fmt.Errorf("failed to get subscription: %w", err)
	}

	// Get current usage
	usage, err := s.usageRepo.FindOrCreate(ctx, uid)
	if err != nil {
		return false, "Failed to get usage", 0, fmt.Errorf("failed to get usage: %w", err)
	}

	// Check if upload would exceed quota
	if !usage.CanUpload(fileSizeBytes, plan.QuotaBytes) {
		availableBytes := usage.GetAvailableBytes(plan.QuotaBytes)
		message := fmt.Sprintf("Storage limit reached. You have %d bytes available, but need %d bytes. Please upgrade your plan.", availableBytes, fileSizeBytes)
		return false, message, availableBytes, nil
	}

	return true, "Upload allowed", usage.GetAvailableBytes(plan.QuotaBytes), nil
}

// UpdateUsage updates the user's storage usage
func (s *BillingService) UpdateUsage(ctx context.Context, userID string, bytesDelta int64, operation string) (int64, error) {
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return 0, fmt.Errorf("invalid user ID: %w", err)
	}

	var usage *models.Usage

	switch operation {
	case "upload":
		err = s.usageRepo.IncrementUsage(ctx, uid, bytesDelta)
		if err != nil {
			return 0, fmt.Errorf("failed to increment usage: %w", err)
		}
		// Get updated usage
		usage, err = s.usageRepo.FindByUserID(ctx, uid)
		if err != nil {
			return 0, fmt.Errorf("failed to get updated usage: %w", err)
		}
		logrus.WithFields(logrus.Fields{
			"user_id":   userID,
			"bytes":     bytesDelta,
			"new_total": usage.UsedBytes,
		}).Info("Usage incremented")

	case "delete":
		err = s.usageRepo.DecrementUsage(ctx, uid, bytesDelta)
		if err != nil {
			return 0, fmt.Errorf("failed to decrement usage: %w", err)
		}
		// Get updated usage
		usage, err = s.usageRepo.FindByUserID(ctx, uid)
		if err != nil {
			return 0, fmt.Errorf("failed to get updated usage: %w", err)
		}
		logrus.WithFields(logrus.Fields{
			"user_id":   userID,
			"bytes":     bytesDelta,
			"new_total": usage.UsedBytes,
		}).Info("Usage decremented")

	default:
		return 0, fmt.Errorf("invalid operation: %s", operation)
	}

	return usage.UsedBytes, nil
}

// HandlePaymentWebhook processes payment webhook events
func (s *BillingService) HandlePaymentWebhook(ctx context.Context, provider, eventType, sessionID, transactionID string) error {
	switch provider {
	case "stripe":
		return s.handleStripeWebhook(ctx, eventType, sessionID, transactionID)
	case "razorpay":
		return fmt.Errorf("Razorpay webhooks not implemented yet")
	default:
		return fmt.Errorf("unsupported payment provider: %s", provider)
	}
}

func (s *BillingService) handleStripeWebhook(ctx context.Context, eventType, sessionID, transactionID string) error {
	// Find subscription by session ID
	subscription, err := s.subscriptionRepo.FindBySessionID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to find subscription: %w", err)
	}

	switch eventType {
	case "checkout.session.completed":
		// Update subscription to active and paid
		subscription.Status = models.SubscriptionStatusActive
		subscription.PaymentStatus = models.PaymentStatusPaid
		subscription.TransactionID = transactionID

		if err := s.subscriptionRepo.Update(ctx, subscription); err != nil {
			return fmt.Errorf("failed to update subscription: %w", err)
		}

		logrus.WithFields(logrus.Fields{
			"subscription_id": subscription.ID.Hex(),
			"user_id":         subscription.UserID.Hex(),
			"transaction_id":  transactionID,
		}).Info("Subscription activated via webhook")

	case "checkout.session.expired":
		// Mark subscription as failed
		subscription.Status = models.SubscriptionStatusCancelled
		subscription.PaymentStatus = models.PaymentStatusFailed

		if err := s.subscriptionRepo.Update(ctx, subscription); err != nil {
			return fmt.Errorf("failed to update subscription: %w", err)
		}

		logrus.WithFields(logrus.Fields{
			"subscription_id": subscription.ID.Hex(),
			"user_id":         subscription.UserID.Hex(),
		}).Warn("Subscription expired via webhook")

	default:
		logrus.WithField("event_type", eventType).Debug("Unhandled webhook event type")
	}

	return nil
}
