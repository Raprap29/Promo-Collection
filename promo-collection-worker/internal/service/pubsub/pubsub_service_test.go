package pubsub_service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"promo-collection-worker/internal/pkg/consts"
	"promo-collection-worker/internal/pkg/models"
)

// --- Mock downstream clients ---
type MockDecrementWalletClient struct {
	mock.Mock
}

func (m *MockDecrementWalletClient) DecrementWalletSubscription(ctx context.Context, req *models.DecrementWalletSubscriptionRequest) (*models.DecrementWalletSubscriptionSuccess, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DecrementWalletSubscriptionSuccess), args.Error(1)
}

type MockRetrieveSubscriberUsageClient struct {
	mock.Mock
}

func (m *MockRetrieveSubscriberUsageClient) RetrieveSubscriberUsage(ctx context.Context, req *models.RetrieveSubscriberUsageRequest) (*models.RetrieveSubscriberUsageSuccess, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RetrieveSubscriberUsageSuccess), args.Error(1)
}

// Helper function to create a valid message
func createValidMessage() models.PromoCollectionPublishedMessage {
	return models.PromoCollectionPublishedMessage{
		Msisdn:           "639175884175",
		WalletKeyword:    "WLT12300",
		WalletAmount:     "1000",
		Unit:             "MB",
		IsRollBack:       false,
		Duration:         "200",
		Channel:          "Dodrio",
		DataToBeDeducted: 100,
	}
}

func TestMessageIgnoreError(t *testing.T) {
	originalErr := errors.New("original error")
	ignoreErr := &MessageIgnoreError{Err: originalErr}

	assert.Contains(t, ignoreErr.Error(), "message ignored for redelivery")
	assert.Contains(t, ignoreErr.Error(), originalErr.Error())
}

func TestMapToDecrementWalletSubscriptionRequest(t *testing.T) {
	msg := createValidMessage()
	req := mapToDecrementWalletSubscriptionRequest(&msg)

	assert.Equal(t, msg.Msisdn, req.MSISDN)
	assert.Equal(t, msg.WalletKeyword, req.Wallet)
	expectedAmount := fmt.Sprintf("%.0f", msg.DataToBeDeducted*1000)
	assert.Equal(t, expectedAmount, req.Amount)
	assert.Equal(t, consts.Unit, req.Unit)
	assert.Equal(t, msg.IsRollBack, req.IsRollBack)
	assert.Equal(t, msg.Duration, req.Duration)
	assert.Equal(t, msg.Channel, req.Channel)
}

func TestMapToRetrieveSubscriberUsageRequest(t *testing.T) {
	msisdn := "639175884175"
	req := mapToRetrieveSubscriberUsageRequest(msisdn)

	assert.Equal(t, msisdn, req.MSISDN)
	assert.Equal(t, consts.IdType, req.IdType)
	assert.Equal(t, consts.QueryType, req.QueryType)
	assert.Equal(t, consts.DetailsFlag, req.DetailsFlag)
	assert.Equal(t, consts.SubscriberType, req.SubscriptionType)
}

func TestHandlePromoCollectionMessage(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid JSON", func(t *testing.T) {
		consumer := &PromoCollectionMessageConsumer{}
		msg := []byte("{invalid-json}")
		err := consumer.HandlePromoCollectionMessage(ctx, msg)
		assert.Error(t, err)
	})

	t.Run("validation error", func(t *testing.T) {
		consumer := &PromoCollectionMessageConsumer{}
		msg := models.PromoCollectionPublishedMessage{
			// Missing Msisdn which is required
			WalletKeyword: "WLT12300",
			WalletAmount:  "1000",
			Unit:          "MB",
			Channel:       "Dodrio",
		}
		data, _ := json.Marshal(msg)
		err := consumer.HandlePromoCollectionMessage(ctx, data)
		assert.Error(t, err)
	})

	t.Run("successful message handling", func(t *testing.T) {
		// Setup mocks
		mockDecrementClient := new(MockDecrementWalletClient)
		mockRetrieveClient := new(MockRetrieveSubscriberUsageClient)

		// Set expectations for successful wallet decrement
		successResp := &models.DecrementWalletSubscriptionSuccess{
			ResultCode:  "0",
			Description: "Success",
		}
		mockDecrementClient.On("DecrementWalletSubscription", mock.Anything, mock.Anything).Return(successResp, nil)

		// Create consumer
		consumer := &PromoCollectionMessageConsumer{
			decrementSubscriptionClient:   mockDecrementClient,
			retrieveSubscriberUsageClient: mockRetrieveClient,
		}

		// Test with valid message
		msg := createValidMessage()
		data, _ := json.Marshal(msg)

		err := consumer.HandlePromoCollectionMessage(ctx, data)

		assert.NoError(t, err)
		mockDecrementClient.AssertExpectations(t)
	})
}

func TestRequestWallet(t *testing.T) {
	ctx := context.Background()
	msg := createValidMessage()

	t.Run("successful wallet decrement", func(t *testing.T) {
		// Setup mocks
		mockDecrementClient := new(MockDecrementWalletClient)
		mockRetrieveClient := new(MockRetrieveSubscriberUsageClient)

		// Set expectations
		successResp := &models.DecrementWalletSubscriptionSuccess{
			ResultCode:  "0",
			Description: "Success",
		}
		mockDecrementClient.On("DecrementWalletSubscription", mock.Anything, mock.Anything).Return(successResp, nil)

		// Create consumer
		consumer := &PromoCollectionMessageConsumer{
			decrementSubscriptionClient:   mockDecrementClient,
			retrieveSubscriberUsageClient: mockRetrieveClient,
		}

		// Test
		action, err := consumer.RequestWallet(ctx, &msg)

		// Assertions
		assert.NoError(t, err)
		assert.Equal(t, consts.ActionAck, action)
		mockDecrementClient.AssertExpectations(t)
	})
}

func TestHandleActionResult(t *testing.T) {
	consumer := &PromoCollectionMessageConsumer{}
	originalErr := errors.New("original error")

	t.Run("action ack", func(t *testing.T) {
		err := consumer.handleActionResult(consts.ActionAck, originalErr)
		assert.NoError(t, err)
	})

	t.Run("action nack", func(t *testing.T) {
		err := consumer.handleActionResult(consts.ActionNack, originalErr)
		assert.Error(t, err)
		assert.Equal(t, originalErr, err)
	})

	t.Run("action ignore", func(t *testing.T) {
		err := consumer.handleActionResult(consts.ActionIgnore, originalErr)
		assert.Error(t, err)
		assert.IsType(t, &MessageIgnoreError{}, err)
		msgIgnoreErr, ok := err.(*MessageIgnoreError)
		assert.True(t, ok)
		assert.Equal(t, originalErr, msgIgnoreErr.Err)
	})

	t.Run("unknown action", func(t *testing.T) {
		err := consumer.handleActionResult("unknown", originalErr)
		assert.NoError(t, err)
	})
}
