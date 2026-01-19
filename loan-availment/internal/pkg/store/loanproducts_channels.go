package store

import (

	"globe/dodrio_loan_availment/internal/pkg/consts"
	"globe/dodrio_loan_availment/internal/pkg/db"
	"globe/dodrio_loan_availment/internal/pkg/logger"
	"globe/dodrio_loan_availment/internal/pkg/models"


	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type LoanProductsChannelRepository struct {
	repo *MongoRepository[models.LoanProductsChannels]
}

func NewLoanProductsChannelsRepository() *LoanProductsChannelRepository {
	collection := db.MDB.Database.Collection(consts.LoanProductsChannelCollection)
	mrepo := NewMongoRepository[models.LoanProductsChannels](collection)
	return &LoanProductsChannelRepository{repo: mrepo}
}

func (r *LoanProductsChannelRepository) LoanProductsChannelsByFilter(filter interface{}) (*models.LoanProductsChannels, error) {

	result, err := r.repo.Read(filter)
	if err != nil {
		return nil, err
	}

	return &result, nil

}

func (r *LoanProductsChannelRepository) ChannelByProductId(productId primitive.ObjectID) (primitive.ObjectID, string, bool, error) {

	channelPipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "productId", Value: productId}}}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: consts.ChannelsCollection},
			{Key: "localField", Value: "channelId"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "channelDetails"},
		}}},
		{{Key: "$unwind", Value: "$channelDetails"}},
		{{Key: "$project", Value: bson.D{
			{Key: "_id", Value: "$channelDetails._id"},
			{Key: "channelCode", Value: "$channelDetails.code"},
			{Key: "isDeleted", Value: "$channelDetails.isDeleted"},
		}}},
	}

	var channelResult struct {
		Id        primitive.ObjectID `bson:"_id"`
		Code      string             `bson:"code"`
		IsDeleted bool               `bson:"isDeleted"`
	}

	if err := r.repo.Aggregate(channelPipeline, &channelResult); err != nil {
		if err == mongo.ErrNoDocuments {
			logger.Warn("No channelResult found")
		} else {
			logger.Error(err)
		}
		return primitive.NilObjectID, "", channelResult.IsDeleted, err
	}

	return channelResult.Id, channelResult.Code, channelResult.IsDeleted, nil

}
