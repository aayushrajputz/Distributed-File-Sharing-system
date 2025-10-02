package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/yourusername/distributed-file-sharing-platform/services/billing-service/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type SubscriptionRepository struct {
	collection *mongo.Collection
}

func NewSubscriptionRepository(db *mongo.Database) *SubscriptionRepository {
	return &SubscriptionRepository{
		collection: db.Collection("subscriptions"),
	}
}

// Create creates a new subscription
func (r *SubscriptionRepository) Create(ctx context.Context, subscription *models.Subscription) error {
	subscription.CreatedAt = time.Now()
	subscription.UpdatedAt = time.Now()

	result, err := r.collection.InsertOne(ctx, subscription)
	if err != nil {
		return fmt.Errorf("failed to create subscription: %w", err)
	}

	subscription.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// FindByUserID finds the active subscription for a user
func (r *SubscriptionRepository) FindByUserID(ctx context.Context, userID primitive.ObjectID) (*models.Subscription, error) {
	var subscription models.Subscription
	err := r.collection.FindOne(ctx, bson.M{
		"userId": userID,
		"status": bson.M{"$in": []models.SubscriptionStatus{
			models.SubscriptionStatusActive,
			models.SubscriptionStatusPending,
		}},
	}).Decode(&subscription)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // No active subscription
		}
		return nil, fmt.Errorf("failed to find subscription: %w", err)
	}

	return &subscription, nil
}

// FindBySessionID finds a subscription by session ID
func (r *SubscriptionRepository) FindBySessionID(ctx context.Context, sessionID string) (*models.Subscription, error) {
	var subscription models.Subscription
	err := r.collection.FindOne(ctx, bson.M{"sessionId": sessionID}).Decode(&subscription)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("subscription not found")
		}
		return nil, fmt.Errorf("failed to find subscription: %w", err)
	}

	return &subscription, nil
}

// Update updates a subscription
func (r *SubscriptionRepository) Update(ctx context.Context, subscription *models.Subscription) error {
	subscription.UpdatedAt = time.Now()

	filter := bson.M{"_id": subscription.ID}
	update := bson.M{"$set": subscription}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("subscription not found")
	}

	return nil
}

// UpdateStatus updates subscription status
func (r *SubscriptionRepository) UpdateStatus(ctx context.Context, subscriptionID primitive.ObjectID, status models.SubscriptionStatus, paymentStatus models.PaymentStatus) error {
	update := bson.M{
		"$set": bson.M{
			"status":        status,
			"paymentStatus": paymentStatus,
			"updatedAt":     time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": subscriptionID}, update)
	if err != nil {
		return fmt.Errorf("failed to update subscription status: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("subscription not found")
	}

	return nil
}

// CancelSubscription cancels a subscription
func (r *SubscriptionRepository) CancelSubscription(ctx context.Context, userID primitive.ObjectID) error {
	filter := bson.M{
		"userId": userID,
		"status": models.SubscriptionStatusActive,
	}
	update := bson.M{
		"$set": bson.M{
			"status":    models.SubscriptionStatusCancelled,
			"updatedAt": time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to cancel subscription: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("no active subscription found")
	}

	return nil
}

// EnsureIndexes creates necessary indexes
func (r *SubscriptionRepository) EnsureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "userId", Value: 1}},
		},
		{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "status", Value: 1},
			},
		},
		{
			Keys: bson.D{{Key: "sessionId", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "transactionId", Value: 1}},
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// FindActiveByUserID finds the active subscription for a user
func (r *SubscriptionRepository) FindActiveByUserID(ctx context.Context, userID primitive.ObjectID) (*models.Subscription, error) {
	var subscription models.Subscription
	err := r.collection.FindOne(ctx, bson.M{
		"userId": userID,
		"status": models.SubscriptionStatusActive,
	}).Decode(&subscription)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // No active subscription
		}
		return nil, fmt.Errorf("failed to find active subscription: %w", err)
	}
	return &subscription, nil
}

// FindByID finds a subscription by ID
func (r *SubscriptionRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.Subscription, error) {
	var subscription models.Subscription
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&subscription)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find subscription: %w", err)
	}
	return &subscription, nil
}

// Cancel cancels a subscription
func (r *SubscriptionRepository) Cancel(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.UpdateOne(ctx,
		bson.M{"_id": id},
		bson.M{
			"$set": bson.M{
				"status":    models.SubscriptionStatusCancelled,
				"updatedAt": time.Now(),
			},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to cancel subscription: %w", err)
	}
	return nil
}
