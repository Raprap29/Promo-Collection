package kafka

import (
	"context"
	"errors"
	"promo-collection-worker/internal/pkg/models"
	"testing"
	"time"
)

const (
	expectedNoErrorMsg = "expected no error, got %v"
	emptyMessage       = "empty message"
	emptyMessageTest   = "empty message"
)

// MockKafkaProducer is a mock implementation of kafka.KafkaProducer for testing.
type MockKafkaProducer struct {
	PublishFunc func(ctx context.Context, data []byte) error
}

func (m *MockKafkaProducer) Publish(ctx context.Context, data []byte) error {
	if m.PublishFunc != nil {
		return m.PublishFunc(ctx, data)
	}
	return nil
}

func TestCollectionWorkerKafkaServicePublishSuccess(t *testing.T) {
	ctx := context.Background()

	t.Run("successful publish", func(t *testing.T) {
		mockProducer := &MockKafkaProducer{
			PublishFunc: func(ctx context.Context, data []byte) error {
				return nil
			},
		}
		svc := NewCollectionWorkerKafkaService(mockProducer)

		msg := models.KafkaMessageForPublishing{
			TransactionId:       "txn123",
			AvailmentId:         "avail123",
			CollectionDateTime:  time.Now(),
			Method:              "method1",
			LoanAgeing:          30,
			CollectedLoanAmount: 100.0,
			CollectedServiceFee: 10.0,
			UnpaidLoanAmount:    200.0,
			UnpaidServiceFee:    20.0,
			Result:              "success",
			CollectionErrorText: "",
			MSISDN:              "1234567890",
			CollectionCategory:  "category1",
			PaymentChannel:      "channel1",
			CollectionType:      "type1",
			CollectedAmount:     110.0,
			TotalUnpaid:         220.0,
			TokenPaymentId:      "token123",
			DataCollected:       50.0,
		}

		err := svc.PublishSuccess(ctx, msg)
		if err != nil {
			t.Errorf(expectedNoErrorMsg, err)
		}
	})

	t.Run("publish error", func(t *testing.T) {
		mockProducer := &MockKafkaProducer{
			PublishFunc: func(ctx context.Context, data []byte) error {
				return errors.New("publish failed")
			},
		}
		svc := NewCollectionWorkerKafkaService(mockProducer)

		msg := models.KafkaMessageForPublishing{
			TransactionId: "txn123",
		}

		err := svc.PublishSuccess(ctx, msg)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})

	t.Run(emptyMessageTest, func(t *testing.T) {
		mockProducer := &MockKafkaProducer{
			PublishFunc: func(ctx context.Context, data []byte) error {
				if len(data) == 0 {
					return errors.New(emptyMessage)
				}
				return nil
			},
		}
		svc := NewCollectionWorkerKafkaService(mockProducer)

		msg := models.KafkaMessageForPublishing{}

		err := svc.PublishSuccess(ctx, msg)
		if err != nil {
			t.Errorf(expectedNoErrorMsg, err)
		}
	})
}

func TestCollectionWorkerKafkaServicePublishFailure(t *testing.T) {
	ctx := context.Background()

	t.Run("successful publish", func(t *testing.T) {
		mockProducer := &MockKafkaProducer{
			PublishFunc: func(ctx context.Context, data []byte) error {
				return nil
			},
		}
		svc := NewCollectionWorkerKafkaService(mockProducer)

		msg := models.KafkaMessageForPublishing{
			TransactionId:       "txn123",
			Result:              "failure",
			CollectionErrorText: "error occurred",
		}

		err := svc.PublishFailure(ctx, msg)
		if err != nil {
			t.Errorf(expectedNoErrorMsg, err)
		}
	})

	t.Run("publish error", func(t *testing.T) {
		mockProducer := &MockKafkaProducer{
			PublishFunc: func(ctx context.Context, data []byte) error {
				return errors.New("publish failed")
			},
		}
		svc := NewCollectionWorkerKafkaService(mockProducer)

		msg := models.KafkaMessageForPublishing{
			TransactionId: "txn123",
		}

		err := svc.PublishFailure(ctx, msg)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})

	t.Run(emptyMessageTest, func(t *testing.T) {
		mockProducer := &MockKafkaProducer{
			PublishFunc: func(ctx context.Context, data []byte) error {
				return nil
			},
		}
		svc := NewCollectionWorkerKafkaService(mockProducer)

		msg := models.KafkaMessageForPublishing{}

		err := svc.PublishFailure(ctx, msg)
		if err != nil {
			t.Errorf(expectedNoErrorMsg, err)
		}
	})

	t.Run("full message", func(t *testing.T) {
		mockProducer := &MockKafkaProducer{
			PublishFunc: func(ctx context.Context, data []byte) error {
				return nil
			},
		}
		svc := NewCollectionWorkerKafkaService(mockProducer)

		msg := models.KafkaMessageForPublishing{
			TransactionId:       "txn456",
			AvailmentId:         "avail456",
			CollectionDateTime:  time.Now(),
			Method:              "method2",
			LoanAgeing:          60,
			CollectedLoanAmount: 200.0,
			CollectedServiceFee: 20.0,
			UnpaidLoanAmount:    400.0,
			UnpaidServiceFee:    40.0,
			Result:              "failure",
			CollectionErrorText: "detailed error",
			MSISDN:              "0987654321",
			CollectionCategory:  "category2",
			PaymentChannel:      "channel2",
			CollectionType:      "type2",
			CollectedAmount:     220.0,
			TotalUnpaid:         440.0,
			TokenPaymentId:      "token456",
			DataCollected:       100.0,
		}

		err := svc.PublishFailure(ctx, msg)
		if err != nil {
			t.Errorf(expectedNoErrorMsg, err)
		}
	})
}

func TestNewCollectionWorkerKafkaService(t *testing.T) {
	mockProducer := &MockKafkaProducer{}
	svc := NewCollectionWorkerKafkaService(mockProducer)
	if svc == nil {
		t.Fatal("expected non-nil service, got nil")
	}
	if svc.KafkaProducer != mockProducer {
		t.Errorf("expected producer to be set, got %v", svc.KafkaProducer)
	}
}
