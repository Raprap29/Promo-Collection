package downstreams

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	// "encoding/json"
	"fmt"
	"io"
	"net/http"

	// "time"

	"globe/dodrio_loan_availment/configs"
	"globe/dodrio_loan_availment/internal/pkg/consts"
	"globe/dodrio_loan_availment/internal/pkg/downstreams/models"
	"globe/dodrio_loan_availment/internal/pkg/logger"
)

// Main function to get details by attribute
func GetDetailsByAttribute(ctx context.Context, MSISDN string) (*models.GetDetailsByAttributesResponseResult, error) {
	clientID := configs.UUP_CLIENT_ID
	clientSecret := configs.UUP_CLIENT_SECRET
	apiKey := configs.UUP_API_KEY
	xApiKey := configs.UUP_X_API_KEY
	username := configs.UUP_USERNAME
	uupAcessTockenUrl := configs.UUP_ACCESSTOKEN_URL
	uupGetAttributeUrl := configs.UUP_SERVICE_URI_GET_DETAILS_BY_ATTRIBUTE

	// Step 1: Call for UUP Access Token
	method := "GET"
	headers := map[string]string{
		"x-api-key":    xApiKey,
		"clientID":     clientID,
		"clientSecret": clientSecret,
		"apiKey":       apiKey,
	}
	body, err := makeAPICall(ctx, uupAcessTockenUrl, method, headers, nil)
	if err != nil {
		return nil, err
	}

	// Parse the access token from the response
	var authResp models.AuthResponse
	err = json.Unmarshal(body, &authResp)
	if err != nil {
		return nil, consts.ErrorUUPAccessTockenFailed
	}
	logger.Info("access token Api response %v", authResp)
	token := authResp.AccessToken

	// Step 2: Call for UUP Get Details by Attribute
	method = "POST"
	headers = map[string]string{
		"x-api-key":    xApiKey,
		"apiKey":       apiKey,
		"clientID":     clientID,
		"clientSecret": clientSecret,
		"username":     username,
		"token":        token, // Pass the token obtained from authentication
		"Content-Type": "application/json",
	}

	payload := fmt.Sprintf(`{
		"resourceType": 1,
		"resourceValue": "%s",
		"attributes": [
        	"subid",
        	"msisdn",
        	"btc",
        	"cfutd",
        	"prodt",
        	"clla",
        	"clsd"
    ]
	}`, MSISDN)

	// Print the JSON payload
	// fmt.Println("JSON Payload:", payload)

	// Use strings.NewReader to pass the JSON payload as an io.Reader
	payloadReader := strings.NewReader(payload)

	// fmt.Println("Get details by attribute:", uupGetAttributeUrl, method, headers)

	// Send the request with payload as JSON string
	logger.Info("subcscriber Details Api Payload %v", payload)
	bodyOfAttributeURL, err := makeAPICall(ctx, uupGetAttributeUrl, method, headers, payloadReader)
	if err != nil {
		logger.Error(ctx, "Error:%v", err)
		return nil, err
	}

	// Assuming the response needs to be unmarshalled into the result struct
	var getDetailsResult models.GetDetailsByAttributesResponseResult

	err = json.Unmarshal(bodyOfAttributeURL, &getDetailsResult)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling get details response: %v", err)
	}
	logger.Info("subcscriber Details Api response %v", getDetailsResult)
	return &getDetailsResult, nil
}

// REST API function
func makeAPICall(ct context.Context, url string, method string, headers map[string]string, payload io.Reader) ([]byte, error) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(configs.TIMEOUT_IN_SECONDS)*time.Second)
	defer cancel()

	// Create a new HTTP request
	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		return nil, err
	}

	// Set headers
	for key, value := range headers {
		req.Header.Add(key, value)
	}

	// Send the request
	// Associate the context with the request
	req = req.WithContext(ctx)
	client := &http.Client{}
	logger.Debug("http hip client certificate required : %v ", configs.HIP_CA_CERTIFICATE_REQUIRED)
	if configs.HIP_CA_CERTIFICATE_REQUIRED {
		rootCA := configs.HIP_CA_CERTIFICATE

		// Create a CA certificate pool and add the server's CA
		caCertPool := x509.NewCertPool()
		if ok := caCertPool.AppendCertsFromPEM([]byte(rootCA)); !ok {
			logger.Error(ctx, consts.ErrorFailedAppendCACertificate)
		}

		// Setup TLS configuration to verify the server certificate
		tlsConfig := &tls.Config{
			RootCAs: caCertPool,
		}

		// Create an HTTP transport with the TLS configuration
		tr := &http.Transport{
			TLSClientConfig: tlsConfig,
		}
		client = &http.Client{Transport: tr}
		logger.Debug("http hip client: %v", client)
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.Error(ct, "Error while making API request to UUP API: %v", err.Error())
		return []byte(""), consts.ErrorDownstreamTimeout
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Check if the response status is 2XX (successful)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logger.Debug(ctx, resp.StatusCode, body)
		if resp.StatusCode == http.StatusInternalServerError {
			var errResp models.GetDetailsByAttributesErrorResponse
			// Try to unmarshal the response body
			if err := json.Unmarshal(body, &errResp); err == nil {
				// If unmarshalling fails, return the original body with a generic error message
				if errResp.ResponseCode == strconv.Itoa(http.StatusInternalServerError) {
					return nil, consts.ErrorMSISDNNotFound
				}
			}

			// If successful, return a more detailed error message with responseCode and responseDescription
			return nil, fmt.Errorf("API error on UUP service call: responseCode: %s, responseDescription: %s, status code: %d", errResp.ResponseCode, errResp.ResponseDescription, resp.StatusCode)
		}
		return nil, fmt.Errorf("API error on UUP service call: %s, status code: %d", string(body), resp.StatusCode)
	}

	return body, nil
}
