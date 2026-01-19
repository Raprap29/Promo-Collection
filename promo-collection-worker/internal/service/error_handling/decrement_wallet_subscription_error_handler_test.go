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
	"promo-collection-worker/internal/pkg/downstream/error_handling"
	"promo-collection-worker/internal/pkg/log_messages"
)

const (
	testDbErr          = "db error"
	businessErr        = "Business error"
	businessErrDetails = "Business error details"
)

func TestDecrementWalletErrorHandlerSystemErrorCode(t *testing.T) {
	// Setup
	mockCollectionsRepo := new(MockCollectionsRepo)
	mockTransactionsRepo := new(MockCollectionTransactionsInProgressRepo)

	handler := &DecrementWalletErrorHandler{
		CollectionsRepo:                  mockCollectionsRepo,
		CollectionTransactionsInProgress: mockTransactionsRepo,
		KafkaService:                     nil,
	}

	ctx := context.Background()
	msg := CreateTestMessage()
	err := &error_handling.DecrementWalletSubscriptionError{
		ResultCode:   "9000",
		Description:  "System error",
		ErrorCode:    "9000",
		ErrorDetails: "System error details",
		StatusCode:   http.StatusInternalServerError,
		Err:          errors.New("system error"),
	}

	objID := primitive.NewObjectID()
	mockCollectionsRepo.On("CreateEntry", ctx, mock.Anything).Return(objID, nil)
	action := handler.DecrementWalletErrorHandler(ctx, msg, err)
	assert.Equal(t, consts.ActionIgnore, action)
	mockCollectionsRepo.AssertExpectations(t)
}

func TestDecrementWalletErrorHandlerOutboundConnError(t *testing.T) {
	// Setup
	mockCollectionsRepo := new(MockCollectionsRepo)
	mockTransactionsRepo := new(MockCollectionTransactionsInProgressRepo)

	handler := &DecrementWalletErrorHandler{
		CollectionsRepo:                  mockCollectionsRepo,
		CollectionTransactionsInProgress: mockTransactionsRepo,
		KafkaService:                     nil,
	}

	ctx := context.Background()
	msg := CreateTestMessage()
	err := &error_handling.DecrementWalletSubscriptionError{
		ResultCode:   "500",
		Description:  "Connection error",
		ErrorCode:    log_messages.OutboundConnError,
		ErrorDetails: "Failed to connect to external service",
		StatusCode:   http.StatusInternalServerError,
		Err:          errors.New("connection error"),
	}

	objID := primitive.NewObjectID()
	mockCollectionsRepo.On("CreateEntry", ctx, mock.Anything).Return(objID, nil)
	action := handler.DecrementWalletErrorHandler(ctx, msg, err)
	assert.Equal(t, consts.ActionIgnore, action)
	mockCollectionsRepo.AssertExpectations(t)
}

func TestDecrementWalletErrorHandlerBusinessError(t *testing.T) {
	// Setup
	mockCollectionsRepo := new(MockCollectionsRepo)
	mockTransactionsRepo := new(MockCollectionTransactionsInProgressRepo)

	handler := &DecrementWalletErrorHandler{
		CollectionsRepo:                  mockCollectionsRepo,
		CollectionTransactionsInProgress: mockTransactionsRepo,
		KafkaService:                     nil,
	}

	ctx := context.Background()
	msg := CreateTestMessage()
	err := &error_handling.DecrementWalletSubscriptionError{
		ResultCode:   "1001",
		Description:  businessErr,
		ErrorCode:    "1001",
		ErrorDetails: businessErrDetails,
		StatusCode:   http.StatusOK,
		Err:          errors.New(businessErr),
	}

	objID := primitive.NewObjectID()
	mockCollectionsRepo.On("CreateEntry", ctx, mock.Anything).Return(objID, nil)
	mockTransactionsRepo.On("DeleteEntry", ctx, msg.Msisdn).Return(nil)
	action := handler.DecrementWalletErrorHandler(ctx, msg, err)
	assert.Equal(t, consts.ActionAck, action)
	mockCollectionsRepo.AssertExpectations(t)
	mockTransactionsRepo.AssertExpectations(t)
}

func TestDecrementWalletErrorHandlerDefaultError(t *testing.T) {
	// Setup
	mockCollectionsRepo := new(MockCollectionsRepo)
	mockTransactionsRepo := new(MockCollectionTransactionsInProgressRepo)

	handler := &DecrementWalletErrorHandler{
		CollectionsRepo:                  mockCollectionsRepo,
		CollectionTransactionsInProgress: mockTransactionsRepo,
		KafkaService:                     nil,
	}

	ctx := context.Background()
	msg := CreateTestMessage()
	err := &error_handling.DecrementWalletSubscriptionError{
		ResultCode:   "4000",
		Description:  "Some other error",
		ErrorCode:    "4000",
		ErrorDetails: "Error details",
		StatusCode:   http.StatusBadRequest,
		Err:          errors.New("other error"),
	}

	objID := primitive.NewObjectID()
	mockCollectionsRepo.On("CreateEntry", ctx, mock.Anything).Return(objID, nil)
	mockTransactionsRepo.On("DeleteEntry", ctx, msg.Msisdn).Return(nil)

	action := handler.DecrementWalletErrorHandler(ctx, msg, err)
	assert.Equal(t, consts.ActionAck, action)
	mockCollectionsRepo.AssertExpectations(t)
	mockTransactionsRepo.AssertExpectations(t)
}

func TestDecrementWalletErrorHandlerCreateEntryError(t *testing.T) {
	// Setup
	mockCollectionsRepo := new(MockCollectionsRepo)
	mockTransactionsRepo := new(MockCollectionTransactionsInProgressRepo)

	handler := &DecrementWalletErrorHandler{
		CollectionsRepo:                  mockCollectionsRepo,
		CollectionTransactionsInProgress: mockTransactionsRepo,
		KafkaService:                     nil,
	}

	ctx := context.Background()
	msg := CreateTestMessage()
	err := &error_handling.DecrementWalletSubscriptionError{
		ResultCode:   "1001",
		Description:  businessErr,
		ErrorCode:    "1001",
		ErrorDetails: businessErrDetails,
		StatusCode:   http.StatusOK,
		Err:          errors.New(businessErr),
	}

	mockCollectionsRepo.On("CreateEntry", ctx, mock.Anything).Return(nil, errors.New(testDbErr))
	mockTransactionsRepo.On("DeleteEntry", ctx, msg.Msisdn).Return(nil)

	action := handler.DecrementWalletErrorHandler(ctx, msg, err)

	assert.Equal(t, consts.ActionAck, action)
	mockCollectionsRepo.AssertExpectations(t)
	mockTransactionsRepo.AssertExpectations(t)
}

func TestDecrementWalletErrorHandlerDeleteEntryError(t *testing.T) {
	mockCollectionsRepo := new(MockCollectionsRepo)
	mockTransactionsRepo := new(MockCollectionTransactionsInProgressRepo)

	handler := &DecrementWalletErrorHandler{
		CollectionsRepo:                  mockCollectionsRepo,
		CollectionTransactionsInProgress: mockTransactionsRepo,
		KafkaService:                     nil,
	}

	ctx := context.Background()
	msg := CreateTestMessage()
	err := &error_handling.DecrementWalletSubscriptionError{
		ResultCode:   "1001",
		Description:  businessErr,
		ErrorCode:    "1001",
		ErrorDetails: businessErrDetails,
		StatusCode:   http.StatusOK,
		Err:          errors.New(businessErr),
	}

	objID := primitive.NewObjectID()
	mockCollectionsRepo.On("CreateEntry", ctx, mock.Anything).Return(objID, nil)
	mockTransactionsRepo.On("DeleteEntry", ctx, msg.Msisdn).Return(errors.New(testDbErr))

	action := handler.DecrementWalletErrorHandler(ctx, msg, err)

	assert.Equal(t, consts.ActionAck, action)
	mockCollectionsRepo.AssertExpectations(t)
	mockTransactionsRepo.AssertExpectations(t)
}

func TestDecrementWalletErrorHandlerHttpOkWithErrorCode(t *testing.T) {
	mockCollectionsRepo := new(MockCollectionsRepo)
	mockTransactionsRepo := new(MockCollectionTransactionsInProgressRepo)

	handler := &DecrementWalletErrorHandler{
		CollectionsRepo:                  mockCollectionsRepo,
		CollectionTransactionsInProgress: mockTransactionsRepo,
		KafkaService:                     nil,
	}

	ctx := context.Background()
	msg := CreateTestMessage()
	err := &error_handling.DecrementWalletSubscriptionError{
		ResultCode:   "2000",
		Description:  "API returned OK but with error code",
		ErrorCode:    "",
		ErrorDetails: "",
		StatusCode:   http.StatusOK,
		Err:          errors.New("api error"),
	}

	objID := primitive.NewObjectID()
	mockCollectionsRepo.On("CreateEntry", ctx, mock.Anything).Return(objID, nil)
	mockTransactionsRepo.On("DeleteEntry", ctx, msg.Msisdn).Return(nil)

	action := handler.DecrementWalletErrorHandler(ctx, msg, err)

	assert.Equal(t, consts.ActionAck, action)
	mockCollectionsRepo.AssertExpectations(t)
	mockTransactionsRepo.AssertExpectations(t)
}

func TestHandleBusinessOrNonOutboundConnError(t *testing.T) {
	mockCollectionsRepo := new(MockCollectionsRepo)
	mockTransactionsRepo := new(MockCollectionTransactionsInProgressRepo)

	handler := &DecrementWalletErrorHandler{
		CollectionsRepo:                  mockCollectionsRepo,
		CollectionTransactionsInProgress: mockTransactionsRepo,
		KafkaService:                     nil,
	}

	ctx := context.Background()
	msg := CreateTestMessage()

	mockTransactionsRepo.On("DeleteEntry", ctx, msg.Msisdn).Return(nil)

	action := handler.HandleBusinessOrNonOutboundConnError(ctx, msg)

	assert.Equal(t, consts.ActionAck, action)
	mockTransactionsRepo.AssertExpectations(t)
}

func TestDecrementWalletErrorHandlerHandleUnknownOrNonAPIError(t *testing.T) {
	mockCollectionsRepo := new(MockCollectionsRepo)
	mockTransactionsRepo := new(MockCollectionTransactionsInProgressRepo)

	handler := &DecrementWalletErrorHandler{
		CollectionsRepo:                  mockCollectionsRepo,
		CollectionTransactionsInProgress: mockTransactionsRepo,
		KafkaService:                     nil,
	}

	ctx := context.Background()
	msg := CreateTestMessage()
	err := errors.New("unknown error format")

	objID := primitive.NewObjectID()
	mockCollectionsRepo.On("CreateEntry", ctx, mock.Anything).Return(objID, nil)
	mockTransactionsRepo.On("DeleteEntry", ctx, msg.Msisdn).Return(nil)

	action := handler.HandleUnknownOrNonAPIError(ctx, msg, err)

	assert.Equal(t, consts.ActionAck, action)
	mockCollectionsRepo.AssertExpectations(t)
	mockTransactionsRepo.AssertExpectations(t)
}

func TestDecrementWalletErrorHandlerHandleUnknownOrNonAPIErrorDBErrors(t *testing.T) {
	mockCollectionsRepo := new(MockCollectionsRepo)
	mockTransactionsRepo := new(MockCollectionTransactionsInProgressRepo)

	handler := &DecrementWalletErrorHandler{
		CollectionsRepo:                  mockCollectionsRepo,
		CollectionTransactionsInProgress: mockTransactionsRepo,
		KafkaService:                     nil,
	}

	ctx := context.Background()
	msg := CreateTestMessage()
	err := errors.New("unknown error format")

	mockCollectionsRepo.On("CreateEntry", ctx, mock.Anything).Return(nil, errors.New(testDbErr))
	mockTransactionsRepo.On("DeleteEntry", ctx, msg.Msisdn).Return(errors.New(testDbErr))

	action := handler.HandleUnknownOrNonAPIError(ctx, msg, err)

	assert.Equal(t, consts.ActionAck, action)
	mockCollectionsRepo.AssertExpectations(t)
	mockTransactionsRepo.AssertExpectations(t)
}
