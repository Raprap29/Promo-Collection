package store

import (
	"context"
	"fmt"
	"globe/dodrio_loan_availment/internal/pkg/consts"
	"globe/dodrio_loan_availment/internal/pkg/db"
	"globe/dodrio_loan_availment/internal/pkg/logger"
	"globe/dodrio_loan_availment/internal/pkg/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UnpaidLoanRepository struct {
	repo *MongoRepository[models.UnpaidLoans]
}

func NewUnpaidLoanRepository() *UnpaidLoanRepository {
	collection := db.MDB.Database.Collection(consts.UnpaidLoanCollection)
	mrepo := NewMongoRepository[models.UnpaidLoans](collection)
	return &UnpaidLoanRepository{repo: mrepo}
}

func (r *UnpaidLoanRepository) CreateUnpaidLoansEntry(ctx context.Context,unpaidLoansDB models.UnpaidLoans) (bool, error) {

	_, err := r.repo.Create(unpaidLoansDB)

	if err != nil {
		logger.Error(ctx,"UnpaidLoans : Error while inserting %v", err.Error())
		return false, fmt.Errorf("UnpaidLoans : error while inserting %v", err.Error())
	}

	return true, nil

}

func (r *UnpaidLoanRepository) UnpaidLoansByFilter(filter interface{}) (*models.UnpaidLoans, error) {
	var unpaidLoansResult models.UnpaidLoans
	opts := options.FindOne().SetSort(bson.D{{"version", -1}})
	err := r.repo.collection.FindOne(context.TODO(), filter, opts).Decode(&unpaidLoansResult)
	if err != nil {
		return nil, err
	}

	return &unpaidLoansResult, nil
}
