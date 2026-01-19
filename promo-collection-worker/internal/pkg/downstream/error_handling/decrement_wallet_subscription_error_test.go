package error_handling

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDecrementWalletSubscriptionError(t *testing.T) {
	t.Run("without status code", func(t *testing.T) {
		err := errors.New("test error")
		apiErr := NewDecrementWalletSubscriptionError(err)

		assert.Equal(t, err, apiErr.Err)
		assert.Equal(t, 0, apiErr.StatusCode)
		assert.Empty(t, apiErr.ResultCode)
		assert.Empty(t, apiErr.Description)
		assert.Empty(t, apiErr.ErrorCode)
		assert.Empty(t, apiErr.ErrorDetails)
	})

	t.Run("with status code", func(t *testing.T) {
		err := errors.New("test error")
		statusCode := 400
		apiErr := NewDecrementWalletSubscriptionError(err, statusCode)

		assert.Equal(t, err, apiErr.Err)
		assert.Equal(t, statusCode, apiErr.StatusCode)
		assert.Empty(t, apiErr.ResultCode)
		assert.Empty(t, apiErr.Description)
		assert.Empty(t, apiErr.ErrorCode)
		assert.Empty(t, apiErr.ErrorDetails)
	})
}

func TestDecrementWalletSubscriptionError_Error(t *testing.T) {
	t.Run("API error", func(t *testing.T) {
		err := errors.New("test error")
		apiErr := &DecrementWalletSubscriptionError{
			Err:        err,
			StatusCode: 400,
			ResultCode: "1001",
		}

		errorString := apiErr.Error()
		assert.Contains(t, errorString, "API error:")
		assert.Contains(t, errorString, "statusCode=400")
		assert.Contains(t, errorString, "test error")
		assert.Contains(t, errorString, "resultCode=1001")
	})

	t.Run("HTTP error", func(t *testing.T) {
		err := errors.New("test error")
		apiErr := &DecrementWalletSubscriptionError{
			Err:          err,
			StatusCode:   400,
			ErrorCode:    "E1001",
			ErrorDetails: "Bad request",
		}

		errorString := apiErr.Error()
		assert.Contains(t, errorString, "HTTP error:")
		assert.Contains(t, errorString, "statusCode=400")
		assert.Contains(t, errorString, "test error")
		assert.Contains(t, errorString, "errorCode=E1001")
		assert.Contains(t, errorString, "desc=Bad request")
	})
}

func TestDecrementWalletSubscriptionError_IsNonAPIError(t *testing.T) {
	t.Run("is non-API error", func(t *testing.T) {
		err := errors.New("test error")
		apiErr := &DecrementWalletSubscriptionError{
			Err: err,
		}

		assert.True(t, apiErr.IsNonAPIError())
	})

	t.Run("with result code", func(t *testing.T) {
		err := errors.New("test error")
		apiErr := &DecrementWalletSubscriptionError{
			Err:        err,
			ResultCode: "1001",
		}

		assert.False(t, apiErr.IsNonAPIError())
	})

	t.Run("with error code", func(t *testing.T) {
		err := errors.New("test error")
		apiErr := &DecrementWalletSubscriptionError{
			Err:       err,
			ErrorCode: "E1001",
		}

		assert.False(t, apiErr.IsNonAPIError())
	})

	t.Run("with error details", func(t *testing.T) {
		err := errors.New("test error")
		apiErr := &DecrementWalletSubscriptionError{
			Err:          err,
			ErrorDetails: "Bad request",
		}

		assert.False(t, apiErr.IsNonAPIError())
	})

	t.Run("with status code", func(t *testing.T) {
		err := errors.New("test error")
		apiErr := &DecrementWalletSubscriptionError{
			Err:        err,
			StatusCode: 400,
		}

		assert.True(t, apiErr.IsNonAPIError())
	})

	t.Run("with description", func(t *testing.T) {
		err := errors.New("test error")
		apiErr := &DecrementWalletSubscriptionError{
			Err:         err,
			Description: "Bad request",
		}

		assert.False(t, apiErr.IsNonAPIError())
	})

	t.Run("nil error", func(t *testing.T) {
		apiErr := &DecrementWalletSubscriptionError{}
		assert.False(t, apiErr.IsNonAPIError())
	})
}
