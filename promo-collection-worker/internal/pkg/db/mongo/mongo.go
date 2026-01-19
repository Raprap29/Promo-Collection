package mongo

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"promo-collection-worker/internal/pkg/config"
	"promo-collection-worker/internal/pkg/logger"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoClient struct {
	Client   *mongo.Client
	Database *mongo.Database
}

func ConnectToMongoDB(ctx context.Context, cfg config.MongoConfig) (*MongoClient, error) {
	return connectWithConnector(ctx, cfg, &DefaultMongoConnector{})
}

func connectWithConnector(ctx context.Context, cfg config.MongoConfig, connector MongoConnector) (*MongoClient, error) {

	mongoURI := fmt.Sprintf("mongodb+srv://%s:%s@%s",
		url.QueryEscape(cfg.Username),
		url.QueryEscape(cfg.Password),
		strings.TrimPrefix(cfg.URI, "mongodb+srv://"),
	)

	// Redact username and password for safe logging
	safeURI := redactMongoURI(mongoURI)

	logger.CtxInfo(ctx, "Connecting to MongoDB",
		slog.String("uri", safeURI),
		slog.String("database", cfg.DBName),
	)

	connectTimeout := cfg.ConnectTimeout
	clientOpts := options.Client().
		ApplyURI(mongoURI).
		SetConnectTimeout(connectTimeout).
		SetServerSelectionTimeout(connectTimeout * 2).
		SetSocketTimeout(connectTimeout * 3).
		SetHeartbeatInterval(10 * time.Second).
		SetMaxConnIdleTime(cfg.MaxConnIdleTime).
		SetMaxPoolSize(cfg.MaxPoolSize).
		SetMinPoolSize(cfg.MinPoolSize)

	client, err := connector.Connect(ctx, clientOpts)
	if err != nil {
		logger.CtxError(ctx, "Failed to connect to MongoDB", err,
			slog.String("uri", safeURI),
			slog.String("database", cfg.DBName),
		)
		return nil, err
	}

	if err := connector.Ping(ctx, client); err != nil {
		logger.CtxError(ctx, "MongoDB ping failed", err,
			slog.String("uri", safeURI),
			slog.String("database", cfg.DBName),
		)
		return nil, err
	}

	logger.CtxInfo(ctx, "Successfully connected to MongoDB",
		slog.String("uri", safeURI),
		slog.String("database", cfg.DBName),
	)

	db := client.Database(cfg.DBName)

	logger.CtxDebug(ctx, "MongoDB client and database initialized",
		slog.String("db_name", cfg.DBName),
		slog.String("mongo_uri", safeURI),
	)

	return &MongoClient{
		Client:   client,
		Database: db,
	}, nil
}

func Disconnect(client *mongo.Client) error {
	return client.Disconnect(context.Background())
}

// redactMongoURI hides username and password from a MongoDB URI
func redactMongoURI(uri string) string {
	parts := strings.SplitN(uri, "@", 2)
	if len(parts) == 2 {
		return "mongodb+srv://***:***@" + parts[1]
	}
	return uri // fallback
}
