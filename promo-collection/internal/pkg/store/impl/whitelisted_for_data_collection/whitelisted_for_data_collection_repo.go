package whitelisted_for_data_collection

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

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type WhitelistedForDataCollectionRepository struct {
	repo interfaces.WhitelistedForDataCollectionStoreInterface
}

func NewWhitelistedForDataCollectionRepository(client *mongodb.MongoClient) *WhitelistedForDataCollectionRepository {
	collection := client.Database.Collection(consts.WhitelistedForDataCollection)
	repo := repository.NewMongoRepository[models.WhitelistedForDataCollection](collection)
	return &WhitelistedForDataCollectionRepository{repo: repo}
}

func NewWhitelistedForDataCollectionWithRepository(
	repo interfaces.WhitelistedForDataCollectionStoreInterface) *WhitelistedForDataCollectionRepository {
	return &WhitelistedForDataCollectionRepository{repo: repo}
}

func (r *WhitelistedForDataCollectionRepository) IsMSISDNWhitelisted(ctx context.Context, msisdn string) (bool, error) {
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

func (r *WhitelistedForDataCollectionRepository) DeleteByMSISDN(ctx context.Context, msisdn string) error {
	filter := bson.M{"MSISDN": msisdn}

	err := r.repo.Delete(ctx, filter)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil
		}
		logger.CtxError(ctx, "Error deleting from whitelisted table", err, slog.String("msisdn", msisdn))
		return err
	}

	return nil
}
