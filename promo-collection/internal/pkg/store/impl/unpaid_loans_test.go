package impl

import (
	"context"
	"errors"
	"testing"

	"promocollection/internal/pkg/store/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockUnpaidLoansRepo struct {
	mock.Mock
}

func (m *MockUnpaidLoansRepo) FindOne(ctx context.Context, filter interface{}, opt *options.FindOneOptions) (models.UnpaidLoans, error) {
	args := m.Called(ctx, filter, opt)
	loan, _ := args.Get(0).(models.UnpaidLoans)
	return loan, args.Error(1)
}

func (m *MockUnpaidLoansRepo) Find(ctx context.Context, filter interface{}) ([]models.UnpaidLoans, error) {
	args := m.Called(ctx, filter)
	loans, _ := args.Get(0).([]models.UnpaidLoans)
	return loans, args.Error(1)
}

func (m *MockUnpaidLoansRepo) Delete(ctx context.Context, filter interface{}) error {
	args := m.Called(ctx, filter)
	return args.Error(0)
}

func TestGetUnpaidLoansByLoanIdSuccess(t *testing.T) {
	mockRepo := new(MockUnpaidLoansRepo)
	loanId := primitive.NewObjectID()
	expectedLoan := models.UnpaidLoans{ID: primitive.NewObjectID(), LoanId: loanId}

	mockRepo.On("FindOne", mock.Anything, mock.Anything, mock.Anything).
		Return(expectedLoan, nil)

	repo := NewUnpaidLoansRepositoryWithInterface(mockRepo)
	loan, err := repo.GetUnpaidLoansByLoanId(context.Background(), loanId)

	assert.NoError(t, err)
	assert.Equal(t, expectedLoan.LoanId, loan.LoanId)
	mockRepo.AssertExpectations(t)
}

func TestGetUnpaidLoansByLoanIdNoDocuments(t *testing.T) {
	mockRepo := new(MockUnpaidLoansRepo)
	loanId := primitive.NewObjectID()

	mockRepo.On("FindOne", mock.Anything, mock.Anything, mock.Anything).
		Return(models.UnpaidLoans{}, mongo.ErrNoDocuments)

	repo := NewUnpaidLoansRepositoryWithInterface(mockRepo)
	loan, err := repo.GetUnpaidLoansByLoanId(context.Background(), loanId)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, mongo.ErrNoDocuments))
	assert.Empty(t, loan.ID) // returned empty loan
	mockRepo.AssertExpectations(t)
}

func TestGetUnpaidLoansByLoanIdUnexpectedError(t *testing.T) {
	mockRepo := new(MockUnpaidLoansRepo)
	loanId := primitive.NewObjectID()
	expectedErr := errors.New("db connection failed")

	mockRepo.On("FindOne", mock.Anything, mock.Anything, mock.Anything).
		Return(models.UnpaidLoans{}, expectedErr)

	repo := NewUnpaidLoansRepositoryWithInterface(mockRepo)
	loan, err := repo.GetUnpaidLoansByLoanId(context.Background(), loanId)

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Empty(t, loan.ID)
	mockRepo.AssertExpectations(t)
}
