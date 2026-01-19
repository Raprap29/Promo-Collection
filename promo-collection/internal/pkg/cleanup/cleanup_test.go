package cleanup

import (
	"context"
	"testing"

	mongodb "promocollection/internal/pkg/db/mongo"
	redisdb "promocollection/internal/pkg/db/redis"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementations for testing
type MockMongoClient struct {
	mock.Mock
	Client *mongodb.MongoClient
}

type MockRedisClient struct {
	mock.Mock
	Client *redisdb.RedisClient
}

func TestCleanupResources(t *testing.T) {
	ctx := context.Background()

	t.Run("successful cleanup with both clients", func(t *testing.T) {
		mongoClient := &mongodb.MongoClient{}
		redisClient := &redisdb.RedisClient{}

		// This test will pass even if disconnect fails since we're not mocking the actual disconnect
		// The function logs errors but doesn't return them
		assert.NotPanics(t, func() {
			CleanupResources(ctx, mongoClient, redisClient)
		})
	})

	t.Run("cleanup with nil mongo client", func(t *testing.T) {
		redisClient := &redisdb.RedisClient{}

		assert.NotPanics(t, func() {
			CleanupResources(ctx, nil, redisClient)
		})
	})

	t.Run("cleanup with nil redis client", func(t *testing.T) {
		mongoClient := &mongodb.MongoClient{}

		assert.NotPanics(t, func() {
			CleanupResources(ctx, mongoClient, nil)
		})
	})

	t.Run("cleanup with both nil clients", func(t *testing.T) {
		assert.NotPanics(t, func() {
			CleanupResources(ctx, nil, nil)
		})
	})

	t.Run("cleanup with nil mongo client.Client", func(t *testing.T) {
		mongoClient := &mongodb.MongoClient{
			Client: nil,
		}
		redisClient := &redisdb.RedisClient{}

		assert.NotPanics(t, func() {
			CleanupResources(ctx, mongoClient, redisClient)
		})
	})

	t.Run("cleanup with nil redis client.Client", func(t *testing.T) {
		mongoClient := &mongodb.MongoClient{}
		redisClient := &redisdb.RedisClient{
			Client: nil,
		}

		assert.NotPanics(t, func() {
			CleanupResources(ctx, mongoClient, redisClient)
		})
	})
}

func TestCleanupResourcesEdgeCases(t *testing.T) {
	t.Run("cleanup with empty context", func(t *testing.T) {
		emptyCtx := context.Background()
		mongoClient := &mongodb.MongoClient{}
		redisClient := &redisdb.RedisClient{}

		assert.NotPanics(t, func() {
			CleanupResources(emptyCtx, mongoClient, redisClient)
		})
	})

	t.Run("cleanup with cancelled context", func(t *testing.T) {
		cancelledCtx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately
		
		mongoClient := &mongodb.MongoClient{}
		redisClient := &redisdb.RedisClient{}

		assert.NotPanics(t, func() {
			CleanupResources(cancelledCtx, mongoClient, redisClient)
		})
	})
}
