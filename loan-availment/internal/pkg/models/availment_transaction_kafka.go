package models

import (
	"time"
)

type AvailmentKafkaMessage struct {
	TransactionID           string    `json:"transactionID"`
	TransactionDatetime     time.Time `json:"transactionDatetime"`
	AvailmentChannel        string    `json:"availmentChannel"`
	MSISDN                  string    `json:"MSISDN"`
	BrandType               string    `json:"brandType"`
	LoanType                string    `json:"loanType"`
	LoanProductName         string    `json:"loanProductName"`
	LoanProductKeyword      string    `json:"loanProductKeyword"`
	ServicingPartner        string    `json:"servicingPartner"`
	LoanAmount              float64   `json:"loanAmount"`
	ServiceFeeAmount        float64   `json:"serviceFeeAmount"`
	TransactionType         string    `json:"transactionType"`
	AvailmentResult         string    `json:"availmentResult"`
	AvailmentErrorText      string    `json:"availmentErrorText"`
	LoanAgeing              int32     `json:"loanAgeing"`
	StatusOfProvisionedLoan string    `json:"statusOfProvisionedLoan"`
}
