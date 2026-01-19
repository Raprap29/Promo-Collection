package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LoanProduct struct {
	ProductId        primitive.ObjectID `bson:"_id,omitempty"`
	Name             string             `bson:"name"`
	Price            float64            `bson:"price"`
	ServiceFee       float64            `bson:"serviceFee"`
	FeeType          string             `bson:"feeType"`
	LoanType         string             `bson:"loanType"`
	WalletID         primitive.ObjectID `bson:"walletId,omitempty"`
	IsScoredProduct  bool               `bson:"isScoredProduct"`
	HasWhitelisting  bool               `bson:"hasWhitelisting"`
	ActivationMethod string             `bson:"activationMethod"`
	Active           bool               `bson:"active"`
	PromoStartDate   *time.Time         `bson:"promoStartDate,omitempty"`
	PromoEndDate     *time.Time         `bson:"promoEndDate,omitempty"`
	CreatedAt        primitive.DateTime `bson:"createdAt,omitempty"`
	UpdatedAt        primitive.DateTime `bson:"updatedAt,omitempty"`
	ProductGUID      string             `bson:"productGUID"`
	Migrated         bool               `bson:"migrated"`
	IsDeleted        bool               `bson:"isDeleted"`
}
