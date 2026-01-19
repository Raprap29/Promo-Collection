package router

import (
	"context"
	"promo-collection-worker/internal/app/handlers"
	"promo-collection-worker/internal/pkg/config"
	"promo-collection-worker/internal/pkg/db/mongo"
	"promo-collection-worker/internal/pkg/kafka"
	"promo-collection-worker/internal/service"

	"github.com/gin-gonic/gin"
)

func SetupRouter(ctx context.Context, mongoClient *mongo.MongoClient,
	kafkaProducer *kafka.KafkaProducer, cfg config.KafkaRetryServiceConfig) *gin.Engine {
	server := gin.Default()

	healthCheckHandler := handlers.NewHealthCheckHandler()
	server.GET("/IntegrationServices/Dodrio/PromoCollectionWorkerService/HealthCheck", healthCheckHandler.HealthCheck)

	kafkaRetryService := service.NewKafkaRetryService(mongoClient, kafkaProducer, cfg)
	kafkaRetryHandler := handlers.NewKafkaRetryHandler(kafkaRetryService)
	server.GET("/IntegrationServices/Dodrio/KafkaRetry", kafkaRetryHandler.RetryKafkaCollectionMessages)

	return server
}
