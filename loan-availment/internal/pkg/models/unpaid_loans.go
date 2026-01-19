package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UnpaidLoans struct {
	ID                 primitive.ObjectID  `bson:"_id"`
	LoanId             primitive.ObjectID  `bson:"loanId"`
	Version            float64             `bson:"version"`
	ValidFrom          time.Time           `bson:"validFrom"`
	ValidTo            *time.Time          `bson:"validTo"`
	TotalLoanAmount    float64             `bson:"totalLoanAmount"`
	UnpaidAmount       float64             `bson:"totalUnpaidAmount"`
	UnpaidServiceFee   float64             `bson:"unpaidServiceFee"`
	LastCollectionId   *primitive.ObjectID `bson:"lastCollectionId"`
	LastCollectionDate *time.Time          `bson:"lastCollectionDateTime"`
	Migrated           bool                `bson:"migrated"`
}
