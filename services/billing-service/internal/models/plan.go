package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Plan represents a subscription plan
type Plan struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name          string             `bson:"name" json:"name"`
	QuotaBytes    int64              `bson:"quotaBytes" json:"quotaBytes"`
	PricePerMonth float64            `bson:"pricePerMonth" json:"pricePerMonth"`
	Description   string             `bson:"description" json:"description"`
	Features      []string           `bson:"features" json:"features"`
	IsPopular     bool               `bson:"isPopular" json:"isPopular"`
	CreatedAt     time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt     time.Time          `bson:"updatedAt" json:"updatedAt"`
}

// PlanName constants
const (
	PlanFree       = "Free"
	PlanPro        = "Pro"
	PlanEnterprise = "Enterprise"
)

// Quota constants (in bytes)
const (
	QuotaFree       = 5 * 1024 * 1024 * 1024        // 5 GB
	QuotaPro        = 100 * 1024 * 1024 * 1024      // 100 GB
	QuotaEnterprise = 1024 * 1024 * 1024 * 1024     // 1 TB
)

// GetDefaultPlans returns the default subscription plans
func GetDefaultPlans() []Plan {
	now := time.Now()
	return []Plan{
		{
			ID:            primitive.NewObjectID(),
			Name:          PlanFree,
			QuotaBytes:    QuotaFree,
			PricePerMonth: 0,
			Description:   "Perfect for personal use",
			Features: []string{
				"5 GB storage",
				"Basic file sharing",
				"Email support",
			},
			IsPopular: false,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:            primitive.NewObjectID(),
			Name:          PlanPro,
			QuotaBytes:    QuotaPro,
			PricePerMonth: 10.00,
			Description:   "Great for professionals",
			Features: []string{
				"100 GB storage",
				"Advanced file sharing",
				"Priority support",
				"Version history",
				"Advanced security",
			},
			IsPopular: true,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:            primitive.NewObjectID(),
			Name:          PlanEnterprise,
			QuotaBytes:    QuotaEnterprise,
			PricePerMonth: 49.00,
			Description:   "Best for teams and businesses",
			Features: []string{
				"1 TB storage",
				"Unlimited file sharing",
				"24/7 premium support",
				"Advanced analytics",
				"Team collaboration",
				"Custom branding",
				"API access",
			},
			IsPopular: false,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}

// GetQuotaGB returns the quota in GB
func (p *Plan) GetQuotaGB() float64 {
	return float64(p.QuotaBytes) / (1024 * 1024 * 1024)
}

