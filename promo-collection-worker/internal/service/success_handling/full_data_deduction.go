package successhandling_service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"promo-collection-worker/internal/pkg/common"
	"promo-collection-worker/internal/pkg/consts"
	mongodb "promo-collection-worker/internal/pkg/db/mongo"
	"promo-collection-worker/internal/pkg/db/redis"
	"promo-collection-worker/internal/pkg/log_messages"
	"promo-collection-worker/internal/pkg/logger"
	"promo-collection-worker/internal/pkg/models"
	"promo-collection-worker/internal/pkg/pubsub"
	closedloans "promo-collection-worker/internal/pkg/store/impl/closed_loans"
	"promo-collection-worker/internal/pkg/store/impl/collection_transaction_in_progress"
	"promo-collection-worker/internal/pkg/store/impl/collections"
	loan_products "promo-collection-worker/internal/pkg/store/impl/loan_products"
	"promo-collection-worker/internal/pkg/store/impl/loans"
	"promo-collection-worker/internal/pkg/store/impl/messages"
	unpaidloans "promo-collection-worker/internal/pkg/store/impl/unpaid_loans"
	storemodels "promo-collection-worker/internal/pkg/store/models"
	serviceinterfaces "promo-collection-worker/internal/service/interfaces"
	"promo-collection-worker/internal/service/kafka"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func FullDataDeductionService(mongoClient *mongodb.MongoClient,
	redisClient *redis.RedisClient,
	pubSubClient serviceinterfaces.RuntimePubSubPublisher,
	kafkaClient *kafka.CollectionWorkerKafkaService,
	msg *models.PromoCollectionPublishedMessage,
	ctx context.Context,
	notificationTopic string,
) error {
	return fullDataDeductionHandlerFunc(mongoClient, redisClient, pubSubClient, kafkaClient, msg, ctx, notificationTopic)
}

var fullDataDeductionHandlerFunc = func(mongoClient *mongodb.MongoClient,
	redisClient *redis.RedisClient,
	pubSubClient serviceinterfaces.RuntimePubSubPublisher,
	kafkaClient *kafka.CollectionWorkerKafkaService,
	msg *models.PromoCollectionPublishedMessage,
	ctx context.Context,
	notificationTopic string,
) error {
	loanRepo := loans.NewLoansRepository(mongoClient)
	closedLoansRepo := closedloans.NewClosedLoansRepository(mongoClient)
	unpaidLoansRepo := unpaidloans.NewUnpaidLoansRepository(mongoClient)
	collectionsRepo := collections.NewCollectionsRepository(mongoClient)
	collectionTransactionsInProgressRepo := collection_transaction_in_progress.
		NewCollectionTransactionsInProgressRepository(mongoClient)

	return FullDataDeductionHandler(
		ctx,
		mongoClient,
		redisClient,
		pubSubClient,
		kafkaClient,
		closedLoansRepo,
		unpaidLoansRepo,
		collectionsRepo,
		loanRepo,
		collectionTransactionsInProgressRepo,
		msg,
		notificationTopic,
	)
}

func FullDataDeductionHandler(
	ctx context.Context,
	mongoClient *mongodb.MongoClient,
	redisClient *redis.RedisClient,
	pubSubClient serviceinterfaces.RuntimePubSubPublisher,
	kafkaClient *kafka.CollectionWorkerKafkaService,
	closedLoansRepo serviceinterfaces.ClosedLoansRepositoryInterface,
	unpaidLoansRepo serviceinterfaces.UnpaidLoansRepositoryInterface,
	collectionsRepo serviceinterfaces.CollectionsRepoInterface,
	loanRepo serviceinterfaces.LoanRepositoryInterface,
	collectionTransactionsInProgressRepo serviceinterfaces.CollectionTransactionsInProgressRepoInterface,
	msg *models.PromoCollectionPublishedMessage,
	notificationTopic string,
) error {
	closedLoanEntry := buildClosedLoanEntry(msg)
	unpaidLoanEntry := buildUnpaidLoanEntry(msg)
	collectionModel := buildFullCollectionModel(msg)

	if err := performFullDataTransactionFunc(
		ctx,
		mongoClient,
		closedLoansRepo,
		unpaidLoansRepo,
		collectionsRepo,
		loanRepo,
		closedLoanEntry,
		unpaidLoanEntry,
		collectionModel,
		msg,
	); err != nil {
		return err
	}

	// publish notification and finalize
	if err := publishNotificationAndFinalizeFullFunc(
		ctx,
		mongoClient,
		pubSubClient,
		kafkaClient,
		collectionTransactionsInProgressRepo,
		msg,
		collectionModel.CreatedAt,
		notificationTopic,
	); err != nil {
		return err
	}

	err := collectionsRepo.UpdatePublishToKafka(ctx, collectionModel.ID)
	if err != nil {
		return err
	}

	return nil
}

var (
	performFullDataTransactionFunc         = performFullDataTransaction
	publishNotificationAndFinalizeFullFunc = publishNotificationPayloadToPubSubAndKafkaFullCollection
	notifyFullFunc                         = notifyFull
)

// hooks for tests to stub repository lookups
var (
	fetchPatternIdFullFunc = func(
		ctx context.Context,
		mongoClient *mongodb.MongoClient,
		event string,
		brandID primitive.ObjectID,
	) (*storemodels.Messages, error) {
		messagesRepo := messages.NewMessagesRepository(mongoClient)
		return messagesRepo.GetPatternIdByEventandBrandId(ctx, event, brandID)
	}

	fetchProductNameFullFunc = func(
		ctx context.Context,
		mongoClient *mongodb.MongoClient,
		loanProductID primitive.ObjectID,
	) (*storemodels.LoanProducts, error) {
		loanProductsRepo := loan_products.NewLoanProductsRepository(mongoClient)
		return loanProductsRepo.GetProductNameById(ctx, loanProductID)
	}
)

// helper to build notification payload for full collection
func buildFullNotification(
	ctx context.Context,
	mongoClient *mongodb.MongoClient,
	msg *models.PromoCollectionPublishedMessage,
) ([]byte, error) {
	// fetch pattern id
	pattern, err := fetchPatternIdFullFunc(
		ctx,
		mongoClient,
		"AutoCollectionSuccessFullDataCollection",
		msg.BrandId,
	)
	var patternId int32
	if err != nil {
		logger.CtxError(ctx, "Failed to fetch notification pattern id", err)
	} else if pattern != nil {
		patternId = pattern.PatternId
	}

	productName := ""
	if msg.LoanProductId != primitive.NilObjectID {
		prod, err := fetchProductNameFullFunc(ctx, mongoClient, msg.LoanProductId)
		if err != nil {
			logger.CtxError(ctx, "Failed to fetch loan product name", err)
		} else if prod != nil {
			productName = prod.Name
		}
	}

	logger.CtxInfo(ctx,
		log_messages.SuccessFullCollectionNotificationData,
		slog.Any("Event", "AutoCollectionSuccessFullDataCollection"),
		slog.Any("PatternId", patternId),
		slog.Any("LoanProduct Name", productName),
	)

	notif := pubsub.LoanDeductionNotificationFormat{
		MSISDN:         msg.Msisdn,
		SmsDbEventName: "AutoCollectionSuccessFullDataCollection",
		PatternId:      patternId,
		NotifParameters: []pubsub.SmsNotificationParameter{
			{Name: pattern.Parameters[0], Value: fmt.Sprintf("%v MB", formatFloat(msg.DataToBeDeducted))},
			{Name: pattern.Parameters[1], Value: fmt.Sprintf("%v", formatFloat(msg.AmountToBeDeductedInPeso))},
			{Name: pattern.Parameters[2], Value: fmt.Sprintf("%v", productName)},
			{Name: pattern.Parameters[3], Value: msg.StartDate.Format(consts.SmsDateFormat)},
		},
	}

	logger.CtxInfo(ctx, log_messages.FullCollectionNotificationPayload, slog.Any("Payload", notif))

	notifBytes, err := json.Marshal(notif)
	if err != nil {
		logger.CtxError(ctx, "Failed to marshal notification payload", err)
		return nil, err
	}
	return notifBytes, nil
}

func notifyFull(
	ctx context.Context,
	mongoClient *mongodb.MongoClient,
	pubSubClient serviceinterfaces.RuntimePubSubPublisher,
	msg *models.PromoCollectionPublishedMessage,
	notificationTopic string,
) error {
	notifBytes, err := buildFullNotification(ctx, mongoClient, msg)
	if err != nil {
		logger.CtxError(ctx, "Failed to build notification payload", err)
		return err
	}

	if err := pubSubClient.Publish(ctx, notificationTopic, notifBytes); err != nil {
		logger.CtxError(ctx, "Failed to publish notification to Pub/Sub", err)
		return err
	}
	logger.CtxInfo(ctx, log_messages.SuccessFullCollectionNotificationPublished)
	return nil
}

// handles notification creation and final operations for full deduction
func publishNotificationPayloadToPubSubAndKafkaFullCollection(
	ctx context.Context,
	mongoClient *mongodb.MongoClient,
	pubSubClient serviceinterfaces.RuntimePubSubPublisher,
	kafkaClient *kafka.CollectionWorkerKafkaService,
	collectionTransactionsInProgressRepo serviceinterfaces.CollectionTransactionsInProgressRepoInterface,
	msg *models.PromoCollectionPublishedMessage,
	createdAt time.Time,
	notificationTopic string,
) error {
	// split notification creation/publish into helper to reduce function size
	if err := notifyFullFunc(ctx, mongoClient, pubSubClient, msg, notificationTopic); err != nil {
		// notifyFull logs details; add explicit log to avoid empty branch
		logger.CtxError(ctx, "notifyFull failed", err)
	}

	kafkaEntry := common.SerializeKafkaMessage(msg, true, "", createdAt)
	logger.CtxInfo(ctx, "Serialized kafka success message", slog.Any("kafkaEntry", kafkaEntry))

	if err := kafkaClient.PublishSuccess(ctx, *kafkaEntry); err != nil {
		logger.CtxError(ctx, "Failed to publish success message to Kafka", err)
		return err
	}

	if err := collectionTransactionsInProgressRepo.DeleteEntry(ctx, msg.Msisdn); err != nil {
		logger.CtxError(ctx, "Failed to delete document from CollectionTransactionInProgress", err)
		return err
	}

	return nil
}

func performFullDataTransaction(
	ctx context.Context,
	mongoClient *mongodb.MongoClient,
	closedLoansRepo serviceinterfaces.ClosedLoansRepositoryInterface,
	unpaidLoansRepo serviceinterfaces.UnpaidLoansRepositoryInterface,
	collectionsRepo serviceinterfaces.CollectionsRepoInterface,
	loanRepo serviceinterfaces.LoanRepositoryInterface,
	closedLoanEntry bson.M,
	unpaidLoanEntry bson.M,
	collectionModel *storemodels.Collections,
	msg *models.PromoCollectionPublishedMessage,
) error {
	_, err := runTransaction(ctx, mongoClient, func(txCtx context.Context) (interface{}, error) {
		if err := closedLoansRepo.CreateClosedLoansEntry(txCtx, closedLoanEntry); err != nil {
			return nil, err
		}

		collectionId, err := collectionsRepo.CreateEntry(txCtx, collectionModel)
		if err != nil {
			return nil, err
		}

		if err := unpaidLoansRepo.UpdatePreviousValidToField(txCtx, msg.LoanId); err != nil {
			return nil, err
		}

		unpaidLoanEntry["lastCollectionId"] = collectionId
		unpaidLoanEntry["lastCollectionDateTime"] = collectionModel.CreatedAt
		unpaidLoanEntry["unpaidServiceFee"] = msg.UnpaidServiceFee - collectionModel.CollectedServiceFee

		if err := unpaidLoansRepo.CreateUnpaidLoanEntry(txCtx, unpaidLoanEntry); err != nil {
			return nil, err
		}

		if err := loanRepo.DeleteActiveLoan(txCtx, msg.LoanId); err != nil {
			return nil, err
		}
		return nil, nil
	})
	if err != nil {
		logger.CtxError(ctx, "MongoDB transaction failed", err)
		return err
	}
	return nil
}

// helper to build closed loan entry
func buildClosedLoanEntry(msg *models.PromoCollectionPublishedMessage) bson.M {
	return bson.M{
		"GUID":                           msg.GUID,
		"MSISDN":                         msg.Msisdn,
		"loanId":                         msg.LoanId,
		"availmentTransactionId":         msg.AvailmentTransactionId,
		"brandId":                        msg.BrandId,
		"loanProductId":                  msg.LoanProductId,
		"loanType":                       msg.LoanType,
		"startDate":                      msg.StartDate,
		"endDate":                        time.Now(),
		"status":                         consts.StatusAuto,
		"totalCollectedAmount":           msg.TotalLoanAmountInPeso,
		"totalLoanAmount":                msg.TotalLoanAmountInPeso,
		"totalWrittenOffOrChurnedAmount": 0.0,
		"writeOffReason":                 consts.ReasonCleanUp,
	}
}

// helper to build unpaid loan entry
func buildUnpaidLoanEntry(msg *models.PromoCollectionPublishedMessage) bson.M {
	return bson.M{
		"loanId":            msg.LoanId,
		"totalUnpaidAmount": 0,
		"validFrom":         time.Now(),
		"validTo":           time.Now(),
		"version":           msg.Version + 1,
		"migrated":          false,
		"totalLoanAmount":   msg.TotalLoanAmountInPeso,
	}
}

// helper to build full collection model
func buildFullCollectionModel(
	msg *models.PromoCollectionPublishedMessage,
) *storemodels.Collections {
	collectionModel := common.SerializeCollections(msg, true, "", "")
	collectionModel.CollectedAmount = msg.AmountToBeDeductedInPeso
	collectionModel.CollectionCategory = consts.LoadTypeData
	collectionModel.CollectionType = consts.FullCollectionType
	collectionModel.Method = consts.Method
	collectionModel.PaymentChannel = msg.Channel
	collectionModel.LoanId = msg.LoanId
	collectionModel.TotalCollectedAmount =
		(msg.TotalLoanAmountInPeso - msg.TotalUnpaidAmountInPeso) + msg.AmountToBeDeductedInPeso
	collectionModel.TotalUnpaid = 0
	collectionModel.UnpaidServiceFee = 0
	collectionModel.ServiceFee = msg.ServiceFee
	collectionModel.Result = true
	collectionModel.PublishedToKafka = false
	collectionModel.CreatedAt = time.Now()
	return collectionModel
}
