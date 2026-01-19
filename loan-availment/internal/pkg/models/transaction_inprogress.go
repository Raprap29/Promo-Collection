package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TransactionInProgress struct {
	Id        primitive.ObjectID `bson:"_id,omitempty"`
	MSISDN    string             `bson:"MSISDN"`
	CreatedAt time.Time          `bson:"createdAt"`
}
