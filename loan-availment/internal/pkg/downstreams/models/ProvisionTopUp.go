package models

import "encoding/xml"

type TopUpSuccessResponse struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    struct {
		TopupResponse struct {
			XMLName xml.Name `xml:"topupResponse"`
			Return  struct {
				ResultCode        int `xml:"resultCode"`
				SessionResultCode int `xml:"sessionResultCode"`
				TransId           int `xml:"transID"`
			} `xml:"return"`
		} `xml:"topupResponse"`
	} `xml:"Body"`
}
