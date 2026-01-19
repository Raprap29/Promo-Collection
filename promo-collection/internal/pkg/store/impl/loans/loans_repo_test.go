package loans

import (
	"context"
	"fmt"
	"promocollection/internal/pkg/store/models"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockMongoRepository struct {
	mock.Mock
}

func (m *MockMongoRepository) InsertOne(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
	args := m.Called(ctx, document, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mongo.InsertOneResult), args.Error(1)
}

func (m *MockMongoRepository) FindOne(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) *mongo.SingleResult {
	args := m.Called(ctx, filter, opts)
	return args.Get(0).(*mongo.SingleResult)
}

func (m *MockMongoRepository) Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (*mongo.Cursor, error) {
	args := m.Called(ctx, pipeline, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mongo.Cursor), args.Error(1)
}

func (m *MockMongoRepository) UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	args := m.Called(ctx, filter, update, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mongo.UpdateResult), args.Error(1)
}

func (m *MockMongoRepository) DeleteOne(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	args := m.Called(ctx, filter, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mongo.DeleteResult), args.Error(1)
}

func (m *MockMongoRepository) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	args := m.Called(ctx, filter, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mongo.Cursor), args.Error(1)
}

func (m *MockMongoRepository) CountDocuments(ctx context.Context, filter interface{}, opts ...*options.CountOptions) (int64, error) {
	args := m.Called(ctx, filter, opts)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockMongoRepository) UpdateMany(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	args := m.Called(ctx, filter, update, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mongo.UpdateResult), args.Error(1)
}

// Mock for repository.MongoRepository
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) FindOne(ctx context.Context, filter interface{}, opt *options.FindOneOptions) (models.Loans, error) {
	args := m.Called(ctx, filter, opt)
	return args.Get(0).(models.Loans), args.Error(1)
}

func (m *MockRepository) Find(ctx context.Context, filter interface{}) ([]models.Loans, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Loans), args.Error(1)
}

func (m *MockRepository) Delete(ctx context.Context, filter interface{}) error {
	args := m.Called(ctx, filter)
	return args.Error(0)
}

func (m *MockRepository) FindAll(ctx context.Context, filter interface{}) ([]models.Loans, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Loans), args.Error(1)
}

func (m *MockRepository) AggregateAll(pipeline interface{}, result interface{}) error {
	args := m.Called(pipeline, result)
	return args.Error(0)
}

func (m *MockRepository) Update(ctx context.Context, filter interface{}, update interface{}) (*mongo.UpdateResult, error) {
	args := m.Called(ctx, filter, update)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mongo.UpdateResult), args.Error(1)
}

func (m *MockRepository) Create(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
	args := m.Called(ctx, document)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mongo.InsertOneResult), args.Error(1)
}

func (m *MockRepository) Aggregate(ctx context.Context, pipeline interface{}, result interface{}) error {
	args := m.Called(ctx, pipeline, result)
	return args.Error(0)
}

func (m *MockRepository) UpdateOne(ctx context.Context, filter interface{}, update interface{}) error {
	args := m.Called(ctx, filter, update)
	return args.Error(0)
}

func (m *MockRepository) CountDocuments(ctx context.Context, filter interface{}) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

// Create a new LoanRepository with a mocked MongoRepository
func setupTest() (*LoanRepository, *MockRepository) {
	mockRepo := new(MockRepository)
	return NewLoanRepositoryWithInterface(mockRepo), mockRepo
}

// Helper to create a sample loan
func createSampleLoan(msisdn string, availmentId string) models.Loans {
	objID, _ := primitive.ObjectIDFromHex(availmentId)
	return models.Loans{
		LoanID:                   primitive.NewObjectID(),
		MSISDN:                   msisdn,
		AvailmentTransactionID:   objID,
		TotalLoanAmount:          100,
		CreatedAt:                time.Now(),
		EducationPeriodTimestamp: time.Now(),
	}
}

func TestGetLoanByMSISDN(t *testing.T) {
	loanRepo, mockRepo := setupTest()
	ctx := context.Background()
	msisdn := "1234567890"

	t.Run("Success", func(t *testing.T) {
		expectedLoan := createSampleLoan(msisdn, "60c8e3b3a6d9e1b2f0a1c3d4")
		mockRepo.On("FindOne", ctx, bson.M{"MSISDN": msisdn}, mock.AnythingOfType("*options.FindOneOptions")).Return(expectedLoan, nil).Once()

		loan, err := loanRepo.GetLoanByMSISDN(ctx, msisdn)

		assert.NoError(t, err)
		assert.Equal(t, &expectedLoan, loan)
		mockRepo.AssertExpectations(t)
	})

	t.Run("No Document Found", func(t *testing.T) {
		mockRepo.On("FindOne", ctx, bson.M{"MSISDN": msisdn}, mock.AnythingOfType("*options.FindOneOptions")).
			Return(models.Loans{}, mongo.ErrNoDocuments).Once()

		loan, err := loanRepo.GetLoanByMSISDN(ctx, msisdn)

		assert.ErrorIs(t, err, mongo.ErrNoDocuments)
		assert.Nil(t, loan)
		mockRepo.AssertExpectations(t)
	})

	t.Run("FindOne Error", func(t *testing.T) {
		testErr := fmt.Errorf("database error")
		mockRepo.On("FindOne", ctx, bson.M{"MSISDN": msisdn}, mock.AnythingOfType("*options.FindOneOptions")).
			Return(models.Loans{}, testErr).Once()

		loan, err := loanRepo.GetLoanByMSISDN(ctx, msisdn)

		assert.ErrorIs(t, err, testErr)
		assert.Nil(t, loan)
		mockRepo.AssertExpectations(t)
	})
}

func TestGetLoanByMSISDNList(t *testing.T) {
	loanRepo, mockRepo := setupTest()
	ctx := context.Background()
	msisdns := []string{"1234567890", "0987654321"}

	t.Run("Success", func(t *testing.T) {
		expectedLoans := []models.Loans{
			createSampleLoan(msisdns[0], "60c8e3b3a6d9e1b2f0a1c3d4"),
			createSampleLoan(msisdns[1], "60c8e3b3a6d9e1b2f0a1c3d5"),
		}
		filter := bson.M{"MSISDN": bson.M{"$in": msisdns}}
		mockRepo.On("Find", ctx, filter).Return(expectedLoans, nil).Once()

		loans, err := loanRepo.GetLoanByMSISDNList(ctx, msisdns)

		assert.NoError(t, err)
		assert.Equal(t, expectedLoans, loans)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Find Error", func(t *testing.T) {
		testErr := fmt.Errorf("database error")
		filter := bson.M{"MSISDN": bson.M{"$in": msisdns}}
		mockRepo.On("Find", ctx, filter).Return(nil, testErr).Once()

		loans, err := loanRepo.GetLoanByMSISDNList(ctx, msisdns)

		assert.ErrorIs(t, err, testErr)
		assert.Nil(t, loans)
		mockRepo.AssertExpectations(t)
	})

	t.Run("No Loans Found", func(t *testing.T) {
		filter := bson.M{"MSISDN": bson.M{"$in": msisdns}}
		mockRepo.On("Find", ctx, filter).Return([]models.Loans{}, nil).Once()

		loans, err := loanRepo.GetLoanByMSISDNList(ctx, msisdns)

		assert.NoError(t, err)
		assert.Empty(t, loans)
		mockRepo.AssertExpectations(t)
	})
}

func TestDeleteActiveLoan(t *testing.T) {
	loanRepo, mockRepo := setupTest()
	ctx := context.Background()
	loanID := primitive.NewObjectID()

	t.Run("Success", func(t *testing.T) {
		filter := bson.M{"_id": loanID}
		mockRepo.On("Delete", ctx, filter).Return(nil).Once()

		err := loanRepo.DeleteActiveLoan(ctx, loanID)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Delete Error", func(t *testing.T) {
		testErr := fmt.Errorf("delete error")
		filter := bson.M{"_id": loanID}
		mockRepo.On("Delete", ctx, filter).Return(testErr).Once()

		err := loanRepo.DeleteActiveLoan(ctx, loanID)

		assert.ErrorIs(t, err, testErr)
		mockRepo.AssertExpectations(t)
	})
}

func TestGetLoanByAvailmentID(t *testing.T) {
	loanRepo, mockRepo := setupTest()
	ctx := context.Background()
	availmentIdHex := "60c8e3b3a6d9e1b2f0a1c3d4"
	availmentId, _ := primitive.ObjectIDFromHex(availmentIdHex)

	t.Run("Success", func(t *testing.T) {
		expectedLoan := createSampleLoan("1234567890", availmentIdHex)
		filter := bson.M{"availmentTransactionId": availmentId}
		mockRepo.On("FindOne", ctx, filter, mock.AnythingOfType("*options.FindOneOptions")).Return(expectedLoan, nil).Once()

		loan, err := loanRepo.GetLoanByAvailmentID(ctx, availmentIdHex)

		assert.NoError(t, err)
		assert.Equal(t, &expectedLoan, loan)
		mockRepo.AssertExpectations(t)
	})

	t.Run("No Availment ID Provided", func(t *testing.T) {
		loan, err := loanRepo.GetLoanByAvailmentID(ctx, "")

		assert.Error(t, err)
		assert.Nil(t, loan)
	})

	t.Run("Invalid Availment ID", func(t *testing.T) {
		loan, err := loanRepo.GetLoanByAvailmentID(ctx, "invalid_id")

		assert.Error(t, err)
		assert.Nil(t, loan)
	})

	t.Run("No Document Found", func(t *testing.T) {
		filter := bson.M{"availmentTransactionId": availmentId}
		mockRepo.On("FindOne", ctx, filter, mock.AnythingOfType("*options.FindOneOptions")).Return(models.Loans{}, mongo.ErrNoDocuments).Once()

		loan, err := loanRepo.GetLoanByAvailmentID(ctx, availmentIdHex)

		assert.ErrorIs(t, err, mongo.ErrNoDocuments)
		assert.Nil(t, loan)
		mockRepo.AssertExpectations(t)
	})

	t.Run("FindOne Error", func(t *testing.T) {
		testErr := fmt.Errorf("database error")
		filter := bson.M{"availmentTransactionId": availmentId}
		mockRepo.On("FindOne", ctx, filter, mock.AnythingOfType("*options.FindOneOptions")).Return(models.Loans{}, testErr).Once()

		loan, err := loanRepo.GetLoanByAvailmentID(ctx, availmentIdHex)

		assert.ErrorIs(t, err, testErr)
		assert.Nil(t, loan)
		mockRepo.AssertExpectations(t)
	})
}

func TestCheckIfOldOrNewLoan(t *testing.T) {
	loanRepo, _ := setupTest()
	ctx := context.Background()

	t.Run("ConversionRate > 0", func(t *testing.T) {
		loan := createSampleLoan("1234567890", "60c8e3b3a6d9e1b2f0a1c3d4")
		loan.ConversionRate = 1.5
		result := loanRepo.CheckIfOldOrNewLoan(ctx, &loan)
		assert.True(t, result)
	})

	t.Run("EducationPeriodTimestamp non-zero", func(t *testing.T) {
		loan := createSampleLoan("1234567890", "60c8e3b3a6d9e1b2f0a1c3d4")
		loan.ConversionRate = 0
		loan.PromoEducationPeriodTimestamp = time.Now()
		result := loanRepo.CheckIfOldOrNewLoan(ctx, &loan)
		assert.True(t, result)
	})

	t.Run("LoanLoadPeriodTimestamp non-zero", func(t *testing.T) {
		loan := createSampleLoan("1234567890", "60c8e3b3a6d9e1b2f0a1c3d4")
		loan.ConversionRate = 0
		loan.LoanLoadPeriodTimestamp = time.Now()
		result := loanRepo.CheckIfOldOrNewLoan(ctx, &loan)
		assert.True(t, result)
	})

	t.Run("All zero values", func(t *testing.T) {
		loan := models.Loans{}
		result := loanRepo.CheckIfOldOrNewLoan(ctx, &loan)
		assert.False(t, result)
	})
}

func TestUpdateOldLoanDocument(t *testing.T) {
	loanRepo, mockRepo := setupTest()
	ctx := context.Background()

	loan := createSampleLoan("1234567890", "60c8e3b3a6d9e1b2f0a1c3d4")
	loanID := loan.LoanID

	conversionRate := 2.5
	promoEducationPeriodTimestamp := time.Now().Add(-24 * time.Hour)
	loanLoadPeriodTimestamp := time.Now()
	maxDataCollectionPercent := 80.0
	minCollectionAmount := 50.0
	totalUnpaidLoan := 500.0

	update := bson.M{
		"conversionRate":                conversionRate,
		"promoEducationPeriodTimestamp": promoEducationPeriodTimestamp,
		"loanLoadPeriodTimestamp":       loanLoadPeriodTimestamp,
		"maxDataCollectionPercent":      maxDataCollectionPercent,
		"minCollectionAmount":           minCollectionAmount,
		"totalUnpaidLoan":               totalUnpaidLoan,
	}
	filter := bson.M{"_id": loanID}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("UpdateOne", ctx, filter, update).Return(nil).Once()

		updatedID, err := loanRepo.UpdateOldLoanDocument(ctx, &loan,
			conversionRate,
			promoEducationPeriodTimestamp,
			loanLoadPeriodTimestamp,
			maxDataCollectionPercent,
			minCollectionAmount,
			totalUnpaidLoan,
		)

		assert.NoError(t, err)
		assert.Equal(t, loanID, updatedID)
		assert.Equal(t, conversionRate, loan.ConversionRate)
		assert.Equal(t, promoEducationPeriodTimestamp, loan.PromoEducationPeriodTimestamp)
		assert.Equal(t, loanLoadPeriodTimestamp, loan.LoanLoadPeriodTimestamp)
		assert.Equal(t, maxDataCollectionPercent, loan.MaxDataCollectionPercent)
		assert.Equal(t, minCollectionAmount, loan.MinCollectionAmount)
		assert.Equal(t, totalUnpaidLoan, loan.TotalUnpaidLoan)

		mockRepo.AssertExpectations(t)
	})

	t.Run("UpdateOne Error", func(t *testing.T) {
		testErr := fmt.Errorf("update error")
		mockRepo.On("UpdateOne", ctx, filter, update).Return(testErr).Once()

		updatedID, err := loanRepo.UpdateOldLoanDocument(ctx, &loan,
			conversionRate,
			promoEducationPeriodTimestamp,
			loanLoadPeriodTimestamp,
			maxDataCollectionPercent,
			minCollectionAmount,
			totalUnpaidLoan,
		)

		assert.ErrorIs(t, err, testErr)
		assert.Equal(t, primitive.NilObjectID, updatedID)

		mockRepo.AssertExpectations(t)
	})
}
