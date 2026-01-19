package interfaces

import (
	"context"
	"promo-collection-worker/internal/pkg/store/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MessagesRepositoryInterface interface {
	GetPatternIdByEventandBrandId(ctx context.Context, event string, brandId primitive.ObjectID) (*models.Messages, error)
}

type MessagesStoreInterface interface {
	FindOne(ctx context.Context, filter interface{}, opt *options.FindOneOptions) (models.Messages, error)
	Find(ctx context.Context, filter interface{}) ([]models.Messages, error)
	Delete(ctx context.Context, filter interface{}) error
}
