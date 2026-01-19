package promocollection

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"

	kafkamodels "promocollection/internal/pkg/models"
	"promocollection/internal/pkg/store/models"
)

type MockSystemLevelRulesRepo struct {
	mock.Mock
}

func (m *MockSystemLevelRulesRepo) FetchSystemLevelRulesConfiguration(ctx context.Context) (models.SystemLevelRules, error) {
	args := m.Called(ctx)
	return args.Get(0).(models.SystemLevelRules), args.Error(1)
}

type MockLoansRepo struct {
	mock.Mock
}

func (m *MockLoansRepo) GetLoanByMSISDN(ctx context.Context, msisdn string) (*models.Loans, error) {
	args := m.Called(ctx, msisdn)
	return args.Get(0).(*models.Loans), args.Error(1)
}

// ---- Mock PubSubPublisherInterface ----
type MockPubSubPublisherForMax struct {
	mock.Mock
}

func (m *MockPubSubPublisherForMax) Close() {}
func (m *MockPubSubPublisherForMax) PublishMessage(ctx context.Context, msg any) (string, error) {
	args := m.Called(ctx, msg)
	return args.String(0), args.Error(1)
}

// ---- Mock CollectionTransactionsInProgressRepo ----
type MockCollectionRepoForMax struct {
	mock.Mock
}

func (m *MockCollectionRepoForMax) DeleteEntry(ctx context.Context, msisdn string) error {
	args := m.Called(ctx, msisdn)
	return args.Error(0)
}

// satisfy the higher-level repository interface as well
func (m *MockCollectionRepoForMax) CheckEntryExists(ctx context.Context, msisdn string) (bool, error) {
	args := m.Called(ctx, msisdn)
	return args.Bool(0), args.Error(1)
}

func (m *MockCollectionRepoForMax) CreateEntry(ctx context.Context, msisdn string) error {
	args := m.Called(ctx, msisdn)
	return args.Error(0)
}

// ---- Mock UnpaidLoansRepositoryInterface ----
type MockUnpaidLoansRepo struct {
	mock.Mock
}

func (m *MockUnpaidLoansRepo) GetUnpaidLoansByLoanId(ctx context.Context, loanId primitive.ObjectID) (*models.UnpaidLoans, error) {
	args := m.Called(ctx, loanId)
	return args.Get(0).(*models.UnpaidLoans), args.Error(1)
}

func (m *MockLoansRepo) GetLoanByMSISDNList(ctx context.Context, msisdns []string) ([]models.Loans, error) {
	panic("not part of scenario")
}
func (m *MockLoansRepo) GetLoanByAvailmentID(ctx context.Context, availmentId string) (*models.Loans, error) {
	panic("not part of test")
}
func (m *MockLoansRepo) DeleteActiveLoan(ctx context.Context, loanID primitive.ObjectID) error {
	panic("not part of test")
}
func (m *MockLoansRepo) CheckIfOldOrNewLoan(ctx context.Context, loan *models.Loans) bool {
	panic("not part")
}
func (m *MockLoansRepo) UpdateOldLoanDocument(ctx context.Context,
	loan *models.Loans,
	conversionRate float64,
	educationPeriodTimestamp time.Time,
	loanLoadPeriodTimestamp time.Time,
	maxDataCollectionPercent float64,
	minCollectionAmount float64,
	totalUnpaidLoan float64) (primitive.ObjectID, error) {
	panic("not part")
}

type MockLoanProductRepo struct {
	mock.Mock
}

func (m *MockLoanProductRepo) GetProductNameById(ctx context.Context, id primitive.ObjectID) (*models.LoanProducts, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.LoanProducts), args.Error(1)
}

func TestCollectionRulesSuccessWithSystemLevelConversion(t *testing.T) {
	ctx := context.Background()

	mockSystemRepo := new(MockSystemLevelRulesRepo)
	mockLoansRepo := new(MockLoansRepo)
	mockUnpaidRepo := new(MockUnpaidLoansRepo)
	mockLoanProductRepo := new(MockLoanProductRepo)

	mockSystemRepo.On("FetchSystemLevelRulesConfiguration", ctx).Return(models.SystemLevelRules{
		MaxDataCollectionPercent: 0.5,
		PesoToDataConversionMatrix: []models.PesoToDataConversion{
			{LoanType: "UNSCORED", ConversionRate: 2.0},
		},
	}, nil)

	mockLoansRepo.On("GetLoanByMSISDN", ctx, "12345").Return(&models.Loans{
		ConversionRate:  0.0,
		TotalUnpaidLoan: 200,
	}, nil)

	mockUnpaidRepo.On("GetUnpaidLoansByLoanId", mock.Anything, mock.Anything).Return(
		&models.UnpaidLoans{
			LoanId:                 primitive.NewObjectID(),
			TotalUnpaidAmount:      1,
			LastCollectionDateTime: time.Now(),
		}, nil)

	mockLoanProductRepo.On("GetProductNameById", mock.Anything, mock.Anything).Return(&models.LoanProducts{
		IsScoredProduct: false,
	}, nil)

	payload := &kafkamodels.PromoEventMessage{
		MSISDN:       12345,
		WALLETAMOUNT: 100,
	}

	mockPub := new(MockPubSubPublisherForMax)
	mockColl := new(MockCollectionRepoForMax)
	mockPub.On("PublishMessage", mock.Anything, mock.Anything).Return("id", nil)
	mockColl.On("DeleteEntry", mock.Anything, mock.Anything).Return(nil)

	err := CollectionRules(ctx, payload, mockSystemRepo, mockLoansRepo, mockPub, mockColl, mockUnpaidRepo, mockLoanProductRepo)

	assert.NoError(t, err)
	mockSystemRepo.AssertExpectations(t)
	mockLoansRepo.AssertExpectations(t)
	mockLoanProductRepo.AssertExpectations(t)
}

func TestCollectionRulesSuccessWithLoanConversion(t *testing.T) {
	ctx := context.Background()

	mockSystemRepo := new(MockSystemLevelRulesRepo)
	mockLoansRepo := new(MockLoansRepo)
	mockUnpaidRepo := new(MockUnpaidLoansRepo)
	mockLoanProductRepo := new(MockLoanProductRepo)

	mockSystemRepo.On("FetchSystemLevelRulesConfiguration", ctx).Return(models.SystemLevelRules{
		MaxDataCollectionPercent: 0.8,
		PesoToDataConversionMatrix: []models.PesoToDataConversion{
			{LoanType: "default", ConversionRate: 1.5},
		},
	}, nil)

	mockLoansRepo.On("GetLoanByMSISDN", ctx, "67890").Return(&models.Loans{
		ConversionRate:  3.0, // should use this instead of system rules
		TotalUnpaidLoan: 50,
	}, nil)

	mockUnpaidRepo.On("GetUnpaidLoansByLoanId", mock.Anything, mock.Anything).Return(
		&models.UnpaidLoans{
			LoanId:                 primitive.NewObjectID(),
			TotalUnpaidAmount:      1,
			LastCollectionDateTime: time.Now(),
		}, nil)

	// Since ConversionRate is not 0.0, loanProductRepo should not be called
	// mockLoanProductRepo.On("GetProductNameById", mock.Anything, mock.Anything).Return(&models.LoanProducts{
	// 	IsScoredProduct: false,
	// }, nil)

	payload := &kafkamodels.PromoEventMessage{
		MSISDN:       67890,
		WALLETAMOUNT: 10,
	}

	mockPub := new(MockPubSubPublisherForMax)
	mockColl := new(MockCollectionRepoForMax)
	mockPub.On("PublishMessage", mock.Anything, mock.Anything).Return("id", nil)
	mockColl.On("DeleteEntry", mock.Anything, mock.Anything).Return(nil)

	err := CollectionRules(ctx, payload, mockSystemRepo, mockLoansRepo, mockPub, mockColl, mockUnpaidRepo, mockLoanProductRepo)

	assert.NoError(t, err)
	mockSystemRepo.AssertExpectations(t)
	mockLoansRepo.AssertExpectations(t)
	// mockLoanProductRepo.AssertExpectations(t) // Not called since ConversionRate != 0.0
}

func TestCollectionRulesFetchSystemRulesError(t *testing.T) {
	ctx := context.Background()

	mockSystemRepo := new(MockSystemLevelRulesRepo)
	mockLoansRepo := new(MockLoansRepo)
	mockUnpaidRepo := new(MockUnpaidLoansRepo)
	mockLoanProductRepo := new(MockLoanProductRepo)

	mockSystemRepo.On("FetchSystemLevelRulesConfiguration", ctx).Return(models.SystemLevelRules{}, assert.AnError)

	// loanProductRepo not called since function returns early on system rules error
	// mockLoanProductRepo.On("GetProductNameById", mock.Anything, mock.Anything).Return(&models.LoanProducts{
	// 	IsScoredProduct: false,
	// }, nil)

	mockUnpaidRepo.On("GetUnpaidLoansByLoanId", mock.Anything, mock.Anything).Return(
		&models.UnpaidLoans{
			LoanId:                 primitive.NewObjectID(),
			TotalUnpaidAmount:      1,
			LastCollectionDateTime: time.Now(),
		}, nil)

	payload := &kafkamodels.PromoEventMessage{
		MSISDN:       11111,
		WALLETAMOUNT: 5,
	}

	mockPub := new(MockPubSubPublisherForMax)
	mockColl := new(MockCollectionRepoForMax)
	mockPub.On("PublishMessage", mock.Anything, mock.Anything).Return("id", nil)
	mockColl.On("DeleteEntry", mock.Anything, mock.Anything).Return(nil)

	err := CollectionRules(ctx, payload, mockSystemRepo, mockLoansRepo, mockPub, mockColl, mockUnpaidRepo, mockLoanProductRepo)

	assert.Error(t, err)
	mockSystemRepo.AssertExpectations(t)
}

func TestCollectionRulesLoansRepoError(t *testing.T) {
	ctx := context.Background()

	mockSystemRepo := new(MockSystemLevelRulesRepo)
	mockLoansRepo := new(MockLoansRepo)
	mockUnpaidRepo := new(MockUnpaidLoansRepo)
	mockLoanProductRepo := new(MockLoanProductRepo)

	// System rules succeed
	mockSystemRepo.On("FetchSystemLevelRulesConfiguration", ctx).Return(models.SystemLevelRules{
		MaxDataCollectionPercent: 0.5,
	}, nil)

	mockUnpaidRepo.On("GetUnpaidLoansByLoanId", mock.Anything, mock.Anything).Return(
		&models.UnpaidLoans{
			LoanId:                 primitive.NewObjectID(),
			TotalUnpaidAmount:      1,
			LastCollectionDateTime: time.Now(),
		}, nil)

	// loanProductRepo not called since loans repo errors
	// mockLoanProductRepo.On("GetProductNameById", mock.Anything, mock.Anything).Return(&models.LoanProducts{
	// 	IsScoredProduct: false,
	// }, nil)

	// Loans repo returns error
	mockLoansRepo.On("GetLoanByMSISDN", ctx, "22222").Return((*models.Loans)(nil), assert.AnError)

	payload := &kafkamodels.PromoEventMessage{
		MSISDN:       22222,
		WALLETAMOUNT: 20,
	}

	mockPub := new(MockPubSubPublisherForMax)
	mockColl := new(MockCollectionRepoForMax)
	mockPub.On("PublishMessage", mock.Anything, mock.Anything).Return("id", nil)
	mockColl.On("DeleteEntry", mock.Anything, mock.Anything).Return(nil)

	err := CollectionRules(ctx, payload, mockSystemRepo, mockLoansRepo, mockPub, mockColl, mockUnpaidRepo, mockLoanProductRepo)
	assert.Error(t, err)

	mockSystemRepo.AssertExpectations(t)
	mockLoansRepo.AssertExpectations(t)
}

func TestCollectionRulesNoLoanData(t *testing.T) {
	ctx := context.Background()

	mockSystemRepo := new(MockSystemLevelRulesRepo)
	mockLoansRepo := new(MockLoansRepo)
	mockUnpaidRepo := new(MockUnpaidLoansRepo)
	mockLoanProductRepo := new(MockLoanProductRepo)

	// Mock system rules
	mockSystemRepo.On("FetchSystemLevelRulesConfiguration", ctx).Return(models.SystemLevelRules{
		MaxDataCollectionPercent: 0.6,
	}, nil)

	// loanProductRepo not called since unpaid amount is 0
	// mockLoanProductRepo.On("GetProductNameById", mock.Anything, mock.Anything).Return(&models.LoanProducts{
	// 	IsScoredProduct: false,
	// }, nil)

	// Loans repo returns empty unpaid balance
	mockLoansRepo.On("GetLoanByMSISDN", ctx, "33333").Return(&models.Loans{}, nil)

	mockUnpaidRepo.On("GetUnpaidLoansByLoanId", mock.Anything, mock.Anything).Return(
		&models.UnpaidLoans{
			LoanId:                 primitive.NewObjectID(),
			TotalUnpaidAmount:      0,
			LastCollectionDateTime: time.Now(),
		}, nil)

	payload := &kafkamodels.PromoEventMessage{
		MSISDN:       33333,
		WALLETAMOUNT: 15,
	}

	mockPub := new(MockPubSubPublisherForMax)
	mockColl := new(MockCollectionRepoForMax)
	mockPub.On("PublishMessage", mock.Anything, mock.Anything).Return("id", nil)
	mockColl.On("DeleteEntry", mock.Anything, mock.Anything).Return(nil)

	err := CollectionRules(ctx, payload, mockSystemRepo, mockLoansRepo, mockPub, mockColl, mockUnpaidRepo, mockLoanProductRepo)
	assert.Error(t, err)
	assert.EqualError(t, err, "no unpaid balance found for loan: ObjectID(\"000000000000000000000000\")")

	mockSystemRepo.AssertExpectations(t)
	mockLoansRepo.AssertExpectations(t)
	mockUnpaidRepo.AssertExpectations(t)
}

func TestCollectionRulesNoConversionRateAvailable(t *testing.T) {
	ctx := context.Background()

	mockSystemRepo := new(MockSystemLevelRulesRepo)
	mockLoansRepo := new(MockLoansRepo)
	mockUnpaidRepo := new(MockUnpaidLoansRepo)
	mockLoanProductRepo := new(MockLoanProductRepo)

	// System rules without conversion matrix
	mockSystemRepo.On("FetchSystemLevelRulesConfiguration", ctx).Return(models.SystemLevelRules{
		MaxDataCollectionPercent:   0.7,
		PesoToDataConversionMatrix: []models.PesoToDataConversion{},
	}, nil)

	mockLoanProductRepo.On("GetProductNameById", mock.Anything, mock.Anything).Return(&models.LoanProducts{
		IsScoredProduct: false,
	}, nil)

	// Loans repo with 0 conversion rate
	mockLoansRepo.On("GetLoanByMSISDN", ctx, "44444").Return(&models.Loans{
		ConversionRate:  0.0,
		TotalUnpaidLoan: 100,
	}, nil)

	mockUnpaidRepo.On("GetUnpaidLoansByLoanId", mock.Anything, mock.Anything).Return(
		&models.UnpaidLoans{
			LoanId:                 primitive.NewObjectID(),
			TotalUnpaidAmount:      0,
			LastCollectionDateTime: time.Now(),
		}, nil)

	payload := &kafkamodels.PromoEventMessage{
		MSISDN:       44444,
		WALLETAMOUNT: 30,
	}

	mockPub := new(MockPubSubPublisherForMax)
	mockColl := new(MockCollectionRepoForMax)
	mockPub.On("PublishMessage", mock.Anything, mock.Anything).Return("id", nil)
	mockColl.On("DeleteEntry", mock.Anything, mock.Anything).Return(nil)

	err := CollectionRules(ctx, payload, mockSystemRepo, mockLoansRepo, mockPub, mockColl, mockUnpaidRepo, mockLoanProductRepo)
	assert.Error(t, err)
	assert.EqualError(t, err, "no conversion rate found in systemlevelrules")

	mockSystemRepo.AssertExpectations(t)
	mockLoansRepo.AssertExpectations(t)
}
