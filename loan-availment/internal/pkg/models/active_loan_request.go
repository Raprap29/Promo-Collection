package models

import "time"

type ActiveLoanRequest struct {
	MSISDN string `json:"msisdn" binding:"required"`
}
type LoanAvaimentLog struct {
	TypeOfTransaction  string    `json:"transaction_type"` // Represents the type of transaction
	Status             string    `json:"status"`           // Status of the transaction
	Channel            string    `json:"channel"`          // Channel through which the transaction occurred
	TAT                float64   `json:"tat"`              // Turnaround time (TAT)
	StartTime          time.Time `json:"start_time"`       // Start time of the transaction (use a timestamp format)
	EndTime            time.Time `json:"end_time"`         // End time of the transaction (use a timestamp format)
	TransactionID      string    `json:"transaction_id"`   // Unique transaction ID
	MSISDN             string    `json:"msisdn"`           // Mobile number (MSISDN)
	ErrorCode          string    `json:"error_code"`       // Error code, if any
	TransactionSubType string    `json:"transaction_sub_type"`
}
