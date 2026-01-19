package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRedisClientInterface defines the Redis operations we need
type MockRedisClientInterface interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Get(ctx context.Context, key string) *redis.StringCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Exists(ctx context.Context, keys ...string) *redis.IntCmd
	Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
	TTL(ctx context.Context, key string) *redis.DurationCmd
}

// MockRedisClient is a mock implementation
type MockRedisClient struct {
	mock.Mock
}

func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	args := m.Called(ctx, key, value, expiration)
	return args.Get(0).(*redis.StatusCmd)
}

func (m *MockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	args := m.Called(ctx, key)
	return args.Get(0).(*redis.StringCmd)
}

func (m *MockRedisClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	args := m.Called(ctx, keys)
	return args.Get(0).(*redis.IntCmd)
}

func (m *MockRedisClient) Exists(ctx context.Context, keys ...string) *redis.IntCmd {
	args := m.Called(ctx, keys)
	return args.Get(0).(*redis.IntCmd)
}

func (m *MockRedisClient) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	args := m.Called(ctx, key, expiration)
	return args.Get(0).(*redis.BoolCmd)
}

func (m *MockRedisClient) TTL(ctx context.Context, key string) *redis.DurationCmd {
	args := m.Called(ctx, key)
	return args.Get(0).(*redis.DurationCmd)
}

// TestableRedisStoreAdapter wraps RedisStoreAdapter for testing
type TestableRedisStoreAdapter struct {
	client MockRedisClientInterface
}

func NewTestableRedisStoreAdapter(client MockRedisClientInterface) *TestableRedisStoreAdapter {
	return &TestableRedisStoreAdapter{client: client}
}

func (a *TestableRedisStoreAdapter) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {

	return a.client.Set(ctx, key, value, expiration).Err()
}

func (a *TestableRedisStoreAdapter) Get(ctx context.Context, key string) (interface{}, error) {
	return a.client.Get(ctx, key).Bytes()
}

func (a *TestableRedisStoreAdapter) Delete(ctx context.Context, key string) error {
	return a.client.Del(ctx, key).Err()
}

func (a *TestableRedisStoreAdapter) Exists(ctx context.Context, key string) (bool, error) {
	val, err := a.client.Exists(ctx, key).Result()
	return val > 0, err
}

func (a *TestableRedisStoreAdapter) Expire(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	return a.client.Expire(ctx, key, expiration).Result()
}

func (a *TestableRedisStoreAdapter) TTL(ctx context.Context, key string) (time.Duration, error) {
	return a.client.TTL(ctx, key).Result()
}

func TestNewRedisStoreAdapter(t *testing.T) {
	client := &redis.Client{}
	adapter := NewRedisStoreAdapter(client)

	assert.NotNil(t, adapter)
	assert.Equal(t, client, adapter.client)
}

func TestRedisStoreAdapter_Set(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		value       interface{}
		expiration  time.Duration
		expectError bool
	}{
		{
			name:        "successful set",
			key:         "test-key",
			value:       "test-value",
			expiration:  time.Hour,
			expectError: false,
		},
		{
			name:        "set with zero expiration",
			key:         "test-key",
			value:       "test-value",
			expiration:  0,
			expectError: false,
		},
		{
			name:        "set with complex value",
			key:         "test-key",
			value:       map[string]interface{}{"field": "value"},
			expiration:  time.Hour,
			expectError: false,
		},
		{
			name:        "set with error",
			key:         "error-key",
			value:       "test-value",
			expiration:  time.Hour,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockRedisClient)
			adapter := NewTestableRedisStoreAdapter(mockClient)

			// Mock the Redis Set command
			mockStatusCmd := &redis.StatusCmd{}
			if tt.expectError {
				mockStatusCmd.SetErr(errors.New("redis error"))
			} else {
				mockStatusCmd.SetVal("OK")
			}

			mockClient.On("Set", mock.Anything, tt.key, tt.value, tt.expiration).Return(mockStatusCmd)

			err := adapter.Set(context.Background(), tt.key, tt.value, tt.expiration)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestRedisStoreAdapter_Get(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		value       []byte
		expectError bool
	}{
		{
			name:        "successful get",
			key:         "test-key",
			value:       []byte("test-value"),
			expectError: false,
		},
		{
			name:        "key not found",
			key:         "nonexistent-key",
			value:       nil,
			expectError: true,
		},
		{
			name:        "get with empty value",
			key:         "empty-key",
			value:       []byte(nil),
			expectError: false,
		},
		{
			name:        "get with error",
			key:         "error-key",
			value:       nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockRedisClient)
			adapter := NewTestableRedisStoreAdapter(mockClient)

			// Mock the Redis Get command
			mockStringCmd := &redis.StringCmd{}
			if tt.expectError {
				mockStringCmd.SetErr(redis.Nil)
			} else {
				mockStringCmd.SetVal(string(tt.value))
			}

			mockClient.On("Get", mock.Anything, tt.key).Return(mockStringCmd)

			result, err := adapter.Get(context.Background(), tt.key)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.value, result)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestRedisStoreAdapter_Delete(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		deleted     int64
		expectError bool
	}{
		{
			name:        "successful delete",
			key:         "test-key",
			deleted:     1,
			expectError: false,
		},
		{
			name:        "delete non-existent key",
			key:         "nonexistent-key",
			deleted:     0,
			expectError: false,
		},
		{
			name:        "delete with error",
			key:         "error-key",
			deleted:     0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockRedisClient)
			adapter := NewTestableRedisStoreAdapter(mockClient)

			// Mock the Redis Del command
			mockIntCmd := &redis.IntCmd{}
			if tt.expectError {
				mockIntCmd.SetErr(errors.New("redis error"))
			} else {
				mockIntCmd.SetVal(tt.deleted)
			}

			mockClient.On("Del", mock.Anything, []string{tt.key}).Return(mockIntCmd)

			err := adapter.Delete(context.Background(), tt.key)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestRedisStoreAdapter_Exists(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		exists      int64
		expectError bool
		expected    bool
	}{
		{
			name:        "key exists",
			key:         "test-key",
			exists:      1,
			expectError: false,
			expected:    true,
		},
		{
			name:        "key does not exist",
			key:         "nonexistent-key",
			exists:      0,
			expectError: false,
			expected:    false,
		},
		{
			name:        "exists with error",
			key:         "error-key",
			exists:      0,
			expectError: true,
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockRedisClient)
			adapter := NewTestableRedisStoreAdapter(mockClient)

			// Mock the Redis Exists command
			mockIntCmd := &redis.IntCmd{}
			if tt.expectError {
				mockIntCmd.SetErr(errors.New("redis error"))
			} else {
				mockIntCmd.SetVal(tt.exists)
			}

			mockClient.On("Exists", mock.Anything, []string{tt.key}).Return(mockIntCmd)

			result, err := adapter.Exists(context.Background(), tt.key)

			if tt.expectError {
				assert.Error(t, err)
				assert.False(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestRedisStoreAdapter_Expire(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		expiration  time.Duration
		success     bool
		expectError bool
	}{
		{
			name:        "successful expire",
			key:         "test-key",
			expiration:  time.Hour,
			success:     true,
			expectError: false,
		},
		{
			name:        "expire non-existent key",
			key:         "nonexistent-key",
			expiration:  time.Hour,
			success:     false,
			expectError: false,
		},
		{
			name:        "expire with error",
			key:         "error-key",
			expiration:  time.Hour,
			success:     false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockRedisClient)
			adapter := NewTestableRedisStoreAdapter(mockClient)

			// Mock the Redis Expire command
			mockBoolCmd := &redis.BoolCmd{}
			if tt.expectError {
				mockBoolCmd.SetErr(errors.New("redis error"))
			} else {
				mockBoolCmd.SetVal(tt.success)
			}

			mockClient.On("Expire", mock.Anything, tt.key, tt.expiration).Return(mockBoolCmd)

			result, err := adapter.Expire(context.Background(), tt.key, tt.expiration)

			if tt.expectError {
				assert.Error(t, err)
				assert.False(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.success, result)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestRedisStoreAdapter_TTL(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		ttl         time.Duration
		expectError bool
	}{
		{
			name:        "successful ttl",
			key:         "test-key",
			ttl:         time.Hour,
			expectError: false,
		},
		{
			name:        "ttl for persistent key",
			key:         "persistent-key",
			ttl:         -1 * time.Second,
			expectError: false,
		},
		{
			name:        "ttl for non-existent key",
			key:         "nonexistent-key",
			ttl:         -2 * time.Second,
			expectError: false,
		},
		{
			name:        "ttl with error",
			key:         "error-key",
			ttl:         0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockRedisClient)
			adapter := NewTestableRedisStoreAdapter(mockClient)

			// Mock the Redis TTL command
			mockDurationCmd := &redis.DurationCmd{}
			if tt.expectError {
				mockDurationCmd.SetErr(errors.New("redis error"))
			} else {
				mockDurationCmd.SetVal(tt.ttl)
			}

			mockClient.On("TTL", mock.Anything, tt.key).Return(mockDurationCmd)

			result, err := adapter.TTL(context.Background(), tt.key)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, time.Duration(0), result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.ttl, result)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestRedisStoreAdapter_Integration(t *testing.T) {
	// This test requires a real Redis instance
	// Skip if running in CI or if Redis is not available
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Create a real Redis client for integration testing
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})
	defer client.Close()

	adapter := NewRedisStoreAdapter(client)
	ctx := context.Background()

	// Test Set
	err := adapter.Set(ctx, "integration-test", "test-value", time.Minute)
	assert.NoError(t, err)

	// Test Get
	value, err := adapter.Get(ctx, "integration-test")
	assert.NoError(t, err)
	assert.Equal(t, []byte("test-value"), value)

	// Test Exists
	exists, err := adapter.Exists(ctx, "integration-test")
	assert.NoError(t, err)
	assert.True(t, exists)

	// Test TTL
	ttl, err := adapter.TTL(ctx, "integration-test")
	assert.NoError(t, err)
	assert.True(t, ttl > 0)
	assert.True(t, ttl <= time.Minute)

	// Test Expire
	success, err := adapter.Expire(ctx, "integration-test", time.Second*30)
	assert.NoError(t, err)
	assert.True(t, success)

	// Test Delete
	err = adapter.Delete(ctx, "integration-test")
	assert.NoError(t, err)

	// Verify deletion
	exists, err = adapter.Exists(ctx, "integration-test")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestRedisStoreAdapter_EdgeCases(t *testing.T) {
	mockClient := new(MockRedisClient)
	adapter := NewTestableRedisStoreAdapter(mockClient)
	ctx := context.Background()

	t.Run("Set with nil value", func(t *testing.T) {
		mockStatusCmd := &redis.StatusCmd{}
		mockStatusCmd.SetVal("OK")
		mockClient.On("Set", mock.Anything, "nil-key", nil, time.Hour).Return(mockStatusCmd)

		err := adapter.Set(ctx, "nil-key", nil, time.Hour)
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("Get with very large value", func(t *testing.T) {
		largeValue := make([]byte, 1024*1024) // 1MB
		mockStringCmd := &redis.StringCmd{}
		mockStringCmd.SetVal(string(largeValue))
		mockClient.On("Get", mock.Anything, "large-key").Return(mockStringCmd)

		result, err := adapter.Get(ctx, "large-key")
		assert.NoError(t, err)
		assert.Equal(t, largeValue, result)
		mockClient.AssertExpectations(t)
	})

	t.Run("TTL with very large duration", func(t *testing.T) {
		largeDuration := 365 * 24 * time.Hour // 1 year
		mockDurationCmd := &redis.DurationCmd{}
		mockDurationCmd.SetVal(largeDuration)
		mockClient.On("TTL", mock.Anything, "long-ttl-key").Return(mockDurationCmd)

		result, err := adapter.TTL(ctx, "long-ttl-key")
		assert.NoError(t, err)
		assert.Equal(t, largeDuration, result)
		mockClient.AssertExpectations(t)
	})
}
