package interfaces

import (
	"context"
	"promo-collection-worker/internal/pkg/store/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ClosedLoansRepositoryInterface interface {
	CreateClosedLoansEntry(ctx context.Context, entry bson.M) error
}

type ClosedLoansStoreInterface interface {
	Create(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error)
	FindOne(ctx context.Context, filter interface{}, opt *options.FindOneOptions) (models.ClosedLoans, error)
	Find(ctx context.Context, filter interface{}) ([]models.ClosedLoans, error)
}
