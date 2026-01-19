package models

// Request
type DecrementWalletSubscriptionRequest struct {
	MSISDN     string `json:"msisdn"`
	Wallet     string `json:"wallet"`
	Amount     string `json:"amount"`
	Unit       string `json:"unit"`
	IsRollBack bool   `json:"isRollBack"`
	Duration   string `json:"duration"`
	Channel    string `json:"channel"`
}

// Success Response (HTTP 200)
type DecrementWalletSubscriptionSuccess struct {
	ResultCode  string `json:"resultCode"`
	Description string `json:"description"`
}

type RetrieveSubscriberUsageRequest struct {
	MSISDN           string `json:"msisdn"`           // Path parameter
	IdType           string `json:"idType"`           // Query parameter
	QueryType        string `json:"queryType"`        // Query parameter
	DetailsFlag      int    `json:"detailsFlag"`      // Query parameter
	SubscriptionType string `json:"subscriptionType"` // Query parameter
}

type RetrieveSubscriberUsageSuccess struct {
	ID              string   `json:"ID"`
	PaymentCategory string   `json:"PaymentCategory"`
	SubscriberId    string   `json:"SubscriberId"`
	CustomerType    string   `json:"CustomerType"`
	CustomerSubType string   `json:"CustomerSubType"`
	SubscriberType  string   `json:"SubscriberType"`
	ActiveDate      string   `json:"ActiveDate"`
	Brand           string   `json:"Brand"`
	Buckets         []Bucket `json:"Buckets"`
}

type Bucket struct {
	BucketId        string         `json:"BucketId"`
	StartDate       string         `json:"StartDate"`
	EndDate         string         `json:"EndDate"`
	State           string         `json:"State"`
	VolumeRemaining int            `json:"VolumeRemaining"`
	TotalAllocated  int            `json:"TotalAllocated"`
	VolumeUsed      int            `json:"VolumeUsed"`
	Type            string         `json:"Type"`
	Unit            string         `json:"Unit"`
	Subscriptions   []Subscription `json:"Subscriptions"`
}

type Subscription struct {
	SubscriptionId string `json:"SubscriptionId"`
	Keyword        string `json:"Keyword"`
	StartDate      string `json:"StartDate"`
	EndDate        string `json:"EndDate"`
	BucketId       string `json:"BucketId"`
	Volume         int    `json:"Volume"`
	Unit           string `json:"Unit"`
}
