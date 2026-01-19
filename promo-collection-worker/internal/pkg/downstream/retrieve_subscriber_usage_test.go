package downstream

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"promo-collection-worker/internal/pkg/config"
	errs "promo-collection-worker/internal/pkg/downstream/error_handling"
	"promo-collection-worker/internal/pkg/models"
)

func TestRetrieveSubscriberUsage(t *testing.T) {
	// Test cases
	tests := []struct {
		name           string
		msisdn         string
		mockResponse   interface{}
		mockStatusCode int
		expectError    bool
		errorType      interface{}
	}{
		{
			name:   "success response",
			msisdn: "639175884175",
			mockResponse: models.RetrieveSubscriberUsageSuccess{
				ID:              "12345",
				PaymentCategory: "Regular",
				SubscriberId:    "639175884175",
				CustomerType:    "Individual",
				SubscriberType:  "Postpaid",
				Buckets:         []models.Bucket{{BucketId: "Data_10GB"}},
			},
			mockStatusCode: http.StatusOK,
			expectError:    false,
		},
		{
			name:   "api error response",
			msisdn: "639175884175",
			mockResponse: errs.RetrieveSubscriberUsageError{
				ErrorCode:    "ERR-1001",
				ErrorMessage: "Invalid subscriber",
				StatusCode:   400,
			},
			mockStatusCode: http.StatusBadRequest,
			expectError:    true,
			errorType:      &errs.RetrieveSubscriberUsageError{},
		},
		{
			name:           "server error",
			msisdn:         "639175884175",
			mockResponse:   nil,
			mockStatusCode: http.StatusInternalServerError,
			expectError:    true,
			errorType:      &errs.RetrieveSubscriberUsageError{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Contains(t, r.URL.Path, tc.msisdn)
				assert.Equal(t, "test-channel", r.Header.Get("sourceChannel"))
				assert.Equal(t, "test-api-key", r.Header.Get("x-api-key"))

				// Verify query parameters
				query := r.URL.Query()
				assert.Equal(t, "MSISDN", query.Get("idType"))
				assert.Equal(t, "Data", query.Get("queryType"))
				assert.Equal(t, "1", query.Get("detailsFlag"))
				assert.Equal(t, "Postpaid", query.Get("subscriptionType"))

				// Return mock response
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.mockStatusCode)

				if tc.mockResponse != nil {
					json.NewEncoder(w).Encode(tc.mockResponse)
				}
			}))
			defer server.Close()

			// Create client with mock server URL
			client := &RetrieveSubscriberUsageClient{
				URL:           server.URL,
				apiKey:        "test-api-key",
				sourceChannel: "test-channel",
				httpClient:    &http.Client{Timeout: 5 * time.Second},
			}

			// Create request
			req := &models.RetrieveSubscriberUsageRequest{
				MSISDN:           tc.msisdn,
				IdType:           "MSISDN",
				QueryType:        "Data",
				DetailsFlag:      1,
				SubscriptionType: "Postpaid",
			}

			// Call the function
			resp, err := client.RetrieveSubscriberUsage(context.Background(), req)

			// Verify results
			if tc.expectError {
				assert.Error(t, err)
				if tc.errorType != nil {
					assert.IsType(t, tc.errorType, err)
				}
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tc.mockResponse.(models.RetrieveSubscriberUsageSuccess).ID, resp.ID)
			}
		})
	}
}

func TestBuildURLWithQueryParams(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		params      map[string]string
		expectedURL string
		expectError bool
	}{
		{
			name:        "valid url with params",
			baseURL:     "https://api.example.com/path",
			params:      map[string]string{"key1": "value1", "key2": "value2"},
			expectedURL: "https://api.example.com/path?key1=value1&key2=value2",
			expectError: false,
		},
		{
			name:        "valid url with special chars in params",
			baseURL:     "https://api.example.com/path",
			params:      map[string]string{"key": "value with spaces", "q": "search&term"},
			expectedURL: "https://api.example.com/path?key=value+with+spaces&q=search%26term",
			expectError: false,
		},
		{
			name:        "invalid url",
			baseURL:     "://invalid-url",
			params:      map[string]string{"key": "value"},
			expectedURL: "",
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := buildURLWithQueryParams(tc.baseURL, tc.params)

			if tc.expectError {
				assert.Error(t, err)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedURL, result)
			}
		})
	}
}

func TestNewRetrieveSubscriberUsageClient(t *testing.T) {
	// Create test config
	cfg := config.HIPConfig{
		RetrieveSubscriberUsageURL:           "https://api.example.com",
		RetrieveSubscriberUsageAPIKey:        "test-key",
		RetrieveSubscriberUsageSourceChannel: "test-channel",
		HTTPTimeout:                          5 * time.Second,
	}

	// Create client
	client := NewRetrieveSubscriberUsageClient(cfg)

	// Verify client
	require.NotNil(t, client)
	assert.Equal(t, cfg.RetrieveSubscriberUsageURL, client.URL)
	assert.Equal(t, cfg.RetrieveSubscriberUsageAPIKey, client.apiKey)
	assert.Equal(t, cfg.RetrieveSubscriberUsageSourceChannel, client.sourceChannel)
	assert.Equal(t, cfg.HTTPTimeout, client.httpClient.Timeout)
}

func TestRetrieveSubscriberUsage_RequestCreationError(t *testing.T) {
	// Create client with invalid URL to trigger NewRequestWithContext error
	client := &RetrieveSubscriberUsageClient{
		URL:           string([]byte{0x7f}), // Invalid URL character
		apiKey:        "test-key",
		sourceChannel: "test-channel",
		httpClient:    &http.Client{},
	}

	// Create request
	req := &models.RetrieveSubscriberUsageRequest{
		MSISDN:           "639175884175",
		IdType:           "MSISDN",
		QueryType:        "Data",
		DetailsFlag:      1,
		SubscriptionType: "Postpaid",
	}

	// Call function
	resp, err := client.RetrieveSubscriberUsage(context.Background(), req)

	// Verify results
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.IsType(t, &errs.RetrieveSubscriberUsageError{}, err)
}

func TestRetrieveSubscriberUsage_HTTPClientError(t *testing.T) {
	// Create client with a timeout of 1 nanosecond to force timeout error
	client := &RetrieveSubscriberUsageClient{
		URL:           "https://api.example.com",
		apiKey:        "test-key",
		sourceChannel: "test-channel",
		httpClient:    &http.Client{Timeout: 1 * time.Nanosecond},
	}

	// Create request
	req := &models.RetrieveSubscriberUsageRequest{
		MSISDN:           "639175884175",
		IdType:           "MSISDN",
		QueryType:        "Data",
		DetailsFlag:      1,
		SubscriptionType: "Postpaid",
	}

	// Call function
	resp, err := client.RetrieveSubscriberUsage(context.Background(), req)

	// Verify results
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.IsType(t, &errs.RetrieveSubscriberUsageError{}, err)
}

func TestRetrieveSubscriberUsage_InvalidJSONResponse(t *testing.T) {
	// Setup mock server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	// Create client
	client := &RetrieveSubscriberUsageClient{
		URL:           server.URL,
		apiKey:        "test-key",
		sourceChannel: "test-channel",
		httpClient:    &http.Client{},
	}

	// Create request
	req := &models.RetrieveSubscriberUsageRequest{
		MSISDN:           "639175884175",
		IdType:           "MSISDN",
		QueryType:        "Data",
		DetailsFlag:      1,
		SubscriptionType: "Postpaid",
	}

	// Call function
	resp, err := client.RetrieveSubscriberUsage(context.Background(), req)

	// Verify results
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.IsType(t, &errs.RetrieveSubscriberUsageError{}, err)
}

func TestRetrieveSubscriberUsage_InvalidErrorResponse(t *testing.T) {
	// Setup mock server that returns error status with invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid error json"))
	}))
	defer server.Close()

	// Create client
	client := &RetrieveSubscriberUsageClient{
		URL:           server.URL,
		apiKey:        "test-key",
		sourceChannel: "test-channel",
		httpClient:    &http.Client{},
	}

	// Create request
	req := &models.RetrieveSubscriberUsageRequest{
		MSISDN:           "639175884175",
		IdType:           "MSISDN",
		QueryType:        "Data",
		DetailsFlag:      1,
		SubscriptionType: "Postpaid",
	}

	// Call function
	resp, err := client.RetrieveSubscriberUsage(context.Background(), req)

	// Verify results
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.IsType(t, &errs.RetrieveSubscriberUsageError{}, err)
}

func TestFindMatchingBucketId(t *testing.T) {
	ctx := context.Background()

	t.Run("matching bucket found", func(t *testing.T) {
		// Create test response with matching bucket
		resp := &models.RetrieveSubscriberUsageSuccess{
			SubscriberId: "639175884175",
			Buckets: []models.Bucket{
				{
					BucketId:        "WLT12300",
					VolumeRemaining: 500,
					Unit:            "MB",
				},
				{
					BucketId:        "WLT45600",
					VolumeRemaining: 200,
					Unit:            "MB",
				},
			},
		}

		// Call function with matching wallet keyword
		walletKeyword := "WLT12300"
		bucket, err := FindMatchingBucketId(ctx, resp, walletKeyword)

		// Verify results
		assert.NoError(t, err)
		assert.Equal(t, walletKeyword, bucket.BucketId)
		assert.Equal(t, 500, bucket.VolumeRemaining)
		assert.Equal(t, "MB", bucket.Unit)
	})

	t.Run("no buckets in response", func(t *testing.T) {
		// Create test response with no buckets
		resp := &models.RetrieveSubscriberUsageSuccess{
			SubscriberId: "639175884175",
			Buckets:      []models.Bucket{},
		}

		// Call function
		walletKeyword := "WLT12300"
		bucket, err := FindMatchingBucketId(ctx, resp, walletKeyword)

		// Verify results
		assert.Error(t, err)
		assert.Empty(t, bucket)
		assert.Contains(t, err.Error(), "no buckets found in subscriber usage response")
	})

	t.Run("no matching bucket found", func(t *testing.T) {
		// Create test response with non-matching buckets
		resp := &models.RetrieveSubscriberUsageSuccess{
			SubscriberId: "639175884175",
			Buckets: []models.Bucket{
				{
					BucketId:        "WLT45600",
					VolumeRemaining: 200,
					Unit:            "MB",
				},
				{
					BucketId:        "WLT78900",
					VolumeRemaining: 300,
					Unit:            "MB",
				},
			},
		}

		// Call function with non-matching wallet keyword
		walletKeyword := "WLT12300"
		bucket, err := FindMatchingBucketId(ctx, resp, walletKeyword)

		// Verify results
		assert.Error(t, err)
		assert.Empty(t, bucket)
		assert.Contains(t, err.Error(), "no matching bucket found for wallet keyword: "+walletKeyword)
	})
}
