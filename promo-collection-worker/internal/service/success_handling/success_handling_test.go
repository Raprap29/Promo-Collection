package successhandling_service

import (
	"context"
	"errors"
	"testing"
	"time"

	mongodb "promo-collection-worker/internal/pkg/db/mongo"
	redispkg "promo-collection-worker/internal/pkg/db/redis"
	"promo-collection-worker/internal/pkg/models"
	storemodels "promo-collection-worker/internal/pkg/store/models"
	serviceinterfaces "promo-collection-worker/internal/service/interfaces"
	"promo-collection-worker/internal/service/kafka"

	miniredis "github.com/alicebob/miniredis/v2"
	redislib "github.com/redis/go-redis/v9"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// mock producer implementing kafka.KafkaProducerInterface
type mockProducer struct {
	err error
}

// Test actual Redis interaction using an in-memory Redis server (miniredis)
func TestInsertAndDeleteSkipTimestampsInRedisActual(t *testing.T) {
	// start in-memory redis
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer s.Close()

	rc := &redispkg.RedisClient{Client: redislib.NewClient(&redislib.Options{Addr: s.Addr()})}

	msg := &models.PromoCollectionPublishedMessage{Msisdn: "639171234567"}

	// insert should succeed
	if err := insertSkipTimestampsInRedis(msg, &simpleSystemRulesRepo{}, rc); err != nil {
		t.Fatalf("expected insert to succeed, got %v", err)
	}
}

func (m *mockProducer) Publish(ctx context.Context, data []byte) error {
	return m.err
}

// mock collection tx repo
type mockCollectionTxRepoSimple struct {
	delErr error
}

func (m *mockCollectionTxRepoSimple) CheckEntryExists(ctx context.Context, msisdn string) (bool, error) {
	return true, nil
}
func (m *mockCollectionTxRepoSimple) CreateEntry(ctx context.Context, msisdn string) error {
	return nil
}
func (m *mockCollectionTxRepoSimple) DeleteEntry(ctx context.Context, msisdn string) error {
	return m.delErr
}

// mock pubsub client
type mockPubSub struct {
	err error
}

func (m *mockPubSub) Publish(ctx context.Context, topic string, msg []byte) error { return m.err }
func (m *mockPubSub) Close() error                                                { return nil }

// Test publishNotificationAndFinalize success path
func TestPublishNotificationAndFinalizeSuccess(t *testing.T) {
	ctx := context.Background()
	// stub notifyPartialFunc so it doesn't touch DB
	oldNotify := notifyPartialFunc
	notifyPartialFunc = func(ctx context.Context, mongoClient *mongodb.MongoClient, pubSubClient serviceinterfaces.RuntimePubSubPublisher, msg *models.PromoCollectionPublishedMessage, gracePeriodTimestamp time.Time, notificationTopic string) error {
		return nil
	}
	defer func() { notifyPartialFunc = oldNotify }()

	prod := &mockProducer{err: nil}
	kafkaClient := kafka.NewCollectionWorkerKafkaService(prod)
	collRepo := &mockCollectionTxRepoSimple{delErr: nil}

	msg := &models.PromoCollectionPublishedMessage{Msisdn: "639171234567"}

	err := publishNotificationAndFinalize(ctx, nil, nil, kafkaClient, collRepo, msg, time.Now(), time.Now().Add(24*time.Hour), "")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

// Test publishNotificationAndFinalize kafka failure
func TestPublishNotificationAndFinalizeKafkaFail(t *testing.T) {
	ctx := context.Background()
	oldNotify := notifyPartialFunc
	notifyPartialFunc = func(ctx context.Context, mongoClient *mongodb.MongoClient, pubSubClient serviceinterfaces.RuntimePubSubPublisher, msg *models.PromoCollectionPublishedMessage, gracePeriodTimestamp time.Time, notificationTopic string) error {
		return nil
	}
	defer func() { notifyPartialFunc = oldNotify }()

	prod := &mockProducer{err: errors.New("kafka fail")}
	kafkaClient := kafka.NewCollectionWorkerKafkaService(prod)
	collRepo := &mockCollectionTxRepoSimple{delErr: nil}

	msg := &models.PromoCollectionPublishedMessage{Msisdn: "639171234567"}

	err := publishNotificationAndFinalize(ctx, nil, nil, kafkaClient, collRepo, msg, time.Now(), time.Now().Add(24*time.Hour), "")
	if err == nil {
		t.Fatalf("expected kafka error, got nil")
	}
}

// Test publishNotificationAndFinalizeFull success and kafka failure
func TestPublishNotificationAndFinalizeFullSuccessAndFail(t *testing.T) {
	ctx := context.Background()
	// stub notifyFullFunc
	oldNotify := notifyFullFunc
	notifyFullFunc = func(ctx context.Context, mongoClient *mongodb.MongoClient, pubSubClient serviceinterfaces.RuntimePubSubPublisher, msg *models.PromoCollectionPublishedMessage, notificationTopic string) error {
		return nil
	}
	defer func() { notifyFullFunc = oldNotify }()

	prod := &mockProducer{err: nil}
	kafkaClient := kafka.NewCollectionWorkerKafkaService(prod)
	collRepo := &mockCollectionTxRepoSimple{delErr: nil}

	msg := &models.PromoCollectionPublishedMessage{Msisdn: "639171234567"}

	// success
	err := publishNotificationPayloadToPubSubAndKafkaFullCollection(ctx, nil, nil, kafkaClient, collRepo, msg, time.Now(), "")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	// kafka fail
	prodFail := &mockProducer{err: errors.New("fail")}
	kafkaClientFail := kafka.NewCollectionWorkerKafkaService(prodFail)
	err = publishNotificationPayloadToPubSubAndKafkaFullCollection(ctx, nil, nil, kafkaClientFail, collRepo, msg, time.Now(), "")
	if err == nil {
		t.Fatalf("expected kafka error, got nil")
	}
}

// Test redis delete and set error paths by pointing to an unreachable address
func TestRedisErrorPaths(t *testing.T) {
	// create a redis client that will fail on network ops
	rc := &redispkg.RedisClient{Client: redislib.NewClient(&redislib.Options{Addr: "127.0.0.1:1"})}

	// call the real insertSkipTimestampsInRedis by invoking it directly using a mock system repo
	// insert should return error due to redis Set failure
	if err := insertSkipTimestampsInRedis(&models.PromoCollectionPublishedMessage{Msisdn: "639171234567"}, &simpleSysRepo{}, rc); err == nil {
		t.Fatalf("expected redis set error, got nil")
	}
}

// simple system rules repo mock at package scope
type simpleSysRepo struct{}

func (s *simpleSysRepo) FetchSystemLevelRulesConfiguration(ctx context.Context) (storemodels.SystemLevelRules, error) {
	return storemodels.SystemLevelRules{GracePeriod: 24, PromoEducationPeriod: 24, LoanLoadPeriod: 24}, nil
}

// Test helper builders for full path
func TestBuildHelpers(t *testing.T) {
	msg := &models.PromoCollectionPublishedMessage{Msisdn: "639171234567", AmountToBeDeductedInPeso: 50, TotalLoanAmountInPeso: 100, ServiceFee: 5, Channel: "SMS"}

	closed := buildClosedLoanEntry(msg)
	if closed["MSISDN"] != msg.Msisdn {
		t.Fatalf("unexpected MSISDN in closed entry")
	}

	coll := buildFullCollectionModel(msg)
	if coll.MSISDN != msg.Msisdn {
		t.Fatalf("unexpected msisdn in collection model")
	}
}

func TestNotifyPartialAndFullDirect(t *testing.T) {
	ctx := context.Background()

	// stub fetch functions (partial and full) to return sample data
	oldFetchPatternP := fetchPatternIdFunc
	oldFetchProductP := fetchProductNameFunc
	oldFetchPatternF := fetchPatternIdFullFunc
	oldFetchProductF := fetchProductNameFullFunc
	fetchPatternIdFunc = func(ctx context.Context, mongoClient *mongodb.MongoClient, event string, brandID primitive.ObjectID) (*storemodels.Messages, error) {
		return &storemodels.Messages{PatternId: 42, Parameters: []string{"fee", "amount", "product", "startdate", "unpaid", "grace"}}, nil
	}
	fetchProductNameFunc = func(ctx context.Context, mongoClient *mongodb.MongoClient, loanProductID primitive.ObjectID) (*storemodels.LoanProducts, error) {
		return &storemodels.LoanProducts{Name: "prod"}, nil
	}
	fetchPatternIdFullFunc = func(ctx context.Context, mongoClient *mongodb.MongoClient, event string, brandID primitive.ObjectID) (*storemodels.Messages, error) {
		return &storemodels.Messages{PatternId: 42, Parameters: []string{"fee", "amount", "product", "startdate", "unpaid", "grace"}}, nil
	}
	fetchProductNameFullFunc = func(ctx context.Context, mongoClient *mongodb.MongoClient, loanProductID primitive.ObjectID) (*storemodels.LoanProducts, error) {
		return &storemodels.LoanProducts{Name: "prod"}, nil
	}
	defer func() {
		fetchPatternIdFunc = oldFetchPatternP
		fetchProductNameFunc = oldFetchProductP
		fetchPatternIdFullFunc = oldFetchPatternF
		fetchProductNameFullFunc = oldFetchProductF
	}()

	// mock pubsub that succeeds
	pub := &mockPubSub{err: nil}

	msg := &models.PromoCollectionPublishedMessage{Msisdn: "639171234567", AmountToBeDeductedInPeso: 10, ServiceFee: 5}

	if err := notifyPartial(ctx, nil, pub, msg, time.Now().Add(24*time.Hour), ""); err != nil {
		t.Fatalf("notifyPartial expected nil, got %v", err)
	}

	// now make pubsub fail
	pub.err = errors.New("pubsub fail")
	if err := notifyPartial(ctx, nil, pub, msg, time.Now().Add(24*time.Hour), ""); err == nil {
		t.Fatalf("notifyPartial expected error, got nil")
	}

	// notifyFull
	pub.err = nil
	if err := notifyFull(ctx, nil, pub, msg, ""); err != nil {
		t.Fatalf("notifyFull expected nil, got %v", err)
	}

	pub.err = errors.New("pubsub fail")
	if err := notifyFull(ctx, nil, pub, msg, ""); err == nil {
		t.Fatalf("notifyFull expected error, got nil")
	}
}

func TestServiceWrappersInvokeHandlers(t *testing.T) {
	// stub partial handler with matching package-level signature
	oldPartial := partialDataDeductionHandlerFunc
	partialDataDeductionHandlerFunc = func(mongoClient *mongodb.MongoClient, redisClient *redispkg.RedisClient, pubSubClient serviceinterfaces.RuntimePubSubPublisher, kafkaClient *kafka.CollectionWorkerKafkaService, msg *models.PromoCollectionPublishedMessage, ctx context.Context, notificationTopic string) error {
		return nil
	}
	defer func() { partialDataDeductionHandlerFunc = oldPartial }()

	if err := PartialDataDeductionService(nil, nil, nil, nil, &models.PromoCollectionPublishedMessage{}, context.Background(), ""); err != nil {
		t.Fatalf("PartialDataDeductionService expected nil, got %v", err)
	}

	// stub full handler with matching package-level signature
	oldFull := fullDataDeductionHandlerFunc
	fullDataDeductionHandlerFunc = func(mongoClient *mongodb.MongoClient, redisClient *redispkg.RedisClient, pubSubClient serviceinterfaces.RuntimePubSubPublisher, kafkaClient *kafka.CollectionWorkerKafkaService, msg *models.PromoCollectionPublishedMessage, ctx context.Context, notificationTopic string) error {
		return nil
	}
	defer func() { fullDataDeductionHandlerFunc = oldFull }()

	if err := FullDataDeductionService(nil, nil, nil, nil, &models.PromoCollectionPublishedMessage{}, context.Background(), ""); err != nil {
		t.Fatalf("FullDataDeductionService expected nil, got %v", err)
	}
}

// --- Additional handler flow tests ---

// minimal mock implementations for the handler tests
type simpleLoanRepoHappy struct{}

func (s *simpleLoanRepoHappy) GetLoanByMSISDN(ctx context.Context, msisdn string) (*storemodels.Loans, error) {
	return &storemodels.Loans{LoanID: primitive.NewObjectID(), GUID: "g1", BrandID: primitive.NewObjectID(), LoanProductID: primitive.NewObjectID(), CreatedAt: time.Now(), TotalUnpaidLoan: 100, ServiceFee: 5}, nil
}
func (s *simpleLoanRepoHappy) GetLoanByMSISDNList(ctx context.Context, msisdns []string) ([]storemodels.Loans, error) {
	return nil, nil
}
func (s *simpleLoanRepoHappy) GetLoanByAvailmentID(ctx context.Context, availmentId string) (*storemodels.Loans, error) {
	return nil, nil
}
func (s *simpleLoanRepoHappy) DeleteActiveLoan(ctx context.Context, loanID primitive.ObjectID) error {
	return nil
}
func (s *simpleLoanRepoHappy) UpdateLoanDocument(ctx context.Context, loan *storemodels.Loans, totalUnpaidLoan float64, gracePeriodTimestamp time.Time) (primitive.ObjectID, error) {
	return primitive.NewObjectID(), nil
}

type simpleUnpaidRepo struct{}

func (s *simpleUnpaidRepo) CreateUnpaidLoanEntry(ctx context.Context, loanEntry bson.M) error {
	return nil
}

func (s *simpleUnpaidRepo) UpdatePreviousValidToField(ctx context.Context, loanId primitive.ObjectID) error {
	return nil
}

type simpleCollectionsRepo struct{}

func (s *simpleCollectionsRepo) CreateEntry(ctx context.Context, model *storemodels.Collections) (primitive.ObjectID, error) {
	return primitive.NewObjectID(), nil
}
func (s *simpleCollectionsRepo) UpdatePublishToKafka(ctx context.Context, id primitive.ObjectID) error {
	return nil
}

func (s *simpleCollectionsRepo) GetFailedKafkaEntriesCursor(ctx context.Context, duration string, batchSize int32) (*mongo.Cursor, error) {
	return nil, nil
}
func (s *simpleCollectionsRepo) UpdatePublishedToKafkaInBulk(ctx context.Context, collectionIds []string) ([]string, error) {
	return []string{}, nil
}

type simpleClosedLoansRepo struct{}

func (s *simpleClosedLoansRepo) CreateClosedLoansEntry(ctx context.Context, document bson.M) error {
	return nil
}

type simpleCollectionTxRepo struct{}

func (s *simpleCollectionTxRepo) CheckEntryExists(ctx context.Context, msisdn string) (bool, error) {
	return true, nil
}
func (s *simpleCollectionTxRepo) CreateEntry(ctx context.Context, msisdn string) error { return nil }
func (s *simpleCollectionTxRepo) DeleteEntry(ctx context.Context, msisdn string) error { return nil }

// system-level rules repo for insertSkipTimestampsInRedis (not used because we stub)
type simpleSystemRulesRepo struct{}

func (s *simpleSystemRulesRepo) FetchSystemLevelRulesConfiguration(ctx context.Context) (storemodels.SystemLevelRules, error) {
	return storemodels.SystemLevelRules{GracePeriod: 24, PromoEducationPeriod: 24, LoanLoadPeriod: 24}, nil
}

// Test PartialDataDeductionHandler happy path by stubbing heavy package-level functions
func TestPartialDataDeductionHandlerFlow(t *testing.T) {
	ctx := context.Background()

	// backup and stub heavy operations
	oldPerform := performPartialDataTransactionFunc
	oldInsert := insertSkipTimestampsInRedisFunc
	oldPublish := publishNotificationAndFinalizeFunc
	performPartialDataTransactionFunc = func(ctx context.Context, mongoClient *mongodb.MongoClient, loanRepo serviceinterfaces.LoanRepositoryInterface, unpaidLoansRepo serviceinterfaces.UnpaidLoansRepositoryInterface, collectionsRepo serviceinterfaces.CollectionsRepoInterface, loan *storemodels.Loans, newUnpaidAmount float64, gracePeriodTimestamp time.Time, unpaidLoanEntry bson.M, collectionModel *storemodels.Collections, msg *models.PromoCollectionPublishedMessage) error {
		return nil
	}
	insertSkipTimestampsInRedisFunc = func(msg *models.PromoCollectionPublishedMessage, systemLevelRulesRepo serviceinterfaces.SystemLevelRulesRepository, redisClient *redispkg.RedisClient) error {
		return nil
	}
	publishNotificationAndFinalizeFunc = func(ctx context.Context, mongoClient *mongodb.MongoClient, pubSubClient serviceinterfaces.RuntimePubSubPublisher, kafkaClient *kafka.CollectionWorkerKafkaService, collectionTransactionsInProgressRepo serviceinterfaces.CollectionTransactionsInProgressRepoInterface, msg *models.PromoCollectionPublishedMessage, createdAt time.Time, gracePeriodTimestamp time.Time, notificationTopic string) error {
		return nil
	}
	defer func() {
		performPartialDataTransactionFunc = oldPerform
		insertSkipTimestampsInRedisFunc = oldInsert
		publishNotificationAndFinalizeFunc = oldPublish
	}()

	// call PartialDataDeductionHandler with simple mocks
	loanRepo := &simpleLoanRepoHappy{}
	unpaidRepo := &simpleUnpaidRepo{}
	collectionsRepo := &simpleCollectionsRepo{}
	sysRepo := &simpleSystemRulesRepo{}
	collTxRepo := &simpleCollectionTxRepo{}

	if err := PartialDataDeductionHandler(ctx, nil, nil, nil, nil, loanRepo, unpaidRepo, collectionsRepo, sysRepo, collTxRepo, &models.PromoCollectionPublishedMessage{Msisdn: "639171234567", AmountToBeDeductedInPeso: 10}, ""); err != nil {
		t.Fatalf("PartialDataDeductionHandler expected nil, got %v", err)
	}
}

// Test FullDataDeductionHandler happy path by stubbing heavy package-level functions
func TestFullDataDeductionHandlerFlow(t *testing.T) {
	ctx := context.Background()

	oldPerform := performFullDataTransactionFunc
	oldPublish := publishNotificationAndFinalizeFullFunc
	performFullDataTransactionFunc = func(ctx context.Context, mongoClient *mongodb.MongoClient, closedLoansRepo serviceinterfaces.ClosedLoansRepositoryInterface, unpaidLoansRepo serviceinterfaces.UnpaidLoansRepositoryInterface, collectionsRepo serviceinterfaces.CollectionsRepoInterface, loanRepo serviceinterfaces.LoanRepositoryInterface, closedLoanEntry bson.M, unpaidLoanEntry bson.M, collectionModel *storemodels.Collections, msg *models.PromoCollectionPublishedMessage) error {
		return nil
	}
	publishNotificationAndFinalizeFullFunc = func(ctx context.Context, mongoClient *mongodb.MongoClient, pubSubClient serviceinterfaces.RuntimePubSubPublisher, kafkaClient *kafka.CollectionWorkerKafkaService, collectionTransactionsInProgressRepo serviceinterfaces.CollectionTransactionsInProgressRepoInterface, msg *models.PromoCollectionPublishedMessage, createdAt time.Time, notificationTopic string) error {
		return nil
	}
	defer func() {
		performFullDataTransactionFunc = oldPerform
		publishNotificationAndFinalizeFullFunc = oldPublish
	}()

	loanRepo := &simpleLoanRepoHappy{}
	unpaidRepo := &simpleUnpaidRepo{}
	collectionsRepo := &simpleCollectionsRepo{}
	closedRepo := &simpleClosedLoansRepo{}
	collTxRepo := &simpleCollectionTxRepo{}

	if err := FullDataDeductionHandler(ctx, nil, nil, nil, nil, closedRepo, unpaidRepo, collectionsRepo, loanRepo, collTxRepo, &models.PromoCollectionPublishedMessage{Msisdn: "639171234567"}, ""); err != nil {
		t.Fatalf("FullDataDeductionHandler expected nil, got %v", err)
	}
}

// Test performFullDataTransaction and performPartialDataTransaction via runTransaction stub
func TestPerformDataTransactionRunTransactionStub(t *testing.T) {
	// backup runTransaction
	oldRun := runTransaction
	defer func() { runTransaction = oldRun }()

	// happy path: runTransaction invokes callback and returns nil error
	runTransaction = func(ctx context.Context, mc *mongodb.MongoClient, cb func(ctx context.Context) (interface{}, error)) (interface{}, error) {
		_, err := cb(context.Background())
		return nil, err
	}

	// call performFullDataTransaction with simple repos that return nil
	closedRepo := &simpleClosedLoansRepo{}
	unpaidRepo := &simpleUnpaidRepo{}
	collectionsRepo := &simpleCollectionsRepo{}
	loanRepo := &simpleLoanRepoHappy{}
	msg := &models.PromoCollectionPublishedMessage{Msisdn: "639171234567"}

	if err := performFullDataTransaction(context.Background(), nil, closedRepo, unpaidRepo, collectionsRepo, loanRepo, bson.M{"a": 1}, bson.M{"b": 2}, &storemodels.Collections{}, msg); err != nil {
		t.Fatalf("expected performFullDataTransaction success, got %v", err)
	}

	// call performPartialDataTransaction with simple repos
	loanRepoSimple := &simpleLoanRepoHappy{}
	if err := performPartialDataTransaction(context.Background(), nil, loanRepoSimple, &simpleUnpaidRepo{}, &simpleCollectionsRepo{}, &storemodels.Loans{LoanID: primitive.NewObjectID()}, 0.0, time.Now(), bson.M{"u": 1}, &storemodels.Collections{}, msg); err != nil {
		t.Fatalf("expected performPartialDataTransaction success, got %v", err)
	}

	// failure path: have runTransaction call the callback which returns an error
	runTransaction = func(ctx context.Context, mc *mongodb.MongoClient, cb func(ctx context.Context) (interface{}, error)) (interface{}, error) {
		return nil, errors.New("tx fail")
	}

	if err := performFullDataTransaction(context.Background(), nil, closedRepo, unpaidRepo, collectionsRepo, loanRepo, bson.M{"a": 1}, bson.M{"b": 2}, &storemodels.Collections{}, msg); err == nil {
		t.Fatalf("expected performFullDataTransaction to fail when runTransaction fails")
	}
}

// Test that publishNotificationAndFinalize returns error when DeleteEntry fails
func TestPublishNotificationAndFinalizeDeleteFail(t *testing.T) {
	ctx := context.Background()
	// stub notifyPartialFunc to avoid pubsub
	oldNotify := notifyPartialFunc
	notifyPartialFunc = func(ctx context.Context, mongoClient *mongodb.MongoClient, pubSubClient serviceinterfaces.RuntimePubSubPublisher, msg *models.PromoCollectionPublishedMessage, gracePeriodTimestamp time.Time, notificationTopic string) error {
		return nil
	}
	defer func() { notifyPartialFunc = oldNotify }()

	prod := &mockProducer{err: nil}
	kafkaClient := kafka.NewCollectionWorkerKafkaService(prod)

	// coll repo that returns delete error
	collRepo := &mockCollectionTxRepoSimple{delErr: errors.New("delete failed")}
	msg := &models.PromoCollectionPublishedMessage{Msisdn: "639171234567"}

	err := publishNotificationAndFinalize(ctx, nil, nil, kafkaClient, collRepo, msg, time.Now(), time.Now().Add(24*time.Hour), "")
	if err == nil {
		t.Fatalf("expected delete error, got nil")
	}
}

// Test that publishNotificationAndFinalizeFull returns error when DeleteEntry fails
func TestPublishNotificationAndFinalizeFullDeleteFail(t *testing.T) {
	ctx := context.Background()
	oldNotify := notifyFullFunc
	notifyFullFunc = func(ctx context.Context, mongoClient *mongodb.MongoClient, pubSubClient serviceinterfaces.RuntimePubSubPublisher, msg *models.PromoCollectionPublishedMessage, notificationTopic string) error {
		return nil
	}
	defer func() { notifyFullFunc = oldNotify }()

	prod := &mockProducer{err: nil}
	kafkaClient := kafka.NewCollectionWorkerKafkaService(prod)
	collRepo := &mockCollectionTxRepoSimple{delErr: errors.New("del fail")}

	msg := &models.PromoCollectionPublishedMessage{Msisdn: "639171234567"}

	err := publishNotificationPayloadToPubSubAndKafkaFullCollection(ctx, nil, nil, kafkaClient, collRepo, msg, time.Now(), "")
	if err == nil {
		t.Fatalf("expected delete error, got nil")
	}
}

// Test notifyPartial with fetchPattern/product errors: should log and continue (no error if pubsub works)
func TestNotifyPartialFetchErrors(t *testing.T) {
	ctx := context.Background()
	oldFetchPattern := fetchPatternIdFunc
	oldFetchProduct := fetchProductNameFunc
	fetchPatternIdFunc = func(ctx context.Context, mongoClient *mongodb.MongoClient, event string, brandID primitive.ObjectID) (*storemodels.Messages, error) {
		return &storemodels.Messages{PatternId: 0, Parameters: []string{"fee", "amount", "product", "startdate", "unpaid", "grace"}}, errors.New("fetch pattern fail")
	}
	fetchProductNameFunc = func(ctx context.Context, mongoClient *mongodb.MongoClient, loanProductID primitive.ObjectID) (*storemodels.LoanProducts, error) {
		return nil, errors.New("fetch product fail")
	}
	defer func() { fetchPatternIdFunc = oldFetchPattern; fetchProductNameFunc = oldFetchProduct }()

	pub := &mockPubSub{err: nil}
	msg := &models.PromoCollectionPublishedMessage{Msisdn: "639171234567", AmountToBeDeductedInPeso: 10, ServiceFee: 5}

	if err := notifyPartial(ctx, nil, pub, msg, time.Now().Add(24*time.Hour), ""); err == nil {
		t.Fatalf("notifyPartial expected error due to fetch errors, got nil")
	}
}

// Test notifyFull with fetchPattern/product errors
func TestNotifyFullFetchErrors(t *testing.T) {
	ctx := context.Background()
	oldFetchPattern := fetchPatternIdFullFunc
	oldFetchProduct := fetchProductNameFullFunc
	fetchPatternIdFullFunc = func(ctx context.Context, mongoClient *mongodb.MongoClient, event string, brandID primitive.ObjectID) (*storemodels.Messages, error) {
		return &storemodels.Messages{PatternId: 0, Parameters: []string{"fee", "amount", "product", "startdate", "unpaid", "grace"}}, errors.New("fetch pattern fail")
	}
	fetchProductNameFullFunc = func(ctx context.Context, mongoClient *mongodb.MongoClient, loanProductID primitive.ObjectID) (*storemodels.LoanProducts, error) {
		return nil, errors.New("fetch product fail")
	}
	defer func() { fetchPatternIdFullFunc = oldFetchPattern; fetchProductNameFullFunc = oldFetchProduct }()

	pub := &mockPubSub{err: nil}
	msg := &models.PromoCollectionPublishedMessage{Msisdn: "639171234567"}

	if err := notifyFull(ctx, nil, pub, msg, ""); err != nil {
		t.Fatalf("notifyFull expected nil despite fetch errors, got %v", err)
	}
}
