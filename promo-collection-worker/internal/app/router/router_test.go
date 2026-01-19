package router

import (
	"context"
	"testing"

	"promo-collection-worker/internal/pkg/config"
	"promo-collection-worker/internal/pkg/db/mongo"
	"promo-collection-worker/internal/pkg/kafka"

	"github.com/stretchr/testify/assert"
)

func TestSetupRouterFunctionExists(t *testing.T) {
	ctx := context.Background()
	cfg := config.KafkaRetryServiceConfig{
		RetryStartDate: "2025-01-01",
		WorkerCount:    5,
		BufferSize:     10,
		MaxBatchSize:   20,
		MongoBatchSize: 100,
	}
	mongoClient := &mongo.MongoClient{}
	kafkaProducer := &kafka.KafkaProducer{}

	assert.Panics(t, func() {
		SetupRouter(ctx, mongoClient, kafkaProducer, cfg)
	}, "SetupRouter should panic due to database dependencies in test environment")
}
