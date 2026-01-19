package tests

import (
	context "context"
	"encoding/json"

	models "globe/dodrio_loan_availment/internal/pkg/models"
	"globe/dodrio_loan_availment/internal/pkg/services"
	"net/http"
	"net/http/httptest"
	reflect "reflect"
	"testing"
	time "time"

	"github.com/gin-gonic/gin"
	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"go.mongodb.org/mongo-driver/bson"
	primitive "go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func TestEligibilityCheckService_EligibilityCheck(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	Mockhelper := NewMockhelper(ctrl)
	mockActiveLoanRepo := NewMockActiveFilterinterface(ctrl)
	mockCheckSubsRepo := NewMockCheckSubsInterface(ctrl)
	mockCheckLoanRepo := NewMockCheckLoanProductService(ctrl)
	mockCheckActiveRepo := NewMockCheckActiveLoanService(ctrl)
	mockCheckWhiteListRepo := NewMockCheckWhitelistSErvice(ctrl)
	mockCheckChannelRepo := NewMockCheckChannelExistService(ctrl)
	mockCheckChannelAndProductRepo := NewMockCheckChannelAndProductService(ctrl)
	mockCheckBrandAndProductRepo := NewMockCheckBrandandProductService(ctrl)
	mockCheckScoreRepo := NewMockCheckScoreEligService(ctrl)
	mockKeywordRepo := NewMockKeywordByFilterservice(ctrl)
	mockBrandRepo := NewMockBrandByFilterservice(ctrl)

	service := services.NewEligibilityCheckService(
		// mockCheckEligb,
		Mockhelper,
		nil, // CheckKeywordInterface
		mockActiveLoanRepo,
		mockCheckSubsRepo,
		mockCheckLoanRepo,
		mockCheckActiveRepo,
		mockCheckWhiteListRepo,
		mockCheckChannelRepo,
		mockCheckChannelAndProductRepo,
		mockCheckBrandAndProductRepo,
		mockCheckScoreRepo,
		mockKeywordRepo,
		mockBrandRepo,
		nil, // services.SubscribersRepository
		nil, // services.LoanAvailmentServicesInt
		nil, // services.UnpaidLoanFilterInterface
	)

	tests := []struct {
		name          string
		msisdn        string
		setupMocks    func()
		expectedCode  int
		expectedError bool
	}{
		{
			name:          "Invalid MSISDN",
			msisdn:        "187345389",
			setupMocks:    func() {},
			expectedCode:  http.StatusNotImplemented,
			expectedError: false,
		},
		{
			name:   "Valid MSISDN No Active Loan",
			msisdn: "9876543210",
			setupMocks: func() {
				Mockhelper.EXPECT().GetCreditScoreCustomerTypeBrandType(gomock.Any(), gomock.Any()).Return(75.00, "", "", nil)

				mockActiveLoanRepo.EXPECT().
					ActiveLoanByFilter(gomock.Any()).
					Return(&models.Loans{}, mongo.ErrNoDocuments)
			},
			expectedCode:  http.StatusOK,
			expectedError: false,
		},
		{
			name:   "Valid MSISDN With Active Loan",
			msisdn: "9876543210",
			setupMocks: func() {

				Mockhelper.EXPECT().
					GetCreditScoreCustomerTypeBrandType(context.Background(), "9876543210").
					Return(float64(800), "CONSUMER", "POSTPAID", nil)

				expectedFilter := bson.M{"MSISDN": "9876543210"}
				activeLoan := &models.Loans{
					MSISDN:          "9876543210",
					LoanProductId:   primitive.NewObjectID(),
					BrandId:         primitive.NewObjectID(),
					TotalLoanAmount: 1000,
					ServiceFee:      100,
				}

				mockActiveLoanRepo.EXPECT().
					ActiveLoanByFilter(expectedFilter).
					Return(activeLoan, nil)

				keywordFilter := bson.D{{Key: "productId", Value: activeLoan.LoanProductId}}
				keyword := &models.Keyword{
					Name: "TestKeyword",
				}
				mockKeywordRepo.EXPECT().
					KeywordByFilter(keywordFilter).
					Return(keyword, nil)

				brandFilter := bson.D{{Key: "_id", Value: activeLoan.BrandId}}
				brand := &models.Brand{
					Code: "TestBrand",
				}
				mockBrandRepo.EXPECT().
					BrandsByFilter(brandFilter).
					Return(brand, nil)
			},
			expectedCode:  http.StatusOK,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/", nil)
			c.Request = c.Request.WithContext(context.Background())

			tt.setupMocks()

			err := service.EligibilityCheck(c, tt.msisdn)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedCode, w.Code)

			if w.Code == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)

			}
		})
	}
}

type MockLoanrepo struct {
	ctrl     *gomock.Controller
	recorder *MockLoanrepoMockRecorder
}

type MockLoanrepoMockRecorder struct {
	mock *MockLoanrepo
}

func NewMockLoanrepo(ctrl *gomock.Controller) *MockLoanrepo {
	mock := &MockLoanrepo{ctrl: ctrl}
	mock.recorder = &MockLoanrepoMockRecorder{mock}
	return mock
}

func (m *MockLoanrepo) EXPECT() *MockLoanrepoMockRecorder {
	return m.recorder
}

func (m *MockLoanrepo) ActiveLoanByFilter(filter interface{}) (*models.Loans, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ActiveLoanByFilter", filter)
	ret0, _ := ret[0].(*models.Loans)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockLoanrepoMockRecorder) ActiveLoanByFilter(filter interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ActiveLoanByFilter", reflect.TypeOf((*MockLoanrepo)(nil).ActiveLoanByFilter), filter)
}


// Mockhelper is a mock of Helper interface.
type Mockhelper struct {
	ctrl     *gomock.Controller
	recorder *MockhelperMockRecorder
}

// MockhelperMockRecorder is the mock recorder for Mockhelper.
type MockhelperMockRecorder struct {
	mock *Mockhelper
}

// NewMockhelper creates a new mock instance.
func NewMockhelper(ctrl *gomock.Controller) *Mockhelper {
	mock := &Mockhelper{ctrl: ctrl}
	mock.recorder = &MockhelperMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *Mockhelper) EXPECT() *MockhelperMockRecorder {
	return m.recorder
}

// GetCreditScoreCustomerTypeBrandType mocks base method.
func (m *Mockhelper) GetCreditScoreCustomerTypeBrandType(ctx context.Context, msisdn string) (float64, string, string, string, *time.Time, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCreditScoreCustomerTypeBrandType", ctx, msisdn)
	ret0, _ := ret[0].(float64)
	ret1, _ := ret[1].(string)
	ret2, _ := ret[2].(string)
	ret3, _ := ret[3].(string)
	ret4, _ := ret[4].(*time.Time)
	ret5, _ := ret[5].(error)

	return ret0, ret1, ret2, ret3, ret4, ret5
}

// GetCreditScoreCustomerTypeBrandType indicates an expected call of GetCreditScoreCustomerTypeBrandType.
func (mr *MockhelperMockRecorder) GetCreditScoreCustomerTypeBrandType(ctx, msisdn interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCreditScoreCustomerTypeBrandType", reflect.TypeOf((*Mockhelper)(nil).GetCreditScoreCustomerTypeBrandType), ctx, msisdn)
}

// MockCheckEligibilityServiceinterface is a mock of CheckEligibilityServiceInterface interface.
type MockCheckEligibilityServiceinterface struct {
	ctrl     *gomock.Controller
	recorder *MockCheckEligibilityServiceinterfaceMockRecorder
}

// MockCheckEligibilityServiceinterfaceMockRecorder is the mock recorder for MockCheckEligibilityServiceinterface.
type MockCheckEligibilityServiceinterfaceMockRecorder struct {
	mock *MockCheckEligibilityServiceinterface
}

// NewMockCheckEligibilityServiceinterface creates a new mock instance.
func NewMockCheckEligibilityServiceinterface(ctrl *gomock.Controller) *MockCheckEligibilityServiceinterface {
	mock := &MockCheckEligibilityServiceinterface{ctrl: ctrl}
	mock.recorder = &MockCheckEligibilityServiceinterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCheckEligibilityServiceinterface) EXPECT() *MockCheckEligibilityServiceinterfaceMockRecorder {
	return m.recorder
}

// CheckEligibility mocks base method.
func (m *MockCheckEligibilityServiceinterface) CheckEligibility(ctx context.Context, msisdn, keyword, channel, customer_type string, credit_score_value float64, brand_type string, brandId primitive.ObjectID) (bool, string, *models.LoanProduct, primitive.ObjectID, time.Time, error) {
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

// CheckEligibility indicates an expected call of CheckEligibility.
func (mr *MockCheckEligibilityServiceinterfaceMockRecorder) CheckEligibility(ctx, msisdn, keyword, channel, customer_type, credit_score_value, brand_type, brandId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckEligibility", reflect.TypeOf((*MockCheckEligibilityServiceinterface)(nil).CheckEligibility), ctx, msisdn, keyword, channel, customer_type, credit_score_value, brand_type, brandId)
}

// MockBrandByFilterservice is a mock of BrandByFilterService interface.
type MockBrandByFilterservice struct {
	ctrl     *gomock.Controller
	recorder *MockBrandByFilterserviceMockRecorder
}

// MockBrandByFilterserviceMockRecorder is the mock recorder for MockBrandByFilterservice.
type MockBrandByFilterserviceMockRecorder struct {
	mock *MockBrandByFilterservice
}

// NewMockBrandByFilterservice creates a new mock instance.
func NewMockBrandByFilterservice(ctrl *gomock.Controller) *MockBrandByFilterservice {
	mock := &MockBrandByFilterservice{ctrl: ctrl}
	mock.recorder = &MockBrandByFilterserviceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBrandByFilterservice) EXPECT() *MockBrandByFilterserviceMockRecorder {
	return m.recorder
}

// BrandsByFilter mocks base method.
func (m *MockBrandByFilterservice) BrandsByFilter(filter interface{}) (*models.Brand, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BrandsByFilter", filter)
	ret0, _ := ret[0].(*models.Brand)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// BrandsByFilter indicates an expected call of BrandsByFilter.
func (mr *MockBrandByFilterserviceMockRecorder) BrandsByFilter(filter interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BrandsByFilter", reflect.TypeOf((*MockBrandByFilterservice)(nil).BrandsByFilter), filter)
}

// MockKeywordByFilterservice is a mock of KeywordByFilterService interface.
type MockKeywordByFilterservice struct {
	ctrl     *gomock.Controller
	recorder *MockKeywordByFilterserviceMockRecorder
}

// MockKeywordByFilterserviceMockRecorder is the mock recorder for MockKeywordByFilterservice.
type MockKeywordByFilterserviceMockRecorder struct {
	mock *MockKeywordByFilterservice
}

// NewMockKeywordByFilterservice creates a new mock instance.
func NewMockKeywordByFilterservice(ctrl *gomock.Controller) *MockKeywordByFilterservice {
	mock := &MockKeywordByFilterservice{ctrl: ctrl}
	mock.recorder = &MockKeywordByFilterserviceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockKeywordByFilterservice) EXPECT() *MockKeywordByFilterserviceMockRecorder {
	return m.recorder
}

// KeywordByFilter mocks base method.
func (m *MockKeywordByFilterservice) KeywordByFilter(filter interface{}) (*models.Keyword, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "KeywordByFilter", filter)
	ret0, _ := ret[0].(*models.Keyword)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// KeywordByFilter indicates an expected call of KeywordByFilter.
func (mr *MockKeywordByFilterserviceMockRecorder) KeywordByFilter(filter interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "KeywordByFilter", reflect.TypeOf((*MockKeywordByFilterservice)(nil).KeywordByFilter), filter)
}

// MockActiveFilterinterface is a mock of ActiveFilterInterface interface.
type MockActiveFilterinterface struct {
	ctrl     *gomock.Controller
	recorder *MockActiveFilterinterfaceMockRecorder
}

// MockActiveFilterinterfaceMockRecorder is the mock recorder for MockActiveFilterinterface.
type MockActiveFilterinterfaceMockRecorder struct {
	mock *MockActiveFilterinterface
}

// NewMockActiveFilterinterface creates a new mock instance.
func NewMockActiveFilterinterface(ctrl *gomock.Controller) *MockActiveFilterinterface {
	mock := &MockActiveFilterinterface{ctrl: ctrl}
	mock.recorder = &MockActiveFilterinterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockActiveFilterinterface) EXPECT() *MockActiveFilterinterfaceMockRecorder {
	return m.recorder
}

// ActiveLoanByFilter mocks base method.
func (m *MockActiveFilterinterface) ActiveLoanByFilter(filter interface{}) (*models.Loans, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ActiveLoanByFilter", filter)
	ret0, _ := ret[0].(*models.Loans)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ActiveLoanByFilter indicates an expected call of ActiveLoanByFilter.
func (mr *MockActiveFilterinterfaceMockRecorder) ActiveLoanByFilter(filter interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ActiveLoanByFilter", reflect.TypeOf((*MockActiveFilterinterface)(nil).ActiveLoanByFilter), filter)
}

// MockCheckSubsInterface is a mock of CheckSubsInterface interface.
type MockCheckSubsInterface struct {
	ctrl     *gomock.Controller
	recorder *MockCheckSubsInterfaceMockRecorder
}

// MockCheckSubsInterfaceMockRecorder is the mock recorder for MockCheckSubsInterface.
type MockCheckSubsInterfaceMockRecorder struct {
	mock *MockCheckSubsInterface
}

// NewMockCheckSubsInterface creates a new mock instance.
func NewMockCheckSubsInterface(ctrl *gomock.Controller) *MockCheckSubsInterface {
	mock := &MockCheckSubsInterface{ctrl: ctrl}
	mock.recorder = &MockCheckSubsInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCheckSubsInterface) EXPECT() *MockCheckSubsInterfaceMockRecorder {
	return m.recorder
}

// CheckSubscriberBlacklisitng mocks base method.
func (m *MockCheckSubsInterface) CheckSubscriberBlacklisitng(ctx context.Context, msisdn string) (*models.Subscribers, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CheckSubscriberBlacklisitng", ctx, msisdn)
	ret0, _ := ret[0].(*models.Subscribers)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CheckSubscriberBlacklisitng indicates an expected call of CheckSubscriberBlacklisitng.
func (mr *MockCheckSubsInterfaceMockRecorder) CheckSubscriberBlacklisitng(ctx, msisdn interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckSubscriberBlacklisitng", reflect.TypeOf((*MockCheckSubsInterface)(nil).CheckSubscriberBlacklisitng), ctx, msisdn)
}

// MockCheckSubscriberBlacklistInterface is a mock of CheckSubscriberBlacklistInterface interface.
type MockCheckSubscriberBlacklistInterface struct {
	ctrl     *gomock.Controller
	recorder *MockCheckSubscriberBlacklistInterfaceMockRecorder
}

// MockCheckSubscriberBlacklistInterfaceMockRecorder is the mock recorder for MockCheckSubscriberBlacklistInterface.
type MockCheckSubscriberBlacklistInterfaceMockRecorder struct {
	mock *MockCheckSubscriberBlacklistInterface
}

// NewMockCheckSubscriberBlacklistInterface creates a new mock instance.
func NewMockCheckSubscriberBlacklistInterface(ctrl *gomock.Controller) *MockCheckSubscriberBlacklistInterface {
	mock := &MockCheckSubscriberBlacklistInterface{ctrl: ctrl}
	mock.recorder = &MockCheckSubscriberBlacklistInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCheckSubscriberBlacklistInterface) EXPECT() *MockCheckSubscriberBlacklistInterfaceMockRecorder {
	return m.recorder
}

// SubscribersByFilter mocks base method.
func (m *MockCheckSubscriberBlacklistInterface) SubscribersByFilter(filter interface{}) (*models.Subscribers, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SubscribersByFilter", filter)
	ret0, _ := ret[0].(*models.Subscribers)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SubscribersByFilter indicates an expected call of SubscribersByFilter.
func (mr *MockCheckSubscriberBlacklistInterfaceMockRecorder) SubscribersByFilter(filter interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SubscribersByFilter", reflect.TypeOf((*MockCheckSubscriberBlacklistInterface)(nil).SubscribersByFilter), filter)
}

// MockCheckLoanProductService is a mock of CheckLoanProductService interface.
type MockCheckLoanProductService struct {
	ctrl     *gomock.Controller
	recorder *MockCheckLoanProductServiceMockRecorder
}

// MockCheckLoanProductServiceMockRecorder is the mock recorder for MockCheckLoanProductService.
type MockCheckLoanProductServiceMockRecorder struct {
	mock *MockCheckLoanProductService
}

// NewMockCheckLoanProductService creates a new mock instance.
func NewMockCheckLoanProductService(ctrl *gomock.Controller) *MockCheckLoanProductService {
	mock := &MockCheckLoanProductService{ctrl: ctrl}
	mock.recorder = &MockCheckLoanProductServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCheckLoanProductService) EXPECT() *MockCheckLoanProductServiceMockRecorder {
	return m.recorder
}

// CheckLoanProduct mocks base method.
func (m *MockCheckLoanProductService) CheckLoanProduct(ctx context.Context, productId primitive.ObjectID) (*models.LoanProduct, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CheckLoanProduct", ctx, productId)
	ret0, _ := ret[0].(*models.LoanProduct)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CheckLoanProduct indicates an expected call of CheckLoanProduct.
func (mr *MockCheckLoanProductServiceMockRecorder) CheckLoanProduct(ctx, productId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckLoanProduct", reflect.TypeOf((*MockCheckLoanProductService)(nil).CheckLoanProduct), ctx, productId)
}

// MockCheckActiveLoanService is a mock of CheckActiveLoanService interface.
type MockCheckActiveLoanService struct {
	ctrl     *gomock.Controller
	recorder *MockCheckActiveLoanServiceMockRecorder
}

// MockCheckActiveLoanServiceMockRecorder is the mock recorder for MockCheckActiveLoanService.
type MockCheckActiveLoanServiceMockRecorder struct {
	mock *MockCheckActiveLoanService
}

// NewMockCheckActiveLoanService creates a new mock instance.
func NewMockCheckActiveLoanService(ctrl *gomock.Controller) *MockCheckActiveLoanService {
	mock := &MockCheckActiveLoanService{ctrl: ctrl}
	mock.recorder = &MockCheckActiveLoanServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCheckActiveLoanService) EXPECT() *MockCheckActiveLoanServiceMockRecorder {
	return m.recorder
}

// CheckActiveLoan mocks base method.
func (m *MockCheckActiveLoanService) CheckActiveLoan(ctx context.Context, msisdn string) (bool, primitive.ObjectID, time.Time, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CheckActiveLoan", ctx, msisdn)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(primitive.ObjectID)
	ret2, _ := ret[2].(time.Time)
	ret3, _ := ret[3].(error)
	return ret0, ret1, ret2, ret3
}

// CheckActiveLoan indicates an expected call of CheckActiveLoan.
func (mr *MockCheckActiveLoanServiceMockRecorder) CheckActiveLoan(ctx, msisdn interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckActiveLoan", reflect.TypeOf((*MockCheckActiveLoanService)(nil).CheckActiveLoan), ctx, msisdn)
}

// MockActiveLaoneService is a mock of ActiveLaoneService interface.
type MockActiveLaoneService struct {
	ctrl     *gomock.Controller
	recorder *MockActiveLaoneServiceMockRecorder
}

// MockActiveLaoneServiceMockRecorder is the mock recorder for MockActiveLaoneService.
type MockActiveLaoneServiceMockRecorder struct {
	mock *MockActiveLaoneService
}

// NewMockActiveLaoneService creates a new mock instance.
func NewMockActiveLaoneService(ctrl *gomock.Controller) *MockActiveLaoneService {
	mock := &MockActiveLaoneService{ctrl: ctrl}
	mock.recorder = &MockActiveLaoneServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockActiveLaoneService) EXPECT() *MockActiveLaoneServiceMockRecorder {
	return m.recorder
}

// IsActiveLoan mocks base method.
func (m *MockActiveLaoneService) IsActiveLoan(ctx context.Context, msisdn string) (bool, *models.Loans, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsActiveLoan", ctx, msisdn)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(*models.Loans)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// IsActiveLoan indicates an expected call of IsActiveLoan.
func (mr *MockActiveLaoneServiceMockRecorder) IsActiveLoan(ctx, msisdn interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsActiveLoan", reflect.TypeOf((*MockActiveLaoneService)(nil).IsActiveLoan), ctx, msisdn)
}

// MockCheckWhitelistSErvice is a mock of CheckWhitelistSErvice interface.
type MockCheckWhitelistSErvice struct {
	ctrl     *gomock.Controller
	recorder *MockCheckWhitelistSErviceMockRecorder
}

// MockCheckWhitelistSErviceMockRecorder is the mock recorder for MockCheckWhitelistSErvice.
type MockCheckWhitelistSErviceMockRecorder struct {
	mock *MockCheckWhitelistSErvice
}

// NewMockCheckWhitelistSErvice creates a new mock instance.
func NewMockCheckWhitelistSErvice(ctrl *gomock.Controller) *MockCheckWhitelistSErvice {
	mock := &MockCheckWhitelistSErvice{ctrl: ctrl}
	mock.recorder = &MockCheckWhitelistSErviceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCheckWhitelistSErvice) EXPECT() *MockCheckWhitelistSErviceMockRecorder {
	return m.recorder
}

// CheckWhitelist mocks base method.
func (m *MockCheckWhitelistSErvice) CheckWhitelist(ctx context.Context, msisdn string, loanProduct *models.LoanProduct) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CheckWhitelist", ctx, msisdn, loanProduct)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CheckWhitelist indicates an expected call of CheckWhitelist.
func (mr *MockCheckWhitelistSErviceMockRecorder) CheckWhitelist(ctx, msisdn, loanProduct interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckWhitelist", reflect.TypeOf((*MockCheckWhitelistSErvice)(nil).CheckWhitelist), ctx, msisdn, loanProduct)
}

// MockIsWhitelistedService is a mock of IsWhitelistedService interface.
type MockIsWhitelistedService struct {
	ctrl     *gomock.Controller
	recorder *MockIsWhitelistedServiceMockRecorder
}

// MockIsWhitelistedServiceMockRecorder is the mock recorder for MockIsWhitelistedService.
type MockIsWhitelistedServiceMockRecorder struct {
	mock *MockIsWhitelistedService
}

// NewMockIsWhitelistedService creates a new mock instance.
func NewMockIsWhitelistedService(ctrl *gomock.Controller) *MockIsWhitelistedService {
	mock := &MockIsWhitelistedService{ctrl: ctrl}
	mock.recorder = &MockIsWhitelistedServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockIsWhitelistedService) EXPECT() *MockIsWhitelistedServiceMockRecorder {
	return m.recorder
}

// IsWhitelisted mocks base method.
func (m *MockIsWhitelistedService) IsWhitelisted(msisdn string, productId primitive.ObjectID) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsWhitelisted", msisdn, productId)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// IsWhitelisted indicates an expected call of IsWhitelisted.
func (mr *MockIsWhitelistedServiceMockRecorder) IsWhitelisted(msisdn, productId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsWhitelisted", reflect.TypeOf((*MockIsWhitelistedService)(nil).IsWhitelisted), msisdn, productId)
}

// MockCheckChannelExistService is a mock of CheckChannelExistService interface.
type MockCheckChannelExistService struct {
	ctrl     *gomock.Controller
	recorder *MockCheckChannelExistServiceMockRecorder
}

// MockCheckChannelExistServiceMockRecorder is the mock recorder for MockCheckChannelExistService.
type MockCheckChannelExistServiceMockRecorder struct {
	mock *MockCheckChannelExistService
}

// NewMockCheckChannelExistService creates a new mock instance.
func NewMockCheckChannelExistService(ctrl *gomock.Controller) *MockCheckChannelExistService {
	mock := &MockCheckChannelExistService{ctrl: ctrl}
	mock.recorder = &MockCheckChannelExistServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCheckChannelExistService) EXPECT() *MockCheckChannelExistServiceMockRecorder {
	return m.recorder
}

// CheckIfChannelExistsAndActive mocks base method.
func (m *MockCheckChannelExistService) CheckIfChannelExistsAndActive(ctx context.Context, channel string) (bool, primitive.ObjectID, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CheckIfChannelExistsAndActive", ctx, channel)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(primitive.ObjectID)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// CheckIfChannelExistsAndActive indicates an expected call of CheckIfChannelExistsAndActive.
func (mr *MockCheckChannelExistServiceMockRecorder) CheckIfChannelExistsAndActive(ctx, channel interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckIfChannelExistsAndActive", reflect.TypeOf((*MockCheckChannelExistService)(nil).CheckIfChannelExistsAndActive), ctx, channel)
}

// MockChannelByFilterServcie is a mock of ChannelByFilterServcie interface.
type MockChannelByFilterServcie struct {
	ctrl     *gomock.Controller
	recorder *MockChannelByFilterServcieMockRecorder
}

// MockChannelByFilterServcieMockRecorder is the mock recorder for MockChannelByFilterServcie.
type MockChannelByFilterServcieMockRecorder struct {
	mock *MockChannelByFilterServcie
}

// NewMockChannelByFilterServcie creates a new mock instance.
func NewMockChannelByFilterServcie(ctrl *gomock.Controller) *MockChannelByFilterServcie {
	mock := &MockChannelByFilterServcie{ctrl: ctrl}
	mock.recorder = &MockChannelByFilterServcieMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockChannelByFilterServcie) EXPECT() *MockChannelByFilterServcieMockRecorder {
	return m.recorder
}

// ChannelsByFilter mocks base method.
func (m *MockChannelByFilterServcie) ChannelsByFilter(ctx context.Context, filter interface{}) (*models.Channels, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ChannelsByFilter", ctx, filter)
	ret0, _ := ret[0].(*models.Channels)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ChannelsByFilter indicates an expected call of ChannelsByFilter.
func (mr *MockChannelByFilterServcieMockRecorder) ChannelsByFilter(ctx, filter interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ChannelsByFilter", reflect.TypeOf((*MockChannelByFilterServcie)(nil).ChannelsByFilter), ctx, filter)
}

// MockCheckChannelAndProductService is a mock of CheckChannelAndProductService interface.
type MockCheckChannelAndProductService struct {
	ctrl     *gomock.Controller
	recorder *MockCheckChannelAndProductServiceMockRecorder
}

// MockCheckChannelAndProductServiceMockRecorder is the mock recorder for MockCheckChannelAndProductService.
type MockCheckChannelAndProductServiceMockRecorder struct {
	mock *MockCheckChannelAndProductService
}

// NewMockCheckChannelAndProductService creates a new mock instance.
func NewMockCheckChannelAndProductService(ctrl *gomock.Controller) *MockCheckChannelAndProductService {
	mock := &MockCheckChannelAndProductService{ctrl: ctrl}
	mock.recorder = &MockCheckChannelAndProductServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCheckChannelAndProductService) EXPECT() *MockCheckChannelAndProductServiceMockRecorder {
	return m.recorder
}

// CheckIfChannelAndProductMapped mocks base method.
func (m *MockCheckChannelAndProductService) CheckIfChannelAndProductMapped(ctx context.Context, channelId, productId primitive.ObjectID) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CheckIfChannelAndProductMapped", ctx, channelId, productId)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CheckIfChannelAndProductMapped indicates an expected call of CheckIfChannelAndProductMapped.
func (mr *MockCheckChannelAndProductServiceMockRecorder) CheckIfChannelAndProductMapped(ctx, channelId, productId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckIfChannelAndProductMapped", reflect.TypeOf((*MockCheckChannelAndProductService)(nil).CheckIfChannelAndProductMapped), ctx, channelId, productId)
}

// MockLoanProductChannelService is a mock of LoanProductChannelService interface.
type MockLoanProductChannelService struct {
	ctrl     *gomock.Controller
	recorder *MockLoanProductChannelServiceMockRecorder
}

// MockLoanProductChannelServiceMockRecorder is the mock recorder for MockLoanProductChannelService.
type MockLoanProductChannelServiceMockRecorder struct {
	mock *MockLoanProductChannelService
}

// NewMockLoanProductChannelService creates a new mock instance.
func NewMockLoanProductChannelService(ctrl *gomock.Controller) *MockLoanProductChannelService {
	mock := &MockLoanProductChannelService{ctrl: ctrl}
	mock.recorder = &MockLoanProductChannelServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockLoanProductChannelService) EXPECT() *MockLoanProductChannelServiceMockRecorder {
	return m.recorder
}

// LoanProductsBrandsByFilter mocks base method.
func (m *MockLoanProductChannelService) LoanProductsBrandsByFilter(ctx context.Context, filter interface{}) (*models.LoanProductsBrands, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LoanProductsBrandsByFilter", ctx, filter)
	ret0, _ := ret[0].(*models.LoanProductsBrands)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// LoanProductsBrandsByFilter indicates an expected call of LoanProductsBrandsByFilter.
func (mr *MockLoanProductChannelServiceMockRecorder) LoanProductsBrandsByFilter(ctx, filter interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LoanProductsBrandsByFilter", reflect.TypeOf((*MockLoanProductChannelService)(nil).LoanProductsBrandsByFilter), ctx, filter)
}

// MockCheckBrandandProductService is a mock of CheckBrandandProductService interface.
type MockCheckBrandandProductService struct {
	ctrl     *gomock.Controller
	recorder *MockCheckBrandandProductServiceMockRecorder
}

// MockCheckBrandandProductServiceMockRecorder is the mock recorder for MockCheckBrandandProductService.
type MockCheckBrandandProductServiceMockRecorder struct {
	mock *MockCheckBrandandProductService
}

// NewMockCheckBrandandProductService creates a new mock instance.
func NewMockCheckBrandandProductService(ctrl *gomock.Controller) *MockCheckBrandandProductService {
	mock := &MockCheckBrandandProductService{ctrl: ctrl}
	mock.recorder = &MockCheckBrandandProductServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCheckBrandandProductService) EXPECT() *MockCheckBrandandProductServiceMockRecorder {
	return m.recorder
}

// CheckIfBrandAndProductMapped mocks base method.
func (m *MockCheckBrandandProductService) CheckIfBrandAndProductMapped(ctx context.Context, brandId, productId primitive.ObjectID) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CheckIfBrandAndProductMapped", ctx, brandId, productId)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CheckIfBrandAndProductMapped indicates an expected call of CheckIfBrandAndProductMapped.
func (mr *MockCheckBrandandProductServiceMockRecorder) CheckIfBrandAndProductMapped(ctx, brandId, productId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckIfBrandAndProductMapped", reflect.TypeOf((*MockCheckBrandandProductService)(nil).CheckIfBrandAndProductMapped), ctx, brandId, productId)
}

// MockCheckScoreEligService is a mock of CheckScoreEligService interface.
type MockCheckScoreEligService struct {
	ctrl     *gomock.Controller
	recorder *MockCheckScoreEligServiceMockRecorder
}

// MockCheckScoreEligServiceMockRecorder is the mock recorder for MockCheckScoreEligService.
type MockCheckScoreEligServiceMockRecorder struct {
	mock *MockCheckScoreEligService
}

// NewMockCheckScoreEligService creates a new mock instance.
func NewMockCheckScoreEligService(ctrl *gomock.Controller) *MockCheckScoreEligService {
	mock := &MockCheckScoreEligService{ctrl: ctrl}
	mock.recorder = &MockCheckScoreEligServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCheckScoreEligService) EXPECT() *MockCheckScoreEligServiceMockRecorder {
	return m.recorder
}

// CheckCreditScoreEligibility mocks base method.
func (m *MockCheckScoreEligService) CheckCreditScoreEligibility(ctx context.Context, creditScore float64, loanProductResult *models.LoanProduct) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CheckCreditScoreEligibility", ctx, creditScore, loanProductResult)
	ret0, _ := ret[0].(bool)
	return ret0
}

// CheckCreditScoreEligibility indicates an expected call of CheckCreditScoreEligibility.
func (mr *MockCheckScoreEligServiceMockRecorder) CheckCreditScoreEligibility(ctx, creditScore, loanProductResult interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckCreditScoreEligibility", reflect.TypeOf((*MockCheckScoreEligService)(nil).CheckCreditScoreEligibility), ctx, creditScore, loanProductResult)
}

// MockLoanProductsChannelByFilterService is a mock of LoanProductsChannelByFilterService interface.
type MockLoanProductsChannelByFilterService struct {
	ctrl     *gomock.Controller
	recorder *MockLoanProductsChannelByFilterServiceMockRecorder
}

// MockLoanProductsChannelByFilterServiceMockRecorder is the mock recorder for MockLoanProductsChannelByFilterService.
type MockLoanProductsChannelByFilterServiceMockRecorder struct {
	mock *MockLoanProductsChannelByFilterService
}

// NewMockLoanProductsChannelByFilterService creates a new mock instance.
func NewMockLoanProductsChannelByFilterService(ctrl *gomock.Controller) *MockLoanProductsChannelByFilterService {
	mock := &MockLoanProductsChannelByFilterService{ctrl: ctrl}
	mock.recorder = &MockLoanProductsChannelByFilterServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockLoanProductsChannelByFilterService) EXPECT() *MockLoanProductsChannelByFilterServiceMockRecorder {
	return m.recorder
}

// LoanProductsChannelsByFilter mocks base method.
func (m *MockLoanProductsChannelByFilterService) LoanProductsChannelsByFilter(filter interface{}) (*models.LoanProductsChannels, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LoanProductsChannelsByFilter", filter)
	ret0, _ := ret[0].(*models.LoanProductsChannels)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// LoanProductsChannelsByFilter indicates an expected call of LoanProductsChannelsByFilter.
func (mr *MockLoanProductsChannelByFilterServiceMockRecorder) LoanProductsChannelsByFilter(filter interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LoanProductsChannelsByFilter", reflect.TypeOf((*MockLoanProductsChannelByFilterService)(nil).LoanProductsChannelsByFilter), filter)
}
