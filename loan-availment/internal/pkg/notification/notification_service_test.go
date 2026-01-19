package notification

import (
	"context"
	"errors"
	"globe/dodrio_loan_availment/internal/pkg/consts"
	"globe/dodrio_loan_availment/internal/pkg/models"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MockMessagesRepo struct {
	mock.Mock
}

func (m *MockMessagesRepo) GetMessageID(ctx context.Context, event string, brand primitive.ObjectID, promoEnabled bool) (*models.MessageResponse, error) {
	args := m.Called(ctx, event, brand, promoEnabled)
	if args.Get(0) != nil {
		return args.Get(0).(*models.MessageResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

type MockSystemLevelRulesRepo struct {
	mock.Mock
}

func (m *MockSystemLevelRulesRepo) SystemLevelRules(filter interface{}) (models.SystemLevelRules, error) {
	args := m.Called(filter)
	if args.Get(0) != nil {
		return *args.Get(0).(*models.SystemLevelRules), args.Error(1)
	}
	return models.SystemLevelRules{}, args.Error(1)
}

type MockActiveLoanRepo struct{ mock.Mock }

func (m *MockActiveLoanRepo) ActiveLoanByFilter(filter interface{}) (*models.Loans, error) {
	args := m.Called(filter)
	if res := args.Get(0); res != nil {
		return res.(*models.Loans), args.Error(1)
	}
	return nil, args.Error(1)
}

type MockLoansProductsRepo struct{ mock.Mock }

func (m *MockLoansProductsRepo) LoanProductByFilter(filter interface{}) (*models.LoanProduct, error) {
	args := m.Called(filter)
	if res := args.Get(0); res != nil {
		return res.(*models.LoanProduct), args.Error(1)
	}
	return nil, args.Error(1)
}

type MockUnpaidLoanRepo struct{ mock.Mock }

func (m *MockUnpaidLoanRepo) UnpaidLoansByFilter(filter interface{}) (*models.UnpaidLoans, error) {
	args := m.Called(filter)
	if res := args.Get(0); res != nil {
		return res.(*models.UnpaidLoans), args.Error(1)
	}
	return nil, args.Error(1)
}

type MockPubSubPublisher struct {
	mock.Mock
}

func (m *MockPubSubPublisher) Publish(ctx context.Context, topic string, data []byte, attributes map[string]string) (string, error) {
	args := m.Called(ctx, topic, data, attributes)
	return args.String(0), args.Error(1)
}

func (m *MockPubSubPublisher) Stop(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockPubSubPublisher) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestNotifyUser_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	brand := primitive.NewObjectID()
	loanId := primitive.NewObjectID()

	mockMsgRepo := new(MockMessagesRepo)
	mockSysRepo := new(MockSystemLevelRulesRepo)
	mockPubSub := new(MockPubSubPublisher)
	mockActiveLoan := new(MockActiveLoanRepo)
	mockLoanProducts := new(MockLoansProductsRepo)
	mockUnpaidLoan := new(MockUnpaidLoanRepo)

	mockSysRepo.On("SystemLevelRules", mock.Anything).
		Return(&models.SystemLevelRules{PromoCollectionEnabled: true}, nil)
	mockMsgRepo.On("GetMessageID", ctx, "LOAN_APPROVED", brand, true).
		Return(&models.MessageResponse{MessageID: 123, Parameters: []string{"LoanDate"}}, nil)
	mockPubSub.On("Publish", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("msg-123", nil)
	mockActiveLoan.On("ActiveLoanByFilter", mock.Anything).
		Return(&models.Loans{LoanId: loanId, LoanProductId: primitive.NewObjectID(), CreatedAt: time.Now()}, nil)
	mockLoanProducts.On("LoanProductByFilter", mock.Anything).Return(&models.LoanProduct{Name: "TestProduct"}, nil).Maybe()
	mockUnpaidLoan.On("UnpaidLoansByFilter", mock.Anything).Return(&models.UnpaidLoans{TotalLoanAmount: 100, UnpaidAmount: 50}, nil).Maybe()

	svc := &NotificationService{
		messageRepo:             mockMsgRepo,
		systemLevelRulesRepo:    mockSysRepo,
		pubsubPublisher:         mockPubSub,
		activeLoanRepo:          mockActiveLoan,
		loansProductsRepository: mockLoanProducts,
		unpaidLoanRepo:          mockUnpaidLoan,
	}

	err := svc.NotifyUser(ctx, "09171234567", "LOAN_APPROVED", brand, &models.LoanProduct{Name: "TestLoan"}, loanId, time.Now())
	require.NoError(t, err)

	mockMsgRepo.AssertExpectations(t)
	mockSysRepo.AssertExpectations(t)
	mockPubSub.AssertExpectations(t)
	mockActiveLoan.AssertExpectations(t)
}

func TestNotifyUser_PubSubFailure(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	brand := primitive.NewObjectID()
	loanId := primitive.NewObjectID()

	mockMsgRepo := new(MockMessagesRepo)
	mockSysRepo := new(MockSystemLevelRulesRepo)
	mockPubSub := new(MockPubSubPublisher)
	mockActiveLoan := new(MockActiveLoanRepo)
	mockLoanProducts := new(MockLoansProductsRepo)
	mockUnpaidLoan := new(MockUnpaidLoanRepo)

	mockSysRepo.On("SystemLevelRules", mock.Anything).
		Return(&models.SystemLevelRules{PromoCollectionEnabled: false}, nil)
	mockMsgRepo.On("GetMessageID", ctx, "LoanAvailmentSuccess", brand, false).
		Return(&models.MessageResponse{MessageID: 456, Parameters: []string{"SkuName"}}, nil)
	mockPubSub.On("Publish", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("", errors.New("pubsub publish failed"))
	mockActiveLoan.On("ActiveLoanByFilter", mock.Anything).
		Return(&models.Loans{LoanId: loanId, LoanProductId: primitive.NewObjectID(), CreatedAt: time.Now()}, nil)
	mockLoanProducts.On("LoanProductByFilter", mock.Anything).Return(&models.LoanProduct{Name: "TestProduct"}, nil).Maybe()
	mockUnpaidLoan.On("UnpaidLoansByFilter", mock.Anything).Return(&models.UnpaidLoans{TotalLoanAmount: 100, UnpaidAmount: 50}, nil).Maybe()

	svc := &NotificationService{
		messageRepo:             mockMsgRepo,
		systemLevelRulesRepo:    mockSysRepo,
		pubsubPublisher:         mockPubSub,
		activeLoanRepo:          mockActiveLoan,
		loansProductsRepository: mockLoanProducts,
		unpaidLoanRepo:          mockUnpaidLoan,
	}

	err := svc.NotifyUser(ctx, "09171234567", "LoanAvailmentSuccess", brand, &models.LoanProduct{Name: "TestLoan"}, loanId, time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to publish to pubsub")

	mockMsgRepo.AssertExpectations(t)
	mockSysRepo.AssertExpectations(t)
	mockPubSub.AssertExpectations(t)
	mockActiveLoan.AssertExpectations(t)
}

func TestNotifyUser_ErrorCases(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	brand := primitive.NewObjectID()
	loanId := primitive.NewObjectID()

	tests := []struct {
		name        string
		setupMocks  func(*MockMessagesRepo, *MockSystemLevelRulesRepo, *MockActiveLoanRepo, *MockLoansProductsRepo, *MockUnpaidLoanRepo, *MockPubSubPublisher)
		expectError bool
	}{
		{
			name: "SystemLevelRules error",
			setupMocks: func(msg *MockMessagesRepo, sys *MockSystemLevelRulesRepo, act *MockActiveLoanRepo, loan *MockLoansProductsRepo, unpaid *MockUnpaidLoanRepo, pub *MockPubSubPublisher) {
				sys.On("SystemLevelRules", mock.Anything).Return(nil, errors.New("system rules error"))
			},
			expectError: true,
		},
		{
			name: "GetMessageID error",
			setupMocks: func(msg *MockMessagesRepo, sys *MockSystemLevelRulesRepo, act *MockActiveLoanRepo, loan *MockLoansProductsRepo, unpaid *MockUnpaidLoanRepo, pub *MockPubSubPublisher) {
				sys.On("SystemLevelRules", mock.Anything).Return(&models.SystemLevelRules{PromoCollectionEnabled: true}, nil)
				msg.On("GetMessageID", ctx, "EVT", brand, true).Return(nil, errors.New("message ID error"))
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockMsg := new(MockMessagesRepo)
			mockSys := new(MockSystemLevelRulesRepo)
			mockAct := new(MockActiveLoanRepo)
			mockLoan := new(MockLoansProductsRepo)
			mockUnpaid := new(MockUnpaidLoanRepo)
			mockPubSub := new(MockPubSubPublisher)

			mockAct.On("ActiveLoanByFilter", mock.Anything).Return(&models.Loans{
				LoanId: primitive.NewObjectID(), LoanProductId: primitive.NewObjectID(), CreatedAt: time.Now(), TotalLoanAmount: 100,
			}, nil).Maybe()
			mockLoan.On("LoanProductByFilter", mock.Anything).Return(&models.LoanProduct{Name: "DefaultProduct", Price: 100, ServiceFee: 10, FeeType: string(consts.Percent)}, nil).Maybe()
			mockUnpaid.On("UnpaidLoansByFilter", mock.Anything).Return(&models.UnpaidLoans{TotalLoanAmount: 100, UnpaidAmount: 50}, nil).Maybe()
			mockPubSub.On("Publish", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("msg-id", nil).Maybe()

			tc.setupMocks(mockMsg, mockSys, mockAct, mockLoan, mockUnpaid, mockPubSub)

			svc := &NotificationService{
				messageRepo: mockMsg, systemLevelRulesRepo: mockSys, activeLoanRepo: mockAct,
				loansProductsRepository: mockLoan, unpaidLoanRepo: mockUnpaid, pubsubPublisher: mockPubSub,
			}

			err := svc.NotifyUser(ctx, "0917xxxxxxx", "EVT", brand, nil, loanId, time.Now())

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			mockMsg.AssertExpectations(t)
			mockSys.AssertExpectations(t)
		})
	}
}

func TestSendNotificationToPubSub(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockPubSub := new(MockPubSubPublisher)
	mockPubSub.On("Publish", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("msg-456", nil)

	mockActiveLoan := new(MockActiveLoanRepo)
	mockLoanProducts := new(MockLoansProductsRepo)
	mockUnpaidLoan := new(MockUnpaidLoanRepo)

	mockActiveLoan.On("ActiveLoanByFilter", mock.Anything).Return(&models.Loans{
		LoanId: primitive.NewObjectID(), LoanProductId: primitive.NewObjectID(), CreatedAt: time.Now(),
	}, nil).Maybe()
	mockLoanProducts.On("LoanProductByFilter", mock.Anything).Return(&models.LoanProduct{Name: "TestProduct"}, nil).Maybe()
	mockUnpaidLoan.On("UnpaidLoansByFilter", mock.Anything).Return(&models.UnpaidLoans{TotalLoanAmount: 100, UnpaidAmount: 50}, nil).Maybe()

	svc := &NotificationService{
		pubsubPublisher: mockPubSub, activeLoanRepo: mockActiveLoan,
		loansProductsRepository: mockLoanProducts, unpaidLoanRepo: mockUnpaidLoan,
	}

	payload := models.SmsNotificationRequestPayload{
		NotificationParameter: []models.NotificationParameter{{Name: "test", Value: "test-value"}},
		PatternID:             123,
	}

	err := svc.sendNotificationToPubSub(ctx, payload, "09171234567", "TEST_EVENT")
	require.NoError(t, err)
	mockPubSub.AssertExpectations(t)
}

func TestGetValuesOfParameters(t *testing.T) {
	t.Parallel()
	loanId := primitive.NewObjectID()
	loanDate := time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC)

	t.Run("All parameter types", func(t *testing.T) {
		t.Parallel()
		activeRepo := new(MockActiveLoanRepo)
		loanRepo := new(MockLoansProductsRepo)
		unpaidRepo := new(MockUnpaidLoanRepo)

		product := &models.LoanProduct{Name: "LoanX", FeeType: string(consts.Percent), Price: 100, ServiceFee: 10}
		activeRepo.On("ActiveLoanByFilter", mock.Anything).Return(&models.Loans{LoanId: loanId, LoanProductId: primitive.NewObjectID(), CreatedAt: loanDate, TotalLoanAmount: 200}, nil)
		loanRepo.On("LoanProductByFilter", mock.Anything).Return(product, nil)
		unpaidRepo.On("UnpaidLoansByFilter", mock.Anything).Return(&models.UnpaidLoans{UnpaidAmount: 50, TotalLoanAmount: 200}, nil)

		svc := &NotificationService{activeLoanRepo: activeRepo, loansProductsRepository: loanRepo, unpaidLoanRepo: unpaidRepo}

		params := []string{consts.SkuName, consts.ServiceFee, consts.LoanFeeAmt, consts.RemainingLoanAmount, consts.LoanDate, consts.AmountCollected, "unknownParam"}

		res := svc.getValuesOfParameters(params, "0917", *product, loanId, loanDate)
		assert.Len(t, res, len(params))
		assert.Equal(t, "LoanX", res[0].Value)
		assert.Equal(t, "10.00", res[1].Value)
		assert.Equal(t, "200.00", res[2].Value)
		assert.Equal(t, "50.00", res[3].Value)
		assert.Equal(t, "2025-09-01", res[4].Value)
		assert.Equal(t, "150.00", res[5].Value)
		assert.Equal(t, "", res[6].Value)
	})

	t.Run("ServiceFee variations", func(t *testing.T) {
		t.Parallel()
		mockActive1 := new(MockActiveLoanRepo)
		mockLoan1 := new(MockLoansProductsRepo)
		mockUnpaid1 := new(MockUnpaidLoanRepo)

		mockActive1.On("ActiveLoanByFilter", mock.Anything).Return(&models.Loans{
			LoanId: primitive.NewObjectID(), LoanProductId: primitive.NewObjectID(), CreatedAt: time.Now(), TotalLoanAmount: 100,
		}, nil)
		mockUnpaid1.On("UnpaidLoansByFilter", mock.Anything).Return(&models.UnpaidLoans{TotalLoanAmount: 100, UnpaidAmount: 50}, nil).Maybe()

		svc1 := &NotificationService{activeLoanRepo: mockActive1, loansProductsRepository: mockLoan1, unpaidLoanRepo: mockUnpaid1}
		product := &models.LoanProduct{Name: "LoanY", FeeType: "Flat", Price: 100, ServiceFee: 25}
		res1 := svc1.getValuesOfParameters([]string{consts.ServiceFee}, "0917", *product, loanId, loanDate)
		assert.Equal(t, "25.00", res1[0].Value)

		activeRepo := new(MockActiveLoanRepo)
		loanRepo := new(MockLoansProductsRepo)
		unpaidRepo := new(MockUnpaidLoanRepo)
		activeRepo.On("ActiveLoanByFilter", mock.Anything).Return(nil, errors.New("db fail"))

		svc2 := &NotificationService{activeLoanRepo: activeRepo, loansProductsRepository: loanRepo, unpaidLoanRepo: unpaidRepo}
		res2 := svc2.getValuesOfParameters([]string{consts.ServiceFee}, "0917", models.LoanProduct{}, primitive.NilObjectID, loanDate)
		assert.Equal(t, "0.00", res2[0].Value)

		mockActive1.AssertExpectations(t)
		activeRepo.AssertExpectations(t)
	})

	t.Run("Error handling", func(t *testing.T) {
		t.Parallel()
		activeRepo := new(MockActiveLoanRepo)
		loanRepo := new(MockLoansProductsRepo)
		unpaidRepo := new(MockUnpaidLoanRepo)

		activeRepo.On("ActiveLoanByFilter", mock.Anything).Return(nil, errors.New("db fail"))
		loanRepo.On("LoanProductByFilter", mock.Anything).Return(&models.LoanProduct{Name: "TestProduct"}, nil).Maybe()
		unpaidRepo.On("UnpaidLoansByFilter", mock.Anything).Return(&models.UnpaidLoans{TotalLoanAmount: 100, UnpaidAmount: 50}, nil).Maybe()

		svc := &NotificationService{activeLoanRepo: activeRepo, loansProductsRepository: loanRepo, unpaidLoanRepo: unpaidRepo}
		res := svc.getValuesOfParameters([]string{consts.LoanFeeAmt}, "0917", models.LoanProduct{}, primitive.NilObjectID, loanDate)
		assert.Equal(t, "0.00", res[0].Value)
	})

	t.Run("Data parameters", func(t *testing.T) {
		t.Parallel()
		activeRepo := new(MockActiveLoanRepo)
		loanRepo := new(MockLoansProductsRepo)
		unpaidRepo := new(MockUnpaidLoanRepo)

		activeRepo.On("ActiveLoanByFilter", mock.Anything).Return(&models.Loans{
			ConversionRate: 1.5, TotalLoanAmount: 100, TotalUnpaidLoan: 50,
			PromoEducationPeriodTimestamp: primitive.NewDateTimeFromTime(time.Now()), LoanProductId: primitive.NewObjectID(),
		}, nil)
		loanRepo.On("LoanProductByFilter", mock.Anything).Return(&models.LoanProduct{Name: "TestProduct"}, nil).Maybe()
		unpaidRepo.On("UnpaidLoansByFilter", mock.Anything).Return(&models.UnpaidLoans{TotalLoanAmount: 100, UnpaidAmount: 50}, nil).Maybe()

		svc := &NotificationService{activeLoanRepo: activeRepo, loansProductsRepository: loanRepo, unpaidLoanRepo: unpaidRepo}
		res := svc.getValuesOfParameters([]string{consts.DataConversion, consts.DataMB, consts.DataCollectionDate}, "0917", models.LoanProduct{}, primitive.NilObjectID, loanDate)

		assert.Equal(t, "150.00", res[0].Value)
		assert.Equal(t, "75.00 MB", res[1].Value)
		assert.NotEmpty(t, res[2].Value)
		activeRepo.AssertExpectations(t)
	})
}
