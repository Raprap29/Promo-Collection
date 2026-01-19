package models

import "encoding/xml"

// ErrorResponse struct for parsing the SOAP fault response
type ErrorResponse struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    Body     `xml:"Body"`
}

// Body struct to hold Fault information
type Body struct {
	Fault Fault `xml:"Fault"`
}

// Fault struct with the fault details
type Fault struct {
	FaultCode   string `xml:"faultcode"`
	FaultString string `xml:"faultstring"`
	Detail      Detail `xml:"detail"`
}

// Detail struct to capture the detailed error message
type Detail struct {
	Text string `xml:"Text"`
}
