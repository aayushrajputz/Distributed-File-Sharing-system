package repository

import (
	"context"
	"errors"
	"time"

	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ErrTemplateNotFound = errors.New("template not found")
)

type TemplateRepository struct {
	collection *mongo.Collection
}

func NewTemplateRepository(database *mongo.Database) *TemplateRepository {
	return &TemplateRepository{
		collection: database.Collection("notification_templates"),
	}
}

// Create creates a new notification template
func (r *TemplateRepository) Create(ctx context.Context, template *models.NotificationTemplate) error {
	template.CreatedAt = time.Now()
	template.UpdatedAt = time.Now()
	
	_, err := r.collection.InsertOne(ctx, template)
	return err
}

// GetByID gets a template by ID
func (r *TemplateRepository) GetByID(ctx context.Context, id string) (*models.NotificationTemplate, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var template models.NotificationTemplate
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&template)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrTemplateNotFound
		}
		return nil, err
	}

	return &template, nil
}

// GetByTemplateID gets a template by template ID
func (r *TemplateRepository) GetByTemplateID(ctx context.Context, templateID string) (*models.NotificationTemplate, error) {
	var template models.NotificationTemplate
	err := r.collection.FindOne(ctx, bson.M{"template_id": templateID}).Decode(&template)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrTemplateNotFound
		}
		return nil, err
	}

	return &template, nil
}

// GetByEventTypeAndChannel gets a template by event type and channel
func (r *TemplateRepository) GetByEventTypeAndChannel(ctx context.Context, eventType models.EventType, channel models.NotificationChannel) (*models.NotificationTemplate, error) {
	filter := bson.M{
		"event_type": eventType,
		"channel":    channel,
		"is_active":  true,
	}

	var template models.NotificationTemplate
	err := r.collection.FindOne(ctx, filter).Decode(&template)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrTemplateNotFound
		}
		return nil, err
	}

	return &template, nil
}

// GetAll gets all templates with pagination
func (r *TemplateRepository) GetAll(ctx context.Context, page, limit int, eventType *models.EventType, channel *models.NotificationChannel) ([]*models.NotificationTemplate, int64, error) {
	filter := bson.M{}
	
	if eventType != nil {
		filter["event_type"] = *eventType
	}
	
	if channel != nil {
		filter["channel"] = *channel
	}

	// Get total count
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Calculate skip
	skip := (page - 1) * limit

	// Find templates
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var templates []*models.NotificationTemplate
	if err = cursor.All(ctx, &templates); err != nil {
		return nil, 0, err
	}

	return templates, total, nil
}

// Update updates a template
func (r *TemplateRepository) Update(ctx context.Context, id string, template *models.NotificationTemplate) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	template.UpdatedAt = time.Now()
	
	filter := bson.M{"_id": objectID}
	update := bson.M{"$set": template}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return ErrTemplateNotFound
	}

	return nil
}

// Delete deletes a template
func (r *TemplateRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return ErrTemplateNotFound
	}

	return nil
}

// SetActive sets the active status of a template
func (r *TemplateRepository) SetActive(ctx context.Context, id string, active bool) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"is_active":  active,
			"updated_at": time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return ErrTemplateNotFound
	}

	return nil
}

// GetActiveTemplates gets all active templates
func (r *TemplateRepository) GetActiveTemplates(ctx context.Context) ([]*models.NotificationTemplate, error) {
	filter := bson.M{"is_active": true}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var templates []*models.NotificationTemplate
	if err = cursor.All(ctx, &templates); err != nil {
		return nil, err
	}

	return templates, nil
}

// CreateIndexes creates necessary indexes
func (r *TemplateRepository) CreateIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "template_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "event_type", Value: 1}, {Key: "channel", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "is_active", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "created_at", Value: -1}},
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
}
