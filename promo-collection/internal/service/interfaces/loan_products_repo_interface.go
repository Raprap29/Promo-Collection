package interfaces

import (
	"context"
	"promocollection/internal/pkg/store/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type LoanProductsRepositoryInterface interface {
	GetProductNameById(ctx context.Context, id primitive.ObjectID) (*models.LoanProducts, error)
}

type LoanProductsStoreInterface interface {
	FindOne(ctx context.Context, filter interface{}, opt *options.FindOneOptions) (models.LoanProducts, error)
	Find(ctx context.Context, filter interface{}) ([]models.LoanProducts, error)
	Delete(ctx context.Context, filter interface{}) error
}
