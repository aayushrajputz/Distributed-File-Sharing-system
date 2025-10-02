package grpc

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/yourusername/distributed-file-sharing-platform/services/billing-service/internal/repository"
	billingv1 "github.com/yourusername/distributed-file-sharing-platform/services/billing-service/pkg/pb/billing/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type BillingHandler struct {
	billingv1.UnimplementedBillingServiceServer
	planRepo         *repository.PlanRepository
	subscriptionRepo *repository.SubscriptionRepository
	usageRepo        *repository.UsageRepository
}

func NewBillingHandler(planRepo *repository.PlanRepository, subscriptionRepo *repository.SubscriptionRepository, usageRepo *repository.UsageRepository) *BillingHandler {
	return &BillingHandler{
		planRepo:         planRepo,
		subscriptionRepo: subscriptionRepo,
		usageRepo:        usageRepo,
	}
}

// ListPlans returns all available subscription plans
func (h *BillingHandler) ListPlans(ctx context.Context, req *billingv1.ListPlansRequest) (*billingv1.ListPlansResponse, error) {
	logrus.Info("ListPlans called")

	// Get plans from repository
	plans, err := h.planRepo.FindAll(ctx)
	if err != nil {
		logrus.Errorf("Failed to get plans: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to get plans")
	}

	// Convert to protobuf format
	var pbPlans []*billingv1.Plan
	for _, plan := range plans {
		pbPlan := &billingv1.Plan{
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
		pbPlans = append(pbPlans, pbPlan)
	}

	return &billingv1.ListPlansResponse{
		Plans: pbPlans,
	}, nil
}
