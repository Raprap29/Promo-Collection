package interfaces

import (
	"context"
	"promocollection/internal/pkg/store/models"

	"go.mongodb.org/mongo-driver/mongo/options"
)

type WhitelistedForDataCollectionRepositoryInterface interface {
	IsMSISDNWhitelisted(ctx context.Context, msisdn string) (bool, error)
	DeleteByMSISDN(ctx context.Context, msisdn string) error
}

type WhitelistedForDataCollectionStoreInterface interface {
	FindOne(ctx context.Context, filter interface{}, opt *options.FindOneOptions) (
		models.WhitelistedForDataCollection, error)
	Delete(ctx context.Context, filter interface{}) error
}
