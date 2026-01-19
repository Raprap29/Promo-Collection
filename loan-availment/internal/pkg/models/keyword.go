package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Keyword struct {
	Id        primitive.ObjectID `bson:"_id"`
	CreatedAt primitive.DateTime `bson:"createdAt"`
	UpdatedAt primitive.DateTime `bson:"updatedAt"`
	Name      string             `bson:"name"`
	ProductId primitive.ObjectID `bson:"productId"`
	IsDeleted bool               `bson:"isDeleted"`
}
