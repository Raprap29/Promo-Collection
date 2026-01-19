package store

import (
	"context"
	"globe/dodrio_loan_availment/internal/pkg/consts"
	"globe/dodrio_loan_availment/internal/pkg/db"
	"globe/dodrio_loan_availment/internal/pkg/logger"
	"globe/dodrio_loan_availment/internal/pkg/models"

	"go.mongodb.org/mongo-driver/mongo"
)

type ProductNameInAmaxRepository struct {
	repo *MongoRepository[models.ProductsNamesInAmax]
}

func NewProductNameInAmaxsRepository() *ProductNameInAmaxRepository {
	collection := db.MDB.Database.Collection(consts.ProductsNamesInAmaxCollection)
	mrepo := NewMongoRepository[models.ProductsNamesInAmax](collection)
	return &ProductNameInAmaxRepository{repo: mrepo}
}

func (r *ProductNameInAmaxRepository) ProductNameInAmaxsByFilter(ctx context.Context, filter interface{}) (*models.ProductsNamesInAmax, error) {

	logger.Debug(ctx, filter)
	result, err := r.repo.Read(filter)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, consts.NOProductNameInAmax
		} else {
			logger.Error(err)
		}
		return nil, err
	}

	return &result, nil

}
