package interfaces

import (
	"context"
	"time"
	redismodel "promo-collection-worker/internal/pkg/store/models"

)

type RedisStoreOperations interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string) (interface{}, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	Expire(ctx context.Context, key string, expiration time.Duration) (bool, error)
	TTL(ctx context.Context, key string) (time.Duration, error)
	SaveSkipTimestamps(
		ctx context.Context,
		msisdn string,
		entry redismodel.SkipTimestamps,
	) error
}
