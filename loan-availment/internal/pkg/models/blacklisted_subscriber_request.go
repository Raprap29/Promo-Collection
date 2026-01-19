package models

type BlackListedSubscriberRequest struct {
	MSISDN      string `json:"msisdn"  binding:"required"`
	Blacklisted bool   `json:"blacklisted"  binding:"required"`
}
