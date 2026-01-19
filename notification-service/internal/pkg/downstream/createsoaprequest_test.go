package downstream

import (
	"encoding/xml"
	"strings"
	"testing"

	"notificationservice/internal/pkg/models"
)

func TestCreateSOAPRequest_NoParams(t *testing.T) {
	messageId := "12345"
	msisdn := "9876543210"

	xmlStr, err := CreateSOAPRequest(messageId, msisdn, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify SOAP envelope tags
	if !strings.Contains(xmlStr, "<soap:Envelope") {
		t.Errorf("Expected SOAP Envelope tag, got: %s", xmlStr)
	}
	if !strings.Contains(xmlStr, "<soap:Body>") {
		t.Errorf("Expected SOAP Body tag, got: %s", xmlStr)
	}

	// Verify data fields
	if !strings.Contains(xmlStr, messageId) {
		t.Errorf("Expected messageId %s in XML, got: %s", messageId, xmlStr)
	}
	if !strings.Contains(xmlStr, msisdn) {
		t.Errorf("Expected MSISDN %s in XML, got: %s", msisdn, xmlStr)
	}
	if strings.Contains(xmlStr, "<NotifParameter>") {
		t.Errorf("Expected no parameters, but found one: %s", xmlStr)
	}

	// Validate that generated XML is well-formed
	var v interface{}
	if err := xml.Unmarshal([]byte(xmlStr), &v); err != nil {
		t.Errorf("Generated XML is invalid: %v\nXML: %s", err, xmlStr)
	}
}

func TestCreateSOAPRequest_WithParams(t *testing.T) {
	messageId := "99999"
	msisdn := "09171234567"
	params := []models.SoapRequestNotifParameter{
		{Name: "AccountNumber", Value: "ABC123"},
		{Name: "Amount", Value: "500"},
	}

	xmlStr, err := CreateSOAPRequest(messageId, msisdn, params)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify structure
	if !strings.Contains(xmlStr, "<NotifParameter>") {
		t.Errorf("Expected NotifParameter elements, got: %s", xmlStr)
	}

	// Verify parameter content
	if !strings.Contains(xmlStr, "AccountNumber") {
		t.Errorf("Expected 'AccountNumber' param in XML, got: %s", xmlStr)
	}
	if !strings.Contains(xmlStr, "ABC123") {
		t.Errorf("Expected 'ABC123' value in XML, got: %s", xmlStr)
	}
	if !strings.Contains(xmlStr, "Amount") {
		t.Errorf("Expected 'Amount' param in XML, got: %s", xmlStr)
	}
	if !strings.Contains(xmlStr, "500") {
		t.Errorf("Expected '500' value in XML, got: %s", xmlStr)
	}

	// Validate XML format
	var v interface{}
	if err := xml.Unmarshal([]byte(xmlStr), &v); err != nil {
		t.Errorf("Generated XML is invalid: %v\nXML: %s", err, xmlStr)
	}
}
