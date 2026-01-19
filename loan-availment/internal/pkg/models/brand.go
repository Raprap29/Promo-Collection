package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Brand struct {
	Id                    primitive.ObjectID `bson:"_id"`
	Code                  string             `bson:"code"`
	IsDeleted             bool               `bson:"isDeleted"`
	CreateAt              primitive.DateTime `bson:"createAt"`
	UpdatedAt             primitive.DateTime `bson:"updatedAt"`
	systemLevelAcceptable bool               `bson:"systemLevelAcceptable"`
}
