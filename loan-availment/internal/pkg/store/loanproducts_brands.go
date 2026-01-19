package store

import (
	"context"
	"globe/dodrio_loan_availment/internal/pkg/consts"
	"globe/dodrio_loan_availment/internal/pkg/db"
	"globe/dodrio_loan_availment/internal/pkg/logger"
	"globe/dodrio_loan_availment/internal/pkg/models"
)

type LoanProductsBrandsRepository struct {
	repo *MongoRepository[models.LoanProductsBrands]
}

func NewLoanProductsBrandsRepository() *LoanProductsBrandsRepository {
	collection := db.MDB.Database.Collection(consts.LoanProductsBrandsCollection)
	mrepo := NewMongoRepository[models.LoanProductsBrands](collection)
	return &LoanProductsBrandsRepository{repo: mrepo}
}

func (r *LoanProductsBrandsRepository) LoanProductsBrandsByFilter(ctx context.Context,filter interface{}) (*models.LoanProductsBrands, error) {

	result, err := r.repo.Read(filter)
	if err != nil {
		logger.Error(ctx,"error inside LoanProductsBrandsByFilter: %v", err)
		return nil, err
	}

	return &result, nil

}
