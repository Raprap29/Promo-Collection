package store

import (
	"context"
	"globe/dodrio_loan_availment/internal/pkg/consts"
	"globe/dodrio_loan_availment/internal/pkg/db"
	"globe/dodrio_loan_availment/internal/pkg/logger"
	"globe/dodrio_loan_availment/internal/pkg/models"
)

type WalletTypesRepository struct {
	repo *MongoRepository[models.WalletTypes]
}

func NewWalletTypesRepository() *WalletTypesRepository {
	collection := db.MDB.Database.Collection(consts.WalletTypes)
	mrepo := NewMongoRepository[models.WalletTypes](collection)
	return &WalletTypesRepository{repo: mrepo}
}

func (r *WalletTypesRepository) WalletTypesByFilter(ctx context.Context, filter interface{}) ([]models.WalletTypes, error) {

	logger.Debug(ctx, filter)
	result, err := r.repo.FindAll(filter)
	if err != nil {
		return nil, err
	}

	return result, nil

}
