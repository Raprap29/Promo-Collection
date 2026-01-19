package loan_products

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

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type LoanProductsRepository struct {
	repo interfaces.LoanProductsStoreInterface
}

func NewLoanProductsRepository(client *mongodb.MongoClient) *LoanProductsRepository {
	collection := client.Database.Collection(consts.LoanProductsCollection)
	repo := repository.NewMongoRepository[models.LoanProducts](collection)
	return &LoanProductsRepository{repo: repo}
}

func NewLoanProductsRepositoryWithInterface(repo interfaces.LoanProductsStoreInterface) *LoanProductsRepository {
	return &LoanProductsRepository{repo: repo}
}

func (lpr *LoanProductsRepository) GetProductNameById(ctx context.Context,
	id primitive.ObjectID) (*models.LoanProducts, error) {
	filter := bson.M{"_id": id}
	loanprods, err := lpr.repo.FindOne(ctx, filter, options.FindOne())
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			logger.CtxWarn(ctx, "No document found for loanProductId", slog.String("_id", id.String()))
			return nil, err
		}
		logger.CtxError(ctx, "Error finding document by loanProductId", err, slog.String("_id", id.String()))
		return nil, err
	}

	logger.CtxDebug(ctx, "Fetched document by loanProductId", slog.String("_id", id.String()))
	return &loanprods, nil
}
