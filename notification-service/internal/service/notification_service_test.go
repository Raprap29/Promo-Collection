package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"notificationservice/internal/pkg/config"
	"notificationservice/internal/pkg/models"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandleMessage_Success(t *testing.T) {
	// 1. Start a mock SOAP server
	mockResponse := `<Envelope>
		<Body>
			<SendSMSNotificationResponse>
				<SendSMSNotificationResult>
					<ResultCode>0</ResultCode>
					<ResultNamespace>Success</ResultNamespace>
				</SendSMSNotificationResult>
			</SendSMSNotificationResponse>
		</Body>
	</Envelope>`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// optional: validate API key
		if r.Header.Get("x-api-key") != "test-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer ts.Close()

	// 2. Prepare notification
	notification := models.PubSubNotificationMessage{
		Msisdn:          "1234567890",
		PatternID:       1001,
		NotifParameters: []models.NotificationParameter{{Name: "param1", Value: "value1"}},
	}
	msgBytes, _ := json.Marshal(notification)

	svc := &NotificationService{
		cfg: &config.AppConfig{
			SMSNotificationConfig: config.SMSNotificationConfig{
				SEND_SMS_NOTIFICATION_URL:       ts.URL,
				SEND_SMS_NOTIFICATION_X_API_KEY: "test-key",
			},
		},
	}

	// 3. Call HandleMessage
	err := svc.HandleMessage(context.Background(), msgBytes)
	assert.NoError(t, err)
}

func TestHandleMessage_InvalidJSON(t *testing.T) {
	msgBytes := []byte(`invalid-json`)

	svc := &NotificationService{
		cfg: &config.AppConfig{},
	}

	err := svc.HandleMessage(context.Background(), msgBytes)
	assert.NoError(t, err)
}
