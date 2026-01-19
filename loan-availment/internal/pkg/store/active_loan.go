package store

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"

	"globe/dodrio_loan_availment/internal/pkg/consts"
	"globe/dodrio_loan_availment/internal/pkg/db"
	"globe/dodrio_loan_availment/internal/pkg/logger"
	"globe/dodrio_loan_availment/internal/pkg/models"
)

type ActiveLoanRepository struct {
	repo *MongoRepository[models.Loans]
}

func NewActiveLoanRepository() *ActiveLoanRepository {
	collection := db.MDB.Database.Collection(consts.LoansCollection)
	mrepo := NewMongoRepository[models.Loans](collection)
	return &ActiveLoanRepository{repo: mrepo}
}

func (r *ActiveLoanRepository) IsActiveLoan(ctx context.Context,msisdn string) (bool, *models.Loans, error) {

	filter := bson.M{"MSISDN": msisdn}

	loans, err := r.repo.Read(filter)
	if err != nil {
		return false, nil, err
	}

	return true, &loans, nil
}

func (r *ActiveLoanRepository) ActiveLoanByFilter(filter interface{}) (*models.Loans, error) {

	result, err := r.repo.Read(filter)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *ActiveLoanRepository) CreateActiveLoanEntry(ctx context.Context,loansDB models.Loans) (bool, error) {

	_, err := r.repo.Create(loansDB)
	if err != nil {
		logger.Error(ctx,"ActiveLoan : Error while inserting %v", err)
		return false, fmt.Errorf("ActiveLoan : error while inserting %v", err.Error())
	}

	return true, nil
}
