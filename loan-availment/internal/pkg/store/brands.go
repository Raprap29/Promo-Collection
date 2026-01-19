package store

import (
	"context"
	"fmt"
	"globe/dodrio_loan_availment/internal/pkg/consts"
	"globe/dodrio_loan_availment/internal/pkg/db"
	"globe/dodrio_loan_availment/internal/pkg/logger"
	"globe/dodrio_loan_availment/internal/pkg/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type BrandRepository struct {
	repo        *MongoRepository[models.Brand]
	mappingRepo *MongoRepository[models.LoanProductsBrands]
}

func NewBrandRepository() *BrandRepository {
	collection := db.MDB.Database.Collection(consts.BrandCollection)
	brandsMappingCollection := db.MDB.Database.Collection(consts.LoanProductsBrandsCollection)
	mrepo := NewMongoRepository[models.Brand](collection)
	mappingRepo := NewMongoRepository[models.LoanProductsBrands](brandsMappingCollection)
	return &BrandRepository{repo: mrepo, mappingRepo: mappingRepo}
}

func (r *BrandRepository) BrandsByFilter(filter interface{}) (*models.Brand, error) {

	result, err := r.repo.Read(filter)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *BrandRepository) BrandsByProductId(ctx context.Context,productId primitive.ObjectID) (string, error) {

	if r.mappingRepo == nil {
		return "", fmt.Errorf("brand mapping repository is not initialized")
	}
	brandPipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "productId", Value: productId}}}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: consts.BrandCollection},
			{Key: "localField", Value: "brandId"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "brands"},
		}}},
		{{Key: "$unwind", Value: "$brands"}},
		{{Key: "$project", Value: bson.D{
			{Key: "_id", Value: "$brands._id"},
			{Key: "code", Value: "$brands.code"},
			{Key: "isDeleted", Value: "$brands.isDeleted"},
		}}},
	}
	var brandResult struct {
		Id        primitive.ObjectID `bson:"_id"`
		Code      string             `bson:"code"`
		IsDeleted bool               `bson:"isDeleted"`
	}
	logger.Debug(ctx,r.mappingRepo)
	if err := r.mappingRepo.Aggregate(brandPipeline, &brandResult); err != nil {
		if err == mongo.ErrNoDocuments {
			logger.Warn(ctx,"No brandResult found")
		} else {
			logger.Error(ctx,err)
		}
		return "", err
	}
	return brandResult.Code, nil
}
