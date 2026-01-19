package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PesoToDataConversion struct {
	LoanType       string  `json:"loanType" bson:"loanType"`
	ConversionRate float64 `json:"conversionRate" bson:"conversionRate"`
}

type SystemLevelRules struct {
	ID                       primitive.ObjectID     `json:"id" bson:"_id"`
	CreditScoreThreshold     int32                  `json:"creditScoreThreshold" bson:"creditScoreThreshold"`
	PartialCollectionEnabled bool                   `json:"partialCollectionEnabled" bson:"partialCollectionEnabled"`
	ReservedAmountForPartial int32                  `json:"reservedAmountForPartialCollection" bson:"reservedAmountForPartialCollection"`
	EducationPeriod          int32                  `json:"educationPeriod" bson:"educationPeriod"`
	DefermentPeriod          int32                  `json:"defermentPeriod" bson:"defermentPeriod"`
	OverdueThreshold         int32                  `json:"overdueThreshold" bson:"overdueThreshold"`
	UpdatedAt                primitive.DateTime     `json:"updatedAt" bson:"updatedAt"`
	UpdatedUserID            primitive.ObjectID     `json:"updatedUserId" bson:"updatedUserId"`
	MaxDataCollectionPercent float64                `json:"maxDataCollectionPercent" bson:"maxDataCollectionPercent"`
	PesoToDataConversion     []PesoToDataConversion `json:"pesoToDataConversionMatrix" bson:"pesoToDataConversionMatrix"`
	MinCollectionAmount      float64                `json:"minCollectionAmount" bson:"minCollectionAmount"`
	PromoCollectionEnabled   bool                   `json:"promoCollectionEnabled" bson:"promoCollectionEnabled"`
	WalletExclusionList      []string               `json:"walletExclusionList" bson:"walletExclusionList"`
	PromoEducationPeriod     int32                  `json:"promoEducationPeriod" bson:"promoEducationPeriod"`
	GracePeriod              int32                  `json:"gracePeriod" bson:"gracePeriod"`
	LoanLoadPeriod           int32                  `json:"loanLoadPeriod" bson:"loanLoadPeriod"`
}
