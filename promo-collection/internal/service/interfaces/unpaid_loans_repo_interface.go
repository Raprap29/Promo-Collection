package interfaces

import (
	"context"
	"promocollection/internal/pkg/store/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UnpaidLoansRepositoryInterface interface {
	GetUnpaidLoansByLoanId(ctx context.Context, loanId primitive.ObjectID) (*models.UnpaidLoans, error)
}

type UnpaidLoansStoreInterface interface {
	FindOne(ctx context.Context, filter interface{}, opt *options.FindOneOptions) (models.UnpaidLoans, error)
	Find(ctx context.Context, filter interface{}) ([]models.UnpaidLoans, error)
	Delete(ctx context.Context, filter interface{}) error
}
