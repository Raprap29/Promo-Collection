package consts

type LoanType string

const (
	LoanTypeGES          LoanType = "GES"
	LoanTypePromo        LoanType = "PROMO SKU"
	LoanTypeLoad         LoanType = "LOAD SKU"
	DeductableAmount              = "Deductable Amount"
	EducationPeriodSpiel          = "PromoPurchaseSuccessWithinPromoEducPeriod"
	LoanTypeScored       LoanType = "SCORED"
	LoanTypeUnScored     LoanType = "UNSCORED"
)
