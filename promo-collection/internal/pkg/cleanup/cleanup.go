package cleanup

import (
	"context"

	"promocollection/internal/pkg/logger"
	mongodb "promocollection/internal/pkg/db/mongo"
	redisdb "promocollection/internal/pkg/db/redis"
)

func CleanupResources(ctx context.Context, mongoClient *mongodb.MongoClient, redisClient *redisdb.RedisClient) {
	if mongoClient != nil && mongoClient.Client != nil {
		if err := mongodb.Disconnect(mongoClient.Client); err != nil {
			logger.CtxError(ctx, "Failed to disconnect from MongoDB", err)
		}
	}
	if redisClient != nil && redisClient.Client != nil {
		if err := redisdb.Disconnect(redisClient.Client); err != nil {
			logger.CtxError(ctx, "Failed to disconnect from Redis", err)
		}
	}
}
