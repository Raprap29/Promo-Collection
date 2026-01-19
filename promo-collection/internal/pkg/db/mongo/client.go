package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoConnector interface {
	Connect(ctx context.Context, opts *options.ClientOptions) (*mongo.Client, error)
	Ping(ctx context.Context, client *mongo.Client) error
}

type DefaultMongoConnector struct{}

func (d *DefaultMongoConnector) Connect(ctx context.Context, opts *options.ClientOptions) (*mongo.Client, error) {
	return mongo.Connect(ctx, opts)
}

func (d *DefaultMongoConnector) Ping(ctx context.Context, client *mongo.Client) error {
	return client.Ping(ctx, nil)
}
