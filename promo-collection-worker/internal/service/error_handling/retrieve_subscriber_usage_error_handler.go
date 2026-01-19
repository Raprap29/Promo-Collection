package error_handling_service

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"promo-collection-worker/internal/pkg/common"
	"promo-collection-worker/internal/pkg/consts"
	mongodb "promo-collection-worker/internal/pkg/db/mongo"
	redis "promo-collection-worker/internal/pkg/db/redis"
	"promo-collection-worker/internal/pkg/gcs"
	"promo-collection-worker/internal/pkg/log_messages"
	"promo-collection-worker/internal/pkg/logger"
	"promo-collection-worker/internal/pkg/models"
	serviceinterfaces "promo-collection-worker/internal/service/interfaces"
	servicekafka "promo-collection-worker/internal/service/kafka"
)

type RetrieveSubscriberUsageErrorHandlerInterface interface {
	HandleError(ctx context.Context, msg *models.PromoCollectionPublishedMessage, err error, statusCode ...int) error
	HandleUnknownOrNonAPIError(ctx context.Context, msg *models.PromoCollectionPublishedMessage, err error) string
	CheckAndProcessBalance(ctx context.Context, msg *models.PromoCollectionPublishedMessage, bucket models.Bucket) error
}

type RetrieveSubscriberUsageErrorHandler struct {
	CollectionsRepo                  serviceinterfaces.CollectionsRepoInterface
	CollectionTransactionsInProgress serviceinterfaces.CollectionTransactionsInProgressRepoInterface
	KafkaService                     *servicekafka.CollectionWorkerKafkaService
	// Runtime resources needed for success path
	MongoClient     *mongodb.MongoClient
	RedisClient     *redis.RedisClient
	PubSubPublisher serviceinterfaces.RuntimePubSubPublisher
	// Callbacks to invoke success handling without importing the success_handling package
	PartialSuccessHandler serviceinterfaces.SuccessHandlerFunc
	FullSuccessHandler    serviceinterfaces.SuccessHandlerFunc
	NotificationTopic     string
	GcsClient             gcs.GcsInterface
}

// NewRetrieveSubscriberUsageErrorHandler creates a new error handler for RetrieveSubscriberUsage API
func NewRetrieveSubscriberUsageErrorHandler(collectionsRepo serviceinterfaces.CollectionsRepoInterface,
	collectionTransactionsInProgress serviceinterfaces.
		CollectionTransactionsInProgressRepoInterface,
	kafkaService *servicekafka.CollectionWorkerKafkaService,
	mongoClient *mongodb.MongoClient,
	redisClient *redis.RedisClient,
	pubsubPublisher serviceinterfaces.RuntimePubSubPublisher,
	partialHandler serviceinterfaces.SuccessHandlerFunc,
	fullHandler serviceinterfaces.SuccessHandlerFunc,
	notificationTopic string,
	gcsClient gcs.GcsInterface,
) *RetrieveSubscriberUsageErrorHandler {
	return &RetrieveSubscriberUsageErrorHandler{
		CollectionsRepo:                  collectionsRepo,
		CollectionTransactionsInProgress: collectionTransactionsInProgress,
		KafkaService:                     kafkaService,
		MongoClient:                      mongoClient,
		RedisClient:                      redisClient,
		PubSubPublisher:                  pubsubPublisher,
		PartialSuccessHandler:            partialHandler,
		FullSuccessHandler:               fullHandler,
		NotificationTopic:                notificationTopic,
		GcsClient:                        gcsClient,
	}
}

// HandleError processes errors from RetrieveSubscriberUsage API calls
// and creates appropriate collection records
func (h *RetrieveSubscriberUsageErrorHandler) HandleError(
	ctx context.Context,
	msg *models.PromoCollectionPublishedMessage,
	err error,
	statusCode ...int,
) error {
	logger.CtxError(ctx, log_messages.ErrorRetrieveSubscriberUsage, err, slog.Time("publish_time", msg.PublishTime))

	var statusCodeInt int
	if len(statusCode) > 0 {
		statusCodeInt = statusCode[0]
	}

	var errorCode string
	var errorMessage string

	if statusCodeInt == http.StatusOK {
		errorCode = log_messages.ErrorCodeMissingSubscriberWalletBucket
		errorMessage = log_messages.ErrorMessageMissingSubscriberWalletBucket
	}

	if statusCodeInt == http.StatusOK {
		collectionEntry := common.SerializeCollections(msg, false, errorMessage, errorCode)
		if _, err := h.CollectionsRepo.CreateEntry(ctx, collectionEntry); err != nil {
			logger.CtxError(ctx, log_messages.ErrorFailedToCreateCollectionEntry, err)
		}

		if err := h.CollectionTransactionsInProgress.DeleteEntry(ctx, msg.Msisdn); err != nil {
			logger.CtxError(ctx, log_messages.ErrorFailedToDeleteTransactionInProgress, err,
				slog.String("msisdn", msg.Msisdn), slog.Time("publish_time", msg.PublishTime))
		}

		publishKafkaFailureIfConfigured(ctx,
			h.KafkaService,
			msg,
			errorMessage,
			collectionEntry.CreatedAt,
			h.CollectionsRepo,
			collectionEntry.ID,
		)
	} else if h.GcsClient != nil {
		if err := h.GcsClient.Upload(ctx, msg); err != nil {
			logger.CtxError(ctx, log_messages.ErrorUploadingToGCSBucket, err, slog.Time("publish_time", msg.PublishTime))
		}
	}

	// Always ACK the message since we've recorded the error
	return nil
}

// Check if the balance is sufficient and process accordingly
func (h *RetrieveSubscriberUsageErrorHandler) CheckAndProcessBalance(
	ctx context.Context,
	msg *models.PromoCollectionPublishedMessage,
	bucket models.Bucket,
) error {
	// Convert volume remaining from KB to MB
	volumeRemainingMB := float64(bucket.VolumeRemaining) / 1024.0
	walletAmount, err := strconv.ParseFloat(msg.WalletAmount, 64)
	if err != nil {
		logger.CtxError(ctx, log_messages.ErrorFailedToParseWalletAmount, err, slog.Time("publish_time", msg.PublishTime))
		return h.HandleError(ctx, msg, nil, 200)
	}
	if walletAmount-volumeRemainingMB >= msg.DataToBeDeducted {
		// call the injected success handler depending on collection type
		switch msg.CollectionType {
		case consts.PartialCollectionType:
			if err := h.PartialSuccessHandler(
				h.MongoClient,
				h.RedisClient,
				h.PubSubPublisher,
				h.KafkaService,
				msg,
				ctx,
				h.NotificationTopic,
			); err != nil {
				logger.CtxError(ctx, "Partial data collection returned error", err, slog.Time("publish_time", msg.PublishTime))
				return err
			}
			logger.CtxInfo(ctx, log_messages.SuccessPartialCollection,
				slog.String("MSISDN", msg.Msisdn), slog.Time("publish_time",
					msg.PublishTime))
		case consts.FullCollectionType:
			if err := h.FullSuccessHandler(
				h.MongoClient,
				h.RedisClient,
				h.PubSubPublisher,
				h.KafkaService,
				msg,
				ctx,
				h.NotificationTopic,
			); err != nil {
				logger.CtxError(ctx, "Full data collection returned error", err, slog.Time("publish_time", msg.PublishTime))
				return err
			}
			logger.CtxInfo(ctx, log_messages.SuccessFullCollection,
				slog.String("MSISDN", msg.Msisdn), slog.Time("publish_time",
					msg.PublishTime))
		default:
			logger.CtxError(ctx, "Unknown collection type for success handling", nil,
				slog.String("collectionType", msg.CollectionType))
		}
		return nil
	} else {
		logger.CtxError(ctx, log_messages.ErrorInsufficientBalanceForDeduction, nil,
			slog.String("msisdn", msg.Msisdn),
			slog.Float64("volumeRemainingMB", volumeRemainingMB),
			slog.Float64("dataToDeduct", msg.DataToBeDeducted),
			slog.Time("publish_time", msg.PublishTime))
		return fmt.Errorf(log_messages.ErrorDeductionDidntHappenInsufficientBalance)
	}
}
