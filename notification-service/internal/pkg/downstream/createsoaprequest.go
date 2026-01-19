package downstream

import (
	"encoding/xml"
	"notificationservice/internal/pkg/models"
)

type soapEnvelope struct {
	XMLName   xml.Name                 `xml:"soap:Envelope"`
	XmlnsSoap string                   `xml:"xmlns:soap,attr"`
	XmlnsNot  string                   `xml:"xmlns:not,attr"`
	Header    models.SoapRequestHeader `xml:"soap:Header"`
	Body      soapBody                 `xml:"soap:Body"`
}

type soapBody struct {
	SendSMS models.SoapRequestSendSMSNotification `xml:"not:SendSMSNotification"`
}

func CreateSOAPRequest(messageID, msisdn string, params []models.SoapRequestNotifParameter) (string, error) {
	sms := models.SoapRequestSendSMSNotification{
		SMSSourceAddress: "",
		MSISDN:           msisdn,
		NotifPatternId:   messageID,
	}
	if len(params) > 0 {
		sms.NotifParameters = &models.SoapRequestNotifParameters{
			NotifParameter: params,
		}
	}

	env := soapEnvelope{
		XmlnsSoap: "http://www.w3.org/2003/05/soap-envelope",
		XmlnsNot:  "http://www.globe.com/warcraft/wsdl/notification/",
		Header:    models.SoapRequestHeader{},
		Body:      soapBody{SendSMS: sms},
	}

	out, err := xml.MarshalIndent(env, "", "  ")
	if err != nil {
		return "", err
	}

	// Optionally, you may want to add the XML header:
	xmlHeader := `<?xml version="1.0" encoding="UTF-8"?>` + "\n"
	return xmlHeader + string(out), nil
}
