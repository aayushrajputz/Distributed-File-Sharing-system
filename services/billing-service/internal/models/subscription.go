package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SubscriptionStatus represents the status of a subscription
type SubscriptionStatus string

const (
	SubscriptionStatusActive    SubscriptionStatus = "active"
	SubscriptionStatusExpired   SubscriptionStatus = "expired"
	SubscriptionStatusCancelled SubscriptionStatus = "cancelled"
	SubscriptionStatusPending   SubscriptionStatus = "pending"
)

// PaymentStatus represents the status of a payment
type PaymentStatus string

const (
	PaymentStatusPending  PaymentStatus = "pending"
	PaymentStatusPaid     PaymentStatus = "paid"
	PaymentStatusFailed   PaymentStatus = "failed"
	PaymentStatusRefunded PaymentStatus = "refunded"
)

// Subscription represents a user's subscription to a plan
type Subscription struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID        primitive.ObjectID `bson:"userId" json:"userId"`
	PlanID        primitive.ObjectID `bson:"planId" json:"planId"`
	Status        SubscriptionStatus `bson:"status" json:"status"`
	PaymentStatus PaymentStatus      `bson:"paymentStatus" json:"paymentStatus"`
	StartDate     time.Time          `bson:"startDate" json:"startDate"`
	EndDate       time.Time          `bson:"endDate" json:"endDate"`
	TransactionID string             `bson:"transactionId" json:"transactionId"`
	PaymentMethod string             `bson:"paymentMethod" json:"paymentMethod"` // "stripe" or "razorpay"
	SessionID     string             `bson:"sessionId,omitempty" json:"sessionId,omitempty"`
	CreatedAt     time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt     time.Time          `bson:"updatedAt" json:"updatedAt"`
}

// IsActive checks if the subscription is currently active
func (s *Subscription) IsActive() bool {
	now := time.Now()
	return s.Status == SubscriptionStatusActive &&
		s.PaymentStatus == PaymentStatusPaid &&
		now.After(s.StartDate) &&
		now.Before(s.EndDate)
}

// IsExpired checks if the subscription has expired
func (s *Subscription) IsExpired() bool {
	return time.Now().After(s.EndDate)
}

// DaysRemaining returns the number of days remaining in the subscription
func (s *Subscription) DaysRemaining() int {
	if s.IsExpired() {
		return 0
	}
	duration := time.Until(s.EndDate)
	return int(duration.Hours() / 24)
}

