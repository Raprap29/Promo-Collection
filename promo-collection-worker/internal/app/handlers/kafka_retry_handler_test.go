package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"promo-collection-worker/internal/pkg/log_messages"
	"promo-collection-worker/internal/service"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockKafkaRetryService struct {
	mock.Mock
}

func (m *MockKafkaRetryService) RetryKafkaCollectionMessages(ctx context.Context) *service.KafkaRetryResponse {
	args := m.Called(ctx)
	return args.Get(0).(*service.KafkaRetryResponse)
}

func TestRetryKafkaCollectionMessages(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Success", func(t *testing.T) {
		mockService := new(MockKafkaRetryService)
		response := &service.KafkaRetryResponse{
			SuccessIDs: []string{"id1", "id2"},
			FailedIDs:  []string{},
			ErrorMsg:   "",
		}
		mockService.On("RetryKafkaCollectionMessages", mock.Anything).Return(response)
		handler := NewKafkaRetryHandler(mockService)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest("POST", "/IntegrationServices/Dodrio/KafkaRetry", nil)
		c.Request = req

		handler.RetryKafkaCollectionMessages(c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"success_ids":["id1","id2"]`)
		mockService.AssertExpectations(t)
	})

	t.Run("Partial success", func(t *testing.T) {
		mockService := new(MockKafkaRetryService)
		response := &service.KafkaRetryResponse{
			SuccessIDs: []string{"id1"},
			FailedIDs:  []string{"id2"},
			ErrorMsg:   log_messages.ErrorUpdatingKafkaFlag,
		}
		mockService.On("RetryKafkaCollectionMessages", mock.Anything).Return(response)
		handler := NewKafkaRetryHandler(mockService)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest("POST", "/IntegrationServices/Dodrio/KafkaRetry", nil)
		c.Request = req

		handler.RetryKafkaCollectionMessages(c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"success_ids":["id1"]`)
		assert.Contains(t, w.Body.String(), `"failed_ids":["id2"]`)
		assert.Contains(t, w.Body.String(), `"error":"error updating Kafka flag in database for transactions with error"`)
		mockService.AssertExpectations(t)
	})

	t.Run("No collections found", func(t *testing.T) {
		mockService := new(MockKafkaRetryService)
		response := &service.KafkaRetryResponse{
			SuccessIDs: []string{},
			FailedIDs:  []string{},
			ErrorMsg:   "",
		}
		mockService.On("RetryKafkaCollectionMessages", mock.Anything).Return(response)
		handler := NewKafkaRetryHandler(mockService)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest("POST", "/IntegrationServices/Dodrio/KafkaRetry", nil)
		c.Request = req

		handler.RetryKafkaCollectionMessages(c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"message":"`+log_messages.NoCollectionInDuration+`"`)
		mockService.AssertExpectations(t)
	})

	t.Run("Complete failure", func(t *testing.T) {
		mockService := new(MockKafkaRetryService)
		response := &service.KafkaRetryResponse{
			SuccessIDs: []string{},
			FailedIDs:  []string{},
			ErrorMsg:   log_messages.NoCollectionInDuration,
		}
		mockService.On("RetryKafkaCollectionMessages", mock.Anything).Return(response)
		handler := NewKafkaRetryHandler(mockService)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest("POST", "/IntegrationServices/Dodrio/KafkaRetry", nil)
		c.Request = req

		handler.RetryKafkaCollectionMessages(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), `"error":"`+response.ErrorMsg+`"`)
		mockService.AssertExpectations(t)
	})
}
