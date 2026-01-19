package downstream

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"promo-collection-worker/internal/pkg/config"
	errs "promo-collection-worker/internal/pkg/downstream/error_handling"
	"promo-collection-worker/internal/pkg/log_messages"
	"promo-collection-worker/internal/pkg/models"

	"github.com/stretchr/testify/assert"
)

// FaultyCloser simulates a ReadCloser that fails on Close()
type FaultyCloser struct {
	io.Reader
}

func (f *FaultyCloser) Close() error { return errors.New("close failed") }

// FaultyRoundTripper simulates a http.Client RoundTripper returning a response with FaultyCloser
type FaultyRoundTripper struct{}

func (f *FaultyRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	body := &FaultyCloser{Reader: bytes.NewBufferString(`{"resultCode":"0","description":"ok"}`)}
	return &http.Response{
		StatusCode: 200,
		Body:       body,
		Header:     make(http.Header),
	}, nil
}

// ErrorResponseServer returns invalid JSON for error responses
type ErrorResponseServer struct {
	*httptest.Server
}

func NewErrorResponseServer() *ErrorResponseServer {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return a 400 status code but with invalid JSON for the error response
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{invalid-json-for-error}"))
	})
	return &ErrorResponseServer{httptest.NewServer(handler)}
}

// Test for error response JSON decode error
func TestDecrementWalletSubscription_ErrorResponseDecodeError(t *testing.T) {
	// Create a test server that returns invalid JSON for error responses
	server := NewErrorResponseServer()
	defer server.Close()

	// Create a client using the test server
	client := NewDecrementWalletSubscriptionClient(config.HIPConfig{
		DecrementWalletSubscriptionURL:    server.URL,
		DecrementWalletSubscriptionAPIKey: "test-key",
	})

	// Create a valid request
	req := &models.DecrementWalletSubscriptionRequest{
		MSISDN:  "639175884175",
		Wallet:  "WLT12300",
		Amount:  "100",
		Unit:    "KB",
		Channel: "WEB",
	}

	// Call the method
	resp, err := client.DecrementWalletSubscription(context.Background(), req)

	// Verify the error
	if resp != nil {
		t.Errorf("Expected nil response, got %v", resp)
	}
	if err == nil {
		t.Fatal("Expected error response decode error, got nil")
	}

	// Verify it's the right type of error
	var apiErr *errs.DecrementWalletSubscriptionError
	if !errors.As(err, &apiErr) {
		t.Errorf("Expected DecrementWalletSubscriptionError, got %T", err)
	}

	// Verify the error message contains "decoded error response"
	if apiErr != nil && !errors.Is(apiErr.Err, nil) {
		errMsg := apiErr.Error()
		if errMsg == "" || !bytes.Contains([]byte(errMsg), []byte("decoded error response")) {
			t.Errorf("Expected error message to contain 'decoded error response', got: %s", errMsg)
		}
	}
}

func TestClient_DecrementWalletSubscription(t *testing.T) {
	req := &models.DecrementWalletSubscriptionRequest{
		MSISDN:  "639175884175",
		Wallet:  "WLT12300",
		Amount:  "100",
		Unit:    "KB",
		Channel: "WEB",
	}

	// --- Success Response ---
	successResp := models.DecrementWalletSubscriptionSuccess{
		ResultCode:  "0",
		Description: "OK",
	}
	successServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(successResp)
	}))
	defer successServer.Close()

	client := NewDecrementWalletSubscriptionClient(config.HIPConfig{
		DecrementWalletSubscriptionURL:    successServer.URL,
		DecrementWalletSubscriptionAPIKey: "key1234",
	})

	okResp, err := client.DecrementWalletSubscription(context.Background(), req)
	if err != nil {
		t.Fatalf("expected success, got err=%v", err)
	}
	if okResp.ResultCode != "0" {
		t.Fatalf("expected resultCode=0, got %v", okResp.ResultCode)
	}

	// --- Error Response ---
	errorResp := errs.DecrementWalletSubscriptionError{
		ErrorCode: "400", ErrorDetails: "Bad request",
	}
	errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(errorResp)
	}))
	defer errorServer.Close()

	client = NewDecrementWalletSubscriptionClient(config.HIPConfig{
		DecrementWalletSubscriptionURL:    errorServer.URL,
		DecrementWalletSubscriptionAPIKey: "key1234",
	})

	okResp, err = client.DecrementWalletSubscription(context.Background(), req)
	if okResp != nil {
		t.Fatalf("expected nil success response, got %v", okResp)
	}
	apiErr := &errs.DecrementWalletSubscriptionError{}
	if !errors.As(err, &apiErr) || apiErr.ErrorCode == "" || apiErr.ErrorCode != "400" {
		t.Fatalf("expected API error 400, got %v", err)
	}

	// --- Invalid JSON Response ---
	badJSONServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{invalid-json}"))
	}))
	defer badJSONServer.Close()

	client = NewDecrementWalletSubscriptionClient(config.HIPConfig{
		DecrementWalletSubscriptionURL:    badJSONServer.URL,
		DecrementWalletSubscriptionAPIKey: "key1234",
	})

	_, err = client.DecrementWalletSubscription(context.Background(), req)
	if err == nil {
		t.Fatalf("expected JSON decode error")
	}

	// --- http.NewRequestWithContext failure (invalid URL) ---
	client = NewDecrementWalletSubscriptionClient(config.HIPConfig{
		DecrementWalletSubscriptionURL:    "http://\x7f", // invalid URL
		DecrementWalletSubscriptionAPIKey: "key1234",
	})
	_, err = client.DecrementWalletSubscription(context.Background(), req)
	if err == nil {
		t.Fatalf("expected http.NewRequest error")
	}

	// --- httpClient.Do failure (unreachable server) ---
	client = NewDecrementWalletSubscriptionClient(config.HIPConfig{
		DecrementWalletSubscriptionURL:    "http://127.0.0.1:0",
		DecrementWalletSubscriptionAPIKey: "key1234",
	})
	_, err = client.DecrementWalletSubscription(context.Background(), req)
	if err == nil {
		t.Fatalf("expected httpClient.Do error")
	}

	// --- Response body Close error ---
	client = NewDecrementWalletSubscriptionClient(config.HIPConfig{
		DecrementWalletSubscriptionURL:    "http://dummy",
		DecrementWalletSubscriptionAPIKey: "key1234",
	})
	client.httpClient = &http.Client{Transport: &FaultyRoundTripper{}}

	okResp, err = client.DecrementWalletSubscription(context.Background(), req)
	if err != nil {
		t.Fatalf("expected success response even with Close error, got err=%v", err)
	}
	if okResp.ResultCode != "0" {
		t.Fatalf("expected resultCode=0, got %v", okResp.ResultCode)
	}
}

func TestDecrementWalletSubscriptionClient_processResponseBody_ErrorMessages(t *testing.T) {
	ctx := context.Background()
	client := &DecrementWalletSubscriptionClient{}

	t.Run("Case 1: ResultCode and Description present", func(t *testing.T) {
		errorResp := errs.DecrementWalletSubscriptionError{
			ResultCode:  "500",
			Description: "Internal server error",
		}

		bodyBytes, _ := json.Marshal(errorResp)

		result, err := client.processResponseBody(ctx, http.StatusInternalServerError, bodyBytes)

		assert.Nil(t, result)
		assert.Error(t, err)

		var apiErr *errs.DecrementWalletSubscriptionError
		assert.True(t, errors.As(err, &apiErr))

		expectedMsg := fmt.Sprintf(log_messages.ErrorApiReturnedError, "Internal server error")
		assert.Equal(t, expectedMsg, apiErr.Err.Error())
		assert.Equal(t, http.StatusInternalServerError, apiErr.StatusCode)
	})

	t.Run("Case 2: ErrorDetails and ErrorCode present", func(t *testing.T) {
		errorResp := errs.DecrementWalletSubscriptionError{
			ErrorCode:    "400",
			ErrorDetails: "Bad request - invalid parameters",
		}

		bodyBytes, _ := json.Marshal(errorResp)

		result, err := client.processResponseBody(ctx, http.StatusBadRequest, bodyBytes)

		assert.Nil(t, result)
		assert.Error(t, err)

		var apiErr *errs.DecrementWalletSubscriptionError
		assert.True(t, errors.As(err, &apiErr))

		expectedMsg := fmt.Sprintf(log_messages.ErrorApiReturnedError, "Bad request - invalid parameters")
		assert.Equal(t, expectedMsg, apiErr.Err.Error())
		assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	})

	t.Run("Case 3: Default case - unknown format (partial or missing fields)", func(t *testing.T) {
		testCases := []struct {
			name   string
			fields errs.DecrementWalletSubscriptionError
		}{
			{"Empty fields", errs.DecrementWalletSubscriptionError{}},
			{"Only ResultCode", errs.DecrementWalletSubscriptionError{ResultCode: "500"}},
			{"Only Description", errs.DecrementWalletSubscriptionError{Description: "Some error"}},
			{"Only ErrorDetails", errs.DecrementWalletSubscriptionError{ErrorDetails: "Some details"}},
			{"Only ErrorCode", errs.DecrementWalletSubscriptionError{ErrorCode: "400"}},
			{"ResultCode + ErrorDetails (both missing pairs)", errs.DecrementWalletSubscriptionError{
				ResultCode:   "500",
				ErrorDetails: "Some details",
			}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				bodyBytes, _ := json.Marshal(tc.fields)

				result, err := client.processResponseBody(ctx, http.StatusBadRequest, bodyBytes)

				assert.Nil(t, result)
				assert.Error(t, err)

				var apiErr *errs.DecrementWalletSubscriptionError
				assert.True(t, errors.As(err, &apiErr))

				assert.Equal(t, log_messages.ErrorUnknownFormatError, apiErr.Err.Error())
				assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
			})
		}
	})
}
