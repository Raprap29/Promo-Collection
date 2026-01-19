package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"promo-collection-worker/internal/pkg/config"
	"promo-collection-worker/internal/pkg/log_messages"
	"promo-collection-worker/internal/pkg/store/models"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type MockCollectionsRepo struct {
	mock.Mock
}

func (m *MockCollectionsRepo) CreateEntry(ctx context.Context, model *models.Collections) (primitive.ObjectID, error) {
	args := m.Called(ctx, model)
	return args.Get(0).(primitive.ObjectID), args.Error(1)
}
func (m *MockCollectionsRepo) UpdatePublishToKafka(ctx context.Context, id primitive.ObjectID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *MockCollectionsRepo) UpdatePublishedToKafkaInBulk(ctx context.Context, collectionIds []string) ([]string, error) {
	args := m.Called(ctx, collectionIds)
	if args.Get(0) != nil {
		return args.Get(0).([]string), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *MockCollectionsRepo) GetFailedKafkaEntriesCursor(ctx context.Context, duration string, batchSize int32) (*mongo.Cursor, error) {
	args := m.Called(ctx, duration, batchSize)
	if args.Get(0) != nil {
		return args.Get(0).(*mongo.Cursor), args.Error(1)
	}
	return nil, args.Error(1)
}

type MockKafkaProducer struct {
	mock.Mock
}

func (m *MockKafkaProducer) Publish(ctx context.Context, msg []byte) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

type MongoCursor struct {
	documents []models.CollectionWithLoanGUID
	index     int
	err       error
}

func NewMongoCursor(docs []models.CollectionWithLoanGUID) *mongo.Cursor {
	return nil
}

func (m *MongoCursor) Next(ctx context.Context) bool {
	return m.index < len(m.documents)
}

func (m *MongoCursor) Decode(val interface{}) error {
	if m.index >= len(m.documents) {
		return errors.New("no more docs")
	}
	if c, ok := val.(*models.CollectionWithLoanGUID); ok {
		*c = m.documents[m.index]
		m.index++
		return nil
	}
	return errors.New("invalid decode type")
}

func (m *MongoCursor) Err() error                                         { return m.err }
func (m *MongoCursor) Close(ctx context.Context) error                    { return nil }
func (m *MongoCursor) ID() int64                                          { return 0 }
func (m *MongoCursor) All(ctx context.Context, results interface{}) error { return nil }
func (m *MongoCursor) TryNext(ctx context.Context) bool                   { return false }
func (m *MongoCursor) RemainingBatchLength() int                          { return 0 }
func (m *MongoCursor) SetBatchSize(int32)                                 {}

type MockCursorHandler struct {
	mock.Mock
}

func (m *MockCursorHandler) StreamDocuments(ctx context.Context, cursor *mongo.Cursor, docChan chan<- models.CollectionWithLoanGUID, errorChan chan<- error) {
	m.Called(ctx, cursor, docChan, errorChan)
}
func (m *MockCursorHandler) DecodeDocument(cursor *mongo.Cursor) (models.CollectionWithLoanGUID, error) {
	args := m.Called(cursor)
	return args.Get(0).(models.CollectionWithLoanGUID), args.Error(1)
}

func createTestCollectionWithLoanGUID(id string) models.CollectionWithLoanGUID {
	availmentID, _ := primitive.ObjectIDFromHex("60d21b5b5058e12345678901")
	loanID, _ := primitive.ObjectIDFromHex("60d21b5b5058e12345678902")
	unpaidLoanID, _ := primitive.ObjectIDFromHex("60d21b5b5058e12345678903")

	return models.CollectionWithLoanGUID{
		ID:                     primitive.NewObjectID(),
		MSISDN:                 "09123456789",
		Ageing:                 30,
		AvailmentTransactionId: availmentID,
		CollectedAmount:        100.0,
		CollectionCategory:     "regular",
		CollectionType:         "full",
		CreatedAt:              time.Now(),
		ErrorText:              "",
		LoanId:                 loanID,
		Method:                 "auto",
		PaymentChannel:         "wallet",
		PublishedToKafka:       false,
		Result:                 true,
		ServiceFee:             10.0,
		TokenPaymentId:         "token123",
		TotalCollectedAmount:   100.0,
		UnpaidLoanId:           unpaidLoanID,
		TotalUnpaid:            0.0,
		LoanGUID:               id,
	}
}

func createTestConfig() config.KafkaRetryServiceConfig {
	return config.KafkaRetryServiceConfig{
		RetryStartDate: "2025-10-01",
		WorkerCount:    5,
		BufferSize:     10,
		MaxBatchSize:   20,
		MongoBatchSize: 100,
		FlushInterval:  500 * time.Millisecond,
	}
}

func TestNewKafkaRetryServiceWithDeps(t *testing.T) {
	repo := &MockCollectionsRepo{}
	kafka := &MockKafkaProducer{}
	cfg := config.KafkaRetryServiceConfig{
		RetryStartDate: "2025-10-01",
		WorkerCount:    5,
		BufferSize:     10,
		MaxBatchSize:   20,
		MongoBatchSize: 100,
	}

	svc := NewKafkaRetryServiceWithDeps(repo, kafka, cfg)
	assert.NotNil(t, svc)
	assert.Equal(t, repo, svc.CollectionRepo)
	assert.Equal(t, kafka, svc.kafkaProducer)
	assert.Equal(t, cfg, svc.workerConfig)
}

func TestKafkaRetryResponse_Error(t *testing.T) {
	resp := &KafkaRetryResponse{
		ErrorMsg: "test error",
	}
	assert.Equal(t, "test error", resp.ErrorMsg)

	resp2 := &KafkaRetryResponse{
		ErrorMsg: "",
	}
	assert.Equal(t, "", resp2.ErrorMsg)
}

func TestRetryKafkaCollectionMessages(t *testing.T) {

	t.Run("Zero workers configuration", func(t *testing.T) {
		repo := &MockCollectionsRepo{}
		kafka := &MockKafkaProducer{}

		cfg := config.KafkaRetryServiceConfig{
			RetryStartDate: "2025-10-01",
			WorkerCount:    0,
			BufferSize:     10,
			MaxBatchSize:   20,
			MongoBatchSize: 100,
			FlushInterval:  500 * time.Millisecond,
		}

		svc := NewKafkaRetryServiceWithDeps(repo, kafka, cfg)

		resp := svc.RetryKafkaCollectionMessages(context.Background())

		assert.NotEmpty(t, resp.ErrorMsg)
		assert.Contains(t, resp.ErrorMsg, log_messages.NoWorkerConfigured)

		assert.Empty(t, resp.SuccessIDs)
		assert.Empty(t, resp.FailedIDs)
	})

	t.Run("Cursor error", func(t *testing.T) {
		repo := &MockCollectionsRepo{}
		kafka := &MockKafkaProducer{}
		repo.On("GetFailedKafkaEntriesCursor", mock.Anything, "2025-10-01", int32(100)).Return(nil, errors.New("cursor error"))

		svc := NewKafkaRetryServiceWithDeps(repo, kafka, createTestConfig())
		resp := svc.RetryKafkaCollectionMessages(context.Background())

		assert.NotNil(t, resp.ErrorMsg)
		assert.Equal(t, "cursor error", resp.ErrorMsg)
		repo.AssertExpectations(t)
	})

	t.Run("Nil cursor", func(t *testing.T) {
		repo := &MockCollectionsRepo{}
		kafka := &MockKafkaProducer{}
		repo.On("GetFailedKafkaEntriesCursor", mock.Anything, "2025-10-01", int32(100)).Return(nil, nil)

		svc := NewKafkaRetryServiceWithDeps(repo, kafka, createTestConfig())
		resp := svc.RetryKafkaCollectionMessages(context.Background())

		assert.Equal(t, "", resp.ErrorMsg)
		assert.Empty(t, resp.SuccessIDs)
		assert.Empty(t, resp.FailedIDs)
		repo.AssertExpectations(t)
	})
}

func TestProcessAndPublishBatch(t *testing.T) {
	t.Run("Empty batch", func(t *testing.T) {
		repo := &MockCollectionsRepo{}
		kafka := &MockKafkaProducer{}
		svc := NewKafkaRetryServiceWithDeps(repo, kafka, createTestConfig())

		success := make(chan []string, 1)
		fail := make(chan []string, 1)
		errch := make(chan error, 1)

		svc.processAndPublishBatch(context.Background(), []models.CollectionWithLoanGUID{}, success, fail, errch)

		select {
		case <-success:
			t.Error("unexpected success")
		case <-fail:
			t.Error("unexpected fail")
		case <-errch:
			t.Error("unexpected error")
		default:
		}
	})

	t.Run("Publish error", func(t *testing.T) {
		repo := &MockCollectionsRepo{}
		kafka := &MockKafkaProducer{}
		svc := NewKafkaRetryServiceWithDeps(repo, kafka, createTestConfig())

		doc := createTestCollectionWithLoanGUID("60d21b5b5058e12345678900")

		kafka.On("Publish", mock.Anything, mock.Anything).Return(errors.New("publish error"))

		repo.On("UpdatePublishedToKafkaInBulk", mock.Anything, []string{}).Return([]string{}, nil)
		repo.On("UpdatePublishedToKafkaInBulk", mock.Anything, []string{doc.ID.Hex()}).Return([]string{}, nil)

		success := make(chan []string, 1)
		fail := make(chan []string, 1)
		errch := make(chan error, 1)

		svc.processAndPublishBatch(context.Background(), []models.CollectionWithLoanGUID{doc}, success, fail, errch)

		select {
		case e := <-errch:
			assert.Contains(t, e.Error(), "publish error")
		case <-time.After(100 * time.Millisecond):
			t.Error("expected publish error")
		}

		select {
		case f := <-fail:
			assert.Contains(t, f, doc.ID.Hex())
		case <-time.After(100 * time.Millisecond):
			t.Error("expected failed ID")
		}
	})

	t.Run("Success", func(t *testing.T) {
		repo := &MockCollectionsRepo{}
		kafka := &MockKafkaProducer{}
		svc := NewKafkaRetryServiceWithDeps(repo, kafka, createTestConfig())

		doc := createTestCollectionWithLoanGUID("60d21b5b5058e12345678900")

		kafka.On("Publish", mock.Anything, mock.Anything).Return(nil)
		repo.On("UpdatePublishedToKafkaInBulk", mock.Anything, []string{doc.ID.Hex()}).Return([]string{}, nil)

		success := make(chan []string, 1)
		fail := make(chan []string, 1)
		errch := make(chan error, 1)

		svc.processAndPublishBatch(context.Background(), []models.CollectionWithLoanGUID{doc}, success, fail, errch)

		select {
		case s := <-success:
			assert.Contains(t, s, doc.ID.Hex())
		case <-time.After(100 * time.Millisecond):
			t.Error("expected success")
		}

		select {
		case <-fail:
			t.Error("unexpected failure")
		case <-errch:
			t.Error("unexpected error")
		default:
		}
	})

	t.Run("Update error", func(t *testing.T) {
		repo := &MockCollectionsRepo{}
		kafka := &MockKafkaProducer{}
		svc := NewKafkaRetryServiceWithDeps(repo, kafka, createTestConfig())

		doc := createTestCollectionWithLoanGUID("60d21b5b5058e12345678900")

		kafka.On("Publish", mock.Anything, mock.Anything).Return(nil)
		repo.On("UpdatePublishedToKafkaInBulk", mock.Anything, []string{doc.ID.Hex()}).Return(
			[]string{}, errors.New("update error"))

		success := make(chan []string, 1)
		fail := make(chan []string, 1)
		errch := make(chan error, 1)

		svc.processAndPublishBatch(context.Background(), []models.CollectionWithLoanGUID{doc}, success, fail, errch)

		select {
		case e := <-errch:
			assert.Contains(t, e.Error(), "update error")
		case <-time.After(100 * time.Millisecond):
			t.Error("expected error")
		}

		select {
		case s := <-success:
			assert.Contains(t, s, doc.ID.Hex())
		case <-time.After(100 * time.Millisecond):
			t.Error("expected success despite DB update error")
		}

		select {
		case <-fail:
			t.Error("unexpected failure")
		default:
		}
	})
}

func TestHandleSuccessIDs(t *testing.T) {
	t.Run("Both success and failed", func(t *testing.T) {
		repo := &MockCollectionsRepo{}
		kafka := &MockKafkaProducer{}
		svc := NewKafkaRetryServiceWithDeps(repo, kafka, createTestConfig())

		successIDs := []string{"id1"}
		failedIDs := []string{"id2"}

		repo.On("UpdatePublishedToKafkaInBulk", mock.Anything, successIDs).Return([]string{}, nil)

		success := make(chan []string, 1)
		fail := make(chan []string, 1)
		errch := make(chan error, 1)

		svc.handleSuccessIDs(context.Background(), successIDs, failedIDs, success, fail, errch)

		var successReceived, failureReceived bool
		for i := 0; i < 2; i++ {
			select {
			case s := <-success:
				assert.Equal(t, successIDs, s)
				successReceived = true
			case f := <-fail:
				assert.Equal(t, failedIDs, f)
				failureReceived = true
			case <-time.After(100 * time.Millisecond):
			}
		}

		assert.True(t, successReceived, "Should have received success IDs")
		assert.True(t, failureReceived, "Should have received failure IDs")
	})
}

func TestProcessDocumentsWorker(t *testing.T) {
	t.Run("Batch processing", func(t *testing.T) {
		repo := &MockCollectionsRepo{}
		kafka := &MockKafkaProducer{}

		cfg := config.KafkaRetryServiceConfig{
			RetryStartDate: "2025-10-01",
			WorkerCount:    1,
			BufferSize:     10,
			MaxBatchSize:   2,
			MongoBatchSize: 100,
			FlushInterval:  500 * time.Millisecond,
		}

		svc := NewKafkaRetryServiceWithDeps(repo, kafka, cfg)

		docs := []models.CollectionWithLoanGUID{
			createTestCollectionWithLoanGUID("60d21b5b5058e12345678900"),
			createTestCollectionWithLoanGUID("60d21b5b5058e12345678901"),
		}

		kafka.On("Publish", mock.Anything, mock.Anything).Return(nil).Times(2)

		repo.On("UpdatePublishedToKafkaInBulk", mock.Anything, []string{
			docs[0].ID.Hex(), docs[1].ID.Hex(),
		}).Return([]string{}, nil)

		docChan := make(chan models.CollectionWithLoanGUID, 10)
		successChan := make(chan []string, 10)
		failureChan := make(chan []string, 10)
		errorChan := make(chan error, 10)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go svc.processDocumentsWorker(ctx, docChan, successChan, failureChan, errorChan)

		for _, doc := range docs {
			docChan <- doc
		}
		close(docChan)

		time.Sleep(200 * time.Millisecond)

		select {
		case s := <-successChan:
			assert.Len(t, s, 2, "Success message should contain both IDs")
			assert.Contains(t, s, docs[0].ID.Hex(), "Success IDs should include first document")
			assert.Contains(t, s, docs[1].ID.Hex(), "Success IDs should include second document")
		case <-time.After(100 * time.Millisecond):
			t.Error("expected success message")
		}

		select {
		case <-failureChan:
			t.Error("unexpected failure")
		case <-errorChan:
			t.Error("unexpected error")
		default:
		}
	})

	t.Run("Timer flush", func(t *testing.T) {
		repo := &MockCollectionsRepo{}
		kafka := &MockKafkaProducer{}

		cfg := config.KafkaRetryServiceConfig{
			RetryStartDate: "2025-10-01",
			WorkerCount:    1,
			BufferSize:     10,
			MaxBatchSize:   100,
			MongoBatchSize: 100,
			FlushInterval:  100 * time.Millisecond,
		}

		svc := NewKafkaRetryServiceWithDeps(repo, kafka, cfg)

		doc := createTestCollectionWithLoanGUID("60d21b5b5058e12345678900")

		kafka.On("Publish", mock.Anything, mock.Anything).Return(nil)
		repo.On("UpdatePublishedToKafkaInBulk", mock.Anything, []string{doc.ID.Hex()}).Return([]string{}, nil)

		docChan := make(chan models.CollectionWithLoanGUID, 10)
		successChan := make(chan []string, 10)
		failureChan := make(chan []string, 10)
		errorChan := make(chan error, 10)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go svc.processDocumentsWorker(ctx, docChan, successChan, failureChan, errorChan)

		docChan <- doc

		time.Sleep(200 * time.Millisecond)
		close(docChan)

		select {
		case s := <-successChan:
			assert.Contains(t, s, doc.ID.Hex())
		case <-time.After(100 * time.Millisecond):
			t.Error("expected success from timer flush")
		}

		select {
		case <-failureChan:
			t.Error("unexpected failure")
		case <-errorChan:
			t.Error("unexpected error")
		default:
		}
	})

	t.Run("Context cancelled", func(t *testing.T) {
		repo := &MockCollectionsRepo{}
		kafka := &MockKafkaProducer{}
		svc := NewKafkaRetryServiceWithDeps(repo, kafka, createTestConfig())

		docChan := make(chan models.CollectionWithLoanGUID, 10)
		successChan := make(chan []string, 10)
		failureChan := make(chan []string, 10)
		errorChan := make(chan error, 10)

		ctx, cancel := context.WithCancel(context.Background())

		go svc.processDocumentsWorker(ctx, docChan, successChan, failureChan, errorChan)

		cancel()

		time.Sleep(100 * time.Millisecond)

		select {
		case <-successChan:
			t.Error("unexpected success")
		case <-failureChan:
			t.Error("unexpected failure")
		case <-errorChan:
			t.Error("unexpected error")
		default:
		}
	})
}

func TestCollectResults_ContextCancelled(t *testing.T) {
	repo := &MockCollectionsRepo{}
	kafka := &MockKafkaProducer{}
	svc := NewKafkaRetryServiceWithDeps(repo, kafka, createTestConfig())

	response := &KafkaRetryResponse{
		SuccessIDs: []string{},
		FailedIDs:  []string{},
	}

	successChan := make(chan []string, 10)
	failureChan := make(chan []string, 10)
	errorChan := make(chan error, 10)
	resultsDone := make(chan struct{})
	var resultErr []error

	ctx, cancel := context.WithCancel(context.Background())

	go svc.collectResults(ctx, response, successChan, failureChan, errorChan, resultsDone, &resultErr)

	cancel()

	<-resultsDone

	assert.Empty(t, response.SuccessIDs)
	assert.Empty(t, response.FailedIDs)
	assert.Nil(t, resultErr)
}

func TestPrepareKafkaMessages(t *testing.T) {
	ctx := context.Background()

	id1 := primitive.NewObjectID()
	availmentId := primitive.NewObjectID()
	now := time.Now()

	doc := createTestCollectionWithLoanGUID(id1.Hex())
	doc.CreatedAt = now
	doc.AvailmentTransactionId = availmentId
	doc.TransactionId = "test-transaction-id"

	repo := &MockCollectionsRepo{}
	kafka := &MockKafkaProducer{}
	svc := NewKafkaRetryServiceWithDeps(repo, kafka, createTestConfig())

	messageValue, failedID := svc.prepareKafkaMessages(ctx, doc)

	assert.Empty(t, failedID, "Should not have failed ID")
	assert.NotEmpty(t, messageValue, "Should have message value")

	messageStr := string(messageValue)
	assert.Contains(t, messageStr, doc.LoanGUID, "Message should contain loan GUID")
}

func TestKafkaPayload(t *testing.T) {
	id := primitive.NewObjectID()
	availmentId := primitive.NewObjectID()
	loanId := primitive.NewObjectID()
	unpaidLoanId := primitive.NewObjectID()
	now := time.Now()
	dateTimeStr := now.Format("01/02/2006 15:04")

	testCases := []struct {
		name           string
		collection     models.CollectionWithLoanGUID
		expectedFields map[int]string
		expectedLength int
	}{
		{
			name: "Regular collection with true result",
			collection: models.CollectionWithLoanGUID{
				ID:                     id,
				TransactionId:          "txn-1",
				KafkaTransactionId:     "kafka-txn-1",
				AvailmentTransactionId: availmentId,
				LoanId:                 loanId,
				UnpaidLoanId:           unpaidLoanId,
				CreatedAt:              now,
				Method:                 "Manual",
				Ageing:                 30,
				TotalCollectedAmount:   100.5,
				CollectedServiceFee:    10.0,
				TotalUnpaid:            20.0,
				UnpaidServiceFee:       2.0,
				Result:                 true,
				ErrorText:              "",
				MSISDN:                 "123456789",
				CollectionCategory:     "Load",
				PaymentChannel:         "Cash",
				CollectionType:         "Cash",
				CollectedAmount:        100.5,
				UnpaidLoanAmount:       20.0,
				TokenPaymentId:         "token123",
				ServiceFee:             12.0,
				LoanGUID:               "loan123",
			},
			expectedFields: map[int]string{
				0:  "kafka-txn-1",
				1:  "loan123",
				2:  dateTimeStr,
				3:  "Manual",
				4:  "30",
				5:  "101",
				6:  "10",
				7:  "20",
				8:  "2",
				9:  "true",
				10: "",
				11: "123456789",
				12: "Cash",
				13: "Cash",
				14: "Load",
				15: "101",
				16: "18",
				17: "",
				18: "",
			},
			expectedLength: 19,
		},
		{
			name: "Data collection category",
			collection: models.CollectionWithLoanGUID{
				ID:                     id,
				TransactionId:          "txn-2",
				AvailmentTransactionId: availmentId,
				LoanId:                 loanId,
				UnpaidLoanId:           unpaidLoanId,
				CreatedAt:              now,
				Method:                 "Auto",
				Ageing:                 15,
				TotalCollectedAmount:   50.25,
				CollectedServiceFee:    5.0,
				TotalUnpaid:            0,
				UnpaidServiceFee:       0,
				Result:                 true,
				ErrorText:              "",
				MSISDN:                 "987654321",
				CollectionCategory:     "Data",
				PaymentChannel:         "Digital",
				CollectionType:         "Data",
				CollectedAmount:        50.25,
				UnpaidLoanAmount:       10.0,
				TokenPaymentId:         "token456",
				DataCollected:          1024.5,
				LoanGUID:               "loan456",
			},
			expectedFields: map[int]string{
				0:  "txn-2",
				1:  "loan456",
				2:  dateTimeStr,
				3:  "Auto",
				4:  "15",
				5:  "50",
				6:  "5",
				7:  "0",
				8:  "0",
				9:  "true",
				10: "",
				11: "987654321",
				12: "Data",
				13: "Digital",
				14: "Data",
				15: "50",
				16: "10",
				17: "token456",
				18: "1025",
			},
			expectedLength: 19,
		},
		{
			name: "Regular collection with false result",
			collection: models.CollectionWithLoanGUID{
				ID:                     id,
				TransactionId:          "txn-3",
				KafkaTransactionId:     "kafka-txn-3",
				AvailmentTransactionId: availmentId,
				LoanId:                 loanId,
				UnpaidLoanId:           unpaidLoanId,
				CreatedAt:              now,
				Method:                 "Auto",
				Ageing:                 45,
				TotalCollectedAmount:   0,
				CollectedServiceFee:    0,
				TotalUnpaid:            150.75,
				UnpaidServiceFee:       15.0,
				Result:                 false,
				ErrorText:              "Payment failed",
				MSISDN:                 "555123456",
				CollectionCategory:     "Load",
				PaymentChannel:         "Bank",
				CollectionType:         "Partial",
				CollectedAmount:        0,
				UnpaidLoanAmount:       135.75,
				TokenPaymentId:         "token789",
				ServiceFee:             15.0,
				LoanGUID:               "loan789",
			},
			expectedFields: map[int]string{
				0:  "kafka-txn-3",
				1:  "loan789",
				2:  dateTimeStr,
				3:  "Auto",
				4:  "45",
				5:  "0",
				6:  "0",
				7:  "151",
				8:  "15",
				9:  "false",
				10: "Payment failed",
				11: "555123456",
				12: "Partial",
				13: "Bank",
				14: "Load",
				15: "0",
				16: "135",
				17: "",
				18: "",
			},
			expectedLength: 19,
		},
	}

	repo := &MockCollectionsRepo{}
	kafka := &MockKafkaProducer{}
	svc := NewKafkaRetryServiceWithDeps(repo, kafka, createTestConfig())

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := svc.kafkaPayload(tc.collection)

			assert.Equal(t, tc.expectedLength, len(result),
				"Expected %d fields in result, got %d", tc.expectedLength, len(result))

			for idx, expectedValue := range tc.expectedFields {
				assert.Equal(t, expectedValue, result[idx],
					"Field at index %d should be '%s', got '%s'", idx, expectedValue, result[idx])
			}

			if tc.collection.CollectionCategory == "Data" {
				collectedServiceFee := int(tc.collection.CollectedServiceFee)
				assert.Equal(t, fmt.Sprintf("%d", collectedServiceFee), result[6],
					"CollectedServiceFee should be %d", collectedServiceFee)

				unpaidLoanAmount := int(tc.collection.UnpaidLoanAmount)
				assert.Equal(t, fmt.Sprintf("%d", unpaidLoanAmount), result[16],
					"UnpaidLoanAmount should be %d", unpaidLoanAmount)

				dataCollected := int(math.Round(tc.collection.DataCollected))
				assert.Equal(t, fmt.Sprintf("%d", dataCollected), result[18],
					"DataCollected should be %d", dataCollected)
			} else {
				serviceFee := int(tc.collection.ServiceFee)
				unpaidServiceFee := int(tc.collection.UnpaidServiceFee)
				collectedServiceFee := serviceFee - unpaidServiceFee

				if tc.collection.Result {
					assert.Equal(t, fmt.Sprintf("%d", collectedServiceFee), result[6],
						"CollectedServiceFee should be %d", collectedServiceFee)
				}
			}
		})
	}
}

func TestRetryKafkaCollectionMessages_EdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		numDocs      int
		workerCount  int
		bufferSize   int
		maxBatchSize int
		mongoBatch   int
		expectErrMsg bool
	}{
		{
			name:         "No records - empty cursor",
			numDocs:      0,
			workerCount:  1,
			bufferSize:   10,
			maxBatchSize: 10,
			mongoBatch:   100,
		},
		{
			name:         "1 worker, 50 records (less than mongo batch size)",
			numDocs:      50,
			workerCount:  1,
			bufferSize:   50,
			maxBatchSize: 50,
			mongoBatch:   100,
		},
		{
			name:         "3 workers, 100 records (exact mongo batch size)",
			numDocs:      100,
			workerCount:  3,
			bufferSize:   10,
			maxBatchSize: 25,
			mongoBatch:   100,
		},
		{
			name:         "1 worker, 301 records (4 pages)",
			numDocs:      301,
			workerCount:  1,
			bufferSize:   5,
			maxBatchSize: 20,
			mongoBatch:   100,
		},
		{
			name:         "2 workers, 150 records (2 pages)",
			numDocs:      150,
			workerCount:  2,
			bufferSize:   10,
			maxBatchSize: 20,
			mongoBatch:   100,
		},
		{
			name:         "1 worker, 301 records (4 pages)",
			numDocs:      301,
			workerCount:  1,
			bufferSize:   1,
			maxBatchSize: 20,
			mongoBatch:   100,
		},
		{
			name:         "1 worker, 101 records (2 pages)",
			numDocs:      101,
			workerCount:  1,
			bufferSize:   1,
			maxBatchSize: 20,
			mongoBatch:   100,
		},
		{
			name:         "1 worker, 101 records (2 pages)",
			numDocs:      101,
			workerCount:  1,
			bufferSize:   10,
			maxBatchSize: 20,
			mongoBatch:   100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			repo := &MockCollectionsRepo{}
			kafka := &MockKafkaProducer{}

			cfg := config.KafkaRetryServiceConfig{
				RetryStartDate: "2025-10-01",
				WorkerCount:    tt.workerCount,
				BufferSize:     tt.bufferSize,
				MaxBatchSize:   tt.maxBatchSize,
				MongoBatchSize: int32(tt.mongoBatch),
				FlushInterval:  500 * time.Millisecond,
			}

			var docs []interface{}
			for i := 0; i < tt.numDocs; i++ {
				doc := createTestCollectionWithLoanGUID(fmt.Sprintf("60d21b5b5058e1234567%03d", i))
				docs = append(docs, doc)
			}

			cursor, _ := mongo.NewCursorFromDocuments(docs, nil, nil)

			repo.On("GetFailedKafkaEntriesCursor", mock.Anything, "2025-10-01", int32(tt.mongoBatch)).
				Return(cursor, nil)

			if tt.numDocs > 0 {
				kafka.On("Publish", mock.Anything, mock.Anything).
					Return(nil).
					Times(tt.numDocs)

				repo.On("UpdatePublishedToKafkaInBulk", mock.Anything, mock.MatchedBy(func(ids []string) bool {
					return len(ids) > 0 && len(ids) <= tt.maxBatchSize
				})).
					Return([]string{}, nil).
					Maybe()
			}

			svc := NewKafkaRetryServiceWithDeps(repo, kafka, cfg)

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			resp := svc.RetryKafkaCollectionMessages(ctx)

			if tt.expectErrMsg {
				assert.NotEmpty(t, resp.ErrorMsg)
			} else {
				assert.Empty(t, resp.ErrorMsg)
			}

			assert.NotNil(t, resp)
			assert.Equal(t, tt.numDocs, len(resp.SuccessIDs)+len(resp.FailedIDs))

			if tt.numDocs > 0 {
				assert.Len(t, resp.SuccessIDs, tt.numDocs)
				assert.Empty(t, resp.FailedIDs)
			} else {
				assert.Empty(t, resp.SuccessIDs)
				assert.Empty(t, resp.FailedIDs)
			}

			repo.AssertExpectations(t)
			kafka.AssertExpectations(t)
		})
	}
}
