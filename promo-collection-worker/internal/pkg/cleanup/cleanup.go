package cleanup

import (
	"context"
	"net/http"
	"promo-collection-worker/internal/pkg/db/mongo"
	"promo-collection-worker/internal/pkg/db/redis"
	"promo-collection-worker/internal/pkg/gcs"
	"promo-collection-worker/internal/pkg/kafka"
	"promo-collection-worker/internal/pkg/log_messages"
	"promo-collection-worker/internal/pkg/logger"
	"time"
)

func CleanupResources(
	ctx context.Context,
	pubsubConsumer interface{ Close() error },
	pubsubPublisher interface{ Close() error },
	kafkaProducer *kafka.KafkaProducer,
	mongoClient *mongo.MongoClient,
	redisClient *redis.RedisClient,
	server *http.Server,
	gcsClient gcs.GcsInterface,
) {
	logger.CtxInfo(ctx, log_messages.CleanupStarted)

	cleanupPubSubResource(pubsubConsumer, "PubSub consumer", ctx)
	cleanupPubSubResource(pubsubPublisher, "PubSub publisher", ctx)

	cleanupKafkaResource(kafkaProducer, ctx)

	cleanupMongoResource(mongoClient, ctx)
	cleanupRedisResource(redisClient, ctx)
	cleanupHTTPServer(server, ctx)
	cleanupGCSResource(gcsClient, ctx)

	logger.CtxInfo(ctx, log_messages.CleanupCompleted)
}

func cleanupPubSubResource(resource interface{ Close() error }, resourceName string, ctx context.Context) {
	if resource == nil {
		return
	}
	if unsubscribeFunc, ok := resource.(interface{ Unsubscribe(context.Context) error }); ok {
		logger.CtxInfo(ctx, "Gracefully unsubscribing from "+resourceName+"...")
		if err := unsubscribeFunc.Unsubscribe(ctx); err != nil {
			logger.CtxError(ctx, "Failed to unsubscribe from "+resourceName, err)
		} else {
			logger.CtxInfo(ctx, "Successfully unsubscribed from "+resourceName)
		}
	}
	if err := resource.Close(); err != nil {
		logger.CtxError(ctx, "Failed to close "+resourceName, err)
	} else {
		logger.CtxInfo(ctx, resourceName+" closed successfully")
	}
}

func cleanupKafkaResource(kafkaProducer *kafka.KafkaProducer, ctx context.Context) {
	if kafkaProducer == nil {
		return
	}
	if err := kafkaProducer.Close(); err != nil {
		logger.CtxError(ctx, "Failed to close Kafka producer", err)
	} else {
		logger.CtxInfo(ctx, "Kafka producer closed successfully")
	}
}

func cleanupMongoResource(mongoClient *mongo.MongoClient, ctx context.Context) {
	if mongoClient == nil || mongoClient.Client == nil {
		return
	}
	mongoCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := mongoClient.Client.Disconnect(mongoCtx); err != nil {
		logger.CtxError(mongoCtx, "Failed to disconnect MongoDB client", err)
	} else {
		logger.CtxInfo(mongoCtx, "MongoDB client disconnected successfully")
	}
}

func cleanupRedisResource(redisClient *redis.RedisClient, ctx context.Context) {
	if redisClient == nil || redisClient.Client == nil {
		return
	}
	if err := redis.Disconnect(redisClient.Client); err != nil {
		logger.CtxError(ctx, "Failed to close Redis client", err)
	} else {
		logger.CtxInfo(ctx, "Redis client closed successfully")
	}
}

func cleanupHTTPServer(server *http.Server, ctx context.Context) {
	if server == nil {
		return
	}
	shutdownCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.CtxError(ctx, "Failed to shutdown HTTP server", err)
	} else {
		logger.CtxInfo(ctx, "HTTP server shutdown successfully")
	}
}

func cleanupGCSResource(gcsClient gcs.GcsInterface, ctx context.Context) {
	if gcsClient == nil {
		return
	}
	gcsClient.Close(ctx)
	logger.CtxInfo(ctx, log_messages.GCSClientClosedSuccessfully)
}
