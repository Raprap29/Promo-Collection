package store

// import (
// 	"context"
// 	"fmt"
// 	"time"

// 	"go.mongodb.org/mongo-driver/bson"
// 	"go.mongodb.org/mongo-driver/mongo"
// )

// // checks if a collection exists in the database
// func DoesCollectionExist(client *mongo.Client, dbName, collectionName string) (bool, error) {
// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	defer cancel()

// 	collections, err := client.Database(dbName).ListCollectionNames(ctx, bson.M{"name": collectionName})
// 	if err != nil {
// 		return false, fmt.Errorf("error listing collections: %w", err)
// 	}

// 	for _, name := range collections {
// 		if name == collectionName {
// 			return true, nil
// 		}
// 	}
// 	return false, nil
// }

// // If collection exists then query the database
// func CollectionExists(client *mongo.Client, dbName, collectionName string) (*mongo.Collection, error) {
// 	collectionExists, err := DoesCollectionExist(client, dbName, collectionName)
// 	if err != nil {
// 		return nil, fmt.Errorf("error checking collection existence: %w", err)
// 	}
// 	if !collectionExists {
// 		return nil, fmt.Errorf("collection '%s' does not exist in database '%s'", collectionName, dbName)
// 	}

// 	return client.Database(dbName).Collection(collectionName), nil
// }
