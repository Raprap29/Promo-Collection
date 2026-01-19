package main

import (
	"context"
	"globe/dodrio_loan_availment/configs"
	"globe/dodrio_loan_availment/internal/app/router"
	"globe/dodrio_loan_availment/internal/pkg/db"
	"globe/dodrio_loan_availment/internal/pkg/kafka/producer"
	"globe/dodrio_loan_availment/internal/pkg/logger"
	"globe/dodrio_loan_availment/internal/pkg/otel"
	"globe/dodrio_loan_availment/internal/pkg/pubsub"
	"globe/dodrio_loan_availment/internal/pkg/redis"
	"globe/dodrio_loan_availment/internal/pkg/utils/worker"
	"log"
	"strconv"
)

func main() {

	// Load Environment Variables
	err := configs.LoadEnv()
	if err != nil {
		logger.Debug("Error loading .env file: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//setup otel collector
	_, err = otel.Setup(ctx, configs.SERVICE_NAME, configs.OTEL_URL)
	if err != nil {
		logger.Error(ctx, "Error setting up OTLP: %v", err)
	}

	// DB Connection
	mdb, dbErr := db.NewMongoDB()
	if dbErr != nil {
		logger.Error(ctx, "Error connecting to MongoDB: %v", dbErr)
	}
	db.MDB = mdb
	defer mdb.Close()

	db.CreateDbTtlIfNotExists()

	topic := configs.KAFKA_TOPIC
	server := configs.KAFKA_SERVER
	kafkaProducer, err := producer.NewKafkaProducer(server, topic)
	if err != nil {
		logger.Error(ctx, "failed to create Kafka Producer error: %v", err)
	}
	logger.Info(ctx, "Kafka Producer Created")
	producer.KafkaProducer = kafkaProducer
	defer kafkaProducer.Close()

	// Initialize Pub/Sub Publisher
	pubsubPublisher, err := pubsub.NewPubSubPublisher(ctx, configs.PROJECT_ID) // Add PROJECT_ID to your configs
	if err != nil {
		logger.Error(ctx, "Failed to create Pub/Sub Publisher: %v", err)
	}
	defer pubsubPublisher.Close()
	logger.Info(ctx, "Pub/Sub Publisher Created")

	numberOfWorkers, er := strconv.Atoi(configs.WORKER_POOL)
	if er != nil {
		logger.Error(ctx, er)
	}

	// Connect to Redis
	redisClient, err := redis.ConnectToRedis(ctx, configs.GetRedisConfig(), nil)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	workerPool := worker.NewWorkerPool(numberOfWorkers)
	defer workerPool.Stop()

	r := router.SetupRouter(workerPool, redisClient.Client, pubsubPublisher)

	port := configs.SERVER_PORT

	if err := r.Run(":" + port); err != nil {
		logger.Error(ctx, "Failed to run server: %v", err)
	}
}
