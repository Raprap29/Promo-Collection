package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RedisLoan represents the structure of loan data stored in Redis.
type RedisLoan struct {
	LoanID                   primitive.ObjectID `json:"loanId"`
	TotalLoanAmount          float64            `json:"totalLoanAmount"`
	ServiceFee               float64            `json:"serviceFee"`
	Outstanding              float64            `json:"outstanding"`
	AvailmentTimestamp       time.Time          `json:"availmentTimestamp"`
	GracePeriodTimestamp     time.Time          `json:"gracePeriodTimestamp"`
	EducationPeriodTimestamp time.Time          `json:"educationPeriodTimestamp"`
	LoanLoadPeriodTimestamp  time.Time          `json:"loanLoadPeriodTimestamp"`
	ConversionRate           float64            `json:"conversionRate"`
}
