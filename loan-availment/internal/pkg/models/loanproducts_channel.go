package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type LoanProductsChannels struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"` 
	ProductID primitive.ObjectID `bson:"productId,omitempty"` 
	ChannelID primitive.ObjectID `bson:"channelId,omitempty"` 
}