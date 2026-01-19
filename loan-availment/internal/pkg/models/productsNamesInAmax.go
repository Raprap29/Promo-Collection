package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ProductsNamesInAmax struct {
	Id          primitive.ObjectID `bson:"_id"`
	Name        string             `bson:"name"`
	IsDeleted   bool               `bson:"isDeleted"`
	CreatedAt   primitive.DateTime `bson:"createdAt"`
	UpdatedAt   primitive.DateTime `bson:"updatedAt"`
	Migrated    bool               `bson:"migrated"`
	ProductGUID string             `bson:"productGUID"`
	ProductId   primitive.ObjectID `bson:"productId"`
	BrandId     primitive.ObjectID `bson:"brandId"`
}
