package error_handling_service

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"promo-collection-worker/internal/pkg/consts"
	"promo-collection-worker/internal/pkg/models"
	dbModels "promo-collection-worker/internal/pkg/store/models"
)

// Mock for CollectionsRepoInterface
type MockCollectionsRepo struct {
	mock.Mock
}

func (m *MockCollectionsRepo) CreateEntry(ctx context.Context, model *dbModels.Collections) (primitive.ObjectID, error) {
	args := m.Called(ctx, model)
	if args.Get(0) == nil {
		return primitive.ObjectID{}, args.Error(1)
	}
	return args.Get(0).(primitive.ObjectID), args.Error(1)
}

func (m *MockCollectionsRepo) UpdatePublishToKafka(ctx context.Context, id primitive.ObjectID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockCollectionsRepo) UpdatePublishedToKafkaInBulk(ctx context.Context, collectionIds []string) ([]string, error) {
	args := m.Called(ctx, collectionIds)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockCollectionsRepo) GetFailedKafkaEntriesCursor(ctx context.Context, duration string, batchSize int32) (*mongo.Cursor, error) {
	args := m.Called(ctx, duration, batchSize)
	return args.Get(0).(*mongo.Cursor), args.Error(1)
}

// Mock for CollectionTransactionsInProgressRepoInterface
type MockCollectionTransactionsInProgressRepo struct {
	mock.Mock
}

func (m *MockCollectionTransactionsInProgressRepo) DeleteEntry(ctx context.Context, msisdn string) error {
	args := m.Called(ctx, msisdn)
	return args.Error(0)
}

func (m *MockCollectionTransactionsInProgressRepo) CreateEntry(ctx context.Context, msisdn string) error {
	args := m.Called(ctx, msisdn)
	return args.Error(0)
}

func (m *MockCollectionTransactionsInProgressRepo) CheckEntryExists(ctx context.Context, msisdn string) (bool, error) {
	args := m.Called(ctx, msisdn)
	return args.Bool(0), args.Error(1)
}

// Helper function to create a test message
func CreateTestMessage() *models.PromoCollectionPublishedMessage {
	loanID, _ := primitive.ObjectIDFromHex("5f50c31f5b5f7c1d5c1d5c1d")
	availmentID, _ := primitive.ObjectIDFromHex("5f50c31f5b5f7c1d5c1d5c1e")
	unpaidLoanID, _ := primitive.ObjectIDFromHex("5f50c31f5b5f7c1d5c1d5c1f")
	brandID, _ := primitive.ObjectIDFromHex("5f50c31f5b5f7c1d5c1d5c20")
	loanProductID, _ := primitive.ObjectIDFromHex("5f50c31f5b5f7c1d5c1d5c21")
	lastCollectionID, _ := primitive.ObjectIDFromHex("5f50c31f5b5f7c1d5c1d5c22")

	return &models.PromoCollectionPublishedMessage{
		Msisdn:                   "09123456789",
		Unit:                     "MB",
		IsRollBack:               false,
		Duration:                 "200",
		Channel:                  "Dodrio",
		SvcId:                    12345,
		Denom:                    "100",
		WalletKeyword:            "TESTDATA",
		WalletAmount:             "100",
		SvcDenomCombined:         "12345100",
		KafkaId:                  67890,
		CollectionType:           "Data",
		Ageing:                   30,
		AvailmentTransactionId:   availmentID,
		LoanId:                   loanID,
		UnpaidLoanId:             unpaidLoanID,
		ServiceFee:               10.0,
		TotalLoanAmountInPeso:    100.0,
		TotalUnpaidAmountInPeso:  50.0,
		AmountToBeDeductedInPeso: 50.0,
		UnpaidServiceFee:         5.0,
		DataToBeDeducted:         100.0,
		LastCollectionDateTime:   time.Now(),
		LastCollectionId:         lastCollectionID,
		StartDate:                time.Now().Add(-30 * 24 * time.Hour),
		BrandId:                  brandID,
		LoanProductId:            loanProductID,
		LoanType:                 consts.LoanTypePromo,
	}
}
