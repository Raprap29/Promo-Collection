package messages

import (
	"context"
	"errors"
	"testing"

	mongodb "promocollection/internal/pkg/db/mongo"
	"promocollection/internal/pkg/store/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mockMessagesStore struct {
	mock.Mock
}

func (m *mockMessagesStore) FindOne(ctx context.Context, filter interface{}, opt *options.FindOneOptions) (models.Messages, error) {
	args := m.Called(ctx, filter, opt)
	if msg, ok := args.Get(0).(models.Messages); ok {
		return msg, args.Error(1)
	}
	return models.Messages{}, args.Error(1)
}

func (m *mockMessagesStore) Find(ctx context.Context, filter interface{}) ([]models.Messages, error) {
	args := m.Called(ctx, filter)
	if msgs, ok := args.Get(0).([]models.Messages); ok {
		return msgs, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockMessagesStore) Delete(ctx context.Context, filter interface{}) error {
	args := m.Called(ctx, filter)
	return args.Error(0)
}

func TestGetPatternIdByEventandBrandId(t *testing.T) {
	ctx := context.Background()
	event := "LoanCreated"
	var patternID int32 = 12345
	brandId := primitive.NilObjectID

	tests := []struct {
		name          string
		setupMock     func(m *mockMessagesStore)
		expectedErr   error
		expectedEmpty bool
	}{
		{
			name: "success - message found",
			setupMock: func(m *mockMessagesStore) {
				m.On("FindOne", ctx, mock.Anything, mock.Anything).
					Return(models.Messages{Event: event, PatternId: patternID}, nil).
					Once()
			},
			expectedErr:   nil,
			expectedEmpty: false,
		},
		{
			name: "failure - no document found",
			setupMock: func(m *mockMessagesStore) {
				m.On("FindOne", ctx, mock.Anything, mock.Anything).
					Return(models.Messages{}, mongo.ErrNoDocuments).
					Once()
			},
			expectedErr:   mongo.ErrNoDocuments,
			expectedEmpty: true,
		},
		{
			name: "failure - unexpected error",
			setupMock: func(m *mockMessagesStore) {
				m.On("FindOne", ctx, mock.Anything, mock.Anything).
					Return(models.Messages{}, errors.New("db error")).
					Once()
			},
			expectedErr:   errors.New("db error"),
			expectedEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(mockMessagesStore)
			tt.setupMock(mockRepo)

			mr := NewMessagesRepositoryWithInterface(mockRepo)
			result, err := mr.GetPatternIdByEventandBrandId(ctx, event, brandId)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.expectedEmpty {
					assert.Equal(t, models.Messages{}, *result)
				} else {
					assert.Equal(t, patternID, result.PatternId)
					assert.Equal(t, event, result.Event)
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestNewMessagesRepository(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("constructor coverage", func(mt *mtest.T) {
		client := &mongodb.MongoClient{Database: mt.DB}
		repo := NewMessagesRepository(client)
		assert.NotNil(t, repo)
		assert.NotNil(t, repo.repo)
	})
}
