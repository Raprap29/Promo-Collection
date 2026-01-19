package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Loans struct {
	LoanId                        primitive.ObjectID `bson:"_id"`
	MSISDN                        string             `bson:"MSISDN"`
	TotalLoanAmount               float64            `bson:"totalLoanAmount"`
	GUID                          string             `bson:"GUID"`
	ServiceFee                    float64            `bson:"serviceFee"`
	LoanProductId                 primitive.ObjectID `bson:"loanProductId"`
	BrandId                       primitive.ObjectID `bson:"brandId"`
	AvailmentTransactionId        primitive.ObjectID `bson:"availmentTransactionId"`
	LoanType                      string             `bson:"loanType"`
	CreatedAt                     time.Time          `bson:"createdAt"`
	Migrated                      bool               `bson:"migrated"`
	TotalUnpaidLoan               float64            `bson:"totalUnpaidLoan"`
	MinCollectionAmount           float64            `bson:"minCollectionAmount"`
	MaxDataCollectionPercent      float64            `bson:"maxDataCollectionPercent"`
	PromoEducationPeriodTimestamp primitive.DateTime `bson:"promoEducationPeriodTimestamp"`
	GracePeriodTimestamp          primitive.DateTime `bson:"gracePeriodTimestamp"`
	LoanLoadPeriodTimestamp       primitive.DateTime `bson:"loanLoadPeriodTimestamp"`
	ConversionRate                float64            `bson:"conversionRate"`
	Version                       int32              `bson:"version"`
}
