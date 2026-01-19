package downstream

import (
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"notificationservice/internal/pkg/models"
	"strings"
)

func SendSOAPRequest(url, apiKey, payloadXML string) (models.SoapResponseEnvelope, error) {
	payload := strings.NewReader(payloadXML)

	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		return models.SoapResponseEnvelope{}, err
	}

	req.Header.Add("x-api-key", apiKey)
	req.Header.Add("Content-Type", "application/xml")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return models.SoapResponseEnvelope{}, err
	}

	defer func() {
		_ = res.Body.Close()
	}()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return models.SoapResponseEnvelope{}, err
	}

	if res.StatusCode != 200 {
		// Return error with raw response
		return models.SoapResponseEnvelope{}, errors.New(string(body))
	}

	var envelope models.SoapResponseEnvelope
	if err := xml.Unmarshal(body, &envelope); err != nil {
		return models.SoapResponseEnvelope{}, err
	}

	return envelope, nil
}
