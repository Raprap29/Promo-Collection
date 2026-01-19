package downstream

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendSOAPRequest_Success(t *testing.T) {
	// Mock SOAP response
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

	// Start a mock HTTP server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-key" {
			http.Error(w, "missing api key", http.StatusUnauthorized)
			return
		}
		bodyBytes := make([]byte, r.ContentLength)
		r.Body.Read(bodyBytes)
		if !strings.Contains(string(bodyBytes), "Envelope") {
			http.Error(w, "invalid SOAP body", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer ts.Close()

	payload := `<Envelope><Body><SendSMSNotification/></Body></Envelope>`
	result, err := SendSOAPRequest(ts.URL, "test-key", payload)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.Body.SendSMSNotificationResponse == nil {
		t.Errorf("Expected SendSMSNotificationResponse, got nil")
	} else if result.Body.SendSMSNotificationResponse.Result.ResultCode != "0" {
		t.Errorf("Expected ResultCode=0, got %s", result.Body.SendSMSNotificationResponse.Result.ResultCode)
	}
}

func TestSendSOAPRequest_Non200(t *testing.T) {
	mockResponse := "Internal Server Error"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, mockResponse, http.StatusInternalServerError)
	}))
	defer ts.Close()

	_, err := SendSOAPRequest(ts.URL, "test-key", "<Envelope></Envelope>")
	if err == nil {
		t.Fatalf("Expected error for non-200 response, got nil")
	}
	if !strings.Contains(err.Error(), "Internal Server Error") {
		t.Errorf("Expected error to contain response body, got: %v", err)
	}
}

func TestSendSOAPRequest_InvalidXML(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not-xml-response"))
	}))
	defer ts.Close()

	_, err := SendSOAPRequest(ts.URL, "test-key", "<Envelope></Envelope>")
	if err == nil {
		t.Fatalf("Expected error due to invalid XML, got nil")
	}

	if !strings.Contains(err.Error(), "not-xml-response") && !strings.Contains(err.Error(), "EOF") {
		t.Logf("Received XML parse error: %v", err)
	}
}
