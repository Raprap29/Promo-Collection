package error_handling

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRetrieveSubscriberUsageError(t *testing.T) {
	t.Run("without status code", func(t *testing.T) {
		err := errors.New("test error")
		apiErr := NewRetrieveSubscriberUsageError(err)

		assert.Equal(t, err, apiErr.Err)
		assert.Equal(t, 0, apiErr.StatusCode)
		assert.Empty(t, apiErr.ErrorCode)
		assert.Empty(t, apiErr.ErrorMessage)
	})

	t.Run("with status code", func(t *testing.T) {
		err := errors.New("test error")
		statusCode := 400
		apiErr := NewRetrieveSubscriberUsageError(err, statusCode)

		assert.Equal(t, err, apiErr.Err)
		assert.Equal(t, statusCode, apiErr.StatusCode)
		assert.Empty(t, apiErr.ErrorCode)
		assert.Empty(t, apiErr.ErrorMessage)
	})
}

func TestRetrieveSubscriberUsageError_Error(t *testing.T) {
	t.Run("API error", func(t *testing.T) {
		err := errors.New("test error")
		apiErr := &RetrieveSubscriberUsageError{
			Err:        err,
			StatusCode: 400,
		}

		errorString := apiErr.Error()
		assert.Contains(t, errorString, "API error:")
		assert.Contains(t, errorString, "statusCode=400")
		assert.Contains(t, errorString, "test error")
		assert.Contains(t, errorString, "errorCode=")
	})

	t.Run("HTTP error", func(t *testing.T) {
		err := errors.New("test error")
		apiErr := &RetrieveSubscriberUsageError{
			Err:          err,
			StatusCode:   400,
			ErrorCode:    "E1001",
			ErrorMessage: "Bad request",
		}

		errorString := apiErr.Error()
		assert.Contains(t, errorString, "HTTP error:")
		assert.Contains(t, errorString, "statusCode=400")
		assert.Contains(t, errorString, "test error")
		assert.Contains(t, errorString, "errorCode=E1001")
		assert.Contains(t, errorString, "errorMessage=Bad request")
	})
}
