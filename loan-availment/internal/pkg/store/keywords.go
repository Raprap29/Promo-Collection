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

type KeywordRepository struct {
	repo *MongoRepository[models.Keyword]
}

func NewKeywordRepository() *KeywordRepository {
	collection := db.MDB.Database.Collection(consts.KeywordCollection)
	mrepo := NewMongoRepository[models.Keyword](collection)
	return &KeywordRepository{repo: mrepo}
}

func (r *KeywordRepository) KeywordByFilter(filter interface{}) (*models.Keyword, error) {

	result, err := r.repo.Read(filter)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *KeywordRepository) BrandCodeAndIdByKeyword(keywordName string) (string, primitive.ObjectID, bool, error) {

	brandPipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "name", Value: keywordName}}}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: consts.LoansProductsCollection},
			{Key: "localField", Value: "productId"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "loanProducts"},
		}}},
		{{Key: "$unwind", Value: "$loanProducts"}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: consts.ProductsNamesInAmaxCollection},
			{Key: "localField", Value: "loanProducts._id"},
			{Key: "foreignField", Value: "productId"},
			{Key: "as", Value: "productNamesInAmax"},
		}}},
		{{Key: "$unwind", Value: "$productNamesInAmax"}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: consts.BrandCollection},
			{Key: "localField", Value: "productNamesInAmax.brandId"},
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

	if err := r.repo.Aggregate(brandPipeline, &brandResult); err != nil {
		if err == mongo.ErrNoDocuments {
			logger.Warn("No brandResult found")
		} else {
			logger.Error(err)
		}
		return "", primitive.NilObjectID, brandResult.IsDeleted, err
	}

	return brandResult.Code, brandResult.Id, brandResult.IsDeleted, nil

}

func (r *KeywordRepository) LoanTypeByKeyword(keywordName string) (string, error) {

	loanTypePipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "name", Value: keywordName}}}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: consts.LoansProductsCollection},
			{Key: "localField", Value: "productId"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "loanProducts"},
		}}},
		{{Key: "$unwind", Value: "$loanProducts"}},
		{{Key: "$project", Value: bson.D{
			{Key: "_id", Value: 0},
			{Key: "loanType", Value: "$loanProducts.loanType"},
		}}},
	}

	var loanTypeResult struct {
		LoanType string `bson:"loanType"`
	}
	if err := r.repo.Aggregate(loanTypePipeline, &loanTypeResult); err != nil {
		if err == mongo.ErrNoDocuments {
			logger.Warn("No loanTypeResult found")
		} else {
			logger.Error(err)
		}
		return "", err
	}

	return loanTypeResult.LoanType, nil

}
func (r *KeywordRepository) ProductNamesInAmaxByKeyword(keywordName string, brandId primitive.ObjectID) (string, error) {
	productNamesInAmaxPipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "name", Value: keywordName}}}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: consts.ProductsNamesInAmaxCollection},
			{Key: "localField", Value: "productId"},
			{Key: "foreignField", Value: "productId"},
			{Key: "as", Value: "productsNamesInAmax"},
		}}}, {{Key: "$match", Value: bson.D{
			{Key: "$and", Value: bson.A{
				bson.D{{Key: "productsNamesInAmax.brandId", Value: brandId}},
				bson.D{{Key: "productsNamesInAmax.isDeleted", Value: bson.D{{Key: "$ne", Value: true}}}},
			}},
		}}},
		{{Key: "$unwind", Value: "$productsNamesInAmax"}},
		{{Key: "$project", Value: bson.D{
			{Key: "_id", Value: 0},
			{Key: "name", Value: "$productsNamesInAmax.name"},
			{Key: "isDeleted", Value: "$productsNamesInAmax.isDeleted"},
			{Key: "brandId", Value: "$productsNamesInAmax.brandId"},
		}}},
	}
	var productsNamesInAmaxName struct {
		Name      string             `bson:"name"`
		IsDeleted bool               `bson:"isDeleted"`
		BrandId   primitive.ObjectID `bson:"brandId"`
	}
	if err := r.repo.Aggregate(productNamesInAmaxPipeline, &productsNamesInAmaxName); err != nil {
		if err == mongo.ErrNoDocuments {
			return "", consts.NOProductNameInAmax
		} else {
			logger.Error(err)
		}
		return "", err
	}
	return productsNamesInAmaxName.Name, nil
}
