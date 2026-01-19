package models

type EligibilityCheckRequest struct {
	MSISDN string `json:"MSISDN" binding:"required"`
}
