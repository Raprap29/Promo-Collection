package db

import (
	"context"
	"globe/dodrio_loan_availment/configs"
	"globe/dodrio_loan_availment/internal/pkg/logger"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDB struct {
	Client   *mongo.Client
	Database *mongo.Database
}

var MDB *MongoDB

func NewMongoDB() (*MongoDB, error) {

	uri := configs.DB_URI
	dbName := configs.DB_NAME

	// Configure client options with connection pooling settings
	clientOptions := options.Client().
		ApplyURI(uri).
		SetMaxPoolSize(configs.DB_MAXPOOLSIZE).                                           // Set maximum number of connections in the pool
		SetMinPoolSize(configs.DB_MINPOOLSIZE).                                           // Set minimum number of connections in the pool
		SetMaxConnIdleTime(time.Duration(configs.DB_MAXIDLETIME_INMINUTES) * time.Minute) // Set maximum idle time for a connection

	// Connect to MongoDB
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		logger.Error("Error in connecting to MongoDB")
		return nil, err
	}

	// Ping the MongoDB server
	if err := client.Ping(context.TODO(), nil); err != nil {
		return nil, err
	}

	db := client.Database(dbName)

	return &MongoDB{
		Client:   client,
		Database: db,
	}, nil
}

func (mdb *MongoDB) Close() error {
	return mdb.Client.Disconnect(context.TODO())
}
