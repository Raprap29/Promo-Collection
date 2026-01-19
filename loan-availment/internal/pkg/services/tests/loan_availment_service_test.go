package tests

import (
	"context"
	"errors"
	"fmt"

	"globe/dodrio_loan_availment/internal/pkg/consts"

	"globe/dodrio_loan_availment/internal/pkg/models"
	"globe/dodrio_loan_availment/internal/pkg/services"

	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestAvailLoan(t *testing.T) {
	tests := []struct {
		name          string
		request       models.AppCreationRequest
		setupMocks    func(*MockHelpers, *MockBrandExists, *MockLoanAvailmentRepo, *MockKafkaService, *MockProcessRequest, *MockTransactionInProgress, *MockNotificationServices)
		expectedCode  int
		expectedError error
	}{
		{
			name: "Success Case",
			request: models.AppCreationRequest{
				MSISDN:        "09123456789",
				TransactionId: "SMS123",
				Keyword:       "PAYLOAN",
				SystemClient:  "test-client",
			},
			setupMocks: func(mh *MockHelpers, mb *MockBrandExists, mar *MockLoanAvailmentRepo, mks *MockKafkaService, mpr *MockProcessRequest, mtp *MockTransactionInProgress, mns *MockNotificationServices) {
				mpr.On("ProcessRequest",
					"PAYLOAN",
					"9123456789",
					"test-client",
					"SMS",
					"test-client",
					"123",
					float64(0),
					mock.Anything,
				).Return()
			},
			expectedCode:  http.StatusOK,
			expectedError: nil,
		},
		{
			name: "Invalid MSISDN Format",
			request: models.AppCreationRequest{
				MSISDN:        "invalid-msisdn",
				TransactionId: "123",
				Keyword:       "PAYLOAN",
				SystemClient:  "test-client",
			},
			setupMocks: func(mh *MockHelpers, mb *MockBrandExists, mar *MockLoanAvailmentRepo, mks *MockKafkaService, mpr *MockProcessRequest, mtp *MockTransactionInProgress, mns *MockNotificationServices) {
				availment := models.Availments{
					ID:        primitive.NewObjectID(),
					ErrorText: consts.ErrorMsisdnFormatValidationFailed.Error(),
				}

				mar.On("InsertFailedAvailment",
					mock.Anything, "invalid-msisdn", "PAYLOAN", mock.Anything, "test-client",
					consts.ErrorMsisdnFormatValidationFailed.Error(), mock.Anything, false,
					float64(0), (*models.LoanProduct)(nil), "",
				).Return(availment, nil, "", "", nil)

				mks.On("PublishAvailmentStatusToKafka",
					mock.Anything,
					mock.MatchedBy(func(id string) bool { return len(id) > 0 }),
					mock.Anything,
					"invalid-msisdn",
					mock.Anything,
					mock.Anything,
					"",
					"PAYLOAN",
					mock.Anything,
					float64(0),
					float64(0),
					consts.FailedAvailmentResult,
					availment.ErrorText,
					int32(0),
					"",
				).Return(nil)

				mar.On("UpdateAvailment",
					mock.Anything,
					availment.ID,
					bson.M{"publishedToKafka": true},
				).Return()
			},
			expectedCode:  http.StatusOK,
			expectedError: consts.ErrorMsisdnFormatValidationFailed,
		},
		{
			name: "Invalid Transaction ID Format",
			request: models.AppCreationRequest{
				MSISDN:        "9123456789",
				TransactionId: "",
				Keyword:       "PAYLOAN",
				SystemClient:  "test-client",
			},
			setupMocks: func(mh *MockHelpers, mb *MockBrandExists, mar *MockLoanAvailmentRepo, mks *MockKafkaService, mpr *MockProcessRequest, mtp *MockTransactionInProgress, mns *MockNotificationServices) {
				availment := models.Availments{
					ID:        primitive.NewObjectID(),
					ErrorText: consts.ErrorTransactionIdFormatValidationFailed.Error(),
				}

				mar.On("InsertFailedAvailment",
					mock.Anything, "9123456789", "PAYLOAN", mock.Anything, "test-client",
					consts.ErrorTransactionIdFormatValidationFailed.Error(), mock.Anything,
					false, float64(0), (*models.LoanProduct)(nil), "",
				).Return(availment, nil, "", "", nil)

				mks.On("PublishAvailmentStatusToKafka",
					mock.Anything, availment.ID.Hex(), mock.Anything, "9123456789",
					mock.Anything, mock.Anything, "", "PAYLOAN", mock.Anything,
					float64(0), mock.Anything, consts.FailedAvailmentResult,
					availment.ErrorText, int32(0), "",
				).Return(nil)

				mar.On("UpdateAvailment",
					mock.Anything,
					availment.ID,
					bson.M{"publishedToKafka": true},
				).Return()

			},
			expectedCode:  http.StatusOK,
			expectedError: consts.ErrorTransactionIdFormatValidationFailed,
		},
		{
			name: "Invalid Keyword Format",
			request: models.AppCreationRequest{
				MSISDN:        "9123456789",
				TransactionId: "SMS123",
				Keyword:       "",
				SystemClient:  "test-client",
			},
			setupMocks: func(mh *MockHelpers, mb *MockBrandExists, mar *MockLoanAvailmentRepo, mks *MockKafkaService, mpr *MockProcessRequest, mtp *MockTransactionInProgress, mns *MockNotificationServices) {
				availment := models.Availments{
					ID:        primitive.NewObjectID(),
					ErrorText: consts.ErrorKeywordFormatValidationFailed.Error(),
				}

				mar.On("InsertFailedAvailment",
					mock.Anything, "9123456789", "", mock.Anything, "test-client",
					consts.ErrorKeywordFormatValidationFailed.Error(), mock.Anything,
					false, float64(0), (*models.LoanProduct)(nil), "",
				).Return(availment, nil, "", "", nil)

				mks.On("PublishAvailmentStatusToKafka",
					mock.Anything, availment.ID.Hex(), mock.Anything, "9123456789",
					mock.Anything, mock.Anything, "", "", mock.Anything,
					float64(0), mock.Anything, consts.FailedAvailmentResult,
					availment.ErrorText, int32(0), "",
				).Return(nil)

				mar.On("UpdateAvailment",
					mock.Anything,
					availment.ID,
					bson.M{"publishedToKafka": true},
				).Return()
				mtp.On("DeleteTransactionInProgressByMsisdn",
					mock.Anything,
					"9123456789",
				).Return(true, nil)
			},
			expectedCode:  http.StatusOK,
			expectedError: consts.ErrorKeywordFormatValidationFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			MockHelpers := new(MockHelpers)
			MockBrandExists := new(MockBrandExists)
			mockLoanAvailmentRepo := new(MockLoanAvailmentRepo)
			mockKafkaService := new(MockKafkaService)
			mockProcessRequest := new(MockProcessRequest)
			mockTransactionInProgress := new(MockTransactionInProgress)
			mockNotificationService := new(MockNotificationServices)

			tt.setupMocks(MockHelpers, MockBrandExists, mockLoanAvailmentRepo, mockKafkaService, mockProcessRequest, mockTransactionInProgress, mockNotificationService)

			service := services.NewLoanAvailmentService(
				nil,
				nil,
				mockKafkaService,
				mockProcessRequest,
				mockTransactionInProgress,
				nil,
				nil,
				nil,
				nil,
				nil,
				mockNotificationService,
			)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", "/", nil)
			c.Request = c.Request.WithContext(context.Background())

			err := service.AvailLoan(c, tt.request)

			time.Sleep(100 * time.Millisecond)

			assert.Equal(t, tt.expectedError, err)
			assert.Equal(t, tt.expectedCode, w.Code)

			MockHelpers.AssertExpectations(t)
			MockBrandExists.AssertExpectations(t)
			mockLoanAvailmentRepo.AssertExpectations(t)
			mockKafkaService.AssertExpectations(t)
			mockProcessRequest.AssertExpectations(t)
			mockTransactionInProgress.AssertExpectations(t)
		})
	}
}

func TestAvailLoanRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Mock dependencies
	// mockAvailmentRepo := NewMockLoanAvailmentServicesInt(ctrl)
	mockProducer := NewMockKafkaServiceInterface(ctrl)
	mockTransactionRepo := NewMockTransactionInProgressInterface(ctrl)
	mockCheckEligibilityRepo := NewMockCheckEligibilityServiceInterface(ctrl)
	mockKeywordRepo := NewMockProductNamesInAmaxService(ctrl)
	mockAmaxService := NewMockProvisionLoanAmaxService(ctrl)
	mockSystemLevelRulesRepo := NewMockSystemLevelRulesRepository(ctrl)
	mockNotificationService := NewMockNotificationServices(ctrl)

	processService := services.NewProcessRequestServiceImpl(
		nil,
		nil,
		nil,
		mockProducer,
		mockTransactionRepo,
		mockCheckEligibilityRepo,
		mockKeywordRepo,
		mockAmaxService,
		nil,
		nil,
		mockSystemLevelRulesRepo,
		mockNotificationService,
	)

	type args struct {
		request models.AppCreationProcessRequest
		ctx     context.Context
	}

	tests := []struct {
		name           string
		args           args
		setupMocks     func()
		expectedResult bool
		expectedError  error
	}{
		{
			name: "Successful eligibility and loan provision",
			args: args{
				request: models.AppCreationProcessRequest{
					MSISDN:        "9234567890",
					Keyword:       "keyword1",
					Channel:       "channel1",
					CustomerType:  "type1",
					CreditScore:   700,
					BrandCode:     "brand1",
					BrandId:       primitive.NewObjectID(),
					TransactionID: "USSD1234",
				},
				ctx: context.Background(),
			},
			setupMocks: func() {
				mockCheckEligibilityRepo.EXPECT().
					CheckEligibility(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(true, "loan", &models.LoanProduct{}, primitive.NewObjectID(), time.Now(), nil)

				mockKeywordRepo.EXPECT().
					ProductNamesInAmaxByKeyword(gomock.Any(), gomock.Any()).
					Return("LoanProductName", nil)
				mockAmaxService.EXPECT().
					ProvisionLoanAmax(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expectedResult: true,
			expectedError:  nil,
		},
		{
			name: "Loan provision failure",
			args: args{
				request: models.AppCreationProcessRequest{
					MSISDN:        "1122334455",
					Keyword:       "loan",
					Channel:       "channel3",
					CustomerType:  "type3",
					CreditScore:   800,
					BrandCode:     "brand3",
					BrandId:       primitive.NewObjectID(),
					TransactionID: "txn789",
				},
				ctx: context.Background(),
			},
			setupMocks: func() {
				mockCheckEligibilityRepo.EXPECT().
					CheckEligibility(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(true, "loan", &models.LoanProduct{}, primitive.NewObjectID(), time.Now(), nil)
				mockKeywordRepo.EXPECT().
					ProductNamesInAmaxByKeyword(gomock.Any(), gomock.Any()).
					Return("LoanProductName", nil)

				mockAmaxService.EXPECT().
					ProvisionLoanAmax(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("provisioning failed"))
			},
			expectedResult: true,
			expectedError:  errors.New("provisioning failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			result, err := processService.AvailLoanRequest(tt.args.request, tt.args.ctx)

			// Assertions
			assert.Equal(t, tt.expectedResult, result)
			if tt.expectedError != nil {
				assert.EqualError(t, err, tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

type MockLoanAvailmentServicesInt struct {
	ctrl     *gomock.Controller
	recorder *MockLoanAvailmentServicesIntMockRecorder
}

type MockLoanAvailmentServicesIntMockRecorder struct {
	mock *MockLoanAvailmentServicesInt
}

func NewMockLoanAvailmentServicesInt(ctrl *gomock.Controller) *MockLoanAvailmentServicesInt {
	mock := &MockLoanAvailmentServicesInt{ctrl: ctrl}
	mock.recorder = &MockLoanAvailmentServicesIntMockRecorder{mock}
	return mock
}

func (m *MockLoanAvailmentServicesInt) EXPECT() *MockLoanAvailmentServicesIntMockRecorder {
	return m.recorder
}

func (m *MockLoanAvailmentServicesInt) InsertFailedAvailment(ctx context.Context, msisdn, keyword, channel, systemClient, errMsg, errCode string, result bool, creditScore float64, loanProduct *models.LoanProduct, brandCode string) (models.Availments, *models.LoanProduct, string, string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InsertFailedAvailment", ctx, msisdn, keyword, channel, systemClient, errMsg, errCode, result, creditScore, loanProduct, brandCode)
	ret0, _ := ret[0].(models.Availments)
	ret1, _ := ret[1].(*models.LoanProduct)
	ret2, _ := ret[2].(string)
	ret3, _ := ret[3].(string)
	ret4, _ := ret[4].(error)
	return ret0, ret1, ret2, ret3, ret4
}

func (mr *MockLoanAvailmentServicesIntMockRecorder) InsertFailedAvailment(ctx, msisdn, keyword, channel, systemClient, errMsg, errCode, result, creditScore, loanProduct, brandCode interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InsertFailedAvailment", reflect.TypeOf((*MockLoanAvailmentServicesInt)(nil).InsertFailedAvailment), ctx, msisdn, keyword, channel, systemClient, errMsg, errCode, result, creditScore, loanProduct, brandCode)
}

func (m *MockLoanAvailmentServicesInt) UpdateAvailment(ctx context.Context, availmentID primitive.ObjectID, updateField bson.M) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "UpdateAvailment", ctx, availmentID, updateField)
}

func (mr *MockLoanAvailmentServicesIntMockRecorder) UpdateAvailment(ctx, availmentID, updateField interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateAvailment", reflect.TypeOf((*MockLoanAvailmentServicesInt)(nil).UpdateAvailment), ctx, availmentID, updateField)
}

type MockKafkaServiceInterface struct {
	ctrl     *gomock.Controller
	recorder *MockKafkaServiceInterfaceMockRecorder
}

type MockKafkaServiceInterfaceMockRecorder struct {
	mock *MockKafkaServiceInterface
}

func NewMockKafkaServiceInterface(ctrl *gomock.Controller) *MockKafkaServiceInterface {
	mock := &MockKafkaServiceInterface{ctrl: ctrl}
	mock.recorder = &MockKafkaServiceInterfaceMockRecorder{mock}
	return mock
}

func (m *MockKafkaServiceInterface) EXPECT() *MockKafkaServiceInterfaceMockRecorder {
	return m.recorder
}

type MockNotificationServices struct {
	ctrl     *gomock.Controller
	recorder *MockNotificationServicesMockRecorder
}

type MockNotificationServicesMockRecorder struct {
	mock *MockNotificationServices
}

func NewMockNotificationServices(ctrl *gomock.Controller) *MockNotificationServices {
	mock := &MockNotificationServices{ctrl: ctrl}
	mock.recorder = &MockNotificationServicesMockRecorder{mock}
	return mock
}

func (m *MockNotificationServices) EXPECT() *MockNotificationServicesMockRecorder {
	return m.recorder
}

func (m *MockNotificationServices) NotifyUser(ctx context.Context, MSISDN, event string, brand primitive.ObjectID, product *models.LoanProduct, loanId primitive.ObjectID, loanDate time.Time) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NotifyUser", ctx, MSISDN, event, brand, product, loanId, loanDate)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockNotificationServicesMockRecorder) NotifyUser(ctx, MSISDN, event, brand, product, loanId, loanDate interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NotifyUser", reflect.TypeOf((*MockNotificationServices)(nil).NotifyUser), ctx, MSISDN, event, brand, product, loanId, loanDate)
}
func (m *MockKafkaServiceInterface) PublishAvailmentStatusToKafka(ctx context.Context, transactionID, availmentChannel, msisdn, brandType, loanType, loanProductName, loanProductKeyword, servicingPartner string, loanAmount, serviceFeeAmount float64, availmentResult, availmentErrorText string, loanAgeing int32, statusOfProvisionedLoan string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PublishAvailmentStatusToKafka", ctx, transactionID, availmentChannel, msisdn, brandType, loanType, loanProductName, loanProductKeyword, servicingPartner, loanAmount, serviceFeeAmount, availmentResult, availmentErrorText, loanAgeing, statusOfProvisionedLoan)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockKafkaServiceInterfaceMockRecorder) PublishAvailmentStatusToKafka(ctx, transactionID, availmentChannel, msisdn, brandType, loanType, loanProductName, loanProductKeyword, servicingPartner, loanAmount, serviceFeeAmount, availmentResult, availmentErrorText, loanAgeing, statusOfProvisionedLoan interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PublishAvailmentStatusToKafka", reflect.TypeOf((*MockKafkaServiceInterface)(nil).PublishAvailmentStatusToKafka), ctx, transactionID, availmentChannel, msisdn, brandType, loanType, loanProductName, loanProductKeyword, servicingPartner, loanAmount, serviceFeeAmount, availmentResult, availmentErrorText, loanAgeing, statusOfProvisionedLoan)
}

type MockCheckEligibilityServiceInterface struct {
	ctrl     *gomock.Controller
	recorder *MockCheckEligibilityServiceInterfaceMockRecorder
}

type MockCheckEligibilityServiceInterfaceMockRecorder struct {
	mock *MockCheckEligibilityServiceInterface
}

func NewMockCheckEligibilityServiceInterface(ctrl *gomock.Controller) *MockCheckEligibilityServiceInterface {
	mock := &MockCheckEligibilityServiceInterface{ctrl: ctrl}
	mock.recorder = &MockCheckEligibilityServiceInterfaceMockRecorder{mock}
	return mock
}

func (m *MockCheckEligibilityServiceInterface) EXPECT() *MockCheckEligibilityServiceInterfaceMockRecorder {
	return m.recorder
}

func (m *MockCheckEligibilityServiceInterface) CheckEligibility(ctx context.Context, msisdn, keyword, channel, customer_type string, credit_score_value float64, brand_type string, brandId primitive.ObjectID) (bool, string, *models.LoanProduct, primitive.ObjectID, time.Time, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CheckEligibility", ctx, msisdn, keyword, channel, customer_type, credit_score_value, brand_type, brandId)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(string)
	ret2, _ := ret[2].(*models.LoanProduct)
	ret3, _ := ret[3].(primitive.ObjectID)
	ret4, _ := ret[4].(time.Time)
	ret5, _ := ret[5].(error)
	return ret0, ret1, ret2, ret3, ret4, ret5
}

func (mr *MockCheckEligibilityServiceInterfaceMockRecorder) CheckEligibility(ctx, msisdn, keyword, channel, customer_type, credit_score_value, brand_type, brandId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckEligibility", reflect.TypeOf((*MockCheckEligibilityServiceInterface)(nil).CheckEligibility), ctx, msisdn, keyword, channel, customer_type, credit_score_value, brand_type, brandId)
}

type MockProcessRequestService struct {
	ctrl     *gomock.Controller
	recorder *MockProcessRequestServiceMockRecorder
}

type MockProcessRequestServiceMockRecorder struct {
	mock *MockProcessRequestService
}

func NewMockProcessRequestService(ctrl *gomock.Controller) *MockProcessRequestService {
	mock := &MockProcessRequestService{ctrl: ctrl}
	mock.recorder = &MockProcessRequestServiceMockRecorder{mock}
	return mock
}

func (m *MockProcessRequestService) EXPECT() *MockProcessRequestServiceMockRecorder {
	return m.recorder
}

func (m *MockProcessRequestService) ProcessRequest(keyword, msisdn, systemClient, channel, transactionId string, creditScore float64, startTime time.Time, ctx context.Context) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "ProcessRequest", keyword, msisdn, systemClient, channel, transactionId, creditScore, ctx)
}

func (mr *MockProcessRequestServiceMockRecorder) ProcessRequest(keyword, msisdn, systemClient, channel, transactionId, creditScore, startTime, ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ProcessRequest", reflect.TypeOf((*MockProcessRequestService)(nil).ProcessRequest), keyword, msisdn, systemClient, channel, systemClient, transactionId, creditScore, ctx)
}

type MockTransactionInProgressInterface struct {
	ctrl     *gomock.Controller
	recorder *MockTransactionInProgressInterfaceMockRecorder
}

type MockTransactionInProgressInterfaceMockRecorder struct {
	mock *MockTransactionInProgressInterface
}

func NewMockTransactionInProgressInterface(ctrl *gomock.Controller) *MockTransactionInProgressInterface {
	mock := &MockTransactionInProgressInterface{ctrl: ctrl}
	mock.recorder = &MockTransactionInProgressInterfaceMockRecorder{mock}
	return mock
}

func (m *MockTransactionInProgressInterface) EXPECT() *MockTransactionInProgressInterfaceMockRecorder {
	return m.recorder
}

func (m *MockTransactionInProgressInterface) CreateTransactionInProgressEntry(ctx context.Context, transactionInProgressDB models.TransactionInProgress) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateTransactionInProgressEntry", ctx, transactionInProgressDB)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockTransactionInProgressInterfaceMockRecorder) CreateTransactionInProgressEntry(ctx, transactionInProgressDB interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateTransactionInProgressEntry", reflect.TypeOf((*MockTransactionInProgressInterface)(nil).CreateTransactionInProgressEntry), ctx, transactionInProgressDB)
}

func (m *MockTransactionInProgressInterface) DeleteTransactionInProgressByMsisdn(ctx context.Context, msisdn string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteTransactionInProgressByMsisdn", ctx, msisdn)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockTransactionInProgressInterfaceMockRecorder) DeleteTransactionInProgressByMsisdn(ctx, msisdn interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteTransactionInProgressByMsisdn", reflect.TypeOf((*MockTransactionInProgressInterface)(nil).DeleteTransactionInProgressByMsisdn), ctx, msisdn)
}

func (m *MockTransactionInProgressInterface) IsAvailmentTransactionInProgress(ctx context.Context, msisdn string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsAvailmentTransactionInProgress", ctx, msisdn)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockTransactionInProgressInterfaceMockRecorder) IsAvailmentTransactionInProgress(ctx, msisdn interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsAvailmentTransactionInProgress", reflect.TypeOf((*MockTransactionInProgressInterface)(nil).IsAvailmentTransactionInProgress), ctx, msisdn)
}

type MockTransactionInprogressAndBrandCheckService struct {
	ctrl     *gomock.Controller
	recorder *MockTransactionInprogressAndBrandCheckServiceMockRecorder
}

type MockTransactionInprogressAndBrandCheckServiceMockRecorder struct {
	mock *MockTransactionInprogressAndBrandCheckService
}

func NewMockTransactionInprogressAndBrandCheckService(ctrl *gomock.Controller) *MockTransactionInprogressAndBrandCheckService {
	mock := &MockTransactionInprogressAndBrandCheckService{ctrl: ctrl}
	mock.recorder = &MockTransactionInprogressAndBrandCheckServiceMockRecorder{mock}
	return mock
}

func (m *MockTransactionInprogressAndBrandCheckService) EXPECT() *MockTransactionInprogressAndBrandCheckServiceMockRecorder {
	return m.recorder
}

// transactionInprogressAndBrandCheck mocks base method.
func (m *MockTransactionInprogressAndBrandCheckService) transactionInprogressAndBrandCheck(ctx context.Context, msisdn string) (float64, string, string, primitive.ObjectID, string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "transactionInprogressAndBrandCheck", ctx, msisdn)
	ret0, _ := ret[0].(float64)
	ret1, _ := ret[1].(string)
	ret2, _ := ret[2].(string)
	ret3, _ := ret[3].(primitive.ObjectID)
	ret4, _ := ret[4].(string)
	ret5, _ := ret[5].(error)
	return ret0, ret1, ret2, ret3, ret4, ret5
}

func (mr *MockTransactionInprogressAndBrandCheckServiceMockRecorder) transactionInprogressAndBrandCheck(ctx, msisdn interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "transactionInprogressAndBrandCheck", reflect.TypeOf((*MockTransactionInprogressAndBrandCheckService)(nil).transactionInprogressAndBrandCheck), ctx, msisdn)
}

type MockBrandByFilterService struct {
	ctrl     *gomock.Controller
	recorder *MockBrandByFilterServiceMockRecorder
}

type MockBrandByFilterServiceMockRecorder struct {
	mock *MockBrandByFilterService
}

func NewMockBrandByFilterService(ctrl *gomock.Controller) *MockBrandByFilterService {
	mock := &MockBrandByFilterService{ctrl: ctrl}
	mock.recorder = &MockBrandByFilterServiceMockRecorder{mock}
	return mock
}

func (m *MockBrandByFilterService) EXPECT() *MockBrandByFilterServiceMockRecorder {
	return m.recorder
}

func (m *MockBrandByFilterService) BrandsByFilter(filter interface{}) (*models.Brand, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BrandsByFilter", filter)
	ret0, _ := ret[0].(*models.Brand)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockBrandByFilterServiceMockRecorder) BrandsByFilter(filter interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BrandsByFilter", reflect.TypeOf((*MockBrandByFilterService)(nil).BrandsByFilter), filter)
}

type MockKeywordByFilterService struct {
	ctrl     *gomock.Controller
	recorder *MockKeywordByFilterServiceMockRecorder
}

type MockKeywordByFilterServiceMockRecorder struct {
	mock *MockKeywordByFilterService
}

func NewMockKeywordByFilterService(ctrl *gomock.Controller) *MockKeywordByFilterService {
	mock := &MockKeywordByFilterService{ctrl: ctrl}
	mock.recorder = &MockKeywordByFilterServiceMockRecorder{mock}
	return mock
}

func (m *MockKeywordByFilterService) EXPECT() *MockKeywordByFilterServiceMockRecorder {
	return m.recorder
}

func (m *MockKeywordByFilterService) KeywordByFilter(filter interface{}) (*models.Keyword, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "KeywordByFilter", filter)
	ret0, _ := ret[0].(*models.Keyword)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockKeywordByFilterServiceMockRecorder) KeywordByFilter(filter interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "KeywordByFilter", reflect.TypeOf((*MockKeywordByFilterService)(nil).KeywordByFilter), filter)
}

type MockCheckKeywordInterface struct {
	ctrl     *gomock.Controller
	recorder *MockCheckKeywordInterfaceMockRecorder
}

type MockCheckKeywordInterfaceMockRecorder struct {
	mock *MockCheckKeywordInterface
}

func NewMockCheckKeywordInterface(ctrl *gomock.Controller) *MockCheckKeywordInterface {
	mock := &MockCheckKeywordInterface{ctrl: ctrl}
	mock.recorder = &MockCheckKeywordInterfaceMockRecorder{mock}
	return mock
}

func (m *MockCheckKeywordInterface) EXPECT() *MockCheckKeywordInterfaceMockRecorder {
	return m.recorder
}

func (m *MockCheckKeywordInterface) CheckKeyword(ctx context.Context, keyword string) (*models.Keyword, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CheckKeyword", ctx, keyword)
	ret0, _ := ret[0].(*models.Keyword)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockCheckKeywordInterfaceMockRecorder) CheckKeyword(ctx, keyword interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckKeyword", reflect.TypeOf((*MockCheckKeywordInterface)(nil).CheckKeyword), ctx, keyword)
}

type MockActiveFilterInterface struct {
	ctrl     *gomock.Controller
	recorder *MockActiveFilterInterfaceMockRecorder
}

type MockActiveFilterInterfaceMockRecorder struct {
	mock *MockActiveFilterInterface
}

func NewMockActiveFilterInterface(ctrl *gomock.Controller) *MockActiveFilterInterface {
	mock := &MockActiveFilterInterface{ctrl: ctrl}
	mock.recorder = &MockActiveFilterInterfaceMockRecorder{mock}
	return mock
}

func (m *MockActiveFilterInterface) EXPECT() *MockActiveFilterInterfaceMockRecorder {
	return m.recorder
}

func (m *MockActiveFilterInterface) ActiveLoanByFilter(filter interface{}) (*models.Loans, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ActiveLoanByFilter", filter)
	ret0, _ := ret[0].(*models.Loans)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockActiveFilterInterfaceMockRecorder) ActiveLoanByFilter(filter interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ActiveLoanByFilter", reflect.TypeOf((*MockActiveFilterInterface)(nil).ActiveLoanByFilter), filter)
}

type MockProductNamesInAmaxService struct {
	ctrl     *gomock.Controller
	recorder *MockProductNamesInAmaxServiceMockRecorder
}

type MockProductNamesInAmaxServiceMockRecorder struct {
	mock *MockProductNamesInAmaxService
}

func NewMockProductNamesInAmaxService(ctrl *gomock.Controller) *MockProductNamesInAmaxService {
	mock := &MockProductNamesInAmaxService{ctrl: ctrl}
	mock.recorder = &MockProductNamesInAmaxServiceMockRecorder{mock}
	return mock
}

func (m *MockProductNamesInAmaxService) EXPECT() *MockProductNamesInAmaxServiceMockRecorder {
	return m.recorder
}

func (m *MockProductNamesInAmaxService) ProductNamesInAmaxByKeyword(keywordName string, brandId primitive.ObjectID) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ProductNamesInAmaxByKeyword", keywordName, brandId)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockProductNamesInAmaxServiceMockRecorder) ProductNamesInAmaxByKeyword(keywordName, brandId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ProductNamesInAmaxByKeyword", reflect.TypeOf((*MockProductNamesInAmaxService)(nil).ProductNamesInAmaxByKeyword), keywordName, brandId)
}

type MockProvisionLoanAmaxService struct {
	ctrl     *gomock.Controller
	recorder *MockProvisionLoanAmaxServiceMockRecorder
}

type MockProvisionLoanAmaxServiceMockRecorder struct {
	mock *MockProvisionLoanAmaxService
}

func NewMockProvisionLoanAmaxService(ctrl *gomock.Controller) *MockProvisionLoanAmaxService {
	mock := &MockProvisionLoanAmaxService{ctrl: ctrl}
	mock.recorder = &MockProvisionLoanAmaxServiceMockRecorder{mock}
	return mock
}

func (m *MockProvisionLoanAmaxService) EXPECT() *MockProvisionLoanAmaxServiceMockRecorder {
	return m.recorder
}

func (m *MockProvisionLoanAmaxService) ProvisionLoanAmax(ctx context.Context, msisdn, transactionId string, loanProduct *models.LoanProduct, keyword, channel, systemClient, productsNameInAmax string, creditScore float64, brand_type string, brandId primitive.ObjectID, startTime time.Time) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ProvisionLoanAmax", ctx, msisdn, transactionId, loanProduct, keyword, channel, systemClient, productsNameInAmax, creditScore, brand_type, brandId, startTime)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockProvisionLoanAmaxServiceMockRecorder) ProvisionLoanAmax(ctx, msisdn, transactionId, loanProduct, keyword, channel, systemClient, productsNameInAmax, creditScore, brand_type, brandId interface{}, startTime interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ProvisionLoanAmax", reflect.TypeOf((*MockProvisionLoanAmaxService)(nil).ProvisionLoanAmax), ctx, msisdn, transactionId, loanProduct, keyword, channel, systemClient, productsNameInAmax, creditScore, brand_type, brandId)
}

// Avail_Loan_mocks
type MockWorkerPool struct {
	mock.Mock
}

type MockHelpers struct {
	mock.Mock
}

type MockBrandExists struct {
	mock.Mock
}

type MockLoanAvailmentRepo struct {
	mock.Mock
}

type MockKafkaService struct {
	mock.Mock
}

type MockProcessRequest struct {
	mock.Mock
}

type MockTransactionInProgress struct {
	mock.Mock
}

// Mock implementations
func (m *MockHelpers) GetCreditScoreCustomerTypeBrandType(ctx context.Context, msisdn string) (float64, string, string, string, *time.Time, error) {
	args := m.Called(ctx, msisdn)
	var timeVal *time.Time
	if t, ok := args.Get(4).(*time.Time); ok {
		timeVal = t
	}
	return args.Get(0).(float64), args.String(1), args.String(2), args.String(3), timeVal, args.Error(5)
}

func (m *MockBrandExists) CheckIfBrandExistsAndActive(ctx context.Context, brand string) (bool, primitive.ObjectID, error) {
	args := m.Called(ctx, brand)
	return args.Bool(0), args.Get(1).(primitive.ObjectID), args.Error(2)
}

func (m *MockLoanAvailmentRepo) InsertFailedAvailment(ctx context.Context, msisdn string, keyword string, channel string, systemClient string, errMsg string, errCode string, result bool, creditScore float64, loanProduct *models.LoanProduct, brandCode string) (models.Availments, *models.LoanProduct, string, string, error) {
	args := m.Called(ctx, msisdn, keyword, channel, systemClient, errMsg, errCode, result, creditScore, loanProduct, brandCode)

	availment := args.Get(0).(models.Availments)
	var loanProd *models.LoanProduct
	if args.Get(1) != nil {
		loanProd = args.Get(1).(*models.LoanProduct)
	}

	return availment, loanProd, args.String(2), args.String(3), args.Error(4)
}
func (m *MockLoanAvailmentRepo) GetAvailmentById(ctx context.Context, id primitive.ObjectID) (models.Availments, error) {
	args := m.Called(ctx, id)

	// Handle nil or invalid assertions to avoid runtime panics
	availment, ok := args.Get(0).(models.Availments)
	if !ok && args.Get(0) != nil {
		return models.Availments{}, fmt.Errorf("invalid type for availment: %v", args.Get(0))
	}

	return availment, args.Error(1) // Ensure index is correct for the error value
}

func (m *MockLoanAvailmentRepo) UpdateAvailment(ctx context.Context, availmentID primitive.ObjectID, updateField bson.M) {
	m.Called(ctx, availmentID, updateField)
}

func (m *MockKafkaService) PublishAvailmentStatusToKafka(ctx context.Context, transactionID string, availmentChannel string, msisdn string, brandType string, loanType string, loanProductName string, loanProductKeyword string, servicingPartner string, loanAmount float64, serviceFeeAmount float64, availmentResult string, availmentErrorText string, loanAgeing int32, statusOfProvisionedLoan string) error {
	args := m.Called(ctx, transactionID, availmentChannel, msisdn, brandType, loanType, loanProductName, loanProductKeyword, servicingPartner, loanAmount, serviceFeeAmount, availmentResult, availmentErrorText, loanAgeing, statusOfProvisionedLoan)
	return args.Error(0)
}

func (m *MockProcessRequest) ProcessRequest(keyword string, msisdn string, systemClient string, channel string, transactionId string, creditScore float64, startTime time.Time, ctx context.Context) {
	m.Called(keyword, msisdn, systemClient, channel, transactionId, creditScore, ctx)
}

func (m *MockTransactionInProgress) DeleteTransactionInProgressByMsisdn(ctx context.Context, msisdn string) (bool, error) {
	args := m.Called(ctx, msisdn)
	return args.Bool(0), args.Error(1)
}
func (m *MockTransactionInProgress) CreateTransactionInProgressEntry(ctx context.Context, transactionInProgressDB models.TransactionInProgress) (bool, error) {
	args := m.Called(ctx, transactionInProgressDB)
	return args.Bool(0), args.Error(1)
}
func (m *MockTransactionInProgress) IsAvailmentTransactionInProgress(ctx context.Context, msisdn string) (bool, error) {
	args := m.Called(ctx, msisdn)
	return args.Bool(0), args.Error(1)
}

// MockSystemLevelRulesRepository is a mock of SystemLevelRulesRepository interface.
type MockSystemLevelRulesRepository struct {
	ctrl     *gomock.Controller
	recorder *MockSystemLevelRulesRepositoryMockRecorder
}

// MockSystemLevelRulesRepositoryMockRecorder is the mock recorder for MockSystemLevelRulesRepository.
type MockSystemLevelRulesRepositoryMockRecorder struct {
	mock *MockSystemLevelRulesRepository
}

// NewMockSystemLevelRulesRepository creates a new mock instance.
func NewMockSystemLevelRulesRepository(ctrl *gomock.Controller) *MockSystemLevelRulesRepository {
	mock := &MockSystemLevelRulesRepository{ctrl: ctrl}
	mock.recorder = &MockSystemLevelRulesRepositoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSystemLevelRulesRepository) EXPECT() *MockSystemLevelRulesRepositoryMockRecorder {
	return m.recorder
}

// SystemLevelRules mocks base method.
func (m *MockSystemLevelRulesRepository) SystemLevelRules(filter interface{}) (models.SystemLevelRules, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SystemLevelRules", filter)
	if ret[0] == nil {
		return models.SystemLevelRules{}, ret[1].(error)
	}
	ret0, _ := ret[0].(*models.SystemLevelRules)
	ret1, _ := ret[1].(error)
	return *ret0, ret1
}

// SystemLevelRules indicates an expected call of SystemLevelRules.
func (mr *MockSystemLevelRulesRepositoryMockRecorder) SystemLevelRules(filter interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SystemLevelRules", reflect.TypeOf((*MockSystemLevelRulesRepository)(nil).SystemLevelRules), filter)
}
