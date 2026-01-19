package handlers

import (
	"context"
	"log/slog"
	"strconv"

	kafkaclient "github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	mongodb "promocollection/internal/pkg/db/mongo"
	kafkaConsumer "promocollection/internal/pkg/kafka/consumer"
	"promocollection/internal/pkg/log_messages"
	"promocollection/internal/pkg/logger"
	"promocollection/internal/pkg/models"
	"promocollection/internal/pkg/pubsub"
	"promocollection/internal/pkg/store/impl"
	"promocollection/internal/pkg/store/impl/collection_transaction_in_progress"
	"promocollection/internal/pkg/store/impl/loan_products"
	"promocollection/internal/pkg/store/impl/loans"
	"promocollection/internal/pkg/store/impl/messages"
	"promocollection/internal/pkg/store/impl/system_level_rules"
	"promocollection/internal/pkg/store/impl/whitelisted_for_data_collection"
	"promocollection/internal/pkg/store/repository"
	promoCollection "promocollection/internal/service/promo_collection"
	interfaces "promocollection/internal/service/interfaces"
	"promocollection/internal/service/kafka"
)

type PromoHandler struct {
	promoKafkaConsumer *kafkaConsumer.KafkaConsumer
}

func NewPromoHandler(ctx context.Context, promoKafkaConsumer *kafkaConsumer.KafkaConsumer) *PromoHandler {
	return &PromoHandler{
		promoKafkaConsumer: promoKafkaConsumer,
	}
}

func (p *PromoHandler) PromoKafkaConsumer(
	ctx context.Context,
	kafkaConsumer *kafkaConsumer.KafkaConsumer,
	mongoClient *mongodb.MongoClient,
	redisClient *redis.Client,
	pubSub, pubSubNotifClient *pubsub.PubSubClient,
) error {
	var kafkaService interfaces.KafkaConsumerServiceInterface = &kafka.KafkaConsumerService{}

	for {
		payload, msg, err := kafkaService.StartKafkaConsumer(ctx, kafkaConsumer)
		if err != nil {
			logger.CtxError(ctx, log_messages.KafkaErrorConsuming, err)
			return err
		}

		if payload == nil {
			logger.CtxError(ctx, "No payload received", err)
		}

		traceCtx := p.setupTraceContext(ctx, payload, msg)

		p.processPromoCollection(traceCtx, mongoClient, redisClient, pubSub, pubSubNotifClient, payload)
	}
}

func (p *PromoHandler) setupTraceContext(ctx context.Context, payload any, msg *kafkaclient.Message) context.Context {
	traceID := uuid.New().String()
	traceCtx := logger.WithTraceID(ctx, traceID)

	logger.CtxInfo(traceCtx, "New collection request started",
		slog.String("trace_id", traceID),
		slog.Any("payload", payload),
	)

	if msg != nil {
		logger.CtxDebug(traceCtx, "Kafka message", slog.String("message", string(msg.Value)))
	}

	return traceCtx
}

func (p *PromoHandler) processPromoCollection(
	ctx context.Context,
	mongoClient *mongodb.MongoClient,
	redisClient *redis.Client,
	pubSub, pubSubNotifClient *pubsub.PubSubClient,
	payload *models.PromoEventMessage,
) {
	systemRulesRepo := system_level_rules.NewSystemLevelRulesRepository(mongoClient)
	loansRepo := loans.NewLoansRepository(mongoClient)
	messagesRepo := messages.NewMessagesRepository(mongoClient)
	loanProdRepo := loan_products.NewLoanProductsRepository(mongoClient)
	whitelistedForDataCollectionRepo :=
		whitelisted_for_data_collection.NewWhitelistedForDataCollectionRepository(mongoClient)
	collectionTransactionsInProgressRepo := collection_transaction_in_progress.
		NewCollectionTransactionsInProgressRepository(mongoClient)
	unpaidRepo := impl.NewUnpaidLoansRepository(mongoClient)

	redisAdapter := repository.NewRedisStoreAdapter(redisClient)

	businessRulesService := promoCollection.NewBusinessLevelRulesService(
		loansRepo,
		messagesRepo,
		loanProdRepo,
		redisAdapter,
		collectionTransactionsInProgressRepo,
		whitelistedForDataCollectionRepo,
		systemRulesRepo,
		pubSub,
		pubSubNotifClient,
		payload,
		unpaidRepo,
	)
	_, promoCollectionError := businessRulesService.CheckBusinessLevelRulesForPromoCollection(
		ctx,
		strconv.Itoa(int(payload.MSISDN)),
	)
	if promoCollectionError != nil {
		logger.CtxError(ctx, "Error while performing promoCollection", promoCollectionError)
	} else {
		logger.CtxInfo(ctx, "Promo collection processing complete")
	}
}
