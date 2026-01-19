package models

import "encoding/xml"

type LoginSessionSuccessResponse struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    struct {
		LoginResponse struct {
			XMLName xml.Name `xml:"loginResponse"`
			Return  struct {
				ResultCode        int `xml:"resultCode"`
				SessionResultCode int `xml:"sessionResultCode"`
			} `xml:"return"`
		} `xml:"loginResponse"`
	} `xml:"Body"`
}
