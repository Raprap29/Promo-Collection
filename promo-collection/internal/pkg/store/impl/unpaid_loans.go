package impl

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
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UnpaidLoanRepository struct {
	repo interfaces.UnpaidLoansStoreInterface
}

func NewUnpaidLoansRepository(client *mongodb.MongoClient) *UnpaidLoanRepository {
	collection := client.Database.Collection(consts.UnpaidLoansCollection)
	repo := repository.NewMongoRepository[models.UnpaidLoans](collection)
	return &UnpaidLoanRepository{repo: repo}
}

func NewUnpaidLoansRepositoryWithInterface(repo interfaces.UnpaidLoansStoreInterface) *UnpaidLoanRepository {
	return &UnpaidLoanRepository{repo: repo}
}

func (ur *UnpaidLoanRepository) GetUnpaidLoansByLoanId(
	ctx context.Context, loanId primitive.ObjectID) (*models.UnpaidLoans, error) {
	var unpaidLoans models.UnpaidLoans

	filter := bson.M{
		"loanId":  loanId,
		"validTo":  primitive.Null{},
	}

	unpaidLoans, err := ur.repo.FindOne(ctx, filter, options.FindOne())
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			logger.CtxWarn(ctx, "No unpaid loan found for loanId", slog.String("loanId", loanId.String()))
			return &models.UnpaidLoans{}, err
		}
		logger.CtxError(ctx, "Error finding unpaid loan by loanId", err, slog.String("loanId", loanId.String()))
		return &models.UnpaidLoans{}, err
	}

	logger.CtxDebug(ctx, "Fetched unpaid loan by loanId", 
		slog.String("loanId", loanId.String()), 
		slog.Any("unpaid_id", unpaidLoans.ID.Hex()))
	return &unpaidLoans, nil
}
