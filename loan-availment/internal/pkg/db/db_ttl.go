package db

import (
	"context"
	"globe/dodrio_loan_availment/configs"
	"globe/dodrio_loan_availment/internal/pkg/consts"
	"globe/dodrio_loan_availment/internal/pkg/logger"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func CreateDbTtlIfNotExists() {
	if MDB == nil || MDB.Database == nil {
		logger.Info("Skipping TTL index setup: MongoDB is not connected")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Access the database and collection
	//db := client.Database("myDatabase")
	collection := MDB.Database.Collection(consts.TransactionsInProgressCollection)

	// TTL settings: field name and expiration time (e.g., 3600 seconds for 1 hour)
	indexField := "createdAt"
	ttlDuration := int32(configs.AVAILMENT_TRANSACTION_TTL_IN_HOURS * 3600)

	// Check if the index already exists
	indexesCursor, err := collection.Indexes().List(ctx)
	if err != nil {
		logger.Error("Failed to list indexes: %v", err)
	}

	indexExists := false
	for indexesCursor.Next(ctx) {

		var index bson.M
		err := indexesCursor.Decode(&index)
		if err != nil {
			logger.Error("Error decoding index information:%v", err)
		}

		// Check if an index on the specific field with TTL exists
		//keyField := index["key"].(bson.M)[indexField]
		expiryValue, hasExpireOption := index["expireAfterSeconds"]

		if hasExpireOption {

			expiryTime := expiryValue.(int32)
			if expiryTime != ttlDuration {
				_, err := collection.Indexes().DropOne(ctx, index["name"].(string))
				if err != nil {
					logger.Error("could not drop index: %v", err)
				}
				indexExists = false
				logger.Info("TTL index deleted.")
				break
			} else {
				indexExists = true
				logger.Info("TTL index already exists.")
				break
			}
		}
	}

	if !indexExists {
		// Create TTL index only if it does not exist
		indexModel := mongo.IndexModel{
			Keys:    bson.D{{Key: indexField, Value: 1}}, // Index on "createdAt" field
			Options: options.Index().SetExpireAfterSeconds(ttlDuration),
		}

		// Create the index
		_, err := collection.Indexes().CreateOne(ctx, indexModel)
		if err != nil {
			logger.Error("Failed to create TTL index:%v", err)
		} else {
			logger.Info("TTL index created successfully.")
		}
	}

}
