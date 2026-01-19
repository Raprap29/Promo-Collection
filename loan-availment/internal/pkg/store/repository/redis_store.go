package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStoreAdapter struct {
	client *redis.Client
}

func NewRedisStoreAdapter(client *redis.Client) *RedisStoreAdapter {
	return &RedisStoreAdapter{client: client}
}

// Implements RedisStoreOperations
func (a *RedisStoreAdapter) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return a.client.Set(ctx, key, value, expiration).Err()
}

func (a *RedisStoreAdapter) Get(ctx context.Context, key string) (interface{}, error) {
	return a.client.Get(ctx, key).Bytes()
}

func (a *RedisStoreAdapter) Delete(ctx context.Context, key string) error {
	return a.client.Del(ctx, key).Err()
}

func (a *RedisStoreAdapter) Exists(ctx context.Context, key string) (bool, error) {
	val, err := a.client.Exists(ctx, key).Result()
	return val > 0, err
}

func (a *RedisStoreAdapter) Expire(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	return a.client.Expire(ctx, key, expiration).Result()
}

func (a *RedisStoreAdapter) TTL(ctx context.Context, key string) (time.Duration, error) {
	return a.client.TTL(ctx, key).Result()
}
