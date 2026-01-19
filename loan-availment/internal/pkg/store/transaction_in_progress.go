package store

import (
	"context"
	"fmt"


	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"globe/dodrio_loan_availment/internal/pkg/consts"
	"globe/dodrio_loan_availment/internal/pkg/db"
	"globe/dodrio_loan_availment/internal/pkg/logger"
	"globe/dodrio_loan_availment/internal/pkg/models"
)

type TransactionInProgressRepository struct {
	repo *MongoRepository[models.TransactionInProgress]
}

func NewTransactionInProgressRepository() *TransactionInProgressRepository {
	collection := db.MDB.Database.Collection(consts.TransactionsInProgressCollection)
	mrepo := NewMongoRepository[models.TransactionInProgress](collection)
	return &TransactionInProgressRepository{repo: mrepo}
}

func (r *TransactionInProgressRepository) DeleteTransactionInProgressByMsisdn(ctx context.Context,msisdn string) (bool, error) {

	filter := bson.M{"MSISDN": msisdn}

	err := r.repo.Delete(filter)

	if err != nil {
		logger.Error(ctx,"TransactionIProgress : Error while deleting %v", err.Error())
		return false, fmt.Errorf("TransactionIProgress : error while deleting %v", err.Error())
	}

	return true, nil
}

func (r *TransactionInProgressRepository) CreateTransactionInProgressEntry(ctx context.Context,transactionInProgressDB models.TransactionInProgress) (bool, error) {

	_, err := r.repo.Create(transactionInProgressDB)

	if err != nil {
		logger.Error(ctx,"TransactionIProgress : Error while inserting %v", err.Error())
		return false, fmt.Errorf("TransactionIProgress : error while inserting %v", err.Error())
	}

	return true, nil
}

func (r *TransactionInProgressRepository) IsAvailmentTransactionInProgress(ctx context.Context,msisdn string) (bool, error) {

	filter := bson.M{"MSISDN": msisdn}

	_, err := r.repo.Read(filter)
	if err != nil {

		if err == mongo.ErrNoDocuments {
			// Log that no document was found
			logger.Warn(ctx,"No transaction in progress found for MSISDN: %v", msisdn)
			return false, nil // Return false to indicate no active transaction
		}
		logger.Error(ctx,"Error querying the database for IsAvailmentTransactionInProgress: %v", err)
		return false, err
	}

	return true, nil

}
