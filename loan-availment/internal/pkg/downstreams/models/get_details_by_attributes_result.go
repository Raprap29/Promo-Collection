package models

type GetDetailsByAttributesResult struct {
	ResponseCode        string            `xml:"ResponseCode"`
	ResponseDescription string            `xml:"ResponseDescription"`
	SubscriberDetails   SubscriberDetails `xml:"SubscriberDetails"`
}

type AuthResponse struct {
	AccessToken string `json:"accessToken"`
}

// Struct for the UUP request payload
type UUPAttributeRequest struct {
	ResourceType  int      `json:"resourceType"`
	ResourceValue string   `json:"resourceValue"`
	Attributes    []string `json:"attributes"`
}
type GetDetailsByAttributesResponseResult struct {
	AttributeValues []AttributeValue `json:"attributeValues"`
}

type AttributeValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
type GetDetailsByAttributesErrorResponse struct {
	ResponseCode        string `json:"ResponseCode"`
	ResponseDescription string `json:"ResponseDesc"`
	ErrorCode           string `json:"ErrorCode"`
	ErrorDetails        string `json:"ErrorDetails"`
}
