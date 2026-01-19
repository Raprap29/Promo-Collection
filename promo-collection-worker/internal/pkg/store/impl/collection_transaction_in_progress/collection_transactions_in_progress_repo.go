package collection_transaction_in_progress

import (
	"context"
	"errors"
	"log/slog"
	"promo-collection-worker/internal/pkg/consts"
	mongodb "promo-collection-worker/internal/pkg/db/mongo"
	"promo-collection-worker/internal/pkg/logger"
	"promo-collection-worker/internal/pkg/store/models"
	"promo-collection-worker/internal/pkg/store/repository"
	"promo-collection-worker/internal/service/interfaces"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// CollectionTransactionsInProgressRepository implements the CollectionTransactionsInProgressRepoInterface
type CollectionTransactionsInProgressRepository struct {
	repo    *repository.MongoRepository[models.CollectionTransactionsInProgress]
	findOne func(ctx context.Context,
		filter interface{}, opts *options.FindOneOptions) (models.CollectionTransactionsInProgress, error)
	create func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error)
	delete func(ctx context.Context, filter interface{}) error
}

// Ensure CollectionTransactionsInProgressRepository implements the CollectionTransactionsInProgressRepoInterface
var _ interfaces.CollectionTransactionsInProgressRepoInterface = (*CollectionTransactionsInProgressRepository)(nil)

func NewCollectionTransactionsInProgressRepository(
	client *mongodb.MongoClient) interfaces.CollectionTransactionsInProgressRepoInterface {
	collection := client.Database.Collection(consts.CollectionTransactionInProgressCollection)
	repo := repository.NewMongoRepository[models.CollectionTransactionsInProgress](collection)
	r := &CollectionTransactionsInProgressRepository{
		repo: repo,
	}
	r.findOne = repo.FindOne
	r.create = repo.Create
	r.delete = repo.Delete
	return r
}

func (r *CollectionTransactionsInProgressRepository) CheckEntryExists(ctx context.Context,
	msisdn string) (bool, error) {
	filter := bson.M{"MSISDN": msisdn}
	_, err := r.findOne(ctx, filter, options.FindOne())
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
	_, err := r.create(ctx, entry)
	if err != nil {
		logger.CtxError(ctx, "Failed to create collection in progress entry", err, slog.String("msisdn", msisdn))
		return err
	}
	logger.CtxInfo(ctx, "Created collection in progress entry", slog.String("msisdn", msisdn))
	return nil
}

func (r *CollectionTransactionsInProgressRepository) DeleteEntry(ctx context.Context, msisdn string) error {
	filter := bson.M{"MSISDN": msisdn}
	err := r.delete(ctx, filter)
	if err != nil {
		logger.CtxError(ctx, "Failed to delete from collection table", err, slog.String("msisdn", msisdn))
		return err
	}
	logger.CtxInfo(ctx, "Successfully deleted entry from collection transaction in progress table",
		slog.String("msisdn", msisdn))
	return nil
}
