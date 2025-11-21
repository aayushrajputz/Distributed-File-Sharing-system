package grpc

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/yourusername/distributed-file-sharing-platform/services/billing-service/internal/models"
	"github.com/yourusername/distributed-file-sharing-platform/services/billing-service/internal/service"
	billingv1 "github.com/yourusername/distributed-file-sharing-platform/services/billing-service/pkg/pb/billing/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type BillingHandler struct {
	billingv1.UnimplementedBillingServiceServer
	service *service.BillingService
}

func NewBillingHandler(service *service.BillingService) *BillingHandler {
	return &BillingHandler{
		service: service,
	}
}

// ListPlans returns all available subscription plans
func (h *BillingHandler) ListPlans(ctx context.Context, req *billingv1.ListPlansRequest) (*billingv1.ListPlansResponse, error) {
	logrus.Info("ListPlans called")

	// Get plans from service
	plans, err := h.service.ListPlans(ctx)
	if err != nil {
		logrus.Errorf("Failed to get plans: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to get plans")
	}

	// Convert to protobuf format
	var pbPlans []*billingv1.Plan
	for _, plan := range plans {
		pbPlans = append(pbPlans, convertPlanToProto(plan))
	}

	return &billingv1.ListPlansResponse{
		Plans: pbPlans,
	}, nil
}

// GetPlan returns a specific plan by ID
func (h *BillingHandler) GetPlan(ctx context.Context, req *billingv1.GetPlanRequest) (*billingv1.GetPlanResponse, error) {
	logrus.WithField("plan_id", req.PlanId).Info("GetPlan called")

	plan, err := h.service.GetPlan(ctx, req.PlanId)
	if err != nil {
		logrus.Errorf("Failed to get plan: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to get plan")
	}

	return &billingv1.GetPlanResponse{
		Plan: convertPlanToProto(*plan),
	}, nil
}

// GetUserSubscription returns the user's current subscription
func (h *BillingHandler) GetUserSubscription(ctx context.Context, req *billingv1.GetUserSubscriptionRequest) (*billingv1.GetUserSubscriptionResponse, error) {
	logrus.WithField("user_id", req.UserId).Info("GetUserSubscription called")

	subscription, plan, err := h.service.GetUserSubscription(ctx, req.UserId)
	if err != nil {
		logrus.Errorf("Failed to get user subscription: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to get user subscription")
	}

	var pbSub *billingv1.Subscription
	if subscription != nil {
		pbSub = convertSubscriptionToProto(subscription, plan)
	}

	return &billingv1.GetUserSubscriptionResponse{
		Subscription:          pbSub,
		HasActiveSubscription: subscription != nil,
	}, nil
}

// CreateSubscription creates a new subscription
func (h *BillingHandler) CreateSubscription(ctx context.Context, req *billingv1.CreateSubscriptionRequest) (*billingv1.CreateSubscriptionResponse, error) {
	logrus.WithFields(logrus.Fields{
		"user_id":        req.UserId,
		"plan_id":        req.PlanId,
		"payment_method": req.PaymentMethod,
	}).Info("CreateSubscription called")

	subscription, paymentURL, sessionID, err := h.service.CreateSubscription(ctx, req.UserId, req.PlanId, req.PaymentMethod)
	if err != nil {
		logrus.Errorf("Failed to create subscription: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to create subscription: %v", err)
	}

	// We need the plan to convert subscription to proto, but CreateSubscription returns the subscription object which has PlanID.
	// Ideally CreateSubscription should return the plan too, or we fetch it.
	// For now, let's fetch the plan again or just pass nil for plan in proto conversion if acceptable,
	// but the proto message has a Plan field.
	// Let's fetch the plan.
	plan, err := h.service.GetPlan(ctx, req.PlanId)
	if err != nil {
		logrus.Errorf("Failed to get plan after subscription creation: %v", err)
		// Continue without plan details in subscription object, or fail.
	}

	return &billingv1.CreateSubscriptionResponse{
		Subscription: convertSubscriptionToProto(subscription, plan),
		PaymentUrl:   paymentURL,
		SessionId:    sessionID,
		ClientSecret: "", // Add if needed
	}, nil
}

// CancelSubscription cancels a subscription
func (h *BillingHandler) CancelSubscription(ctx context.Context, req *billingv1.CancelSubscriptionRequest) (*billingv1.CancelSubscriptionResponse, error) {
	logrus.WithFields(logrus.Fields{
		"user_id":         req.UserId,
		"subscription_id": req.SubscriptionId,
	}).Info("CancelSubscription called")

	err := h.service.CancelSubscription(ctx, req.UserId, req.SubscriptionId)
	if err != nil {
		logrus.Errorf("Failed to cancel subscription: %v", err)
		return &billingv1.CancelSubscriptionResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &billingv1.CancelSubscriptionResponse{
		Success: true,
		Message: "Subscription cancelled successfully",
	}, nil
}

// GetUsage returns usage stats
func (h *BillingHandler) GetUsage(ctx context.Context, req *billingv1.GetUsageRequest) (*billingv1.GetUsageResponse, error) {
	logrus.WithField("user_id", req.UserId).Info("GetUsage called")

	usage, err := h.service.GetUsage(ctx, req.UserId)
	if err != nil {
		logrus.Errorf("Failed to get usage: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to get usage")
	}

	return &billingv1.GetUsageResponse{
		Usage: &billingv1.Usage{
			UserId:           usage.UserID,
			PlanName:         usage.PlanName,
			QuotaBytes:       usage.QuotaBytes,
			UsedBytes:        usage.UsedBytes,
			QuotaGb:          usage.QuotaGB,
			UsedGb:           usage.UsedGB,
			PercentUsed:      usage.PercentUsed,
			UpgradeAvailable: usage.UpgradeAvailable,
			QuotaExceeded:    usage.QuotaExceeded,
		},
	}, nil
}

// CheckQuota checks if upload is allowed
func (h *BillingHandler) CheckQuota(ctx context.Context, req *billingv1.CheckQuotaRequest) (*billingv1.CheckQuotaResponse, error) {
	allowed, message, availableBytes, err := h.service.CheckQuota(ctx, req.UserId, req.FileSizeBytes)
	if err != nil {
		logrus.Errorf("Failed to check quota: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to check quota")
	}

	// Calculate quota and used bytes if needed, but service returns available.
	// For now, we'll just return what we have.
	return &billingv1.CheckQuotaResponse{
		Allowed:        allowed,
		Message:        message,
		AvailableBytes: availableBytes,
	}, nil
}

// UpdateUsage updates usage stats
func (h *BillingHandler) UpdateUsage(ctx context.Context, req *billingv1.UpdateUsageRequest) (*billingv1.UpdateUsageResponse, error) {
	newUsedBytes, err := h.service.UpdateUsage(ctx, req.UserId, req.BytesDelta, req.Operation)
	if err != nil {
		logrus.Errorf("Failed to update usage: %v", err)
		return &billingv1.UpdateUsageResponse{
			Success: false,
		}, nil
	}

	return &billingv1.UpdateUsageResponse{
		Success:      true,
		NewUsedBytes: newUsedBytes,
	}, nil
}

// HandlePaymentWebhook handles payment webhooks
func (h *BillingHandler) HandlePaymentWebhook(ctx context.Context, req *billingv1.PaymentWebhookRequest) (*billingv1.PaymentWebhookResponse, error) {
	logrus.WithFields(logrus.Fields{
		"provider":   req.Provider,
		"event_type": req.EventType,
	}).Info("HandlePaymentWebhook called")

	err := h.service.HandlePaymentWebhook(ctx, req.Provider, req.EventType, req.SessionId, req.TransactionId, req.RawPayload)
	if err != nil {
		logrus.Errorf("Failed to handle webhook: %v", err)
		return &billingv1.PaymentWebhookResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &billingv1.PaymentWebhookResponse{
		Success: true,
		Message: "Webhook processed successfully",
	}, nil
}

// Helper functions

func convertPlanToProto(plan models.Plan) *billingv1.Plan {
	return &billingv1.Plan{
		Id:            plan.ID.Hex(),
		Name:          plan.Name,
		QuotaBytes:    plan.QuotaBytes,
		PricePerMonth: plan.PricePerMonth,
		Description:   plan.Description,
		Features:      plan.Features,
		IsPopular:     plan.IsPopular,
		CreatedAt:     timestamppb.New(plan.CreatedAt),
		UpdatedAt:     timestamppb.New(plan.UpdatedAt),
	}
}

func convertSubscriptionToProto(sub *models.Subscription, plan *models.Plan) *billingv1.Subscription {
	pbSub := &billingv1.Subscription{
		Id:            sub.ID.Hex(),
		UserId:        sub.UserID.Hex(),
		PlanId:        sub.PlanID.Hex(),
		Status:        convertSubscriptionStatus(sub.Status),
		PaymentStatus: convertPaymentStatus(sub.PaymentStatus),
		StartDate:     timestamppb.New(sub.StartDate),
		EndDate:       timestamppb.New(sub.EndDate),
		TransactionId: sub.TransactionID,
		PaymentMethod: sub.PaymentMethod,
		CreatedAt:     timestamppb.New(sub.CreatedAt),
		UpdatedAt:     timestamppb.New(sub.UpdatedAt),
	}

	if plan != nil {
		pbSub.Plan = convertPlanToProto(*plan)
	}

	return pbSub
}

func convertSubscriptionStatus(status models.SubscriptionStatus) billingv1.SubscriptionStatus {
	switch status {
	case models.SubscriptionStatusActive:
		return billingv1.SubscriptionStatus_SUBSCRIPTION_STATUS_ACTIVE
	case models.SubscriptionStatusExpired:
		return billingv1.SubscriptionStatus_SUBSCRIPTION_STATUS_EXPIRED
	case models.SubscriptionStatusCancelled:
		return billingv1.SubscriptionStatus_SUBSCRIPTION_STATUS_CANCELLED
	case models.SubscriptionStatusPending:
		return billingv1.SubscriptionStatus_SUBSCRIPTION_STATUS_PENDING
	default:
		return billingv1.SubscriptionStatus_SUBSCRIPTION_STATUS_UNSPECIFIED
	}
}

func convertPaymentStatus(status models.PaymentStatus) billingv1.PaymentStatus {
	switch status {
	case models.PaymentStatusPaid:
		return billingv1.PaymentStatus_PAYMENT_STATUS_PAID
	case models.PaymentStatusPending:
		return billingv1.PaymentStatus_PAYMENT_STATUS_PENDING
	case models.PaymentStatusFailed:
		return billingv1.PaymentStatus_PAYMENT_STATUS_FAILED
	case models.PaymentStatusRefunded:
		return billingv1.PaymentStatus_PAYMENT_STATUS_REFUNDED
	default:
		return billingv1.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED
	}
}
