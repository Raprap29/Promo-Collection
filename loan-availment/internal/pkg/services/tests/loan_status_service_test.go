package tests

import (
	"context"
	"globe/dodrio_loan_availment/internal/pkg/models"
	"globe/dodrio_loan_availment/internal/pkg/notification"
	"globe/dodrio_loan_availment/internal/pkg/pubsub"
	"globe/dodrio_loan_availment/internal/pkg/services"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func TestLoanStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testLoanID := primitive.NewObjectID()
	testProductID := primitive.NewObjectID()
	testBrandID := primitive.NewObjectID()

	tests := []struct {
		name          string
		msisdn        string
		setupMocks    func(*MockLoanRepo, *MockLoanProductRepo, *MockNotificationService, *MockHelper, *MockBrandExist)
		expectedCode  int
		expectedError error
	}{
		{
			name:   "Success - Active Loan Found",
			msisdn: "9123456789",
			setupMocks: func(lr *MockLoanRepo, lpr *MockLoanProductRepo, ns *MockNotificationService, h *MockHelper, be *MockBrandExist) {
				loan := &models.Loans{
					LoanId:        testLoanID,
					LoanProductId: testProductID,
					BrandId:       testBrandID,
					MSISDN:        "9123456789",
					CreatedAt:     time.Now(),
				}

				product := &models.LoanProduct{
					ProductId: testProductID,
					Name:      "Test Product",
					Price:     1000.00,
					Active:    true,
				}

				lr.On("ActiveLoanByFilter", bson.M{"MSISDN": "9123456789"}).Return(loan, nil)
				lpr.On("LoanProductByFilter", bson.M{"_id": testProductID}).Return(product, nil)

			},
			expectedCode:  http.StatusOK,
			expectedError: nil,
		},
		{
			name:   "No Active Loan Found",
			msisdn: "9123456789",
			setupMocks: func(lr *MockLoanRepo, lpr *MockLoanProductRepo, ns *MockNotificationService, h *MockHelper, be *MockBrandExist) {
				lr.On("ActiveLoanByFilter", bson.M{"MSISDN": "9123456789"}).Return(nil, mongo.ErrNoDocuments)

				h.On("GetCreditScoreCustomerTypeBrandType",
					mock.MatchedBy(func(ctx context.Context) bool { return true }),
					"9123456789",
				).Return(float64(700), "REGULAR", "BRAND1", nil)

				be.On("CheckIfBrandExistsAndActive",
					mock.MatchedBy(func(ctx context.Context) bool { return true }),
					"BRAND1",
				).Return(true, testBrandID, nil)
			},
			expectedCode:  http.StatusOK,
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockLoanRepo := new(MockLoanRepo)
			mockLoanProductRepo := new(MockLoanProductRepo)
			mockHelper := new(MockHelper)
			mockBrandExist := new(MockBrandExist)
			mockNotificationService := new(MockNotificationService)

			tt.setupMocks(mockLoanRepo, mockLoanProductRepo, mockNotificationService, mockHelper, mockBrandExist)

			pubsubPublisher, _ := pubsub.NewPubSubPublisher(context.Background(), "test-project")
			service := services.NewLoanStatusService(
				nil,
				mockLoanRepo,
				mockLoanProductRepo,
				mockHelper,
				mockBrandExist,
				notification.NewNotificationService(pubsubPublisher),
			)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", "/", nil)
			c.Request = c.Request.WithContext(context.Background())

			request := models.AppCreationRequest{
				MSISDN: tt.msisdn,
			}

			err := service.LoanStatus(c, request)
			time.Sleep(100 * time.Millisecond)

			assert.Equal(t, tt.expectedError, err)
			assert.Equal(t, tt.expectedCode, w.Code)

			mockLoanRepo.AssertExpectations(t)
			mockLoanProductRepo.AssertExpectations(t)
			mockHelper.AssertExpectations(t)
			mockBrandExist.AssertExpectations(t)
		})
	}
}

// Mock structs
type MockLoanRepo struct {
	mock.Mock
}

type MockLoanProductRepo struct {
	mock.Mock
}

type MockHelper struct {
	mock.Mock
}

type MockBrandExist struct {
	mock.Mock
}

type MockNotificationService struct {
	mock.Mock
}

// Implement interface methods for mocks
func (m *MockLoanRepo) ActiveLoanByFilter(filter interface{}) (*models.Loans, error) {
	args := m.Called(filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Loans), args.Error(1)
}

func (m *MockLoanProductRepo) LoanProductByFilter(filter interface{}) (*models.LoanProduct, error) {
	args := m.Called(filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.LoanProduct), args.Error(1)
}

func (m *MockHelper) GetCreditScoreCustomerTypeBrandType(ctx context.Context, msisdn string) (float64, string, string, string, *time.Time, error) {
	args := m.Called(ctx, msisdn)
	var timeVal *time.Time
	if t, ok := args.Get(4).(*time.Time); ok {
		timeVal = t
	}
	return args.Get(0).(float64), args.String(1), args.String(2), args.String(3), timeVal, args.Error(5)
}

func (m *MockBrandExist) CheckIfBrandExistsAndActive(ctx context.Context, brand string) (bool, primitive.ObjectID, error) {
	args := m.Called(ctx, brand)
	return args.Get(0).(bool), args.Get(1).(primitive.ObjectID), args.Error(2)
}

func (m *MockNotificationService) NotifyUser(ctx context.Context, MSISDN string, event string, brand primitive.ObjectID, product *models.LoanProduct, loanId primitive.ObjectID, loanDate time.Time) error {
	args := m.Called(ctx, MSISDN, event, brand, product, loanId, loanDate)
	return args.Error(0)
}
