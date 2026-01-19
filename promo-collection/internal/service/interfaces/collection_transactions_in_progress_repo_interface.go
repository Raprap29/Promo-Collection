package interfaces

import (
	"context"
	"promocollection/internal/pkg/store/models"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CollectionTransactionsInProgressRepositoryInterface interface {
	CheckEntryExists(ctx context.Context, msisdn string) (bool, error)
	CreateEntry(ctx context.Context, msisdn string) error
	DeleteEntry(ctx context.Context, msisdn string) error
}

type CollectionTransactionsInProgressStoreInterface interface {
	FindOne(ctx context.Context, filter interface{}, opt *options.FindOneOptions) (
		models.CollectionTransactionsInProgress, error)
	Create(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error)
	Delete(ctx context.Context, filter interface{}) error
}
