package unpaidloans

import (
	"context"
	"log/slog"
	"promo-collection-worker/internal/pkg/consts"
	mongodb "promo-collection-worker/internal/pkg/db/mongo"
	"promo-collection-worker/internal/pkg/log_messages"
	"promo-collection-worker/internal/pkg/logger"
	"promo-collection-worker/internal/pkg/store/models"
	"promo-collection-worker/internal/pkg/store/repository"
	"promo-collection-worker/internal/service/interfaces"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UnpaidLoansRepository struct {
	repo interfaces.UnpaidLoansStoreInterface
}

func NewUnpaidLoansRepository(client *mongodb.MongoClient) *UnpaidLoansRepository {
	collection := client.Database.Collection(consts.UnpaidLoansCollection)
	repo := repository.NewMongoRepository[models.UnpaidLoans](collection)
	return &UnpaidLoansRepository{repo: repo}
}

func NewUnpaidLoansRepositoryWithInterface(repo interfaces.UnpaidLoansStoreInterface) *UnpaidLoansRepository {
	return &UnpaidLoansRepository{repo: repo}
}

func (ur UnpaidLoansRepository) CreateUnpaidLoanEntry(ctx context.Context, loanEntry bson.M) error {
	result, err := ur.repo.Create(ctx, loanEntry)
	if err != nil {
		logger.CtxError(ctx, log_messages.ErrorCreatingUnpaidLoansMongoDbDocument, err)
		return err
	}
	logger.CtxInfo(ctx, log_messages.SuccessUnpaidLoansCreation, slog.Any("UnpaidLoansId", result.InsertedID))

	return nil
}

func (ur UnpaidLoansRepository) UpdatePreviousValidToField(ctx context.Context, loanId primitive.ObjectID) error {
	findOpts := options.FindOne().SetSort(bson.M{"_id": -1})
	latestDoc, err := ur.repo.FindOne(ctx, bson.M{"loanId": loanId}, findOpts)
	if err != nil {
		logger.CtxError(ctx, log_messages.ErrorFetchingLatestUnpaidLoansMongoDbDocument, err)
		return err
	}

	filter := bson.M{"_id": latestDoc.ID}
	update := bson.M{
		"$set": bson.M{
			"validTo": time.Now(),
		},
	}
	result, err := ur.repo.UpdateMany(ctx, filter, update)
	if err != nil {
		logger.CtxError(ctx, log_messages.ErrorUpdatingUnpaidLoansMongoDbDocument, err)
		return err
	}
	logger.CtxInfo(ctx, log_messages.SuccessUpdatedUnpaidLoansDocument, slog.Any("unpaidLoansId", result.UpsertedID))

	return nil
}
