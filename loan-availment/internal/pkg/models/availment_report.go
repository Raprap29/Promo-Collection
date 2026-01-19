package models

import "time"

type AvailmentReport struct {
	AvailmentId             string    `json:"availmentId"`
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
	LoanAgeing              *int      `json:"loanAgeing"`
	CreditScoreAtTime       int       `json:"creditScoreAtTime"`
	StatusOfProvisionedLoan string    `json:"statusOfProvisionedLoan"`
	FromMigration           string    `json:"fromMigration"`
}
