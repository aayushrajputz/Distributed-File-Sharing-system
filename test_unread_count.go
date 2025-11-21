package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Mock models to avoid importing the whole project
type NotificationStatus string

const (
	StatusUnread NotificationStatus = "unread"
	StatusRead   NotificationStatus = "read"
)

func main() {
	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	db := client.Database("file-sharing")
	collection := db.Collection("notifications")

	// Test Data
	userIDString := "691f370c5e516413ceb3e644"
	userIDObjectID, _ := primitive.ObjectIDFromHex(userIDString)

	// 1. Clean up existing test data
	collection.DeleteMany(ctx, bson.M{"title": "Test Notification"})

	// 2. Insert notification with String UserID
	_, err = collection.InsertOne(ctx, bson.M{
		"user_id":    userIDString,
		"status":     StatusUnread,
		"title":      "Test Notification (String ID)",
		"created_at": time.Now(),
	})
	if err != nil {
		log.Fatal(err)
	}

	// 3. Insert notification with ObjectId UserID
	_, err = collection.InsertOne(ctx, bson.M{
		"user_id":    userIDObjectID,
		"status":     StatusUnread,
		"title":      "Test Notification (ObjectID)",
		"created_at": time.Now(),
	})
	if err != nil {
		log.Fatal(err)
	}

	// 4. Query using the Fixed Logic
	count, err := GetUnreadCount(ctx, collection, userIDString)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d unread notifications (Expected 2)\n", count)

	if count == 2 {
		fmt.Println("✅ SUCCESS: Handled both String and ObjectId user_ids!")
	} else {
		fmt.Println("❌ FAILED: Did not find both notifications.")
	}
}

// GetUnreadCount - The fixed function
func GetUnreadCount(ctx context.Context, collection *mongo.Collection, userID string) (int64, error) {
	// Handle both string and ObjectId user_id
	userIDs := []interface{}{userID}
	if objID, err := primitive.ObjectIDFromHex(userID); err == nil {
		userIDs = append(userIDs, objID)
	}

	filter := bson.M{
		"user_id": bson.M{"$in": userIDs},
		"$or": []bson.M{
			{"status": bson.M{"$ne": StatusRead}},
			{"read_at": bson.M{"$exists": false}},
		},
	}

	return collection.CountDocuments(ctx, filter)
}
