package models

type EligibilityCheckResponse struct {
	MSISDN          string   `json:"MSISDN" binding:"required"`
	SubscriberScore *string  `json:"SubscriberScore,omitempty"`
	RemainingScore  *string  `json:"RemainingScore,omitempty"`
	ExistingLoan    *bool    `json:"ExistingLoan,omitempty"`
	LoanKeyWord     *string  `json:"LoanKeyWord,omitempty"`
	LoanAmount      *float64 `json:"LoanAmount,omitempty"`
	// Brand           *string  `json:"Brand,omitempty"`
	// SkuAvailed                 string `json:"SKU availed"`
	// DateOfAvailment            string `json:"Date of Availment"`
	// OutstandingPrincipalAmount int32  `json:"Outstanding Principal Amount"`
	// OutstandingServiceFee      int32  `json:"Outstanding Service Fee"`
	Result     string `json:"Result,omitempty"`
	StatusCode string `json:"StatusCode,omitempty"`
}
