package models

import "encoding/xml"

type SoapRequestNotifParameter struct {
	Name  string
	Value string
}

type SoapRequestEnvelope struct {
	XMLName xml.Name          `xml:"soap:Envelope"`
	Header  SoapRequestHeader `xml:"soap:Header"`
	Body    SoapRequestBody   `xml:"soap:Body"`
}

type SoapRequestHeader struct{}

type SoapRequestBody struct {
	SendSMSNotification SoapRequestSendSMSNotification `xml:"not:SendSMSNotification"`
}

type SoapRequestSendSMSNotification struct {
	SMSSourceAddress string                      `xml:"SMSSourceAddress"`
	MSISDN           string                      `xml:"MSISDN"`
	NotifPatternId   string                      `xml:"NotifPatternId"`
	NotifParameters  *SoapRequestNotifParameters `xml:"NotifParameters,omitempty"`
}

type SoapRequestNotifParameters struct {
	NotifParameter []SoapRequestNotifParameter `xml:"NotifParameter,omitempty"`
}
