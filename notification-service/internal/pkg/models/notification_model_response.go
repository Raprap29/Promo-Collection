package models

import "encoding/xml"

// Struct for the entire SOAP envelope
type SoapResponseEnvelope struct {
	XMLName xml.Name           `xml:"Envelope"`
	Header  SoapResponseHeader `xml:"Header"`
	Body    SoapResponseBody   `xml:"Body"`
}

// Struct for the SOAP header
type SoapResponseHeader struct {
	WarcraftHeader SoapResponseWarcraftHeader `xml:"WarcraftHeader"`
}

// Struct for the specific header content
type SoapResponseWarcraftHeader struct {
	EsbMessageID        string `xml:"EsbMessageID"`
	EsbRequestDateTime  string `xml:"EsbRequestDateTime"`
	EsbResponseDateTime string `xml:"EsbResponseDateTime"`
	EsbIMLNumber        string `xml:"EsbIMLNumber"`
	EsbOperationName    string `xml:"EsbOperationName"`
}

// Struct for the SOAP body, handling both Fault and Response
type SoapResponseBody struct {
	Fault                       *Fault                       `xml:"Fault,omitempty"`
	SendSMSNotificationResponse *SendSMSNotificationResponse `xml:"SendSMSNotificationResponse,omitempty"`
}

// Struct for the SOAP fault
type Fault struct {
	Code   FaultCode   `xml:"Code"`
	Reason FaultReason `xml:"Reason"`
	Detail FaultDetail `xml:"Detail"`
}

type FaultCode struct {
	Value string `xml:"Value"`
}

type FaultReason struct {
	Text string `xml:"Text"`
}

type FaultDetail struct {
	Text string `xml:"Text"`
}

// Struct for the SendSMSNotificationResponse
type SendSMSNotificationResponse struct {
	Result SendSMSNotificationResult `xml:"SendSMSNotificationResult"`
}

type SendSMSNotificationResult struct {
	ResultCode      string `xml:"ResultCode"`
	ResultNamespace string `xml:"ResultNamespace"`
}
