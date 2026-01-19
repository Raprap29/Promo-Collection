package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Messages struct {
	ID         primitive.ObjectID `bson:"_id"`
	CreatedAt  primitive.DateTime `bson:"createdAt"`
	UpdatedAt  primitive.DateTime `bson:"updatedAt"`
	PatternId  int32              `bson:"patternId"`
	Parameters []string           `bson:"parameters"`
	Event      string             `bson:"event"`
	BrandId    string             `bson:"brandId"`
	Labels     []string           `bson:"labels"`
	Migrated   bool               `bson:"migrated"`
	IsDeleted  bool               `bson:"isDeleted"`
}
