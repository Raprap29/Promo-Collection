package runtime

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"promo-collection-worker/internal/app/router"
	"promo-collection-worker/internal/pkg/cleanup"
	"promo-collection-worker/internal/pkg/config"
	"promo-collection-worker/internal/pkg/db/mongo"
	"promo-collection-worker/internal/pkg/db/redis"
	"promo-collection-worker/internal/pkg/gcs"
	"promo-collection-worker/internal/pkg/kafka"

	servicekafka "promo-collection-worker/internal/service/kafka"
	successhandling_service "promo-collection-worker/internal/service/success_handling"

	"promo-collection-worker/internal/pkg/log_messages"
	"promo-collection-worker/internal/pkg/logger"
	"promo-collection-worker/internal/pkg/pubsub"
	pubsubService "promo-collection-worker/internal/service/pubsub"
)

var (
	connectMongoDB = mongo.ConnectToMongoDB
	connectRedisDB = func(ctx context.Context, cfg config.RedisConfig) (*redis.RedisClient, error) {
		return redis.ConnectToRedis(ctx, cfg, nil)
	}
	newKafkaProducer = kafka.NewKafkaProducer
)

// PubSub defines the contract for any PubSub consumer
type PubSubConsumer interface {
	Close() error
	Consume(ctx context.Context, sub string, handler func(ctx context.Context, msg []byte) error) error
	StartConsumer(subscription string, handler func(ctx context.Context, msg []byte) error)
}

// PubSubPublisher defines the contract for any PubSub publisher
type PubSubPublisher interface {
	Close() error
	Publish(ctx context.Context, topic string, msg []byte) error
}

// App encapsulates application resources and lifecycle.
type App struct {
	Cfg             *config.AppConfig
	PubSubConsumer  PubSubConsumer
	PubSubPublisher PubSubPublisher
	KafkaProducer   *kafka.KafkaProducer
	KafkaService    *servicekafka.CollectionWorkerKafkaService
	MongoClient     *mongo.MongoClient
	RedisClient     *redis.RedisClient
	HTTPServer      *http.Server
	GcsClient       gcs.GcsInterface
}

func New(ctx context.Context) (*App, error) {
	cfg, err := config.LoadFromConfig()
	if err != nil {
		logger.CtxError(ctx, log_messages.FailedLoadingConfiguration, err)
		return nil, err
	}
	logger.Init(cfg.Logging.LogLevel)

	pubsubConsumer, err := pubsub.NewPubSubConsumer(ctx, cfg.PubSub.ProjectID)
	if err != nil {
		logger.CtxError(ctx, log_messages.FailureInPubsubConsumerCreation, err)
		return nil, err
	}

	pubsubPublisher, err := pubsub.NewPubSubPublisher(ctx, cfg.PubSub.ProjectID)
	if err != nil {
		logger.CtxError(ctx, "Failure in PubSub publisher creation", err)
		return nil, err
	}

	kafkaProducer, err := newKafkaProducer(cfg.Kafka)
	if err != nil {
		logger.CtxError(ctx, "Failure in Kafka producer creation", err)
		return nil, err
	}

	// Wrap kafkaProducer with the service used by the rest of the app
	kafkaService := servicekafka.NewCollectionWorkerKafkaService(kafkaProducer)

	mClient, err := connectMongoDB(ctx, cfg.Mongo)
	if err != nil {
		logger.CtxError(ctx, "Failed to connect to MongoDB", err)
		return nil, err
	}

	rClient, err := connectRedisDB(ctx, cfg.Redis)
	if err != nil {
		logger.CtxError(ctx, "Failed to connect to Redis", err)
		return nil, err
	}

	gcsClient, err := gcs.NewGCSClient(ctx, cfg.GCS.BucketName)
	if err != nil {
		logger.CtxError(ctx, "Failed to create GCS client", err)
		return nil, err
	}

	return &App{
		Cfg:             cfg,
		PubSubConsumer:  pubsubConsumer,
		PubSubPublisher: pubsubPublisher,
		KafkaProducer:   kafkaProducer,
		KafkaService:    kafkaService,
		MongoClient:     mClient,
		RedisClient:     rClient,
		GcsClient:       gcsClient,
	}, nil
}

// Run starts the PubSub consumer and HTTP server, then blocks until shutdown.
func (a *App) Run(ctx context.Context) error {
	// Start PubSub consumer
	pubSubConsumerService := pubsubService.NewPromoCollectionMessageConsumer(
		a.Cfg.HIP,
		a.MongoClient,
		a.PubSubPublisher,
		a.KafkaService,
		a.RedisClient,
		successhandling_service.PartialDataDeductionService,
		successhandling_service.FullDataDeductionService,
		a.Cfg.PubSub.NotificationTopic,
		a.GcsClient,
	)
	go a.PubSubConsumer.StartConsumer(a.Cfg.PubSub.Subscription, pubSubConsumerService.HandlePromoCollectionMessage)

	// Start HTTP server
	engine := router.SetupRouter(ctx, a.MongoClient, a.KafkaProducer, a.Cfg.KafkaRetryService)
	a.HTTPServer = &http.Server{
		Addr:              fmt.Sprintf(":%d", a.Cfg.Server.Port),
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := a.HTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.CtxError(ctx, log_messages.ServerStartFailure, err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	a.Shutdown(ctx)
	logger.CtxInfo(ctx, log_messages.ServerExiting)
	return nil
}

// Shutdown gracefully closes all resources with bounded timeouts.
func (a *App) Shutdown(ctx context.Context) {
	cleanup.CleanupResources(ctx,
		a.PubSubConsumer,
		a.PubSubPublisher,
		a.KafkaProducer,
		a.MongoClient,
		a.RedisClient,
		a.HTTPServer,
		a.GcsClient,
	)
}
