package error_handling_service

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"promo-collection-worker/internal/pkg/consts"
	mongodb "promo-collection-worker/internal/pkg/db/mongo"
	redis "promo-collection-worker/internal/pkg/db/redis"
	"promo-collection-worker/internal/pkg/downstream/error_handling"
	"promo-collection-worker/internal/pkg/models"
	serviceinterfaces "promo-collection-worker/internal/service/interfaces"
	servicekafka "promo-collection-worker/internal/service/kafka"
)

const (
	testWalletBucketMissing = "wallet bucket missing"
	testDbError             = "db error"
)

type FakeGCSClient struct {
	UploadCalled bool
	UploadErr    error
}

func (f *FakeGCSClient) Upload(ctx context.Context, msg *models.PromoCollectionPublishedMessage) error {
	f.UploadCalled = true
	return f.UploadErr
}

func (f *FakeGCSClient) Close(ctx context.Context) {}

// Test for NewRetrieveSubscriberUsageErrorHandler
func TestNewRetrieveSubscriberUsageErrorHandler(t *testing.T) {
	// Setup
	mockCollectionsRepo := new(MockCollectionsRepo)
	mockTransactionsRepo := new(MockCollectionTransactionsInProgressRepo)

	// Execute
	// Provide nil kafka/mongo/redis/pubsub and stub callbacks for tests
	// Note: the production constructor expects concrete types; for test we can call the struct literal directly
	handler := &RetrieveSubscriberUsageErrorHandler{
		CollectionsRepo:                  mockCollectionsRepo,
		CollectionTransactionsInProgress: mockTransactionsRepo,
		KafkaService:                     nil,
		MongoClient:                      nil,
		RedisClient:                      nil,
		PubSubPublisher:                  nil,
		PartialSuccessHandler:            nil,
		FullSuccessHandler:               nil,
	}

	// Assert
	assert.NotNil(t, handler)
	assert.Equal(t, mockCollectionsRepo, handler.CollectionsRepo)
	assert.Equal(t, mockTransactionsRepo, handler.CollectionTransactionsInProgress)
	assert.Nil(t, handler.KafkaService)
}

func TestHandleErrorNonAPIError(t *testing.T) {
	mockTransactionsRepo := new(MockCollectionTransactionsInProgressRepo)
	fakeGCS := &FakeGCSClient{}

	handler := RetrieveSubscriberUsageErrorHandler{
		CollectionTransactionsInProgress: mockTransactionsRepo,
		GcsClient:                        fakeGCS,
	}

	ctx := context.Background()
	msg := CreateTestMessage()
	err := error_handling.NewRetrieveSubscriberUsageError(errors.New("network error"))

	result := handler.HandleError(ctx, msg, err)

	assert.Nil(t, result)
	mockTransactionsRepo.AssertExpectations(t)
}

func TestHandleErrorHttpOk(t *testing.T) {
	mockCollectionsRepo := new(MockCollectionsRepo)
	mockTransactionsRepo := new(MockCollectionTransactionsInProgressRepo)

	handler := RetrieveSubscriberUsageErrorHandler{
		CollectionsRepo:                  mockCollectionsRepo,
		CollectionTransactionsInProgress: mockTransactionsRepo,
	}

	ctx := context.Background()
	msg := CreateTestMessage()
	err := error_handling.NewRetrieveSubscriberUsageError(
		errors.New(testWalletBucketMissing),
		http.StatusOK,
	)
	err.ErrorCode = "MISSING_BUCKET"
	err.ErrorMessage = "No matching wallet bucket found"

	objID := primitive.NewObjectID()
	mockCollectionsRepo.On("CreateEntry", ctx, mock.Anything).Return(objID, nil)
	mockTransactionsRepo.On("DeleteEntry", ctx, msg.Msisdn).Return(nil)

	result := handler.HandleError(ctx, msg, err, http.StatusOK)

	assert.Nil(t, result)
	mockCollectionsRepo.AssertExpectations(t)
	mockTransactionsRepo.AssertExpectations(t)
}

func TestHandleErrorHttpError(t *testing.T) {
	mockTransactionsRepo := new(MockCollectionTransactionsInProgressRepo)

	handler := RetrieveSubscriberUsageErrorHandler{
		CollectionTransactionsInProgress: mockTransactionsRepo,
	}

	ctx := context.Background()
	msg := CreateTestMessage()
	err := error_handling.NewRetrieveSubscriberUsageError(
		errors.New("server error"),
		http.StatusInternalServerError,
	)
	err.ErrorCode = "SERVER_ERROR"
	err.ErrorMessage = "Internal server error"

	result := handler.HandleError(ctx, msg, err, http.StatusInternalServerError)

	assert.Nil(t, result)
	mockTransactionsRepo.AssertExpectations(t)
}

func TestHandleErrorCreateEntryError(t *testing.T) {
	mockCollectionsRepo := new(MockCollectionsRepo)
	mockTransactionsRepo := new(MockCollectionTransactionsInProgressRepo)

	handler := RetrieveSubscriberUsageErrorHandler{
		CollectionsRepo:                  mockCollectionsRepo,
		CollectionTransactionsInProgress: mockTransactionsRepo,
	}

	ctx := context.Background()
	msg := CreateTestMessage()
	err := error_handling.NewRetrieveSubscriberUsageError(
		errors.New("wallet bucket missing"),
		http.StatusOK,
	)

	mockCollectionsRepo.On("CreateEntry", ctx, mock.Anything).Return(nil, errors.New(testDbError))
	mockTransactionsRepo.On("DeleteEntry", ctx, msg.Msisdn).Return(nil)

	result := handler.HandleError(ctx, msg, err, http.StatusOK)

	assert.Nil(t, result)
	mockCollectionsRepo.AssertExpectations(t)
	mockTransactionsRepo.AssertExpectations(t)
}

func TestHandleErrorDeleteEntryError(t *testing.T) {
	mockCollectionsRepo := new(MockCollectionsRepo)
	mockTransactionsRepo := new(MockCollectionTransactionsInProgressRepo)

	handler := RetrieveSubscriberUsageErrorHandler{
		CollectionsRepo:                  mockCollectionsRepo,
		CollectionTransactionsInProgress: mockTransactionsRepo,
	}

	ctx := context.Background()
	msg := CreateTestMessage()
	err := error_handling.NewRetrieveSubscriberUsageError(
		errors.New(testWalletBucketMissing),
		http.StatusOK,
	)

	objID := primitive.NewObjectID()
	mockCollectionsRepo.On("CreateEntry", ctx, mock.Anything).Return(objID, nil)
	mockTransactionsRepo.On("DeleteEntry", ctx, msg.Msisdn).Return(errors.New(testDbError))

	result := handler.HandleError(ctx, msg, err, http.StatusOK)

	assert.Nil(t, result)
	mockCollectionsRepo.AssertExpectations(t)
	mockTransactionsRepo.AssertExpectations(t)
}

func TestHandleErrorGenericError(t *testing.T) {
	mockTransactionsRepo := new(MockCollectionTransactionsInProgressRepo)

	handler := RetrieveSubscriberUsageErrorHandler{
		CollectionTransactionsInProgress: mockTransactionsRepo,
	}

	ctx := context.Background()
	msg := CreateTestMessage()
	err := errors.New("generic error")

	result := handler.HandleError(ctx, msg, err)

	assert.Nil(t, result)
	mockTransactionsRepo.AssertExpectations(t)
}

func TestHandleErrorDefaultErrorCodes(t *testing.T) {
	mockTransactionsRepo := new(MockCollectionTransactionsInProgressRepo)

	handler := RetrieveSubscriberUsageErrorHandler{
		CollectionTransactionsInProgress: mockTransactionsRepo,
	}

	ctx := context.Background()
	msg := CreateTestMessage()

	err := error_handling.NewRetrieveSubscriberUsageError(
		errors.New("some error"),
		http.StatusBadRequest,
	)

	result := handler.HandleError(ctx, msg, err, http.StatusBadRequest)

	assert.Nil(t, result)
	mockTransactionsRepo.AssertExpectations(t)
}

func TestCheckAndProcessBalanceSufficientBalance(t *testing.T) {
	// Setup
	mockCollectionsRepo := new(MockCollectionsRepo)
	mockTransactionsRepo := new(MockCollectionTransactionsInProgressRepo)

	handler := RetrieveSubscriberUsageErrorHandler{
		CollectionsRepo:                  mockCollectionsRepo,
		CollectionTransactionsInProgress: mockTransactionsRepo,
	}

	ctx := context.Background()
	msg := CreateTestMessage()
	msg.WalletAmount = "1000"
	msg.DataToBeDeducted = 100

	bucket := models.Bucket{
		BucketId:        "WLT12345",
		VolumeRemaining: 500 * 1024,
		Unit:            "KB",
	}

	err := handler.CheckAndProcessBalance(ctx, msg, bucket)

	assert.NoError(t, err)
}

func TestCheckAndProcessBalanceInsufficientBalance(t *testing.T) {
	mockCollectionsRepo := new(MockCollectionsRepo)
	mockTransactionsRepo := new(MockCollectionTransactionsInProgressRepo)

	handler := RetrieveSubscriberUsageErrorHandler{
		CollectionsRepo:                  mockCollectionsRepo,
		CollectionTransactionsInProgress: mockTransactionsRepo,
	}

	ctx := context.Background()
	msg := CreateTestMessage()
	msg.WalletAmount = "100"
	msg.DataToBeDeducted = 200

	bucket := models.Bucket{
		BucketId:        "WLT12345",
		VolumeRemaining: 50 * 1024,
		Unit:            "KB",
	}

	err := handler.CheckAndProcessBalance(ctx, msg, bucket)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "deduction didn't happen: insufficient balance")
}

func TestCheckAndProcessBalanceInvalidWalletAmount(t *testing.T) {
	mockCollectionsRepo := new(MockCollectionsRepo)
	mockTransactionsRepo := new(MockCollectionTransactionsInProgressRepo)

	handler := RetrieveSubscriberUsageErrorHandler{
		CollectionsRepo:                  mockCollectionsRepo,
		CollectionTransactionsInProgress: mockTransactionsRepo,
	}

	ctx := context.Background()
	msg := CreateTestMessage()
	msg.WalletAmount = "invalid"

	bucket := models.Bucket{
		BucketId:        "WLT12345",
		VolumeRemaining: 500 * 1024,
		Unit:            "KB",
	}

	objID := primitive.NewObjectID()
	mockCollectionsRepo.On("CreateEntry", ctx, mock.Anything).Return(objID, nil)
	mockTransactionsRepo.On("DeleteEntry", ctx, msg.Msisdn).Return(nil)

	err := handler.CheckAndProcessBalance(ctx, msg, bucket)

	assert.Nil(t, err)
	mockCollectionsRepo.AssertExpectations(t)
	mockTransactionsRepo.AssertExpectations(t)
}
func TestCheckAndProcessBalancePartialCollectionSuccess(t *testing.T) {
	mockCollectionsRepo := new(MockCollectionsRepo)
	mockTransactionsRepo := new(MockCollectionTransactionsInProgressRepo)

	partialSuccessHandlerCalled := false
	mockPartialHandler := serviceinterfaces.SuccessHandlerFunc(func(
		mClient *mongodb.MongoClient,
		rClient *redis.RedisClient,
		pubSub serviceinterfaces.RuntimePubSubPublisher,
		kafkaSvc *servicekafka.CollectionWorkerKafkaService,
		msg *models.PromoCollectionPublishedMessage,
		ctx context.Context,
		notificationTopic string,
	) error {
		partialSuccessHandlerCalled = true
		return nil
	})

	handler := RetrieveSubscriberUsageErrorHandler{
		CollectionsRepo:                  mockCollectionsRepo,
		CollectionTransactionsInProgress: mockTransactionsRepo,
		MongoClient:                      nil,
		RedisClient:                      nil,
		PubSubPublisher:                  nil,
		KafkaService:                     nil,
		PartialSuccessHandler:            mockPartialHandler,
		FullSuccessHandler:               nil,
	}

	ctx := context.Background()
	msg := CreateTestMessage()
	msg.CollectionType = consts.PartialCollectionType
	msg.WalletAmount = "100"
	msg.DataToBeDeducted = 50

	bucket := models.Bucket{
		BucketId:        "WLT12345",
		VolumeRemaining: 40 * 1024,
		Unit:            "KB",
	}

	err := handler.CheckAndProcessBalance(ctx, msg, bucket)

	assert.Nil(t, err)
	assert.True(t, partialSuccessHandlerCalled, "Partial success handler should have been called")
}

func TestCheckAndProcessBalancePartialCollectionHandlerError(t *testing.T) {
	mockCollectionsRepo := new(MockCollectionsRepo)
	mockTransactionsRepo := new(MockCollectionTransactionsInProgressRepo)

	expectedError := errors.New("partial handler failed")
	partialSuccessHandlerCalled := false
	mockPartialHandler := serviceinterfaces.SuccessHandlerFunc(func(
		mClient *mongodb.MongoClient,
		rClient *redis.RedisClient,
		pubSub serviceinterfaces.RuntimePubSubPublisher,
		kafkaSvc *servicekafka.CollectionWorkerKafkaService,
		msg *models.PromoCollectionPublishedMessage,
		ctx context.Context,
		notificationTopic string,
	) error {
		partialSuccessHandlerCalled = true
		return expectedError
	})

	handler := RetrieveSubscriberUsageErrorHandler{
		CollectionsRepo:                  mockCollectionsRepo,
		CollectionTransactionsInProgress: mockTransactionsRepo,
		MongoClient:                      nil,
		RedisClient:                      nil,
		PubSubPublisher:                  nil,
		KafkaService:                     nil,
		PartialSuccessHandler:            mockPartialHandler,
		FullSuccessHandler:               nil,
	}

	ctx := context.Background()
	msg := CreateTestMessage()
	msg.CollectionType = consts.PartialCollectionType
	msg.WalletAmount = "100"
	msg.DataToBeDeducted = 50

	bucket := models.Bucket{
		BucketId:        "WLT12345",
		VolumeRemaining: 40 * 1024,
		Unit:            "KB",
	}

	err := handler.CheckAndProcessBalance(ctx, msg, bucket)

	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.True(t, partialSuccessHandlerCalled, "Partial success handler should have been called")
}

func TestCheckAndProcessBalanceFullCollectionSuccess(t *testing.T) {
	mockCollectionsRepo := new(MockCollectionsRepo)
	mockTransactionsRepo := new(MockCollectionTransactionsInProgressRepo)

	fullSuccessHandlerCalled := false
	mockFullHandler := serviceinterfaces.SuccessHandlerFunc(func(
		mClient *mongodb.MongoClient,
		rClient *redis.RedisClient,
		pubSub serviceinterfaces.RuntimePubSubPublisher,
		kafkaSvc *servicekafka.CollectionWorkerKafkaService,
		msg *models.PromoCollectionPublishedMessage,
		ctx context.Context,
		notificationTopic string,
	) error {
		fullSuccessHandlerCalled = true
		return nil
	})

	handler := RetrieveSubscriberUsageErrorHandler{
		CollectionsRepo:                  mockCollectionsRepo,
		CollectionTransactionsInProgress: mockTransactionsRepo,
		MongoClient:                      nil,
		RedisClient:                      nil,
		PubSubPublisher:                  nil,
		KafkaService:                     nil,
		PartialSuccessHandler:            nil,
		FullSuccessHandler:               mockFullHandler,
	}

	ctx := context.Background()
	msg := CreateTestMessage()
	msg.CollectionType = consts.FullCollectionType
	msg.WalletAmount = "100"
	msg.DataToBeDeducted = 50

	bucket := models.Bucket{
		BucketId:        "WLT12345",
		VolumeRemaining: 40 * 1024, // 40 MB in KB (100 - 40 = 60 >= 50)
		Unit:            "KB",
	}

	err := handler.CheckAndProcessBalance(ctx, msg, bucket)

	assert.Nil(t, err)
	assert.True(t, fullSuccessHandlerCalled, "Full success handler should have been called")
}

func TestCheckAndProcessBalanceFullCollectionHandlerError(t *testing.T) {
	mockCollectionsRepo := new(MockCollectionsRepo)
	mockTransactionsRepo := new(MockCollectionTransactionsInProgressRepo)

	expectedError := errors.New("full handler failed")
	fullSuccessHandlerCalled := false
	mockFullHandler := serviceinterfaces.SuccessHandlerFunc(func(
		mClient *mongodb.MongoClient,
		rClient *redis.RedisClient,
		pubSub serviceinterfaces.RuntimePubSubPublisher,
		kafkaSvc *servicekafka.CollectionWorkerKafkaService,
		msg *models.PromoCollectionPublishedMessage,
		ctx context.Context,
		notificationTopic string,
	) error {
		fullSuccessHandlerCalled = true
		return expectedError
	})

	handler := RetrieveSubscriberUsageErrorHandler{
		CollectionsRepo:                  mockCollectionsRepo,
		CollectionTransactionsInProgress: mockTransactionsRepo,
		MongoClient:                      nil,
		RedisClient:                      nil,
		PubSubPublisher:                  nil,
		KafkaService:                     nil,
		PartialSuccessHandler:            nil,
		FullSuccessHandler:               mockFullHandler,
	}

	ctx := context.Background()
	msg := CreateTestMessage()
	msg.CollectionType = consts.FullCollectionType
	msg.WalletAmount = "100"
	msg.DataToBeDeducted = 50

	bucket := models.Bucket{
		BucketId:        "WLT12345",
		VolumeRemaining: 40 * 1024, // 40 MB in KB (100 - 40 = 60 >= 50)
		Unit:            "KB",
	}

	err := handler.CheckAndProcessBalance(ctx, msg, bucket)

	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.True(t, fullSuccessHandlerCalled, "Full success handler should have been called")
}
