package router

import (
	"context"
	"promocollection/internal/app/handlers"
	mongodb "promocollection/internal/pkg/db/mongo"
	kafkaConsumer "promocollection/internal/pkg/kafka/consumer"
	"promocollection/internal/pkg/logger"
	"promocollection/internal/pkg/pubsub"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func SetupRouter(ctx context.Context,
	promoKafkaConsumer *kafkaConsumer.KafkaConsumer,
	pubsub, pubsubnotifclient *pubsub.PubSubClient,
	mongoClient *mongodb.MongoClient,
	redisClient *redis.Client) *gin.Engine {
	server := gin.Default()
	go func() {

		promoHandler := handlers.NewPromoHandler(ctx, promoKafkaConsumer)

		err := promoHandler.PromoKafkaConsumer(ctx, promoKafkaConsumer, mongoClient, redisClient, pubsub, pubsubnotifclient)
		if err != nil {
			logger.CtxError(ctx, "failed to start Kafka consumer", err)
		}
	}()

	healthCheckHandler := handlers.NewHealthCheckHandler()
	server.GET("/IntegrationServices/Dodrio/PromoCollection/HealthCheck", healthCheckHandler.HealthCheck)

	return server
}
