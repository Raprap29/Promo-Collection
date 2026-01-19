package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Channels struct {
	Id        primitive.ObjectID `bson:"_id"`
	Code      string             `bson:"code"`
	IsDeleted bool               `bson:"isDeleted"`
	CreatedAt primitive.DateTime `bson:"createdAt"`
	UpdatedAt primitive.DateTime `bson:"updatedAt"`
}
