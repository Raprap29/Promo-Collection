package models

import "encoding/xml"

type CreateSessionSuccessResponse struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    struct {
		CreateSessionResponse struct {
			XMLName xml.Name `xml:"createSessionResponse"`
			Return  struct {
				SessionResultCode int    `xml:"resultCode"`
				Session           string `xml:"session"`
			} `xml:"return"`
		} `xml:"createSessionResponse"`
	} `xml:"Body"`
}
