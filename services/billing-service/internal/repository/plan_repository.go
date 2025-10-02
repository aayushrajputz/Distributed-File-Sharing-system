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

type PlanRepository struct {
	collection *mongo.Collection
}

func NewPlanRepository(db *mongo.Database) *PlanRepository {
	return &PlanRepository{
		collection: db.Collection("plans"),
	}
}

// Create creates a new plan
func (r *PlanRepository) Create(ctx context.Context, plan *models.Plan) error {
	plan.CreatedAt = time.Now()
	plan.UpdatedAt = time.Now()

	result, err := r.collection.InsertOne(ctx, plan)
	if err != nil {
		return fmt.Errorf("failed to create plan: %w", err)
	}

	plan.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// FindAll returns all plans
func (r *PlanRepository) FindAll(ctx context.Context) ([]models.Plan, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to find plans: %w", err)
	}
	defer cursor.Close(ctx)

	var plans []models.Plan
	if err := cursor.All(ctx, &plans); err != nil {
		return nil, fmt.Errorf("failed to decode plans: %w", err)
	}

	return plans, nil
}

// FindByID finds a plan by ID
func (r *PlanRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.Plan, error) {
	var plan models.Plan
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&plan)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("plan not found")
		}
		return nil, fmt.Errorf("failed to find plan: %w", err)
	}

	return &plan, nil
}

// FindByName finds a plan by name
func (r *PlanRepository) FindByName(ctx context.Context, name string) (*models.Plan, error) {
	var plan models.Plan
	err := r.collection.FindOne(ctx, bson.M{"name": name}).Decode(&plan)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("plan not found")
		}
		return nil, fmt.Errorf("failed to find plan: %w", err)
	}

	return &plan, nil
}

// Update updates a plan
func (r *PlanRepository) Update(ctx context.Context, plan *models.Plan) error {
	plan.UpdatedAt = time.Now()

	filter := bson.M{"_id": plan.ID}
	update := bson.M{"$set": plan}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update plan: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("plan not found")
	}

	return nil
}

// Delete deletes a plan
func (r *PlanRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete plan: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("plan not found")
	}

	return nil
}

// InitializeDefaultPlans creates the default plans if they don't exist
func (r *PlanRepository) InitializeDefaultPlans(ctx context.Context) error {
	// Check if plans already exist
	count, err := r.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return fmt.Errorf("failed to count plans: %w", err)
	}

	if count > 0 {
		return nil // Plans already exist
	}

	// Create default plans
	defaultPlans := models.GetDefaultPlans()
	for i := range defaultPlans {
		if err := r.Create(ctx, &defaultPlans[i]); err != nil {
			return fmt.Errorf("failed to create default plan %s: %w", defaultPlans[i].Name, err)
		}
	}

	return nil
}

