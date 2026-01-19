package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"promocollection/internal/app/router"
	"promocollection/internal/pkg/cleanup"
	config "promocollection/internal/pkg/config"
	mongodb "promocollection/internal/pkg/db/mongo"
	redisdb "promocollection/internal/pkg/db/redis"
	kafkaConsumer "promocollection/internal/pkg/kafka/consumer"
	"promocollection/internal/pkg/log_messages"
	"promocollection/internal/pkg/logger"
	"promocollection/internal/pkg/pubsub"

	gcppubsub "cloud.google.com/go/pubsub"
)

func main() {

	ctx := context.Background()

	logger.Init()

	cfg, err := config.LoadFromConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Connect to MongoDB
	mongoClient, err := mongodb.ConnectToMongoDB(ctx, cfg.Mongo)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// Connect to Redis
	redisClient, err := redisdb.ConnectToRedis(ctx, cfg.Redis, nil)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// kafka consumer
	kafkaConsumer, err := kafkaConsumer.NewKafkaConsumer(cfg.Kafka)
	if err != nil {
		log.Fatalf("Failed to create kafka consumer: %v", err)
	}

	logger.Debug("Kafka Consumer Created")

	// kafka subscribe
	err = kafkaConsumer.Subscribe(cfg.Kafka.PromoTopic)
	if err != nil {
		logger.CtxError(ctx, "failed to create Kafka Subscribe ", err)
		return
	}
	// pubsub publisher for promo-collection-worker
	pubsubclient, err := initPubSubClient(ctx, cfg.PubSub.ProjectID, cfg.PubSub.Topic, "(1/2)")
	if err != nil {
		log.Fatalf("Failed to create Pub/Sub client: %v", err)
	}

	// pubsub publisher for notification-service
	pubsubnotifclient, err := initPubSubClient(ctx, cfg.PubSub.ProjectID, cfg.PubSub.NotificationTopic, "(2/2)")
	if err != nil {
		log.Fatalf("Failed to create Pub/Sub client: %v", err)
	}

	server := router.SetupRouter(ctx, kafkaConsumer, pubsubclient, pubsubnotifclient, mongoClient, redisClient.Client)
	port := cfg.Server.Port

	if err := server.Run(":" + fmt.Sprintf("%d", port)); err != nil {
		logger.CtxError(ctx, "Failed to start server", err)
	}

	defer cleanup.CleanupResources(ctx, mongoClient, redisClient)

}

func initPubSubClient(ctx context.Context, projectID, topic, label string) (*pubsub.PubSubClient, error) {
	client, err := pubsub.NewPubSubClient(ctx, projectID, topic, gcppubsub.NewClient)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", log_messages.ErrorPubSubClientCreation, err)
	}

	logger.Info("successful pubsub client creation "+label,
		slog.String("pubsub_topic", topic),
		slog.Any("pubsub_client", client),
	)

	return client, nil
}
