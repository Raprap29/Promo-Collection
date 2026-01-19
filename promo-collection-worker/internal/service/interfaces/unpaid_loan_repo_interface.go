package interfaces

import (
	"context"
	"promo-collection-worker/internal/pkg/store/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UnpaidLoansRepositoryInterface interface {
	CreateUnpaidLoanEntry(ctx context.Context, loanEntry bson.M) error
	UpdatePreviousValidToField(ctx context.Context, loanId primitive.ObjectID) error
}

type UnpaidLoansStoreInterface interface {
	Create(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error)
	FindOne(ctx context.Context, filter interface{}, opt *options.FindOneOptions) (models.UnpaidLoans, error)
	Find(ctx context.Context, filter interface{}) ([]models.UnpaidLoans, error)
	Delete(ctx context.Context, filter interface{}) error
	UpdateMany(ctx context.Context, filter interface{}, update interface{}) (*mongo.UpdateResult, error)
}
