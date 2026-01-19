package downstream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"promo-collection-worker/internal/pkg/config"
	errs "promo-collection-worker/internal/pkg/downstream/error_handling"
	"promo-collection-worker/internal/pkg/log_messages"
	"promo-collection-worker/internal/pkg/logger"
	"promo-collection-worker/internal/pkg/models"
)

type RetrieveSubscriberUsageAPI interface {
	RetrieveSubscriberUsage(ctx context.Context, req *models.RetrieveSubscriberUsageRequest) (
		*models.RetrieveSubscriberUsageSuccess, error)
}

type RetrieveSubscriberUsageClient struct {
	URL           string
	apiKey        string
	sourceChannel string
	httpClient    *http.Client
}

func NewRetrieveSubscriberUsageClient(cfg config.HIPConfig) *RetrieveSubscriberUsageClient {
	return &RetrieveSubscriberUsageClient{
		URL:           cfg.RetrieveSubscriberUsageURL,
		apiKey:        cfg.RetrieveSubscriberUsageAPIKey,
		sourceChannel: cfg.RetrieveSubscriberUsageSourceChannel,
		httpClient: &http.Client{
			Timeout: cfg.HTTPTimeout,
		},
	}
}

// prepareURL formats the URL with query parameters for RetrieveSubscriberUsage
func (c *RetrieveSubscriberUsageClient) prepareURL(
	ctx context.Context,
	req *models.RetrieveSubscriberUsageRequest,
) (string, error) {
	// Format the URL with the MSISDN as a path parameter
	baseURL := fmt.Sprintf("%s/%s", c.URL, req.MSISDN)
	logger.CtxInfo(ctx, "RetrieveSubscriberUsageRequest", slog.String("baseURL", baseURL))

	// Add query parameters
	queryParams := make(map[string]string)
	queryParams["idType"] = req.IdType
	queryParams["queryType"] = req.QueryType
	queryParams["detailsFlag"] = fmt.Sprintf("%d", req.DetailsFlag)
	queryParams["subscriptionType"] = req.SubscriptionType

	// Build URL with query parameters
	urlWithParams, err := buildURLWithQueryParams(baseURL, queryParams)
	if err != nil {
		logger.CtxError(ctx, log_messages.ErrorFailedToBuildURLWithQueryParams, err)
		return "", fmt.Errorf(log_messages.ErrorFailedToBuildURLWithQueryParams, err)
	}

	return urlWithParams, nil
}

// processResponseBody processes the response body from RetrieveSubscriberUsage API
func (c *RetrieveSubscriberUsageClient) processResponseBody(
	ctx context.Context,
	statusCode int,
	body io.Reader,
) (*models.RetrieveSubscriberUsageSuccess, error) {
	logger.CtxInfo(ctx, log_messages.ReceivedRetrieveSubscriberUsageResponse, slog.Int("status", statusCode))

	if statusCode == http.StatusOK {
		var respData models.RetrieveSubscriberUsageSuccess
		if err := json.NewDecoder(body).Decode(&respData); err != nil {
			logger.CtxError(ctx, log_messages.ErrorFailedToDecodeRetrieveSubscriberUsageSuccessResponse, err)
			return nil, errs.NewRetrieveSubscriberUsageError(
				fmt.Errorf(log_messages.ErrorFailedToDecodeRetrieveSubscriberUsageSuccessResponse, err),
			)
		}
		return &respData, nil
	}

	var apiErr errs.RetrieveSubscriberUsageError
	if err := json.NewDecoder(body).Decode(&apiErr); err != nil {
		logger.CtxError(ctx, log_messages.ErrorFailedToDecodeRetrieveSubscriberUsageErrorResponse, err)
		return nil, errs.NewRetrieveSubscriberUsageError(fmt.Errorf(
			log_messages.ErrorFailedToDecodeRetrieveSubscriberUsageErrorResponse, err))
	}

	// Set the status code in the error
	apiErr.StatusCode = statusCode

	// Create an error message based on the error message
	var errorMsg string
	switch {
	case apiErr.ErrorMessage != "" && apiErr.ErrorCode != "":
		errorMsg = fmt.Sprintf(log_messages.ErrorApiReturnedError, apiErr.ErrorMessage)
	default:
		errorMsg = log_messages.ErrorUnknownFormatError
	}

	apiErr.Err = errors.New(errorMsg)

	return nil, &apiErr
}

func (c *RetrieveSubscriberUsageClient) RetrieveSubscriberUsage(
	ctx context.Context,
	req *models.RetrieveSubscriberUsageRequest,
) (*models.RetrieveSubscriberUsageSuccess, error) {

	logger.CtxInfo(ctx, "RetrieveSubscriberUsageRequest", slog.Any("req", req))

	urlWithParams, err := c.prepareURL(ctx, req)
	if err != nil {
		return nil, errs.NewRetrieveSubscriberUsageError(err)
	}

	logger.CtxInfo(ctx, log_messages.ErrorPreparingToSendRetrieveSubscriberUsageRequest,
		slog.String("url", urlWithParams),
		slog.String("sourceChannel", c.sourceChannel),
		slog.String("msisdn", req.MSISDN),
		slog.String("idType", req.IdType),
		slog.String("queryType", req.QueryType),
		slog.Int("detailsFlag", req.DetailsFlag),
		slog.String("subscriptionType", req.SubscriptionType),
	)

	// Create GET request without body
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, urlWithParams, nil)
	if err != nil {
		logger.CtxError(ctx, log_messages.ErrorFailedToBuildRetrieveSubscriberUsageRequest, err)
		return nil, errs.NewRetrieveSubscriberUsageError(fmt.Errorf(
			log_messages.ErrorFailedToBuildRetrieveSubscriberUsageRequest, err))
	}
	httpReq.Header.Set("sourceChannel", c.sourceChannel)
	httpReq.Header.Set("x-api-key", c.apiKey)

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		logger.CtxError(ctx, log_messages.ErrorFailedToSendRetrieveSubscriberUsageRequest,
			err,
			slog.String("url", urlWithParams),
			slog.String("sourceChannel", c.sourceChannel),
		)

		return nil, errs.NewRetrieveSubscriberUsageError(fmt.Errorf(
			log_messages.ErrorFailedToSendRetrieveSubscriberUsageRequest, err), -1)
	}

	defer func() {
		if cerr := httpResp.Body.Close(); cerr != nil {
			logger.CtxError(ctx, log_messages.ErrorFailedToCloseRetrieveSubscriberUsageResponseBody, cerr)
		}
	}()

	return c.processResponseBody(ctx, httpResp.StatusCode, httpResp.Body)
}

// buildURLWithQueryParams constructs a URL with query parameters
func buildURLWithQueryParams(baseURL string, params map[string]string) (string, error) {
	// Parse the base URL
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	// Get existing query parameters
	q := u.Query()

	// Add new parameters
	for key, value := range params {
		q.Set(key, value)
	}

	// Set the updated query parameters
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// FindMatchingBucketId searches for a matching bucket by wallet keyword in the response
// Returns the bucket ID if found, or an error if no matching bucket is found
func FindMatchingBucketId(
	ctx context.Context,
	resp *models.RetrieveSubscriberUsageSuccess,
	walletKeyword string,
) (models.Bucket, error) {
	if len(resp.Buckets) == 0 {
		logger.CtxError(ctx, log_messages.ErrorNoBucketsFoundInSubscriberUsageResponse, nil)
		return models.Bucket{}, fmt.Errorf(log_messages.ErrorNoBucketsFoundInSubscriberUsageResponse)
	}

	// Search for matching bucket by wallet keyword
	for i := range resp.Buckets {
		if resp.Buckets[i].BucketId == walletKeyword {
			return resp.Buckets[i], nil
		}
	}

	// If no matching bucket found
	logger.CtxError(ctx, log_messages.ErrorNoMatchingBucketFoundForWalletKeyword, nil,
		slog.String("walletKeyword", walletKeyword))
	return models.Bucket{}, fmt.Errorf(log_messages.ErrorNoMatchingBucketFoundForWalletKeyword, walletKeyword)
}
