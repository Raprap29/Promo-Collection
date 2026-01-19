package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RedisLoan struct {
	EducationPeriodTimestamp primitive.DateTime `json:"educationPeriodTimestamp"`
	LoanLoadPeriodTimestamp  primitive.DateTime `json:"loanLoadPeriodTimestamp"`
}
