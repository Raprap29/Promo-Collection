package closedloans

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

	"go.mongodb.org/mongo-driver/bson"
)

type ClosedLoansRepository struct {
	repo interfaces.ClosedLoansStoreInterface
}

func NewClosedLoansRepository(client *mongodb.MongoClient) *ClosedLoansRepository {
	collection := client.Database.Collection(consts.ClosedLoansCollection)
	repo := repository.NewMongoRepository[models.ClosedLoans](collection)
	return &ClosedLoansRepository{repo: repo}
}

func NewClosedLoansRepositoryWithInterface(repo interfaces.ClosedLoansStoreInterface) *ClosedLoansRepository {
	return &ClosedLoansRepository{repo: repo}
}

func (cl *ClosedLoansRepository) CreateClosedLoansEntry(ctx context.Context, entry bson.M) error {
	result, err := cl.repo.Create(ctx, entry)
	if err != nil {
		logger.CtxError(ctx, log_messages.ErrorCreatingClosedLoansMongoDbDocument, err)
		return err
	}
	logger.CtxInfo(ctx, log_messages.SuccessClosedLoansCreation, slog.Any("ClosedLoansId", result.InsertedID))

	return nil
}
