package promocollection

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"

	kafkamodels "promocollection/internal/pkg/models"
	storeModels "promocollection/internal/pkg/store/models"
)

// ---- Mock SystemLevelRulesRepository ----
type MockSystemLevelRulesRepoForMin struct {
	mock.Mock
}

func (m *MockSystemLevelRulesRepoForMin) FetchSystemLevelRulesConfiguration(ctx context.Context) (storeModels.SystemLevelRules, error) {
	args := m.Called(ctx)
	return args.Get(0).(storeModels.SystemLevelRules), args.Error(1)
}

// ---- Mock LoanRepositoryInterface ----
type MockLoansRepoForMin struct {
	mock.Mock
}

func (m *MockLoansRepoForMin) GetLoanByMSISDN(ctx context.Context, msisdn string) (*storeModels.Loans, error) {
	args := m.Called(ctx, msisdn)
	return args.Get(0).(*storeModels.Loans), args.Error(1)
}

// satisfy the expanded LoanRepositoryInterface used by the service
func (m *MockLoansRepoForMin) GetLoanByMSISDNList(ctx context.Context, msisdns []string) ([]storeModels.Loans, error) {
	panic("not part of scenario")
}
func (m *MockLoansRepoForMin) GetLoanByAvailmentID(ctx context.Context, availmentId string) (*storeModels.Loans, error) {
	panic("not part of test")
}
func (m *MockLoansRepoForMin) DeleteActiveLoan(ctx context.Context, loanID primitive.ObjectID) error {
	panic("not part of test")
}
func (m *MockLoansRepoForMin) CheckIfOldOrNewLoan(ctx context.Context, loan *storeModels.Loans) bool {
	args := m.Called(ctx, loan)
	return args.Bool(0)
}
func (m *MockLoansRepoForMin) UpdateOldLoanDocument(ctx context.Context,
	loan *storeModels.Loans,
	conversionRate float64,
	educationPeriodTimestamp time.Time,
	loanLoadPeriodTimestamp time.Time,
	maxDataCollectionPercent float64,
	minCollectionAmount float64,
	totalUnpaidLoan float64) (primitive.ObjectID, error) {
	args := m.Called(ctx, loan, conversionRate, educationPeriodTimestamp, loanLoadPeriodTimestamp, maxDataCollectionPercent, minCollectionAmount, totalUnpaidLoan)
	return args.Get(0).(primitive.ObjectID), args.Error(1)
}

// ---- Mock PubSubPublisherInterface ----
type MockPubSubPublisher struct {
	mock.Mock
}

func (m *MockPubSubPublisher) Close() {
	// no-op
}

func (m *MockPubSubPublisher) PublishMessage(ctx context.Context, msg any) (string, error) {
	args := m.Called(ctx, msg)
	return args.String(0), args.Error(1)
}

// ---- Mock CollectionTransactionsInProgressRepo ----
type MockCollectionTransactionsInProgressRepo struct {
	mock.Mock
}

func (m *MockCollectionTransactionsInProgressRepo) DeleteEntry(ctx context.Context, msisdn string) error {
	args := m.Called(ctx, msisdn)
	return args.Error(0)
}

func (m *MockCollectionTransactionsInProgressRepo) CheckEntryExists(ctx context.Context, msisdn string) (bool, error) {
	args := m.Called(ctx, msisdn)
	return args.Bool(0), args.Error(1)
}

func (m *MockCollectionTransactionsInProgressRepo) CreateEntry(ctx context.Context, msisdn string) error {
	if m == nil {
		return nil
	}
	args := m.Called(ctx, msisdn)
	return args.Error(0)
}

type MockUnpaidLoansRepositoryInterface struct {
	mock.Mock
}

func (m *MockUnpaidLoansRepositoryInterface) GetUnpaidLoansByLoanId(ctx context.Context, loanId primitive.ObjectID) (*storeModels.UnpaidLoans, error) {
	args := m.Called(ctx, loanId)
	if obj := args.Get(0); obj != nil {
		return obj.(*storeModels.UnpaidLoans), args.Error(1)
	}
	return nil, args.Error(1)
}

// ---- Mock LoanProductsRepositoryInterface ----
type MockLoanProductsRepo struct {
	mock.Mock
}

func (m *MockLoanProductsRepo) GetProductNameById(ctx context.Context, id primitive.ObjectID) (*storeModels.LoanProducts, error) {
	args := m.Called(ctx, id)
	if obj := args.Get(0); obj != nil {
		return obj.(*storeModels.LoanProducts), args.Error(1)
	}
	return nil, args.Error(1)
}

var mockLoan = &storeModels.Loans{
	LoanID:                   primitive.NewObjectID(),
	MSISDN:                   "9262025405",
	TotalLoanAmount:          1000, // in Peso
	GUID:                     "mock-loan-guid-123",
	ServiceFee:               50,
	LoanProductID:            primitive.NewObjectID(),
	BrandID:                  primitive.NewObjectID(),
	AvailmentTransactionID:   primitive.NewObjectID(),
	LoanType:                 "STANDARD",
	CreatedAt:                time.Now().AddDate(0, 0, -10),
	OldBrandType:             "TM",
	OldAvailmentID:           12345,
	OldProductID:             678,
	GracePeriodTimestamp:     time.Time{},
	EducationPeriodTimestamp: time.Time{},
	LoanLoadPeriodTimestamp:  time.Time{},
	ConversionRate:           1.0,
	Version:                  1,
	TotalUnpaidLoan:          500,
	MinCollectionAmount:      100,
	MaxDataCollectionPercent: 50,
}

var mockUnpaidLoan = &storeModels.UnpaidLoans{
	ID:                     primitive.NewObjectID(),
	LastCollectionID:       primitive.NewObjectID(),
	Version:                1,
	ValidFrom:              time.Now().AddDate(0, -1, 0),
	ValidTo:                time.Now().AddDate(0, 1, 0),
	UnpaidServiceFee:       50,
	Migrated:               true,
	TotalLoanAmount:        600,
	TotalUnpaidAmount:      200,
	LastCollectionDateTime: time.Now().AddDate(0, 0, -7),
	LoanId:                 primitive.NewObjectID(),
	DueDate:                time.Time{},
}

// ----------------- Tests -----------------

func TestMinPromoDataCollectionSuccessFullyClosesLoanUsingSystemConversion(t *testing.T) {
	ctx := context.Background()

	mockSystemRepo := new(MockSystemLevelRulesRepoForMin)
	mockLoansRepo := new(MockLoansRepoForMin)
	mockPubSub := new(MockPubSubPublisher)

	// system rules provide conversion rate and max percent such that deductable == outstanding
	mockSystemRepo.On("FetchSystemLevelRulesConfiguration", ctx).Return(storeModels.SystemLevelRules{
		MaxDataCollectionPercent: 0.5,
		PesoToDataConversionMatrix: []storeModels.PesoToDataConversion{
			{LoanType: "UNSCORED", ConversionRate: 2.0},
		},
	}, nil)

	mockLoansRepo.On("GetLoanByMSISDN", ctx, "12345").Return(&storeModels.Loans{
		ConversionRate:  0.0, // force system-level conversion
		TotalUnpaidLoan: 100,
	}, nil)

	// expect pubsub publish for full collection
	mockPubSub.On("PublishMessage", mock.Anything, mock.Anything).Return("msg-id", nil)

	payload := &kafkamodels.PromoEventMessage{
		MSISDN:        12345,
		WALLETAMOUNT:  100,
		WALLETKEYWORD: "wallet",
	}

	mockPub := mockPubSub
	mockColl := new(MockCollectionTransactionsInProgressRepo)
	mockColl.On("DeleteEntry", mock.Anything, mock.Anything).Return(nil)
	mockUnpaid := new(MockUnpaidLoansRepositoryInterface)
	mockUnpaid.On("GetUnpaidLoansByLoanId", mock.Anything, mock.Anything).Return(
		&storeModels.UnpaidLoans{
			LoanId:                 primitive.NewObjectID(),
			TotalUnpaidAmount:      1,
			LastCollectionDateTime: time.Now(),
		}, nil)
	mockLoanProd := new(MockLoanProductsRepo)
	mockLoanProd.On("GetProductNameById", mock.Anything, mock.Anything).Return(&storeModels.LoanProducts{
		IsScoredProduct: false,
	}, nil)

	err := CollectionRules(ctx, payload, mockSystemRepo, mockLoansRepo, mockPub, mockColl, mockUnpaid, mockLoanProd)

	assert.NoError(t, err)
	mockSystemRepo.AssertExpectations(t)
	mockLoansRepo.AssertExpectations(t)
	mockPubSub.AssertExpectations(t)
}

func TestMinPromoDataCollectionSuccessPartialWithLoanConversion(t *testing.T) {
	ctx := context.Background()

	mockSystemRepo := new(MockSystemLevelRulesRepoForMin)
	mockLoansRepo := new(MockLoansRepoForMin)
	mockPubSub := new(MockPubSubPublisher)

	mockSystemRepo.On("FetchSystemLevelRulesConfiguration", ctx).Return(storeModels.SystemLevelRules{
		MaxDataCollectionPercent: 0.8,
		PesoToDataConversionMatrix: []storeModels.PesoToDataConversion{
			{LoanType: "default", ConversionRate: 1.5},
		},
		MinCollectionAmount: 1,
	}, nil)

	mockLoansRepo.On("GetLoanByMSISDN", ctx, "67890").Return(&storeModels.Loans{
		ConversionRate:  3.0, // should use this
		TotalUnpaidLoan: 50,
	}, nil)

	// expect pubsub publish for partial collection (since deductionPesoEquivalent >= MinCollectionAmount)
	mockPubSub.On("PublishMessage", mock.Anything, mock.Anything).Return("partial-msg-id", nil)

	payload := &kafkamodels.PromoEventMessage{
		MSISDN:        67890,
		WALLETAMOUNT:  10,
		WALLETKEYWORD: "wallet",
	}

	mockPub := mockPubSub
	mockColl := new(MockCollectionTransactionsInProgressRepo)
	mockColl.On("DeleteEntry", mock.Anything, mock.Anything).Return(nil)
	mockUnpaid := new(MockUnpaidLoansRepositoryInterface)
	mockUnpaid.On("GetUnpaidLoansByLoanId", mock.Anything, mock.Anything).Return(
		&storeModels.UnpaidLoans{
			LoanId:                 primitive.NewObjectID(),
			TotalUnpaidAmount:      1,
			LastCollectionDateTime: time.Now(),
		}, nil)

	mockLoanProd := new(MockLoanProductsRepo)
	mockLoanProd.On("GetProductNameById", mock.Anything, mock.Anything).Return(&storeModels.LoanProducts{
		IsScoredProduct: false,
	}, nil)

	err := CollectionRules(ctx, payload, mockSystemRepo, mockLoansRepo, mockPub, mockColl, mockUnpaid, mockLoanProd)

	assert.NoError(t, err)
	mockSystemRepo.AssertExpectations(t)
	mockLoansRepo.AssertExpectations(t)
	mockPubSub.AssertExpectations(t)
}

func TestMinPromoDataCollectionBelowMinDeletesInProgressEntry(t *testing.T) {
	ctx := context.Background()

	mockSystemRepo := new(MockSystemLevelRulesRepoForMin)
	mockLoansRepo := new(MockLoansRepoForMin)
	mockCollectionRepo := new(MockCollectionTransactionsInProgressRepo)

	// system rules with minCollectionAmount > deductionPesoEquivalent so branch triggers
	mockSystemRepo.On("FetchSystemLevelRulesConfiguration", ctx).Return(storeModels.SystemLevelRules{
		MaxDataCollectionPercent: 0.5,
		PesoToDataConversionMatrix: []storeModels.PesoToDataConversion{
			{LoanType: "UNSCORED", ConversionRate: 2.0},
		},
		MinCollectionAmount: 10, // high min to force below-min path
	}, nil)

	// loan with small unpaid balance so deductionPesoEquivalent < MinCollectionAmount
	mockLoansRepo.On("GetLoanByMSISDN", ctx, "55555").Return(&storeModels.Loans{
		ConversionRate:  0.0,
		TotalUnpaidLoan: 1,
	}, nil)

	// expect DeleteEntry called
	mockCollectionRepo.On("DeleteEntry", mock.Anything, "55555").Return(nil)

	payload := &kafkamodels.PromoEventMessage{
		MSISDN:       55555,
		WALLETAMOUNT: 1,
	}
	mockUnpaid := new(MockUnpaidLoansRepositoryInterface)
	mockUnpaid.On("GetUnpaidLoansByLoanId", mock.Anything, mock.Anything).Return(
		&storeModels.UnpaidLoans{
			LoanId:                 primitive.NewObjectID(),
			TotalUnpaidAmount:      1,
			LastCollectionDateTime: time.Now(),
		}, nil)
	mockLoanProd := new(MockLoanProductsRepo)
	mockLoanProd.On("GetProductNameById", mock.Anything, mock.Anything).Return(&storeModels.LoanProducts{
		IsScoredProduct: false,
	}, nil)

	mockPub := new(MockPubSubPublisher)
	mockPub.On("PublishMessage", mock.Anything, mock.Anything).Return("id", nil)
	err := CollectionRules(ctx, payload, mockSystemRepo, mockLoansRepo, mockPub, mockCollectionRepo, mockUnpaid, mockLoanProd)
	assert.NoError(t, err)

	mockSystemRepo.AssertExpectations(t)
	mockLoansRepo.AssertExpectations(t)
	mockCollectionRepo.AssertExpectations(t)
}

func TestMinPromoDataCollectionFullClosePublishesCorrectMsg(t *testing.T) {
	ctx := context.Background()

	mockSystemRepo := new(MockSystemLevelRulesRepoForMin)
	mockLoansRepo := new(MockLoansRepoForMin)
	mockPubSub := new(MockPubSubPublisher)

	mockSystemRepo.On("FetchSystemLevelRulesConfiguration", ctx).Return(storeModels.SystemLevelRules{
		MaxDataCollectionPercent: 1.0,
		PesoToDataConversionMatrix: []storeModels.PesoToDataConversion{
			{LoanType: "UNSCORED", ConversionRate: 1.0},
		},
	}, nil)

	mockLoansRepo.On("GetLoanByMSISDN", ctx, "99999").Return(&storeModels.Loans{
		ConversionRate:  0.0,
		TotalUnpaidLoan: 10,
	}, nil)

	// capture the published message and assert its fields
	mockPubSub.On("PublishMessage", mock.Anything, mock.MatchedBy(func(msg any) bool {
		m, ok := msg.(kafkamodels.PublishtoWorkerMsgFormat)
		if !ok {
			return false
		}
		// expect MSISDN string, WalletAmount equal to outstanding (10 MB with conversion 1.0)
		return m.Msisdn == "99999" && m.Unit == "MB" && m.WalletAmount == "10"
	})).Return("full-msg-id", nil)

	payload := &kafkamodels.PromoEventMessage{
		MSISDN:        99999,
		WALLETAMOUNT:  10,
		WALLETKEYWORD: "wallet",
	}

	mockPub := mockPubSub
	mockColl := new(MockCollectionTransactionsInProgressRepo)
	mockColl.On("DeleteEntry", mock.Anything, mock.Anything).Return(nil)
	mockUnpaid := new(MockUnpaidLoansRepositoryInterface)
	mockUnpaid.On("GetUnpaidLoansByLoanId", mock.Anything, mock.Anything).Return(
		&storeModels.UnpaidLoans{
			LoanId:                 primitive.NewObjectID(),
			TotalUnpaidAmount:      1,
			LastCollectionDateTime: time.Now(),
		}, nil)

	mockLoanProd := new(MockLoanProductsRepo)
	mockLoanProd.On("GetProductNameById", mock.Anything, mock.Anything).Return(&storeModels.LoanProducts{
		IsScoredProduct: false,
	}, nil)

	err := CollectionRules(ctx, payload, mockSystemRepo, mockLoansRepo, mockPub, mockColl, mockUnpaid, mockLoanProd)
	assert.NoError(t, err)

	mockSystemRepo.AssertExpectations(t)
	mockLoansRepo.AssertExpectations(t)
	mockPubSub.AssertExpectations(t)
}

func TestMinPromoDataCollectionFetchSystemRulesError(t *testing.T) {
	ctx := context.Background()

	mockSystemRepo := new(MockSystemLevelRulesRepoForMin)
	mockLoansRepo := new(MockLoansRepoForMin)

	mockSystemRepo.On("FetchSystemLevelRulesConfiguration", ctx).Return(storeModels.SystemLevelRules{}, assert.AnError)

	payload := &kafkamodels.PromoEventMessage{
		MSISDN:       11111,
		WALLETAMOUNT: 5,
	}

	mockPub := new(MockPubSubPublisher)
	mockPub.On("PublishMessage", mock.Anything, mock.Anything).Return("id", nil)
	mockUnpaid := new(MockUnpaidLoansRepositoryInterface)
	mockUnpaid.On("GetUnpaidLoansByLoanId", mock.Anything, mock.Anything).Return(
		&storeModels.UnpaidLoans{
			LoanId:                 primitive.NewObjectID(),
			TotalUnpaidAmount:      1,
			LastCollectionDateTime: time.Now(),
		}, nil)

	mockLoanProd := new(MockLoanProductsRepo)
	mockLoanProd.On("GetProductNameById", mock.Anything, mock.Anything).Return(&storeModels.LoanProducts{
		IsScoredProduct: false,
	}, nil)

	err := CollectionRules(ctx, payload, mockSystemRepo, mockLoansRepo, mockPub, nil, mockUnpaid, mockLoanProd)

	assert.Error(t, err)
	mockSystemRepo.AssertExpectations(t)
}

func TestMinPromoDataCollectionLoansRepoError(t *testing.T) {
	ctx := context.Background()

	mockSystemRepo := new(MockSystemLevelRulesRepoForMin)
	mockLoansRepo := new(MockLoansRepoForMin)

	mockSystemRepo.On("FetchSystemLevelRulesConfiguration", ctx).Return(storeModels.SystemLevelRules{
		MaxDataCollectionPercent: 0.5,
	}, nil)

	mockLoansRepo.On("GetLoanByMSISDN", ctx, "22222").Return((*storeModels.Loans)(nil), assert.AnError)

	payload := &kafkamodels.PromoEventMessage{
		MSISDN:       22222,
		WALLETAMOUNT: 20,
	}

	mockPub := new(MockPubSubPublisher)
	mockPub.On("PublishMessage", mock.Anything, mock.Anything).Return("id", nil)
	mockUnpaid := new(MockUnpaidLoansRepositoryInterface)
	mockUnpaid.On("GetUnpaidLoansByLoanId", mock.Anything, mock.Anything).Return(
		&storeModels.UnpaidLoans{
			LoanId:                 primitive.NewObjectID(),
			TotalUnpaidAmount:      1,
			LastCollectionDateTime: time.Now(),
		}, nil)

	mockLoanProd := new(MockLoanProductsRepo)
	mockLoanProd.On("GetProductNameById", mock.Anything, mock.Anything).Return(&storeModels.LoanProducts{
		IsScoredProduct: false,
	}, nil)

	err := CollectionRules(ctx, payload, mockSystemRepo, mockLoansRepo, mockPub, nil, mockUnpaid, mockLoanProd)
	assert.Error(t, err)

	mockSystemRepo.AssertExpectations(t)
	mockLoansRepo.AssertExpectations(t)
}

func TestMinPromoDataCollectionNoLoanData(t *testing.T) {
	ctx := context.Background()

	mockSystemRepo := new(MockSystemLevelRulesRepoForMin)
	mockLoansRepo := new(MockLoansRepoForMin)

	mockSystemRepo.On("FetchSystemLevelRulesConfiguration", ctx).Return(storeModels.SystemLevelRules{
		MaxDataCollectionPercent: 0.6,
	}, nil)

	mockLoansRepo.On("GetLoanByMSISDN", ctx, "33333").Return(&storeModels.Loans{
		TotalUnpaidLoan: 0,
	}, nil)

	payload := &kafkamodels.PromoEventMessage{
		MSISDN:       33333,
		WALLETAMOUNT: 15,
	}

	mockPub := new(MockPubSubPublisher)
	mockPub.On("PublishMessage", mock.Anything, mock.Anything).Return("id", nil)
	mockUnpaid := new(MockUnpaidLoansRepositoryInterface)
	mockUnpaid.On("GetUnpaidLoansByLoanId", mock.Anything, mock.Anything).Return(&storeModels.UnpaidLoans{
		ID:                primitive.ObjectID{},
		TotalUnpaidAmount: 0,
	}, nil)
	mockLoanProd := new(MockLoanProductsRepo)
	mockLoanProd.On("GetProductNameById", mock.Anything, mock.Anything).Return(&storeModels.LoanProducts{
		IsScoredProduct: false,
	}, nil)

	err := CollectionRules(ctx, payload, mockSystemRepo, mockLoansRepo, mockPub, nil, mockUnpaid, mockLoanProd)
	assert.Error(t, err)
	assert.EqualError(t, err, "no unpaid balance found for loan: ObjectID(\"000000000000000000000000\")")

	mockSystemRepo.AssertExpectations(t)
	mockLoansRepo.AssertExpectations(t)
}

func TestMinPromoDataCollectionNoConversionRateAvailable(t *testing.T) {
	ctx := context.Background()

	mockSystemRepo := new(MockSystemLevelRulesRepoForMin)
	mockLoansRepo := new(MockLoansRepoForMin)

	mockSystemRepo.On("FetchSystemLevelRulesConfiguration", ctx).Return(storeModels.SystemLevelRules{
		MaxDataCollectionPercent:   0.7,
		PesoToDataConversionMatrix: []storeModels.PesoToDataConversion{},
	}, nil)

	mockLoansRepo.On("GetLoanByMSISDN", ctx, "44444").Return(&storeModels.Loans{
		ConversionRate:  0.0,
		TotalUnpaidLoan: 100,
	}, nil)

	payload := &kafkamodels.PromoEventMessage{
		MSISDN:       44444,
		WALLETAMOUNT: 30,
	}

	mockPub := new(MockPubSubPublisher)
	mockPub.On("PublishMessage", mock.Anything, mock.Anything).Return("id", nil)
	mockUnpaid := new(MockUnpaidLoansRepositoryInterface)
	mockUnpaid.On("GetUnpaidLoansByLoanId", mock.Anything, mock.Anything).Return(
		&storeModels.UnpaidLoans{
			LoanId:                 primitive.NewObjectID(),
			TotalUnpaidAmount:      1,
			LastCollectionDateTime: time.Now(),
		}, nil)

	mockLoanProd := new(MockLoanProductsRepo)
	mockLoanProd.On("GetProductNameById", mock.Anything, mock.Anything).Return(&storeModels.LoanProducts{
		IsScoredProduct: false,
	}, nil)

	err := CollectionRules(ctx, payload, mockSystemRepo, mockLoansRepo, mockPub, nil, mockUnpaid, mockLoanProd)
	assert.Error(t, err)
	assert.EqualError(t, err, "no conversion rate found in systemlevelrules")

	mockSystemRepo.AssertExpectations(t)
	mockLoansRepo.AssertExpectations(t)
}

func TestMinPromoDataCollectionFullClosePublishesDirectly(t *testing.T) {
	ctx := context.Background()

	// pubsub mock
	mockPub := new(MockPubSubPublisher)
	// Expect a publish where WalletAmount == "10
	mockPub.On("PublishMessage", mock.Anything, mock.MatchedBy(func(msg any) bool {
		m, ok := msg.(kafkamodels.PublishtoWorkerMsgFormat)
		return ok && m.Unit == "MB"
	})).Return("full-id", nil)

	// params: LoanUnpaidAmount * ConversionRate == DeductableDataAmount
	params := DeductionParams{
		MinCollectionAmount:  1,
		ConversionRate:       2.0,
		LoanUnpaidAmount:     5.0,
		DeductableDataAmount: 10.0,
	}

	payload := &kafkamodels.PromoEventMessage{MSISDN: 11111, WALLETKEYWORD: "w"}

	err := DataToBeDeductedCalculation(ctx, payload, params, mockPub, nil, mockLoan, mockUnpaidLoan)
	assert.NoError(t, err)
	mockPub.AssertExpectations(t)
}

func TestMinPromoDataCollectionPartialPublishesFlooredAmount(t *testing.T) {
	ctx := context.Background()

	mockPub := new(MockPubSubPublisher)
	// deductionPesoEquivalent = 3.5 -> floor = 3 -> *2 = 6
	mockPub.On("PublishMessage", mock.Anything, mock.MatchedBy(func(msg any) bool {
		m, ok := msg.(kafkamodels.PublishtoWorkerMsgFormat)
		return ok && m.Unit == "MB"
	})).Return("partial-id", nil)

	params := DeductionParams{
		MinCollectionAmount:  1,
		ConversionRate:       2.0,
		LoanUnpaidAmount:     20.0,
		DeductableDataAmount: 7.0,
	}

	payload := &kafkamodels.PromoEventMessage{MSISDN: 22222, WALLETKEYWORD: "w"}

	err := DataToBeDeductedCalculation(ctx, payload, params, mockPub, nil, mockLoan, mockUnpaidLoan)
	assert.NoError(t, err)
	mockPub.AssertExpectations(t)
}

func TestMinPromoDataCollectionBelowMinDeletesWhenRepoProvided(t *testing.T) {
	ctx := context.Background()

	mockPub := new(MockPubSubPublisher)
	// no publish expected

	mockRepo := new(MockCollectionTransactionsInProgressRepo)
	mockRepo.On("DeleteEntry", mock.Anything, "33333").Return(nil)

	params := DeductionParams{
		MinCollectionAmount:  5,
		ConversionRate:       2.0,
		LoanUnpaidAmount:     2.0,
		DeductableDataAmount: 1.0, // deductionPesoEquivalent = 0.5 < MinCollectionAmount
	}

	payload := &kafkamodels.PromoEventMessage{MSISDN: 33333}

	err := DataToBeDeductedCalculation(ctx, payload, params, mockPub, mockRepo, mockLoan, mockUnpaidLoan)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMinPromoDataCollectionBelowMinNoRepoSkipsDelete(t *testing.T) {
	ctx := context.Background()

	mockPub := new(MockPubSubPublisher)

	params := DeductionParams{
		MinCollectionAmount:  5,
		ConversionRate:       2.0,
		LoanUnpaidAmount:     2.0,
		DeductableDataAmount: 1.0,
	}

	payload := &kafkamodels.PromoEventMessage{MSISDN: 44444}

	// pass nil repo; expect no error
	err := DataToBeDeductedCalculation(ctx, payload, params, mockPub, nil, mockLoan, mockUnpaidLoan)
	assert.NoError(t, err)
}

func TestMinPromoDataCollectionPublishErrorReturnsError(t *testing.T) {
	ctx := context.Background()

	mockPub := new(MockPubSubPublisher)
	mockPub.On("PublishMessage", mock.Anything, mock.Anything).Return("", assert.AnError)

	params := DeductionParams{
		MinCollectionAmount:  1,
		ConversionRate:       2.0,
		LoanUnpaidAmount:     5.0,
		DeductableDataAmount: 10.0,
	}

	payload := &kafkamodels.PromoEventMessage{MSISDN: 55555}

	err := DataToBeDeductedCalculation(ctx, payload, params, mockPub, nil, mockLoan, mockUnpaidLoan)
	assert.Error(t, err)
}

func TestMinPromoDataCollectionDeleteErrorPropagates(t *testing.T) {
	ctx := context.Background()

	mockPub := new(MockPubSubPublisher)
	mockRepo := new(MockCollectionTransactionsInProgressRepo)
	mockRepo.On("DeleteEntry", mock.Anything, "66666").Return(assert.AnError)

	params := DeductionParams{
		MinCollectionAmount:  5,
		ConversionRate:       2.0,
		LoanUnpaidAmount:     2.0,
		DeductableDataAmount: 1.0,
	}
	payload := &kafkamodels.PromoEventMessage{MSISDN: 66666}

	err := DataToBeDeductedCalculation(ctx, payload, params, mockPub, mockRepo, mockLoan, mockUnpaidLoan)
	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}
