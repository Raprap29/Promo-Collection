package whitelisted_for_data_collection

import (
	"context"
	"errors"
	"log/slog"

	"promo-collection-worker/internal/pkg/consts"
	mongodb "promo-collection-worker/internal/pkg/db/mongo"
	"promo-collection-worker/internal/pkg/logger"
	"promo-collection-worker/internal/pkg/store/models"
	"promo-collection-worker/internal/pkg/store/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type WhitelistFinder interface {
	FindOne(ctx context.Context, filter interface{},
		opts *options.FindOneOptions) (models.WhitelistedForDataCollection, error)
}

type WhitelistedRepository struct {
	repo WhitelistFinder
}

func NewWhitelistedRepository(client *mongodb.MongoClient) *WhitelistedRepository {
	collection := client.Database.Collection(consts.WhitelistedForDataCollection)
	repo := repository.NewMongoRepository[models.WhitelistedForDataCollection](collection)
	return &WhitelistedRepository{repo: repo}
}

func (r *WhitelistedRepository) IsMSISDNWhitelisted(ctx context.Context, msisdn string) (bool, error) {
	filter := bson.M{"MSISDN": msisdn}
	_, err := r.repo.FindOne(ctx, filter, nil)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return false, nil
		}
		logger.CtxError(ctx, "Error checking whitelisted table", err, slog.String("msisdn", msisdn))
		return false, err
	}
	return true, nil
}
