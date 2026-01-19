package error_handling_service

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"promo-collection-worker/internal/pkg/common"
	"promo-collection-worker/internal/pkg/consts"
	"promo-collection-worker/internal/pkg/downstream/error_handling"
	"promo-collection-worker/internal/pkg/log_messages"
	"promo-collection-worker/internal/pkg/logger"
	"promo-collection-worker/internal/pkg/models"
	"promo-collection-worker/internal/service/interfaces"
	servicekafka "promo-collection-worker/internal/service/kafka"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ErrorHandler defines the interface for handling wallet decrement errors
type ErrorHandler interface {
	DecrementWalletErrorHandler(ctx context.Context, msg *models.PromoCollectionPublishedMessage,
		err *error_handling.DecrementWalletSubscriptionError) string
}

// DecrementWalletErrorHandler handles wallet decrement errors
type DecrementWalletErrorHandler struct {
	CollectionsRepo                  interfaces.CollectionsRepoInterface
	CollectionTransactionsInProgress interfaces.CollectionTransactionsInProgressRepoInterface
	KafkaService                     *servicekafka.CollectionWorkerKafkaService
}

// DecrementWalletErrorHandler processes errors from DecrementWalletSubscription API
// and creates appropriate collection records based on error status codes
// Returns an action string: consts.ActionAck, consts.ActionNack, or consts.ActionIgnore
func (h *DecrementWalletErrorHandler) DecrementWalletErrorHandler(
	ctx context.Context,
	msg *models.PromoCollectionPublishedMessage,
	err *error_handling.DecrementWalletSubscriptionError,
) string {
	var errorCodeInt int
	var errorCodeString string
	var errorText string

	if err.StatusCode != http.StatusOK {
		errorCodeInt, _ = strconv.Atoi(err.ErrorCode)
		errorCodeString = err.ErrorCode
		errorText = err.ErrorDetails
	} else {
		errorCodeInt, _ = strconv.Atoi(err.ResultCode)
		errorCodeString = err.ResultCode
		errorText = err.Description
	}

	logger.CtxInfo(ctx, "DecrementWalletSubscription error handler",
		slog.String("msisdn", msg.Msisdn),
		slog.String("errorCode", errorCodeString),
		slog.String("errorText", errorText),
		slog.Time("publish_time", msg.PublishTime),
	)
	// Create and save collection entry for all error types
	collectionEntry := common.SerializeCollections(msg, false, errorText, errorCodeString)
	if _, err := h.CollectionsRepo.CreateEntry(ctx, collectionEntry); err != nil {
		logger.CtxError(ctx, log_messages.ErrorFailedToCreateCollectionEntry, err)
	}
	// publish kafka failure event in helper
	publishKafkaFailureIfConfigured(ctx,
		h.KafkaService,
		msg,
		errorText,
		collectionEntry.CreatedAt,
		h.CollectionsRepo,
		collectionEntry.ID,
	)
	// Handle based on status code
	switch {
	case errorCodeInt == consts.SystemErrorCode:
		logger.CtxInfo(ctx, log_messages.ErrorHandlingSystemErrorCode,
			slog.String("msisdn", msg.Msisdn),
			slog.String("errorType", "system error"),
			slog.Time("publish_time", msg.PublishTime))
		return consts.ActionIgnore
	case err.ErrorCode == log_messages.OutboundConnError:
		logger.CtxInfo(ctx, log_messages.ErrorHandlingOutboundConnError,
			slog.String("msisdn", msg.Msisdn),
			slog.String("errorType", "outbound conn error"),
			slog.Time("publish_time", msg.PublishTime))
		return consts.ActionIgnore
	default:
		logger.CtxInfo(ctx, log_messages.ErrorHandlingBusinessOrNonOutboundConnError,
			slog.String("msisdn", msg.Msisdn),
			slog.String("errorCode", errorCodeString),
			slog.String("errorText", errorText),
			slog.Time("publish_time", msg.PublishTime))
		return h.HandleBusinessOrNonOutboundConnError(ctx, msg)
	}
}

func publishKafkaFailureIfConfigured(
	ctx context.Context,
	kafkaSvc *servicekafka.CollectionWorkerKafkaService,
	msg *models.PromoCollectionPublishedMessage,
	errorText string,
	createdAt time.Time,
	collectionsRepo interfaces.CollectionsRepoInterface,
	id primitive.ObjectID,
) {
	kafkaEntry := common.SerializeKafkaMessage(msg, false, errorText, createdAt)
	logger.CtxInfo(ctx, "Serialized kafka message", slog.Any("kafkaEntry", kafkaEntry))
	if kafkaSvc != nil {
		if err := kafkaSvc.PublishFailure(ctx, *kafkaEntry); err != nil {
			logger.CtxError(ctx,
				"Failed to publish failure message to Kafka",
				err,
				slog.Time("publish_time",
					msg.PublishTime))
		} else {
			logger.CtxInfo(ctx,
				"Published failure event to Kafka",
				slog.Any("kafkaEntry", kafkaEntry),
				slog.Time("publish_time",
					msg.PublishTime))
			err := collectionsRepo.UpdatePublishToKafka(ctx, id)
			if err != nil {
				logger.CtxError(ctx, "Failed to update publishedToKafka in collections document", err)
			}
		}
	} else {
		logger.CtxInfo(ctx,
			"Kafka service not configured; skipping publish",
			slog.Time("publish_time",
				msg.PublishTime))
	}

}

// HandleBusinessOrNonOutboundConnError handles business errors and non outbound conn errors
func (h *DecrementWalletErrorHandler) HandleBusinessOrNonOutboundConnError(
	ctx context.Context,
	msg *models.PromoCollectionPublishedMessage,
) string {

	// Delete transaction in progress entry
	if err := h.CollectionTransactionsInProgress.DeleteEntry(ctx, msg.Msisdn); err != nil {
		logger.CtxError(ctx, log_messages.ErrorFailedToDeleteTransactionInProgress, err,
			slog.String("msisdn", msg.Msisdn), slog.Time("publish_time", msg.PublishTime))
	}

	return consts.ActionAck
}

func (h *DecrementWalletErrorHandler) HandleUnknownOrNonAPIError(
	ctx context.Context,
	msg *models.PromoCollectionPublishedMessage,
	err error,
) string {
	logger.CtxError(ctx, log_messages.ErrorTextUnknownResponseOrNonAPIErrorStructFormatDWS,
		err,
		slog.Time("publish_time",
			msg.PublishTime))
	collectionEntry := common.SerializeCollections(msg, false,
		log_messages.ErrorTextUnknownResponseOrNonAPIErrorStructFormatDWS,
		log_messages.ErrorUnknownResponseStructFormatDWS)
	if _, err := h.CollectionsRepo.CreateEntry(ctx, collectionEntry); err != nil {
		logger.CtxError(ctx,
			log_messages.ErrorFailedToCreateCollectionEntry,
			err,
			slog.Time("publish_time",
				msg.PublishTime))
	}
	if err := h.CollectionTransactionsInProgress.DeleteEntry(ctx, msg.Msisdn); err != nil {
		logger.CtxError(ctx, log_messages.ErrorFailedToDeleteTransactionInProgress, err,
			slog.String("msisdn", msg.Msisdn), slog.Time("publish_time", msg.PublishTime))
	}
	// Publish failure event to Kafka if kafka service is configured
	kafkaEntry := common.SerializeKafkaMessage(
		msg,
		false,
		log_messages.ErrorTextUnknownResponseOrNonAPIErrorStructFormatDWS,
		collectionEntry.CreatedAt,
	)
	if h.KafkaService != nil {
		if err := h.KafkaService.PublishFailure(ctx, *kafkaEntry); err != nil {
			logger.CtxError(ctx,
				"Failed to publish failure message to Kafka",
				err,
				slog.Time("publish_time",
					msg.PublishTime))
		} else {
			logger.CtxInfo(ctx,
				"Published failure event to Kafka",
				slog.Any("kafkaEntry", kafkaEntry),
				slog.Time("publish_time",
					msg.PublishTime))
		}
	} else {
		logger.CtxInfo(ctx, "Kafka service not configured; skipping publish", slog.Time("publish_time", msg.PublishTime))
	}

	return consts.ActionAck
}
