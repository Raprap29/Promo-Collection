package loan_products

import (
	"context"
	"errors"
	"testing"

	mongodb "promo-collection-worker/internal/pkg/db/mongo"
	"promo-collection-worker/internal/pkg/store/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mockLoanProductsStore struct {
	mock.Mock
}

func (m *mockLoanProductsStore) FindOne(ctx context.Context, filter interface{}, opt *options.FindOneOptions) (models.LoanProducts, error) {
	args := m.Called(ctx, filter, opt)
	if avail, ok := args.Get(0).(models.LoanProducts); ok {
		return avail, args.Error(1)
	}
	return models.LoanProducts{}, args.Error(1)
}

func (m *mockLoanProductsStore) Find(ctx context.Context, filter interface{}) ([]models.LoanProducts, error) {
	args := m.Called(ctx, filter)
	if avails, ok := args.Get(0).([]models.LoanProducts); ok {
		return avails, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockLoanProductsStore) Delete(ctx context.Context, filter interface{}) error {
	args := m.Called(ctx, filter)
	return args.Error(0)
}

const optionsFindOneOptionsString = "*options.FindOneOptions"

func TestGetProductNameById(t *testing.T) {
	ctx := context.Background()
	id := primitive.NewObjectID()

	tests := []struct {
		name          string
		setupMock     func(m *mockLoanProductsStore)
		expectedErr   error
		expectedEmpty bool
	}{
		{
			name: "success - document found",
			setupMock: func(m *mockLoanProductsStore) {
				m.On("FindOne", ctx, mock.Anything, mock.AnythingOfType(optionsFindOneOptionsString)).
					Return(models.LoanProducts{ID: id}, nil).
					Once()
			},
			expectedErr:   nil,
			expectedEmpty: false,
		},
		{
			name: "failure - no document found",
			setupMock: func(m *mockLoanProductsStore) {
				m.On("FindOne", ctx, mock.Anything, mock.AnythingOfType(optionsFindOneOptionsString)).
					Return(models.LoanProducts{}, mongo.ErrNoDocuments).
					Once()
			},
			expectedErr:   mongo.ErrNoDocuments,
			expectedEmpty: true,
		},
		{
			name: "failure - unexpected error",
			setupMock: func(m *mockLoanProductsStore) {
				m.On("FindOne", ctx, mock.Anything, mock.AnythingOfType(optionsFindOneOptionsString)).
					Return(models.LoanProducts{}, errors.New("db error")).
					Once()
			},
			expectedErr:   errors.New("db error"),
			expectedEmpty: true,
		},
		{
			name: "success - empty doc returned but no error",
			setupMock: func(m *mockLoanProductsStore) {
				m.On("FindOne", ctx, mock.Anything, mock.AnythingOfType(optionsFindOneOptionsString)).
					Return(models.LoanProducts{}, nil).
					Once()
			},
			expectedErr:   nil,
			expectedEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(mockLoanProductsStore)
			tt.setupMock(mockRepo)

			lpr := NewLoanProductsRepositoryWithInterface(mockRepo)
			result, err := lpr.GetProductNameById(ctx, id)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)

				if tt.expectedEmpty {
					assert.Equal(t, models.LoanProducts{}, *result)
				} else {
					assert.Equal(t, id, result.ID)
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestNewLoanProductsRepository(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("constructor works", func(mt *mtest.T) {
		client := &mongodb.MongoClient{
			Database: mt.DB,
		}

		repo := NewLoanProductsRepository(client)

		assert.NotNil(t, repo)
		assert.NotNil(t, repo.repo)
	})
}
