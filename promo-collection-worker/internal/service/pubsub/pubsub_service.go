package pubsub_service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"promo-collection-worker/internal/pkg/config"
	"promo-collection-worker/internal/pkg/consts"
	mongodb "promo-collection-worker/internal/pkg/db/mongo"
	"promo-collection-worker/internal/pkg/db/redis"
	"promo-collection-worker/internal/pkg/downstream"
	error_handling "promo-collection-worker/internal/pkg/downstream/error_handling"
	"promo-collection-worker/internal/pkg/gcs"
	"promo-collection-worker/internal/pkg/log_messages"
	"promo-collection-worker/internal/pkg/logger"
	"promo-collection-worker/internal/pkg/models"
	"promo-collection-worker/internal/pkg/store/impl/collection_transaction_in_progress"
	"promo-collection-worker/internal/pkg/store/impl/collections"
	error_handling_service "promo-collection-worker/internal/service/error_handling"
	serviceinterfaces "promo-collection-worker/internal/service/interfaces"
	servicekafka "promo-collection-worker/internal/service/kafka"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate = validator.New()

// MessageIgnoreError is a special error type that signals the PubSub consumer
// to neither ACK nor NACK the message, effectively letting it be redelivered
// after the redelivery timeout
type MessageIgnoreError struct {
	Err error
}

func (e *MessageIgnoreError) Error() string {
	return fmt.Sprintf("message ignored for redelivery: %v", e.Err)
}

type PromoCollectionMessageConsumerInterface interface {
	RequestWallet(ctx context.Context, req *models.PromoCollectionPublishedMessage) (string, error)
	HandlePromoCollectionMessage(ctx context.Context, msg []byte) error
}

type PromoCollectionMessageConsumer struct {
	decrementSubscriptionClient         downstream.DecrementWalletSubscriptionAPI
	decrementWalletErrorHandler         *error_handling_service.DecrementWalletErrorHandler
	retrieveSubscriberUsageErrorHandler *error_handling_service.RetrieveSubscriberUsageErrorHandler
	retrieveSubscriberUsageClient       downstream.RetrieveSubscriberUsageAPI
	// Additional dependencies passed from runtime
	MongoClient           *mongodb.MongoClient
	RedisClient           *redis.RedisClient
	PubSubPublisher       serviceinterfaces.RuntimePubSubPublisher
	KafkaService          *servicekafka.CollectionWorkerKafkaService
	PartialSuccessHandler serviceinterfaces.SuccessHandlerFunc
	FullSuccessHandler    serviceinterfaces.SuccessHandlerFunc
	NotificationTopic     string
	GcsClient             gcs.GcsInterface
}

// NewPromoCollectionMessageConsumer creates a PubSub consumer service with a real downstream client.
func NewPromoCollectionMessageConsumer(cfg config.HIPConfig,
	mongoClient *mongodb.MongoClient,
	pubSubPublisher serviceinterfaces.RuntimePubSubPublisher,
	kafkaService *servicekafka.CollectionWorkerKafkaService,
	redisClient *redis.RedisClient,
	partialHandler serviceinterfaces.SuccessHandlerFunc,
	fullHandler serviceinterfaces.SuccessHandlerFunc,
	notificationTopic string,
	gcsClient gcs.GcsInterface,
) PromoCollectionMessageConsumerInterface {
	collectionsRepo := collections.NewCollectionsRepository(mongoClient)
	collectionTransactionsInProgressRepo := collection_transaction_in_progress.
		NewCollectionTransactionsInProgressRepository(mongoClient)

	// Initialize the error handlers
	decrementWalletErrorHandler := &error_handling_service.DecrementWalletErrorHandler{
		CollectionsRepo:                  collectionsRepo,
		CollectionTransactionsInProgress: collectionTransactionsInProgressRepo,
		KafkaService:                     kafkaService,
	}

	// Create retrieve subscriber usage error handler with callbacks to success handlers
	retrieveSubscriberUsageErrorHandler := error_handling_service.NewRetrieveSubscriberUsageErrorHandler(
		collectionsRepo,
		collectionTransactionsInProgressRepo,
		kafkaService,
		mongoClient,
		redisClient,
		pubSubPublisher,
		partialHandler,
		fullHandler,
		notificationTopic,
		gcsClient,
	)

	return &PromoCollectionMessageConsumer{
		decrementSubscriptionClient:         downstream.NewDecrementWalletSubscriptionClient(cfg),
		retrieveSubscriberUsageClient:       downstream.NewRetrieveSubscriberUsageClient(cfg),
		decrementWalletErrorHandler:         decrementWalletErrorHandler,
		retrieveSubscriberUsageErrorHandler: retrieveSubscriberUsageErrorHandler,
		MongoClient:                         mongoClient,
		RedisClient:                         redisClient,
		PubSubPublisher:                     pubSubPublisher,
		KafkaService:                        kafkaService,
		PartialSuccessHandler:               partialHandler,
		FullSuccessHandler:                  fullHandler,
		NotificationTopic:                   notificationTopic,
		GcsClient:                           gcsClient,
	}
}

// NewPromoCollectionMessageConsumerWithClient allows injecting a mock client for testing.
func NewPromoCollectionMessageConsumerWithClient(
	decrementSubscriptionClient downstream.DecrementWalletSubscriptionAPI,
	retrieveSubscriberUsageClient downstream.RetrieveSubscriberUsageAPI,
	mongoClient *mongodb.MongoClient) PromoCollectionMessageConsumerInterface {

	// Initialize repositories
	collectionsRepo := collections.NewCollectionsRepository(mongoClient)
	collectionTransactionsInProgressRepo := collection_transaction_in_progress.
		NewCollectionTransactionsInProgressRepository(mongoClient)

	// Initialize the error handlers
	decrementWalletErrorHandler := &error_handling_service.DecrementWalletErrorHandler{
		CollectionsRepo:                  collectionsRepo,
		CollectionTransactionsInProgress: collectionTransactionsInProgressRepo,
	}

	retrieveSubscriberUsageErrorHandler := &error_handling_service.RetrieveSubscriberUsageErrorHandler{
		CollectionsRepo:                  collectionsRepo,
		CollectionTransactionsInProgress: collectionTransactionsInProgressRepo,
	}

	return &PromoCollectionMessageConsumer{
		decrementSubscriptionClient:         decrementSubscriptionClient,
		retrieveSubscriberUsageClient:       retrieveSubscriberUsageClient,
		decrementWalletErrorHandler:         decrementWalletErrorHandler,
		retrieveSubscriberUsageErrorHandler: retrieveSubscriberUsageErrorHandler,
	}
}

// HandleMessage processes incoming PubSub messages
func (c *PromoCollectionMessageConsumer) HandlePromoCollectionMessage(ctx context.Context, msg []byte) error {
	logger.CtxInfo(ctx, "Handling PubSub message", slog.String("payload", string(msg)))

	var event models.PromoCollectionPublishedMessage
	if err := json.Unmarshal(msg, &event); err != nil {
		logger.CtxError(ctx, "Failed to unmarshal PubSub message", err)
		return err
	}

	// Extract publish time from context (use the shared key from models)
	if publishTime, ok := ctx.Value(models.PublishTimeKey).(time.Time); ok {
		event.PublishTime = publishTime
	}

	// Validate the event
	if err := validate.Struct(event); err != nil {
		logger.CtxError(ctx, "Validation failed for PubSub message", err)
		return err
	}

	// Set the DataCollectionRequestTraceId in context for tracing
	ctx = logger.WithTraceID(ctx, event.DataCollectionRequestTraceId)

	// Log important fields
	logger.CtxInfo(ctx, "Event details", slog.String("event", event.String()))

	// Delegate to decrementWalletSubscription.
	action, err := c.RequestWallet(ctx, &event)
	return c.handleActionResult(action, err)
}

func (p *PromoCollectionMessageConsumer) RequestWallet(
	ctx context.Context,
	msg *models.PromoCollectionPublishedMessage,
) (string, error) {
	req := mapToDecrementWalletSubscriptionRequest(msg)
	okResp, err := p.decrementSubscriptionClient.DecrementWalletSubscription(ctx, req)

	if err != nil {
		logger.CtxError(ctx, log_messages.ErrorDecrementWalletSubscription, err,
			slog.String("subscriber", msg.Msisdn+"|"+msg.WalletKeyword))
		return p.handleDecrementError(ctx, msg, err)
	}

	// Handle success response: call the configured success handler callbacks
	switch msg.CollectionType {
	case consts.PartialCollectionType:
		if err := p.PartialSuccessHandler(
			p.MongoClient,
			p.RedisClient,
			p.PubSubPublisher,
			p.KafkaService,
			msg,
			ctx,
			p.NotificationTopic,
		); err != nil {
			logger.CtxError(ctx, "Partial data collection returned error", err)
			return consts.ActionAck, err
		}
		logger.CtxInfo(ctx, log_messages.SuccessPartialCollection, slog.String("MSISDN", msg.Msisdn),
			slog.Time("publish_time", msg.PublishTime))
	case consts.FullCollectionType:
		if err := p.FullSuccessHandler(
			p.MongoClient,
			p.RedisClient,
			p.PubSubPublisher,
			p.KafkaService,
			msg,
			ctx,
			p.NotificationTopic,
		); err != nil {
			logger.CtxError(ctx, "Full data collection returned error", err)
			return consts.ActionAck, err
		}
		logger.CtxInfo(ctx, log_messages.SuccessFullCollection, slog.String("MSISDN", msg.Msisdn),
			slog.Time("publish_time", msg.PublishTime))
	}
	logger.CtxInfo(ctx, "DecrementWalletSubscription success",
		slog.String("subscriber", msg.Msisdn+"|"+msg.WalletKeyword),
		slog.String("details", "resultCode="+okResp.ResultCode+", description="+okResp.Description),
	)

	return consts.ActionAck, nil
}

// Handle errors from the decrement wallet subscription API
func (p *PromoCollectionMessageConsumer) handleDecrementError(
	ctx context.Context,
	msg *models.PromoCollectionPublishedMessage,
	err error,
) (string, error) {
	var errResp *error_handling.DecrementWalletSubscriptionError
	if (!errors.As(err, &errResp) || errResp.IsNonAPIError()) && errResp.StatusCode != -1 {
		// If the error doesn't match our expected type or non api error
		action := p.decrementWalletErrorHandler.HandleUnknownOrNonAPIError(ctx, msg, err)
		return action, err
	}

	// Handle timeout/no response case
	if errResp.StatusCode == -1 {
		return p.handleNoResponseCase(ctx, msg)
	}

	// Handle API errors with status codes
	action := p.decrementWalletErrorHandler.DecrementWalletErrorHandler(ctx, msg, errResp)
	return action, err
}

// Handle the case when there's no response from the decrement API
func (p *PromoCollectionMessageConsumer) handleNoResponseCase(
	ctx context.Context,
	msg *models.PromoCollectionPublishedMessage,
) (string, error) {
	req := mapToRetrieveSubscriberUsageRequest(msg.Msisdn)
	resp, err := p.retrieveSubscriberUsageClient.RetrieveSubscriberUsage(ctx, req)

	if err != nil {
		err := p.retrieveSubscriberUsageErrorHandler.HandleError(ctx, msg, err)
		return consts.ActionAck, err
	}

	// Find matching bucket
	matchingBucket, err := downstream.FindMatchingBucketId(ctx, resp, msg.WalletKeyword)
	if err != nil {
		err := p.retrieveSubscriberUsageErrorHandler.HandleError(ctx, msg, nil, 200)
		return consts.ActionAck, err
	}

	err = p.retrieveSubscriberUsageErrorHandler.CheckAndProcessBalance(ctx, msg, matchingBucket)

	// Check if the error is specifically about insufficient balance
	if err != nil && err.Error() == log_messages.ErrorDeductionDidntHappenInsufficientBalance {
		return consts.ActionNack, err
	}

	return consts.ActionAck, err
}

// Map action to appropriate return value
func (p *PromoCollectionMessageConsumer) handleActionResult(action string, originalErr error) error {
	switch action {
	case consts.ActionAck:
		return nil
	case consts.ActionNack:
		return originalErr
	case consts.ActionIgnore:
		return &MessageIgnoreError{Err: originalErr}
	default:
		return nil
	}
}

func mapToDecrementWalletSubscriptionRequest(msg *models.
	PromoCollectionPublishedMessage) *models.DecrementWalletSubscriptionRequest {
	// Convert MB to KB by multiplying by 1000
	amountInKB := fmt.Sprintf("%.0f", msg.DataToBeDeducted*1000)
	return &models.DecrementWalletSubscriptionRequest{
		MSISDN:     msg.Msisdn,
		Wallet:     msg.WalletKeyword,
		Amount:     amountInKB,
		Unit:       consts.Unit,
		IsRollBack: msg.IsRollBack,
		Duration:   msg.Duration,
		Channel:    msg.Channel,
	}
}

func mapToRetrieveSubscriberUsageRequest(msisdn string) *models.RetrieveSubscriberUsageRequest {
	return &models.RetrieveSubscriberUsageRequest{
		MSISDN:           msisdn,
		IdType:           consts.IdType,
		QueryType:        consts.QueryType,
		DetailsFlag:      consts.DetailsFlag,
		SubscriptionType: consts.SubscriberType,
	}
}
