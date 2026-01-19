package redis

import (
	"context"

	goredis "github.com/redis/go-redis/v9"
)

type RedisConnector interface {
	Connect(ctx context.Context, opts *goredis.Options) (*goredis.Client, error)
	Ping(ctx context.Context, client *goredis.Client) error
}

type DefaultRedisConnector struct{}

func (d *DefaultRedisConnector) Connect(ctx context.Context, opts *goredis.Options) (*goredis.Client, error) {
	client := goredis.NewClient(opts)
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return client, nil
}

func (d *DefaultRedisConnector) Ping(ctx context.Context, client *goredis.Client) error {
	return client.Ping(ctx).Err()
}
