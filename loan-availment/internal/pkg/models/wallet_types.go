package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type WalletTypes struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Name      string             `bson:"name"`
	CreatedAt primitive.DateTime `bson:"createdAt"`
	UpdatedAt primitive.DateTime `bson:"updatedAt"`
	Number    int                `bson:"number"`
}
