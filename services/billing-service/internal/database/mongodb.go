package database

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDB struct {
	Client   *mongo.Client
	Database *mongo.Database
}

func NewMongoDB(uri, database string) (*MongoDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping the database
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	logrus.Info("MongoDB connected successfully")

	db := client.Database(database)

	// Create indexes
	if err := createIndexes(ctx, db); err != nil {
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	return &MongoDB{
		Client:   client,
		Database: db,
	}, nil
}

func createIndexes(ctx context.Context, db *mongo.Database) error {
	// Plans collection indexes
	plansCollection := db.Collection("plans")
	_, err := plansCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "name", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create plans indexes: %w", err)
	}

	// Subscriptions collection indexes
	subscriptionsCollection := db.Collection("subscriptions")
	_, err = subscriptionsCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "userId", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "userId", Value: 1}, {Key: "status", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "sessionId", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "transactionId", Value: 1}},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create subscriptions indexes: %w", err)
	}

	// Usage collection indexes
	usageCollection := db.Collection("usage")
	_, err = usageCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "userId", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create usage indexes: %w", err)
	}

	logrus.Info("MongoDB indexes created successfully")
	return nil
}

func (m *MongoDB) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return m.Client.Disconnect(ctx)
}
