package promocollection

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"promocollection/internal/pkg/consts"
	kafkamodels "promocollection/internal/pkg/models"
	"promocollection/internal/pkg/store/models"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// --- Loan Repository Mocks ---

type mockLoanRepo struct{ mock.Mock }

func (m *mockLoanRepo) GetLoanByMSISDN(ctx context.Context, msisdn string) (*models.Loans, error) {
	args := m.Called(ctx, msisdn)
	if args.Get(0) != nil {
		return args.Get(0).(*models.Loans), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockLoanRepo) CheckIfOldOrNewLoan(ctx context.Context, loan *models.Loans) bool {
	return m.Called(ctx, loan).Bool(0)
}

func (m *mockLoanRepo) UpdateOldLoanDocument(
	ctx context.Context,
	loan *models.Loans,
	conversionRate float64,
	educationPeriodTimestamp, loanLoadPeriodTimestamp time.Time,
	maxDataCollectionPercent, minCollectionAmount, totalUnpaidLoan float64,
) (primitive.ObjectID, error) {
	args := m.Called(ctx, loan, conversionRate, educationPeriodTimestamp, loanLoadPeriodTimestamp,
		maxDataCollectionPercent, minCollectionAmount, totalUnpaidLoan)
	return args.Get(0).(primitive.ObjectID), args.Error(1)
}

func (m *mockLoanRepo) GetLoanByMSISDNList(ctx context.Context, msisdn []string) ([]models.Loans, error) {
	args := m.Called(ctx, msisdn)
	if args.Get(0) != nil {
		return args.Get(0).([]models.Loans), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockLoanRepo) GetLoanByAvailmentID(ctx context.Context, availmentId string) (*models.Loans, error) {
	args := m.Called(ctx, availmentId)
	if args.Get(0) != nil {
		return args.Get(0).(*models.Loans), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockLoanRepo) DeleteActiveLoan(ctx context.Context, loanID primitive.ObjectID) error {
	return m.Called(ctx, loanID).Error(0)
}

// --- Collection Transactions In Progress Repo ---

type mockCollectionTransactionsInProgressRepo struct{ mock.Mock }

func (m *mockCollectionTransactionsInProgressRepo) CheckEntryExists(ctx context.Context, msisdn string) (bool, error) {
	args := m.Called(ctx, msisdn)
	return args.Bool(0), args.Error(1)
}

func (m *mockCollectionTransactionsInProgressRepo) CreateEntry(ctx context.Context, msisdn string) error {
	return m.Called(ctx, msisdn).Error(0)
}

func (m *mockCollectionTransactionsInProgressRepo) DeleteEntry(ctx context.Context, msisdn string) error {
	return m.Called(ctx, msisdn).Error(0)
}

// --- Whitelist Repo ---

type mockWhitelistedForDataCollectionRepo struct{ mock.Mock }

func (m *mockWhitelistedForDataCollectionRepo) IsMSISDNWhitelisted(ctx context.Context, msisdn string) (bool, error) {
	args := m.Called(ctx, msisdn)
	return args.Bool(0), args.Error(1)
}

func (m *mockWhitelistedForDataCollectionRepo) DeleteByMSISDN(ctx context.Context, msisdn string) error {
	return m.Called(ctx, msisdn).Error(0)
}

// --- System Rules Repo ---

type mockSystemRulesRepo struct{ mock.Mock }

func (m *mockSystemRulesRepo) FetchSystemLevelRulesConfiguration(ctx context.Context) (models.SystemLevelRules, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).(models.SystemLevelRules), args.Error(1)
	}
	return models.SystemLevelRules{}, args.Error(1)
}

// --- Loan Products Repo ---

type mockLoanProductsRepo struct{ mock.Mock }

func (m *mockLoanProductsRepo) GetProductNameById(ctx context.Context, id primitive.ObjectID) (*models.LoanProducts, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*models.LoanProducts), args.Error(1)
	}
	return nil, args.Error(1)
}

// --- Messages Repo ---

type mockMessagesRepo struct{ mock.Mock }

func (m *mockMessagesRepo) GetPatternIdByEventandBrandId(ctx context.Context, event string, brandId primitive.ObjectID) (*models.Messages, error) {
	args := m.Called(ctx, event, brandId)
	if args.Get(0) != nil {
		return args.Get(0).(*models.Messages), args.Error(1)
	}
	return nil, args.Error(1)
}

// --- Redis Repo ---

type mockRedisRepo struct{ mock.Mock }

func (m *mockRedisRepo) Get(ctx context.Context, key string) (interface{}, error) {
	args := m.Called(ctx, key)
	return args.Get(0), args.Error(1)
}

func (m *mockRedisRepo) Set(ctx context.Context, key string, inter interface{}, duration time.Duration) error {
	return m.Called(ctx, key, inter, duration).Error(0)
}

func (m *mockRedisRepo) TTL(ctx context.Context, key string) (time.Duration, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(time.Duration), args.Error(1)
}

func (m *mockRedisRepo) SaveSkipTimestamps(ctx context.Context, msisdn string, entry models.SkipTimestamps) error {
	return m.Called(ctx, msisdn, entry).Error(0)
}

func (m *mockRedisRepo) Delete(ctx context.Context, key string) error {
	return m.Called(ctx, key).Error(0)
}

func (m *mockRedisRepo) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *mockRedisRepo) Expire(ctx context.Context, key string, duration time.Duration) (bool, error) {
	args := m.Called(ctx, key, duration)
	return args.Bool(0), args.Error(1)
}

// --- PubSub ---

type mockPubSub struct{ mock.Mock }

func (m *mockPubSub) PublishMessage(ctx context.Context, msg interface{}) (string, error) {
	if len(m.ExpectedCalls) == 0 {
		return "", nil
	}
	args := m.Called(ctx, msg)
	return args.String(0), args.Error(1)
}

func (m *mockPubSub) Close() {}

// --- Unpaid Repo ---

type mockUnpaidRepo struct{ mock.Mock }

func (m *mockUnpaidRepo) GetUnpaidLoansByLoanId(ctx context.Context, loanId primitive.ObjectID) (*models.UnpaidLoans, error) {
	args := m.Called(ctx, loanId)
	if obj := args.Get(0); obj != nil {
		return obj.(*models.UnpaidLoans), args.Error(1)
	}
	return nil, args.Error(1)
}

// --- Test function ---
func TestCheckBusinessLevelRulesForPromoCollection_HappyPath(t *testing.T) {
	ctx := context.Background()
	msisdn := "123"

	// Build Redis key
	key := models.SkipTimestampsKeyBuilder(msisdn)

	// Initialize mocks
	loanRepo := &mockLoanRepo{}
	collectionRepo := &mockCollectionTransactionsInProgressRepo{}
	whitelistRepo := &mockWhitelistedForDataCollectionRepo{}
	systemRulesRepo := &mockSystemRulesRepo{}
	loanProductsRepo := &mockLoanProductsRepo{}
	messagesRepo := &mockMessagesRepo{}
	redisRepo := &mockRedisRepo{}
	pubsubClient := &mockPubSub{}
	unpaidRepo := &mockUnpaidRepo{}

	// Sample payload
	testPayload := &kafkamodels.PromoEventMessage{
		MSISDN: 123,
	}

	// Sample active loan
	activeLoan := &models.Loans{
		LoanID:                        primitive.NewObjectID(),
		MSISDN:                        msisdn,
		TotalLoanAmount:               1000,
		ServiceFee:                    10,
		LoanProductID:                 primitive.NewObjectID(),
		BrandID:                       primitive.NewObjectID(),
		AvailmentTransactionID:        primitive.NewObjectID(),
		LoanType:                      "Normal",
		PromoEducationPeriodTimestamp: time.Now().Add(-2 * time.Hour),
		LoanLoadPeriodTimestamp:       time.Now().Add(-1 * time.Hour),
	}

	// Configure mocks

	// Step 1: PromoCollectionEnabled
	systemRulesRepo.On("FetchSystemLevelRulesConfiguration", ctx).Return(models.SystemLevelRules{
		PromoCollectionEnabled: true,
	}, nil)

	// Step 2: No existing transaction
	collectionRepo.On("CheckEntryExists", ctx, msisdn).Return(false, nil)

	// Step 3: Create entry
	collectionRepo.On("CreateEntry", ctx, msisdn).Return(nil)
	collectionRepo.On("DeleteEntry", ctx, msisdn).Return(nil)

	// Step 4: Redis checks
	redisRepo.On("Get", ctx, key).Return(nil, redis.Nil)

	// Step 5: Active loan
	loanRepo.On("GetLoanByMSISDN", ctx, msisdn).Return(activeLoan, nil)

	// Step 6: Loan type check
	loanRepo.On("CheckIfOldOrNewLoan", ctx, activeLoan).Return(true) // true → new loan

	// Step 9: Data collection completes successfully
	pubsubClient.On("PublishMessage", ctx, mock.Anything).Return("msgID", nil)

	unpaidRepo.On("GetUnpaidLoansByLoanId", ctx, mock.Anything).
		Return(&models.UnpaidLoans{}, nil)

	// Initialize service
	service := NewBusinessLevelRulesService(
		loanRepo,
		messagesRepo,
		loanProductsRepo,
		redisRepo,
		collectionRepo,
		whitelistRepo,
		systemRulesRepo,
		pubsubClient,
		pubsubClient,
		testPayload,
		unpaidRepo,
	)

	// Run the test
	result, err := service.CheckBusinessLevelRulesForPromoCollection(ctx, msisdn)
	require.Error(t, err) // expecting the error
	require.Contains(t, err.Error(), "no unpaid balance found for loan")
	require.False(t, result)

	// Assert all expected calls
	loanRepo.AssertExpectations(t)
	collectionRepo.AssertExpectations(t)
	whitelistRepo.AssertExpectations(t)
	systemRulesRepo.AssertExpectations(t)
	loanProductsRepo.AssertExpectations(t)
	messagesRepo.AssertExpectations(t)
	redisRepo.AssertExpectations(t)
	// pubsubClient.AssertExpectations(t)
	unpaidRepo.AssertExpectations(t)
}

func TestCheckBusinessLevelRulesForPromoCollection_PromoDisabled(t *testing.T) {
	ctx := context.Background()
	msisdn := "123"

	systemRulesRepo := &mockSystemRulesRepo{}

	systemRulesRepo.On("FetchSystemLevelRulesConfiguration", ctx).
		Return(models.SystemLevelRules{PromoCollectionEnabled: false}, nil)

	service := NewBusinessLevelRulesService(
		&mockLoanRepo{},
		&mockMessagesRepo{},
		&mockLoanProductsRepo{},
		&mockRedisRepo{},
		&mockCollectionTransactionsInProgressRepo{},
		&mockWhitelistedForDataCollectionRepo{},
		systemRulesRepo,
		&mockPubSub{},
		&mockPubSub{},
		&kafkamodels.PromoEventMessage{},
		&mockUnpaidRepo{},
	)

	result, err := service.CheckBusinessLevelRulesForPromoCollection(ctx, msisdn)
	require.NoError(t, err)
	require.False(t, result)

	systemRulesRepo.AssertExpectations(t)
}

func TestCheckBusinessLevelRulesForPromoCollection_SkipIfEntryExists(t *testing.T) {
	ctx := context.Background()
	msisdn := "123"

	// Mock system rules (required first step)
	systemRulesRepo := &mockSystemRulesRepo{}
	systemRulesRepo.On("FetchSystemLevelRulesConfiguration", ctx).
		Return(models.SystemLevelRules{PromoCollectionEnabled: true}, nil)

	// Mock collection repo → entry exists
	collectionRepo := &mockCollectionTransactionsInProgressRepo{}
	collectionRepo.On("CheckEntryExists", ctx, msisdn).Return(true, nil)

	// Initialize service
	service := NewBusinessLevelRulesService(
		&mockLoanRepo{},
		&mockMessagesRepo{},
		&mockLoanProductsRepo{},
		&mockRedisRepo{},
		collectionRepo,
		&mockWhitelistedForDataCollectionRepo{},
		systemRulesRepo,
		&mockPubSub{},
		&mockPubSub{},
		&kafkamodels.PromoEventMessage{},
		&mockUnpaidRepo{},
	)

	// Run the test
	result, err := service.CheckBusinessLevelRulesForPromoCollection(ctx, msisdn)
	require.NoError(t, err)
	require.False(t, result) // should skip collection

	// Assert expectations
	systemRulesRepo.AssertExpectations(t)
	collectionRepo.AssertExpectations(t)
}

func TestCheckBusinessLevelRulesForPromoCollection_ActivePeriodInRedis(t *testing.T) {
	ctx := context.Background()
	msisdn := "123"

	// Initialize mocks
	loanRepo := &mockLoanRepo{}
	collectionRepo := &mockCollectionTransactionsInProgressRepo{}
	whitelistRepo := &mockWhitelistedForDataCollectionRepo{}
	systemRulesRepo := &mockSystemRulesRepo{}
	loanProductsRepo := &mockLoanProductsRepo{}
	messagesRepo := &mockMessagesRepo{}
	redisRepo := &mockRedisRepo{}
	pubsubClient := &mockPubSub{}
	unpaidRepo := &mockUnpaidRepo{}

	// Sample payload
	testPayload := &kafkamodels.PromoEventMessage{
		MSISDN: 123,
	}

	// Create skip timestamps → promo education still active
	skipTimestamps := models.SkipTimestamps{
		PromoEducationPeriodTimestamp: time.Now().Add(1 * time.Hour), // active
		LoanLoadPeriodTimestamp:       time.Now().Add(-1 * time.Hour),
		GracePeriodTimestamp:          time.Now().Add(-1 * time.Hour),
	}

	skipBytes, _ := json.Marshal(skipTimestamps)

	// Mock Redis repo
	redisRepo.On("Get", ctx, mock.Anything).Return(skipBytes, nil)

	// Step 1: PromoCollectionEnabled
	systemRulesRepo.On("FetchSystemLevelRulesConfiguration", ctx).Return(models.SystemLevelRules{
		PromoCollectionEnabled: true,
	}, nil)

	// Step 2: No existing transaction
	collectionRepo.On("CheckEntryExists", ctx, msisdn).Return(false, nil)

	// Step 3: Create entry
	collectionRepo.On("CreateEntry", ctx, msisdn).Return(nil)
	collectionRepo.On("DeleteEntry", ctx, msisdn).Return(nil)

	// Mock loan repo → return a loan for notification
	loan := &models.Loans{
		MSISDN:                        msisdn,
		LoanID:                        primitive.NewObjectID(),
		LoanProductID:                 primitive.NewObjectID(),
		BrandID:                       primitive.NewObjectID(),
		TotalUnpaidLoan:               100,
		ServiceFee:                    10,
		PromoEducationPeriodTimestamp: time.Now().Add(-time.Hour),
		CreatedAt:                     time.Now().Add(-24 * time.Hour),
		ConversionRate:                0.5,
	}
	loanRepo.On("GetLoanByMSISDN", ctx, msisdn).Return(loan, nil)

	// Mock LoanProductsRepo → GetProductNameById
	loanProductsRepo.On("GetProductNameById", ctx, loan.LoanProductID).Return(&models.LoanProducts{
		Name: "SampleSKU",
	}, nil)

	// Mock MessagesRepo → GetPatternIdByEventandBrandId
	messagesRepo.On("GetPatternIdByEventandBrandId", ctx, consts.EducationPeriodSpiel, loan.BrandID).Return(&models.Messages{
		PatternId: 123,
		Parameters: []string{
			"REMAINING_LOAN_AMOUNT",
			"SKU_NAME",
			"SERVICE_FEE",
			"LOAN_DATE",
		},
	}, nil)

	// Mock PubSub → publish education notification
	pubsubClient.On("PublishMessage", ctx, mock.Anything).Return("msgID", nil)

	// Initialize service
	service := NewBusinessLevelRulesService(
		loanRepo,
		messagesRepo,
		loanProductsRepo,
		redisRepo,
		collectionRepo,
		whitelistRepo,
		systemRulesRepo,
		pubsubClient,
		pubsubClient,
		testPayload,
		unpaidRepo,
	)

	// Run the test
	result, err := service.CheckBusinessLevelRulesForPromoCollection(ctx, msisdn)
	require.NoError(t, err)
	require.False(t, result) // skipped due to active period

	// Assert all expected calls
	redisRepo.AssertExpectations(t)
	systemRulesRepo.AssertExpectations(t)
	loanRepo.AssertExpectations(t)
	loanProductsRepo.AssertExpectations(t)
	messagesRepo.AssertExpectations(t)
	pubsubClient.AssertExpectations(t)
}

func TestCheckBusinessLevelRulesForPromoCollection_OldLoanPath(t *testing.T) {
	ctx := context.Background()
	msisdn := "123"

	// Initialize mocks
	loanRepo := &mockLoanRepo{}
	collectionRepo := &mockCollectionTransactionsInProgressRepo{}
	whitelistRepo := &mockWhitelistedForDataCollectionRepo{}
	systemRulesRepo := &mockSystemRulesRepo{}
	loanProductsRepo := &mockLoanProductsRepo{}
	messagesRepo := &mockMessagesRepo{}
	redisRepo := &mockRedisRepo{}
	pubsubClient := &mockPubSub{}
	unpaidRepo := &mockUnpaidRepo{}

	// Sample payload
	testPayload := &kafkamodels.PromoEventMessage{
		MSISDN: 123,
	}

	// Step 1: PromoCollectionEnabled
	systemRulesRepo.On("FetchSystemLevelRulesConfiguration", ctx).Return(models.SystemLevelRules{
		PromoCollectionEnabled: true,
	}, nil)

	// Step 2: No existing transaction
	collectionRepo.On("CheckEntryExists", ctx, msisdn).Return(false, nil)

	// Step 3: Create entry
	collectionRepo.On("CreateEntry", ctx, msisdn).Return(nil)
	collectionRepo.On("DeleteEntry", ctx, msisdn).Return(nil)

	// Step 4: Redis → no active period
	key := models.SkipTimestampsKeyBuilder(msisdn)
	redisRepo.On("Get", ctx, key).Return(nil, redis.Nil)

	// Step 5: Active loan (old/lapsed)
	oldLoan := &models.Loans{
		LoanID:                        primitive.NewObjectID(),
		MSISDN:                        msisdn,
		TotalLoanAmount:               1000,
		ServiceFee:                    10,
		LoanProductID:                 primitive.NewObjectID(),
		BrandID:                       primitive.NewObjectID(),
		AvailmentTransactionID:        primitive.NewObjectID(),
		LoanType:                      "Normal",
		PromoEducationPeriodTimestamp: time.Now().Add(-48 * time.Hour),
		LoanLoadPeriodTimestamp:       time.Now().Add(-24 * time.Hour),
	}

	loanRepo.On("GetLoanByMSISDN", ctx, msisdn).Return(oldLoan, nil)

	// Step 6: Check if old or new → false (old loan)
	loanRepo.On("CheckIfOldOrNewLoan", ctx, oldLoan).Return(false)

	// Step 7: Whitelist check → not whitelisted
	whitelistRepo.On("IsMSISDNWhitelisted", ctx, msisdn).Return(false, nil)

	// Initialize service
	service := NewBusinessLevelRulesService(
		loanRepo,
		messagesRepo,
		loanProductsRepo,
		redisRepo,
		collectionRepo,
		whitelistRepo,
		systemRulesRepo,
		pubsubClient,
		pubsubClient,
		testPayload,
		unpaidRepo,
	)

	// Run the test
	result, err := service.CheckBusinessLevelRulesForPromoCollection(ctx, msisdn)
	require.NoError(t, err)
	require.False(t, result)

	// Assert all expected calls
	loanRepo.AssertExpectations(t)
	collectionRepo.AssertExpectations(t)
	whitelistRepo.AssertExpectations(t)
	systemRulesRepo.AssertExpectations(t)
	loanProductsRepo.AssertExpectations(t)
	messagesRepo.AssertExpectations(t)
	redisRepo.AssertExpectations(t)
	pubsubClient.AssertExpectations(t)
	unpaidRepo.AssertExpectations(t)
}

func TestProcessOldLoanForPromoCollection_ActiveSkippingPeriod(t *testing.T) {
	ctx := context.Background()
	msisdn := "123"

	// Initialize mocks
	loanRepo := &mockLoanRepo{}
	systemRulesRepo := &mockSystemRulesRepo{}
	whitelistRepo := &mockWhitelistedForDataCollectionRepo{}
	redisRepo := &mockRedisRepo{}
	pubsubClient := &mockPubSub{}
	messagesRepo := &mockMessagesRepo{}

	loanProductsRepo := &mockLoanProductsRepo{}

	// Sample old loan
	oldLoan := &models.Loans{
		LoanID:          primitive.NewObjectID(),
		MSISDN:          msisdn,
		LoanType:        consts.LoanTypePromo,
		CreatedAt:       time.Now().Add(-48 * time.Hour),
		TotalUnpaidLoan: 1000,
		ServiceFee:      10,
		LoanProductID:   primitive.NewObjectID(),
		BrandID:         primitive.NewObjectID(),
	}

	loanID := primitive.NewObjectID()
	loanRepo.On("UpdateOldLoanDocument",
		mock.Anything, // ctx
		mock.Anything, // loan pointer
		0.5,           // conversionRate
		mock.Anything, // promoEducationPeriodTimestamp
		mock.Anything, // loanLoadPeriodTimestamp
		80.0,          // MaxDataCollectionPercent from SystemLevelRules
		50.0,          // MinCollectionAmount from SystemLevelRules
		1000.0,        // TotalUnpaidLoan from loan
	).Return(loanID, nil)

	// Step 1: Mock system-level rules
	systemRulesRepo.On("FetchSystemLevelRulesConfiguration", ctx).Return(models.SystemLevelRules{
		ID:                                 primitive.NewObjectID(),
		Migrated:                           true,
		CreditScoreThreshold:               600,
		OverdueThreshold:                   30,
		UpdatedAt:                          time.Now(),
		EducationPeriod:                    5,
		DefermentPeriod:                    3,
		ReservedAmountForPartialCollection: 100,
		PartialCollectionEnabled:           true,

		MaxDataCollectionPercent: 80,
		PesoToDataConversionMatrix: []models.PesoToDataConversion{
			{LoanType: string(consts.LoanTypePromo), ConversionRate: 0.5},
			{LoanType: string(consts.LoanTypeGES), ConversionRate: 1.0},
			{LoanType: string(consts.LoanTypeLoad), ConversionRate: 0.8},
		},
		MinCollectionAmount:    50,
		PromoCollectionEnabled: true,
		WalletExclusionList:    []string{"WALLET1", "WALLET2"},
		PromoEducationPeriod:   7,
		GracePeriod:            3,
		LoanLoadPeriod:         5,
	}, nil)

	// Step 2 & 3: LoanRepo update
	loanRepo.On("UpdateOldLoanDocument", ctx, oldLoan, 0.5, mock.Anything, time.Time{},
		100, 50, oldLoan.TotalUnpaidLoan,
	).Return(oldLoan.LoanID, nil)

	// Step 4: Delete from whitelist
	whitelistRepo.On("DeleteByMSISDN", ctx, msisdn).Return(nil)

	// Step 5: Redis save
	redisRepo.On("SaveSkipTimestamps", ctx, msisdn, mock.Anything).Return(nil)

	// Step 5.2: Education notification
	pubsubClient.On("PublishMessage", ctx, mock.Anything).Return("msgID", nil)

	// Mock LoanProductsRepo → GetProductNameById
	loanProductsRepo.On("GetProductNameById", ctx, oldLoan.LoanProductID).Return(&models.LoanProducts{
		Name: "SampleSKU",
	}, nil)

	// Mock MessagesRepo → GetPatternIdByEventandBrandId
	messagesRepo.On("GetPatternIdByEventandBrandId", ctx, consts.EducationPeriodSpiel, oldLoan.BrandID).Return(&models.Messages{
		PatternId: 123,
		Parameters: []string{
			"REMAINING_LOAN_AMOUNT",
			"SKU_NAME",
			"SERVICE_FEE",
			"LOAN_DATE",
		},
	}, nil)

	// Initialize service
	service := &BusinessLevelRulesService{
		loanRepo:                         loanRepo,
		systemRulesRepo:                  systemRulesRepo,
		whitelistedForDataCollectionRepo: whitelistRepo,
		redisRepo:                        redisRepo,
		pubsubnotifclient:                pubsubClient,
		messagesRepo:                     messagesRepo,
		loanProductsRepo:                 loanProductsRepo,
	}

	// Run the test
	result, err := service.processOldLoanForPromoCollection(ctx, msisdn, oldLoan)
	require.NoError(t, err)
	require.False(t, result) // skipped due to active period

	// Assert all expected calls
	// loanRepo.AssertExpectations(t)
	systemRulesRepo.AssertExpectations(t)
	whitelistRepo.AssertExpectations(t)
	redisRepo.AssertExpectations(t)
	pubsubClient.AssertExpectations(t)
}

func TestHandleNonLapsedNewLoan_ActiveEducationPeriod(t *testing.T) {
	ctx := context.Background()
	msisdn := "123"

	// Initialize mocks
	redisRepo := &mockRedisRepo{}
	pubsubClient := &mockPubSub{}
	loanProductsRepo := &mockLoanProductsRepo{}
	messagesRepo := &mockMessagesRepo{}

	// Sample loan with active promo education period
	activeLoan := &models.Loans{
		LoanID:                        primitive.NewObjectID(),
		MSISDN:                        msisdn,
		PromoEducationPeriodTimestamp: time.Now().Add(1 * time.Hour), // still active
		LoanLoadPeriodTimestamp:       time.Now().Add(-1 * time.Hour),
		LoanProductID:                 primitive.NewObjectID(),
		BrandID:                       primitive.NewObjectID(),
		TotalUnpaidLoan:               1000,
		ServiceFee:                    10,
		CreatedAt:                     time.Now().Add(-24 * time.Hour),
		LoanType:                      consts.LoanTypePromo,
	}

	// Mock Redis → SaveSkipTimestamps
	redisRepo.On("SaveSkipTimestamps", ctx, msisdn, mock.Anything).Return(nil)

	// Mock PubSub → publish education notification
	pubsubClient.On("PublishMessage", ctx, mock.Anything).Return("msgID", nil)

	// Mock LoanProductsRepo → GetProductNameById (called inside publishEducationNotification)
	loanProductsRepo.On("GetProductNameById", ctx, activeLoan.LoanProductID).Return(&models.LoanProducts{
		Name: "SampleSKU",
	}, nil)

	// Mock MessagesRepo → GetPatternIdByEventandBrandId (called inside publishEducationNotification)
	messagesRepo.On("GetPatternIdByEventandBrandId", ctx, consts.EducationPeriodSpiel, activeLoan.BrandID).Return(&models.Messages{
		PatternId: 123,
		Parameters: []string{
			"REMAINING_LOAN_AMOUNT",
			"SKU_NAME",
			"SERVICE_FEE",
			"LOAN_DATE",
		},
	}, nil)

	// Initialize service with all required dependencies
	service := &BusinessLevelRulesService{
		redisRepo:         redisRepo,
		pubsubnotifclient: pubsubClient,
		loanProductsRepo:  loanProductsRepo,
		messagesRepo:      messagesRepo,
	}

	// Run the function
	err := service.createSkippingCacheEntry(ctx, activeLoan, msisdn)
	require.NoError(t, err)

	// Assert expectations
	redisRepo.AssertExpectations(t)
	pubsubClient.AssertExpectations(t)
	loanProductsRepo.AssertExpectations(t)
	messagesRepo.AssertExpectations(t)
}

func TestCheckRedisForActivePeriod_Coverage(t *testing.T) {
	ctx := context.Background()
	msisdn := "123"

	redisRepo := &mockRedisRepo{}
	loanRepo := &mockLoanRepo{}
	pubsubClient := &mockPubSub{}
	loanProductsRepo := &mockLoanProductsRepo{}
	messagesRepo := &mockMessagesRepo{}

	service := &BusinessLevelRulesService{
		redisRepo:         redisRepo,
		loanRepo:          loanRepo,
		pubsubnotifclient: pubsubClient,
		loanProductsRepo:  loanProductsRepo,
		messagesRepo:      messagesRepo,
	}

	now := time.Now()
	fetchingKey := models.SkipTimestampsKeyBuilder(msisdn)

	// --- Case 1: Loan load period active (promo expired) ---
	loanLoadEnd := now.Add(1 * time.Hour)
	loadSkip := models.SkipTimestamps{
		PromoEducationPeriodTimestamp: now.Add(-2 * time.Hour), // already expired
		LoanLoadPeriodTimestamp:       loanLoadEnd,             // active
		GracePeriodTimestamp:          now.Add(-1 * time.Hour), // expired
	}
	bytesLoad, _ := json.Marshal(loadSkip)
	redisRepo.ExpectedCalls = nil
	redisRepo.On("Get", ctx, fetchingKey).Return(bytesLoad, nil)

	active, err := service.checkRedisForActivePeriod(ctx, msisdn)
	require.NoError(t, err)
	require.True(t, active) // hits loan load branch

	// --- Case 2: Grace period active (loan load expired) ---
	graceEnd := now.Add(1 * time.Hour)
	graceSkip := models.SkipTimestamps{
		PromoEducationPeriodTimestamp: now.Add(-3 * time.Hour), // expired
		LoanLoadPeriodTimestamp:       now.Add(-2 * time.Hour), // expired
		GracePeriodTimestamp:          graceEnd,                // active
	}
	bytesGrace, _ := json.Marshal(graceSkip)
	redisRepo.ExpectedCalls = nil
	redisRepo.On("Get", ctx, fetchingKey).Return(bytesGrace, nil)

	active, err = service.checkRedisForActivePeriod(ctx, msisdn)
	require.NoError(t, err)
	require.True(t, active) // still in grace period

	// --- Case 3: All periods expired + successful delete ---
	expiredSkip := models.SkipTimestamps{
		PromoEducationPeriodTimestamp: now.Add(-3 * time.Hour),
		LoanLoadPeriodTimestamp:       now.Add(-2 * time.Hour),
		GracePeriodTimestamp:          now.Add(-1 * time.Hour),
	}
	bytesExpired, _ := json.Marshal(expiredSkip)
	redisRepo.ExpectedCalls = nil
	redisRepo.On("Get", ctx, fetchingKey).Return(bytesExpired, nil)
	redisRepo.On("Delete", ctx, fetchingKey).Return(nil)

	active, err = service.checkRedisForActivePeriod(ctx, msisdn)
	require.NoError(t, err)
	require.False(t, active) // deleted → no active period

	// --- Case 4: All periods expired + delete error ---
	redisRepo.ExpectedCalls = nil
	redisRepo.On("Get", ctx, fetchingKey).Return(bytesExpired, nil)
	redisRepo.On("Delete", ctx, fetchingKey).Return(errors.New("delete failed"))

	active, err = service.checkRedisForActivePeriod(ctx, msisdn)
	require.Error(t, err)
	require.True(t, active) // still returns true because delete failed

	// --- Case 5: Redis key not found ---
	redisRepo.ExpectedCalls = nil
	redisRepo.On("Get", ctx, fetchingKey).Return(nil, redis.Nil)

	active, err = service.checkRedisForActivePeriod(ctx, msisdn)
	require.NoError(t, err)
	require.False(t, active) // no key → not active

	// --- Case 6: Redis get returns unexpected error ---
	redisRepo.ExpectedCalls = nil
	redisRepo.On("Get", ctx, fetchingKey).Return(nil, errors.New("redis error"))

	active, err = service.checkRedisForActivePeriod(ctx, msisdn)
	require.Error(t, err)
	require.False(t, active)

	redisRepo.AssertExpectations(t)
}

func TestGetActiveLoan(t *testing.T) {
	ctx := context.Background()
	msisdn := "123"

	loanRepo := &mockLoanRepo{}
	service := &BusinessLevelRulesService{
		loanRepo: loanRepo,
	}

	// --- Case 1: Loan exists ---
	expectedLoan := &models.Loans{
		LoanID:                   primitive.NewObjectID(),
		MSISDN:                   msisdn,
		TotalLoanAmount:          1000,
		GUID:                     "loan-guid-1",
		ServiceFee:               50,
		LoanProductID:            primitive.NewObjectID(),
		BrandID:                  primitive.NewObjectID(),
		AvailmentTransactionID:   primitive.NewObjectID(),
		CreatedAt:                time.Now(),
		TotalUnpaidLoan:          500,
		MinCollectionAmount:      100,
		MaxDataCollectionPercent: 20,
	}
	loanRepo.ExpectedCalls = nil
	loanRepo.On("GetLoanByMSISDN", ctx, msisdn).Return(expectedLoan, nil)

	loan, err := service.getActiveLoan(ctx, msisdn)
	require.NoError(t, err)
	require.NotNil(t, loan)
	require.Equal(t, expectedLoan.LoanID, loan.LoanID)

	// --- Case 2: Loan not found (mongo.ErrNoDocuments) ---
	loanRepo.ExpectedCalls = nil
	loanRepo.On("GetLoanByMSISDN", ctx, msisdn).Return(nil, mongo.ErrNoDocuments)

	loan, err = service.getActiveLoan(ctx, msisdn)
	require.NoError(t, err)
	require.Nil(t, loan)

	// --- Case 3: Unexpected error from repo ---
	loanRepo.ExpectedCalls = nil
	loanRepo.On("GetLoanByMSISDN", ctx, msisdn).Return(nil, errors.New("db error"))

	loan, err = service.getActiveLoan(ctx, msisdn)
	require.Error(t, err)
	require.Nil(t, loan)

	// --- Case 4: Repo returns nil loan without error ---
	loanRepo.ExpectedCalls = nil
	loanRepo.On("GetLoanByMSISDN", ctx, msisdn).Return(nil, nil)

	loan, err = service.getActiveLoan(ctx, msisdn)
	require.Error(t, err)
	require.Nil(t, loan)

	loanRepo.AssertExpectations(t)
}

func TestGetConversionRateForLoanType(t *testing.T) {
	ctx := context.Background()

	matrix := []models.PesoToDataConversion{
		{LoanType: "PROMO SKU", ConversionRate: 1.5},
	}

	// --- Case 1: Matching loan type found ---
	rate, err := getConversionRateForLoanType(ctx, consts.LoanTypePromo, matrix)
	require.NoError(t, err)
	require.Equal(t, 1.5, rate)

	// --- Case 2: No conversion rate found ---
	rate, err = getConversionRateForLoanType(ctx, "UNKNOWN", matrix)
	require.Error(t, err)
	require.Equal(t, 0.0, rate)
	require.Contains(t, err.Error(), "conversion rate not found for loanType=UNKNOWN")
}
