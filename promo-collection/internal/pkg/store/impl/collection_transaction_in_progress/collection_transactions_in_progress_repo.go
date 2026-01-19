package collection_transaction_in_progress

import (
	"context"
	"errors"
	"log/slog"
	"promocollection/internal/pkg/consts"
	mongodb "promocollection/internal/pkg/db/mongo"
	"promocollection/internal/pkg/logger"
	"promocollection/internal/pkg/store/models"
	"promocollection/internal/pkg/store/repository"
	"promocollection/internal/service/interfaces"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CollectionTransactionsInProgressRepository struct {
	repo interfaces.CollectionTransactionsInProgressStoreInterface
}

func NewCollectionTransactionsInProgressRepository(
	client *mongodb.MongoClient) *CollectionTransactionsInProgressRepository {
	collection := client.Database.Collection(consts.CollectionTransactionInProgressCollection)
	repo := repository.NewMongoRepository[models.CollectionTransactionsInProgress](collection)
	return &CollectionTransactionsInProgressRepository{repo: repo}
}

func NewCollectionTransactionsInProgressRepositoryWithInterface(
	repo interfaces.CollectionTransactionsInProgressStoreInterface) *CollectionTransactionsInProgressRepository {
	return &CollectionTransactionsInProgressRepository{repo: repo}
}

func (r *CollectionTransactionsInProgressRepository) CheckEntryExists(ctx context.Context,
	msisdn string) (bool, error) {
	filter := bson.M{"MSISDN": msisdn}
	_, err := r.repo.FindOne(ctx, filter, options.FindOne())
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return false, nil
		}
		logger.CtxError(ctx, "Error checking collection in progress", err, slog.String("msisdn", msisdn))
		return false, err
	}
	return true, nil
}

func (r *CollectionTransactionsInProgressRepository) CreateEntry(ctx context.Context,
	msisdn string) error {
	entry := models.CollectionTransactionsInProgress{
		MSISDN:    msisdn,
		CreatedAt: time.Now(),
	}
	_, err := r.repo.Create(ctx, entry)
	if err != nil {
		logger.CtxError(ctx, "Failed to create collection in progress entry", err, slog.String("msisdn", msisdn))
		return err
	}
	logger.CtxInfo(ctx, "Created collection in progress entry", slog.String("msisdn", msisdn))
	return nil
}

func (r *CollectionTransactionsInProgressRepository) DeleteEntry(ctx context.Context, msisdn string) error {
	filter := bson.M{"MSISDN": msisdn}
	err := r.repo.Delete(ctx, filter)
	if err != nil {
		logger.CtxError(ctx, "Failed to delete from collection table", err, slog.String("msisdn", msisdn))
		return err
	}
	logger.CtxInfo(ctx, "Successfully deleted entry from collection table", slog.String("msisdn", msisdn))
	return nil
}
