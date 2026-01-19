package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type WhitelistedSubs struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	ProductID primitive.ObjectID `bson:"productId"`
	MSISDN    string             `bson:"MSISDN"`
}
