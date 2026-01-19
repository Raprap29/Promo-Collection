package repository

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	redismodel "promocollection/internal/pkg/store/models"
)

func TestNewRedisStoreAdapter(t *testing.T) {
	db, mock := redismock.NewClientMock()

	adapter := NewRedisStoreAdapter(db)

	assert.NotNil(t, adapter)
	assert.Equal(t, db, adapter.client)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRedisStoreAdapter_Set(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		adapter := NewRedisStoreAdapter(db)
		ctx := context.Background()
		key := "test-key"
		value := "test-value"
		expiration := 5 * time.Minute

		mock.ExpectSet(key, value, expiration).SetVal("OK")

		err := adapter.Set(ctx, key, value, expiration)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Error", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		adapter := NewRedisStoreAdapter(db)
		ctx := context.Background()
		key := "test-key"
		value := "test-value"
		expiration := 5 * time.Minute

		mock.ExpectSet(key, value, expiration).SetErr(redis.Nil)

		err := adapter.Set(ctx, key, value, expiration)

		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRedisStoreAdapter_Get(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		adapter := NewRedisStoreAdapter(db)
		ctx := context.Background()
		key := "test-key"
		expectedValue := []byte("test-value")

		mock.ExpectGet(key).SetVal(string(expectedValue))

		result, err := adapter.Get(ctx, key)

		assert.NoError(t, err)
		assert.Equal(t, expectedValue, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Error", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		adapter := NewRedisStoreAdapter(db)
		ctx := context.Background()
		key := "test-key"

		mock.ExpectGet(key).SetErr(redis.Nil)

		result, err := adapter.Get(ctx, key)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRedisStoreAdapter_Delete(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		adapter := NewRedisStoreAdapter(db)
		ctx := context.Background()
		key := "test-key"

		mock.ExpectDel(key).SetVal(1)

		err := adapter.Delete(ctx, key)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Error", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		adapter := NewRedisStoreAdapter(db)
		ctx := context.Background()
		key := "test-key"

		mock.ExpectDel(key).SetErr(redis.Nil)

		err := adapter.Delete(ctx, key)

		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRedisStoreAdapter_Exists(t *testing.T) {
	t.Run("Key exists", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		adapter := NewRedisStoreAdapter(db)
		ctx := context.Background()
		key := "test-key"

		mock.ExpectExists(key).SetVal(1)

		exists, err := adapter.Exists(ctx, key)

		assert.NoError(t, err)
		assert.True(t, exists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Key does not exist", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		adapter := NewRedisStoreAdapter(db)
		ctx := context.Background()
		key := "test-key"

		mock.ExpectExists(key).SetVal(0)

		exists, err := adapter.Exists(ctx, key)

		assert.NoError(t, err)
		assert.False(t, exists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Error", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		adapter := NewRedisStoreAdapter(db)
		ctx := context.Background()
		key := "test-key"

		mock.ExpectExists(key).SetErr(redis.Nil)

		exists, err := adapter.Exists(ctx, key)

		assert.Error(t, err)
		assert.False(t, exists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRedisStoreAdapter_Expire(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		adapter := NewRedisStoreAdapter(db)
		ctx := context.Background()
		key := "test-key"
		expiration := 5 * time.Minute

		mock.ExpectExpire(key, expiration).SetVal(true)

		result, err := adapter.Expire(ctx, key, expiration)

		assert.NoError(t, err)
		assert.True(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Error", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		adapter := NewRedisStoreAdapter(db)
		ctx := context.Background()
		key := "test-key"
		expiration := 5 * time.Minute

		mock.ExpectExpire(key, expiration).SetErr(redis.Nil)

		result, err := adapter.Expire(ctx, key, expiration)

		assert.Error(t, err)
		assert.False(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRedisStoreAdapter_TTL(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		adapter := NewRedisStoreAdapter(db)
		ctx := context.Background()
		key := "test-key"
		expectedTTL := 5 * time.Minute

		mock.ExpectTTL(key).SetVal(expectedTTL)

		result, err := adapter.TTL(ctx, key)

		assert.NoError(t, err)
		assert.Equal(t, expectedTTL, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Error", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		adapter := NewRedisStoreAdapter(db)
		ctx := context.Background()
		key := "test-key"

		mock.ExpectTTL(key).SetErr(redis.Nil)

		result, err := adapter.TTL(ctx, key)

		assert.Error(t, err)
		assert.Equal(t, time.Duration(0), result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRedisStoreAdapter_SaveSkipTimestamps(t *testing.T) {

	t.Run("Success with PromoEducationPeriodTimestamp later", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		adapter := NewRedisStoreAdapter(db)
		ctx := context.Background()
		msisdn := "639171234567"

		now := time.Now()
		entry := redismodel.SkipTimestamps{
			GracePeriodTimestamp:          now.Add(-1 * time.Hour),
			PromoEducationPeriodTimestamp: now.Add(2 * time.Hour),
			LoanLoadPeriodTimestamp:       now.Add(1 * time.Hour),
		}

		expectedKey := redismodel.SkipTimestampsKeyBuilder(msisdn)
		expectedTTL := time.Until(entry.PromoEducationPeriodTimestamp)

		data, _ := json.Marshal(entry)  
		mock.ExpectSet(expectedKey, data, expectedTTL).SetVal("OK") 

		err := adapter.SaveSkipTimestamps(ctx, msisdn, entry)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Success with LoanLoadPeriodTimestamp later", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		adapter := NewRedisStoreAdapter(db)
		ctx := context.Background()
		msisdn := "639171234567"

		now := time.Now()
		entry := redismodel.SkipTimestamps{
			GracePeriodTimestamp:          now.Add(-1 * time.Hour),
			PromoEducationPeriodTimestamp: now.Add(1 * time.Hour),
			LoanLoadPeriodTimestamp:       now.Add(2 * time.Hour),
		}

		expectedKey := redismodel.SkipTimestampsKeyBuilder(msisdn)
		expectedTTL := time.Until(entry.LoanLoadPeriodTimestamp)

		data, _ := json.Marshal(entry) 
		mock.ExpectSet(expectedKey, data, expectedTTL).SetVal("OK") 

		err := adapter.SaveSkipTimestamps(ctx, msisdn, entry)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Success with negative TTL (sets minimum TTL)", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		adapter := NewRedisStoreAdapter(db)
		ctx := context.Background()
		msisdn := "639171234567"

		now := time.Now()
		entry := redismodel.SkipTimestamps{
			GracePeriodTimestamp:          now.Add(1 * time.Hour),
			PromoEducationPeriodTimestamp: now.Add(-1 * time.Hour),
			LoanLoadPeriodTimestamp:       now.Add(-2 * time.Hour),
		}

		expectedKey := redismodel.SkipTimestampsKeyBuilder(msisdn)
		expectedTTL := time.Second

		data, _ := json.Marshal(entry) 
		mock.ExpectSet(expectedKey, data, expectedTTL).SetVal("OK") 

		err := adapter.SaveSkipTimestamps(ctx, msisdn, entry)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Error from Set", func(t *testing.T) {
		db, mock := redismock.NewClientMock()
		adapter := NewRedisStoreAdapter(db)
		ctx := context.Background()
		msisdn := "639171234567"

		now := time.Now()
		entry := redismodel.SkipTimestamps{
			GracePeriodTimestamp:          now.Add(-1 * time.Hour),
			PromoEducationPeriodTimestamp: now.Add(2 * time.Hour),
			LoanLoadPeriodTimestamp:       now.Add(1 * time.Hour),
		}

		expectedKey := redismodel.SkipTimestampsKeyBuilder(msisdn)
		expectedTTL := time.Until(entry.PromoEducationPeriodTimestamp)

		data, _ := json.Marshal(entry) 
		mock.ExpectSet(expectedKey, data, expectedTTL).SetErr(redis.Nil) 

		err := adapter.SaveSkipTimestamps(ctx, msisdn, entry)

		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
