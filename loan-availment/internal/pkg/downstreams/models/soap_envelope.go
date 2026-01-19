package models

import "encoding/xml"

type SoapEnvelope struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    SoapBody `xml:"Body"`
}
