package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type LoanProductsBrands struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"` 
	BrandID   primitive.ObjectID `bson:"brandId,omitempty"` 
	ProductID primitive.ObjectID `bson:"productId,omitempty"` 
}