package downstream

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"promo-collection-worker/internal/pkg/config"
	"promo-collection-worker/internal/pkg/consts"
	errs "promo-collection-worker/internal/pkg/downstream/error_handling"
	"promo-collection-worker/internal/pkg/log_messages"
	"promo-collection-worker/internal/pkg/logger"
	"promo-collection-worker/internal/pkg/models"
)

// DecrementWalletSubscriptionAPI interface (for mocking & testing)
type DecrementWalletSubscriptionAPI interface {
	DecrementWalletSubscription(ctx context.Context, req *models.DecrementWalletSubscriptionRequest) (
		*models.DecrementWalletSubscriptionSuccess, error)
}

type DecrementWalletSubscriptionClient struct {
	URL        string
	apiKey     string
	httpClient *http.Client
}

func NewDecrementWalletSubscriptionClient(cfg config.HIPConfig) *DecrementWalletSubscriptionClient {
	return &DecrementWalletSubscriptionClient{
		URL:    cfg.DecrementWalletSubscriptionURL,
		apiKey: cfg.DecrementWalletSubscriptionAPIKey,
		httpClient: &http.Client{
			Timeout: cfg.HTTPTimeout,
		},
	}
}

// processResponseBody processes the response body from DecrementWalletSubscription API
func (c *DecrementWalletSubscriptionClient) processResponseBody(
	ctx context.Context,
	statusCode int,
	bodyBytes []byte,
) (*models.DecrementWalletSubscriptionSuccess, error) {
	logger.CtxInfo(ctx, "Processing DecrementWalletSubscription response", slog.Int("status", statusCode))

	if statusCode == http.StatusOK {
		var respData models.DecrementWalletSubscriptionSuccess
		if err := json.NewDecoder(bytes.NewBuffer(bodyBytes)).Decode(&respData); err != nil {
			logger.CtxError(ctx, "failed to decode DecrementWalletSubscription success response", err)
			return nil, errs.NewDecrementWalletSubscriptionError(
				fmt.Errorf("decoded success response: %w", err),
			)
		}
		if respData.ResultCode == "0" {
			return &respData, nil
		}
	}

	var apiErr errs.DecrementWalletSubscriptionError
	if err := json.NewDecoder(bytes.NewBuffer(bodyBytes)).Decode(&apiErr); err != nil {
		logger.CtxError(ctx, "failed to decode DecrementWalletSubscription error response", err)
		return nil, errs.NewDecrementWalletSubscriptionError(fmt.Errorf("decoded error response: %w", err))
	}

	// Set the status code in the error
	apiErr.StatusCode = statusCode

	// Create an error message based on the description or error details
	var errorMsg string
	switch {
	case apiErr.ResultCode != "" && apiErr.Description != "":
		errorMsg = fmt.Sprintf(log_messages.ErrorApiReturnedError, apiErr.Description)
	case apiErr.ErrorDetails != "" && apiErr.ErrorCode != "":
		errorMsg = fmt.Sprintf(log_messages.ErrorApiReturnedError, apiErr.ErrorDetails)
	default:
		errorMsg = log_messages.ErrorUnknownFormatError
	}

	apiErr.Err = errors.New(errorMsg)

	return nil, &apiErr
}

func (c *DecrementWalletSubscriptionClient) DecrementWalletSubscription(
	ctx context.Context,
	req *models.DecrementWalletSubscriptionRequest,
) (*models.DecrementWalletSubscriptionSuccess, error) {
	url := c.URL

	body, err := json.Marshal(req)
	if err != nil {
		logger.CtxError(ctx, "failed to marshal request for DecrementWalletSubscription", err)
		return nil, errs.NewDecrementWalletSubscriptionError(fmt.Errorf("marshal request: %w", err))
	}

	logger.CtxInfo(ctx, "Preparing to send DecrementWalletSubscription request",
		slog.String("url", url),
		slog.String("content-type", consts.ContentType),
		slog.String("body", string(body)),
	)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		logger.CtxError(ctx, "failed to build DecrementWalletSubscription request", err)
		return nil, errs.NewDecrementWalletSubscriptionError(fmt.Errorf("build request: %w", err))
	}
	httpReq.Header.Set("Content-Type", consts.ContentType)
	httpReq.Header.Set("x-api-key", c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		logger.CtxError(ctx, "failed to send request to DecrementWalletSubscription",
			err,
			slog.String("url", url),
			slog.String("content-type", consts.ContentType),
			slog.String("body", string(body)),
		)

		// Using -1 as a special indicator that there was no HTTP response
		return nil, errs.NewDecrementWalletSubscriptionError(
			fmt.Errorf("failed to send request: %w", err),
			-1)
	}

	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			logger.CtxError(ctx, "failed to close DecrementWalletSubscription response body", cerr)
		}
	}()

	logger.CtxInfo(ctx, "Received DecrementWalletSubscription response", slog.Int("status", resp.StatusCode))

	// Read the response body once
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.CtxError(ctx, "failed to read DecrementWalletSubscription response body", err)
		return nil, errs.NewDecrementWalletSubscriptionError(
			fmt.Errorf("read response body: %w", err),
			resp.StatusCode,
		)
	}

	return c.processResponseBody(ctx, resp.StatusCode, bodyBytes)
}
