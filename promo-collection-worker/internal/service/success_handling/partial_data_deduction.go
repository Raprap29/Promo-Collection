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
	"promo-collection-worker/internal/pkg/store/impl/collection_transaction_in_progress"
	"promo-collection-worker/internal/pkg/store/impl/collections"
	loan_products "promo-collection-worker/internal/pkg/store/impl/loan_products"
	"promo-collection-worker/internal/pkg/store/impl/loans"
	"promo-collection-worker/internal/pkg/store/impl/messages"
	"promo-collection-worker/internal/pkg/store/impl/system_level_rules"
	unpaidloans "promo-collection-worker/internal/pkg/store/impl/unpaid_loans"
	storemodels "promo-collection-worker/internal/pkg/store/models"
	serviceinterfaces "promo-collection-worker/internal/service/interfaces"
	"promo-collection-worker/internal/service/kafka"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// runTransaction is an injectable hook that runs a transaction. Tests can replace
var runTransaction = func(
	ctx context.Context,
	mc *mongodb.MongoClient,
	cb func(ctx context.Context) (interface{}, error),
) (interface{}, error) {
	session, err := mc.Client.StartSession()
	if err != nil {
		return nil, fmt.Errorf("failed to start MongoDB session: %w", err)
	}
	defer session.EndSession(context.Background())

	wrapper := func(sc mongo.SessionContext) (interface{}, error) {
		return cb(sc)
	}
	return session.WithTransaction(ctx, wrapper)
}

// It delegates to PartialDataDeductionHandler which is easier to unit-test
func PartialDataDeductionService(mongoClient *mongodb.MongoClient,
	redisClient *redis.RedisClient,
	pubSubClient serviceinterfaces.RuntimePubSubPublisher,
	kafkaClient *kafka.CollectionWorkerKafkaService,
	msg *models.PromoCollectionPublishedMessage,
	ctx context.Context,
	notificationTopic string,
) error {
	return partialDataDeductionHandlerFunc(
		mongoClient,
		redisClient,
		pubSubClient,
		kafkaClient,
		msg,
		ctx,
		notificationTopic,
	)
}

var partialDataDeductionHandlerFunc = func(
	mongoClient *mongodb.MongoClient,
	redisClient *redis.RedisClient,
	pubSubClient serviceinterfaces.RuntimePubSubPublisher,
	kafkaClient *kafka.CollectionWorkerKafkaService,
	msg *models.PromoCollectionPublishedMessage,
	ctx context.Context,
	notificationTopic string,
) error {
	loanRepo, unpaidLoansRepo, collectionsRepo, systemLevelRulesRepo,
		collectionTransactionsInProgressRepo := initPartialRepos(mongoClient)
	return PartialDataDeductionHandler(
		ctx,
		mongoClient,
		redisClient,
		pubSubClient,
		kafkaClient,
		loanRepo,
		unpaidLoansRepo,
		collectionsRepo,
		systemLevelRulesRepo,
		collectionTransactionsInProgressRepo,
		msg,
		notificationTopic,
	)
}

func PartialDataDeductionHandler(
	ctx context.Context,
	mongoClient *mongodb.MongoClient,
	redisClient *redis.RedisClient,
	pubSubClient serviceinterfaces.RuntimePubSubPublisher,
	kafkaClient *kafka.CollectionWorkerKafkaService,
	loanRepo serviceinterfaces.LoanRepositoryInterface,
	unpaidLoansRepo serviceinterfaces.UnpaidLoansRepositoryInterface,
	collectionsRepo serviceinterfaces.CollectionsRepoInterface,
	systemLevelRulesRepo serviceinterfaces.SystemLevelRulesRepository,
	collectionTransactionsInProgressRepo serviceinterfaces.CollectionTransactionsInProgressRepoInterface,
	msg *models.PromoCollectionPublishedMessage,
	notificationTopic string,
) error {
	loan, newUnpaidAmount, gracePeriodTimestamp, err := fetchLoanAndCompute(ctx, loanRepo, msg, systemLevelRulesRepo)
	if err != nil {
		return err
	}
	unpaidLoanEntry := buildPartialUnpaidLoanEntry(msg)
	// Build a Collections model instance for insertion
	collectionModel := preparePartialCollectionModel(msg, newUnpaidAmount)

	if err := performPartialDataTransactionFunc(
		ctx,
		mongoClient,
		loanRepo,
		unpaidLoansRepo,
		collectionsRepo,
		loan,
		newUnpaidAmount,
		gracePeriodTimestamp,
		unpaidLoanEntry,
		collectionModel,
		msg,
	); err != nil {
		return err
	}

	// Insert skip timestamps in Redis
	if err := insertSkipTimestampsInRedisFunc(msg, systemLevelRulesRepo, redisClient); err != nil {
		logger.CtxError(ctx, "Failed to insert skip timestamps in Redis", err)
	}

	// publish notification, kafka and finalize
	if err := publishNotificationAndFinalizeFunc(
		ctx,
		mongoClient,
		pubSubClient,
		kafkaClient,
		collectionTransactionsInProgressRepo,
		msg,
		collectionModel.CreatedAt,
		gracePeriodTimestamp,
		notificationTopic,
	); err != nil {
		return err
	}

	err = collectionsRepo.UpdatePublishToKafka(ctx, collectionModel.ID)
	if err != nil {
		return err
	}

	return nil
}

var (
	performPartialDataTransactionFunc  = performPartialDataTransaction
	publishNotificationAndFinalizeFunc = publishNotificationAndFinalize
	insertSkipTimestampsInRedisFunc    = insertSkipTimestampsInRedis
	notifyPartialFunc                  = notifyPartial
)

var (
	fetchPatternIdFunc = func(
		ctx context.Context,
		mongoClient *mongodb.MongoClient,
		event string,
		brandID primitive.ObjectID,
	) (*storemodels.Messages, error) {
		messagesRepo := messages.NewMessagesRepository(mongoClient)
		return messagesRepo.GetPatternIdByEventandBrandId(ctx, event, brandID)
	}

	fetchProductNameFunc = func(
		ctx context.Context,
		mongoClient *mongodb.MongoClient,
		loanProductID primitive.ObjectID,
	) (*storemodels.LoanProducts, error) {
		loanProductsRepo := loan_products.NewLoanProductsRepository(mongoClient)
		prod, err := loanProductsRepo.GetProductNameById(ctx, loanProductID)
		return prod, err
	}
)

func buildPartialCollectionModel(
	msg *models.PromoCollectionPublishedMessage,
	createdAt time.Time,
	newUnpaidAmount float64,
) *storemodels.Collections {
	var unpaidservicefee float64

	if msg.AmountToBeDeductedInPeso >= msg.UnpaidServiceFee {
		unpaidservicefee = 0.0
	} else {
		unpaidservicefee = msg.UnpaidServiceFee - msg.AmountToBeDeductedInPeso
	}

	collectionModel := common.SerializeCollections(msg, true, "", "")
	collectionModel.CollectedAmount = msg.AmountToBeDeductedInPeso
	collectionModel.CollectionCategory = consts.LoadTypeData
	collectionModel.CollectionType = consts.PartialCollectionType
	collectionModel.Method = consts.Method
	collectionModel.PaymentChannel = msg.Channel
	collectionModel.LoanId = msg.LoanId
	collectionModel.TotalCollectedAmount = (msg.TotalLoanAmountInPeso -
		msg.TotalUnpaidAmountInPeso) + msg.AmountToBeDeductedInPeso
	collectionModel.TotalUnpaid = newUnpaidAmount
	collectionModel.UnpaidServiceFee = unpaidservicefee
	collectionModel.ServiceFee = msg.ServiceFee
	collectionModel.Result = true
	collectionModel.PublishedToKafka = false
	collectionModel.CreatedAt = createdAt
	collectionModel.DataCollected = msg.DataToBeDeducted
	collectionModel.TransactionId = msg.DataCollectionRequestTraceId
	collectionModel.CollectedServiceFee = msg.ServiceFee - unpaidservicefee
	collectionModel.UnpaidLoanAmount = newUnpaidAmount - (msg.ServiceFee - collectionModel.CollectedServiceFee)
	return collectionModel
}

func insertSkipTimestampsInRedis(msg *models.PromoCollectionPublishedMessage,
	systemLevelRulesRepo serviceinterfaces.SystemLevelRulesRepository,
	redisClient *redis.RedisClient,
) error {
	ctx := context.Background()

	systemRules, err := systemLevelRulesRepo.FetchSystemLevelRulesConfiguration(ctx)
	if err != nil {
		logger.CtxError(ctx, log_messages.ErrorFetchingSLRMongoDbDocument, err)
		return err
	}

	// Calculate timestamps based on current time + period values
	now := time.Now()
	graceDuration := time.Duration(systemRules.GracePeriod) * time.Hour
	skipTimestamps := map[string]interface{}{
		"gracePeriodTimestamp": now.Add(graceDuration),
	}

	timestampData, err := json.Marshal(skipTimestamps)
	if err != nil {
		logger.CtxError(ctx, "Failed to marshal skip timestamps", err)
		return err
	}

	key := fmt.Sprintf("skipTimestamps:%s", msg.Msisdn)

	err = redisClient.Client.Set(ctx, key, timestampData, graceDuration).Err()
	if err != nil {
		logger.CtxError(ctx, "Failed to set skip timestamps in Redis", err)
		return err
	}

	logger.CtxInfo(ctx, fmt.Sprintf("Successfully stored skip timestamps for MSISDN: %s", msg.Msisdn))
	return nil
}

func buildPartialUnpaidLoanEntry(msg *models.PromoCollectionPublishedMessage) bson.M {
	return bson.M{
		"loanId":            msg.LoanId,
		"totalUnpaidAmount": msg.TotalUnpaidAmountInPeso - msg.AmountToBeDeductedInPeso,
		"validFrom":         time.Now(),
		"validTo":           nil,
		"version":           msg.Version + 1,
		"migrated":          false,
	}
}

// publishNotificationAndFinalize handles notification creation, publishing to pubsub, kafka publish and final cleanup
func publishNotificationAndFinalize(ctx context.Context,
	mongoClient *mongodb.MongoClient,
	pubSubClient serviceinterfaces.RuntimePubSubPublisher,
	kafkaClient *kafka.CollectionWorkerKafkaService,
	collectionTransactionsInProgressRepo serviceinterfaces.CollectionTransactionsInProgressRepoInterface,
	msg *models.PromoCollectionPublishedMessage,
	createdAt time.Time,
	gracePeriodTimestamp time.Time,
	notificationTopic string,
) error {
	// break out notification into helper to reduce function length
	if err := notifyPartialFunc(ctx, mongoClient, pubSubClient, msg, gracePeriodTimestamp, notificationTopic); err != nil {
		logger.CtxError(ctx, "notifyPartial failed", err)
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

func notifyPartial(
	ctx context.Context,
	mongoClient *mongodb.MongoClient,
	pubSubClient serviceinterfaces.RuntimePubSubPublisher,
	msg *models.PromoCollectionPublishedMessage,
	gracePeriodTimestamp time.Time,
	notificationTopic string,
) error {
	notifBytes, err := buildPartialNotification(ctx, mongoClient, msg, gracePeriodTimestamp)
	if err != nil {
		logger.CtxError(ctx, "Failed to build notification payload", err)
		return err
	}

	if err := pubSubClient.Publish(ctx, notificationTopic, notifBytes); err != nil {
		logger.CtxError(ctx, "Failed to publish notification to Pub/Sub", err)
		return err
	}
	logger.CtxInfo(ctx, log_messages.SuccessPartialCollectionNotificationPublished)
	return nil
}

// helper to build notification payload for partial collection
func buildPartialNotification(
	ctx context.Context,
	mongoClient *mongodb.MongoClient,
	msg *models.PromoCollectionPublishedMessage,
	gracePeriodTimestamp time.Time,
) ([]byte, error) {
	// fetch pattern id
	pattern, err := fetchPatternIdFunc(
		ctx,
		mongoClient,
		"AutoCollectionSuccessPartialDataCollection",
		msg.BrandId,
	)
	var patternId int32
	if err != nil {
		// log and continue; pattern id will be zero if not found
		logger.CtxError(ctx, "Failed to fetch notification pattern id", err)
		return nil, err
	} else if pattern != nil {
		patternId = pattern.PatternId
	}

	productName := ""
	if msg.LoanProductId != primitive.NilObjectID {
		prodIface, err := fetchProductNameFunc(ctx, mongoClient, msg.LoanProductId)
		if err != nil {
			logger.CtxError(ctx, "Failed to fetch loan product name", err)
			return nil, err
		} else if prodIface != nil {
			productName = prodIface.Name
		}
	}
	logger.CtxInfo(ctx,
		log_messages.SuccessPartialCollectionNotificationData,
		slog.Any("Event", "AutoCollectionSuccessPartialDataCollection"),
		slog.Any("PatternId", patternId),
		slog.Any("LoanProduct Name", productName),
	)
	now := time.Now().UTC()
	diff := gracePeriodTimestamp.Sub(now)
	daysRemaining := int(diff.Hours() / 24)
	notif := pubsub.LoanDeductionNotificationFormat{
		MSISDN:         msg.Msisdn,
		SmsDbEventName: "AutoCollectionSuccessPartialDataCollection",
		PatternId:      patternId,
		NotifParameters: []pubsub.SmsNotificationParameter{
			{Name: pattern.Parameters[1], Value: fmt.Sprintf("%v", formatFloat(msg.AmountToBeDeductedInPeso))},
			{Name: pattern.Parameters[2], Value: productName},
			{Name: pattern.Parameters[0], Value: fmt.Sprintf("%v MB", formatFloat(msg.DataToBeDeducted))},
			{Name: pattern.Parameters[3], Value: msg.StartDate.Format(consts.SmsDateFormat)},
			{Name: pattern.Parameters[4], Value: fmt.Sprintf("%v",
				formatFloat(msg.TotalUnpaidAmountInPeso-msg.AmountToBeDeductedInPeso))},
			{Name: pattern.Parameters[5], Value: fmt.Sprintf("%v", daysRemaining)},
		},
	}

	logger.CtxInfo(ctx, log_messages.PartialCollectionNotificationPayload, slog.Any("Payload", notif))
	notifBytes, err := json.Marshal(notif)
	if err != nil {
		logger.CtxError(ctx, "Failed to marshal notification payload", err)
		return nil, err
	}
	return notifBytes, nil
}

func formatFloat(value float64) string {
	return fmt.Sprintf(consts.FloatTwoDecimalFormat, value)
}

func initPartialRepos(mongoClient *mongodb.MongoClient) (
	*loans.LoanRepository,
	*unpaidloans.UnpaidLoansRepository,
	serviceinterfaces.CollectionsRepoInterface,
	*system_level_rules.SystemLevelRulesRepository,
	serviceinterfaces.CollectionTransactionsInProgressRepoInterface,
) {
	loanRepo := loans.NewLoansRepository(mongoClient)
	unpaidLoansRepo := unpaidloans.NewUnpaidLoansRepository(mongoClient)
	collectionsRepo := collections.NewCollectionsRepository(mongoClient)
	systemLevelRulesRepo := system_level_rules.NewSystemLevelRulesRepository(mongoClient)
	collectionTransactionsInProgressRepo := collection_transaction_in_progress.
		NewCollectionTransactionsInProgressRepository(mongoClient)
	return loanRepo, unpaidLoansRepo, collectionsRepo, systemLevelRulesRepo, collectionTransactionsInProgressRepo
}

func performPartialDataTransaction(
	ctx context.Context,
	mongoClient *mongodb.MongoClient,
	loanRepo serviceinterfaces.LoanRepositoryInterface,
	unpaidLoansRepo serviceinterfaces.UnpaidLoansRepositoryInterface,
	collectionsRepo serviceinterfaces.CollectionsRepoInterface,
	loan *storemodels.Loans,
	newUnpaidAmount float64,
	gracePeriodTimestamp time.Time,
	unpaidLoanEntry bson.M,
	collectionModel *storemodels.Collections,
	msg *models.PromoCollectionPublishedMessage,
) error {
	// Use injectable runTransaction so tests can stub transaction execution.
	_, err := runTransaction(ctx, mongoClient, func(txCtx context.Context) (interface{}, error) {
		if _, err := loanRepo.UpdateLoanDocument(txCtx, loan, newUnpaidAmount, gracePeriodTimestamp); err != nil {
			return nil, err
		}

		if err := unpaidLoansRepo.UpdatePreviousValidToField(txCtx, loan.LoanID); err != nil {
			return nil, err
		}

		collectionId, err := collectionsRepo.CreateEntry(txCtx, collectionModel)
		if err != nil {
			return nil, err
		}

		unpaidLoanEntry["lastCollectionId"] = collectionId
		unpaidLoanEntry["lastCollectionDateTime"] = collectionModel.CreatedAt
		unpaidLoanEntry["unpaidServiceFee"] = msg.UnpaidServiceFee - collectionModel.CollectedServiceFee

		if err := unpaidLoansRepo.CreateUnpaidLoanEntry(txCtx, unpaidLoanEntry); err != nil {
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

func fetchLoanAndCompute(
	ctx context.Context,
	loanRepo serviceinterfaces.LoanRepositoryInterface,
	msg *models.PromoCollectionPublishedMessage,
	systemLevelRulesRepo serviceinterfaces.SystemLevelRulesRepository,
) (*storemodels.Loans, float64, time.Time, error) {
	loan, err := loanRepo.GetLoanByMSISDN(ctx, msg.Msisdn)
	if err != nil {
		logger.CtxError(ctx, log_messages.ErrorFetchingLoansMongoDbDocument, err)
		return nil, 0, time.Time{}, err
	}
	newUnpaidAmount := msg.TotalUnpaidAmountInPeso - msg.AmountToBeDeductedInPeso

	systemRules, err := systemLevelRulesRepo.FetchSystemLevelRulesConfiguration(ctx)
	if err != nil {
		logger.CtxError(ctx, log_messages.ErrorFetchingSLRMongoDbDocument, err)
		return nil, 0, time.Time{}, err
	}

	graceDuration := time.Duration(systemRules.GracePeriod) * time.Hour
	gracePeriodTimestamp := time.Now().Add(graceDuration)
	return loan, newUnpaidAmount, gracePeriodTimestamp, nil
}

func preparePartialCollectionModel(
	msg *models.PromoCollectionPublishedMessage,
	newUnpaidAmount float64,
) *storemodels.Collections {
	createdAt := time.Now()
	collectionModel := buildPartialCollectionModel(msg, createdAt, newUnpaidAmount)
	return collectionModel
}
