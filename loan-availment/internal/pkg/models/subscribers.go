package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Subscribers struct {
	ID                           primitive.ObjectID `bson:"_id"`
	MSISDN                       string             `bson:"MSISDN"`
	Blacklisted                  bool               `bson:"blacklisted"`
	Migrated                     bool               `bson:"migrated"`
	ExclusionReason              string             `bson:"exclusionReason"`
	SubscriberType               string             `bson:"subscriberType"`
	CustomerType                 string             `bson:"customerType"`
	CreditLoanLimitAmount        float64            `bson:"creditLoanLimitAmount"`
	CreditLoanLimitAmountGenDate *time.Time         `bson:"creditLoanLimitAmountGenDate"`
	UpdatedAt                    *time.Time         `bson:"updatedAt"`
	CreatedAt                    time.Time          `bson:"createdAt"`
}
