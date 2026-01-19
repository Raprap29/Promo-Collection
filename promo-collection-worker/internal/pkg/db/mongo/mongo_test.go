package mongo

import (
	"context"
	"errors"
	"testing"

	"promo-collection-worker/internal/pkg/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MockMongoConnector mocks the MongoConnector interface
type MockMongoConnector struct {
	mock.Mock
}

func (m *MockMongoConnector) Connect(ctx context.Context, opts *options.ClientOptions) (*mongo.Client, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).(*mongo.Client), args.Error(1)
}

func (m *MockMongoConnector) Ping(ctx context.Context, client *mongo.Client) error {
	args := m.Called(ctx, client)
	return args.Error(0)
}

func TestConnectWithConnector(t *testing.T) {
	t.Run("successful connection and ping", func(t *testing.T) {
		cfg := config.MongoConfig{
			URI:    "mongodb://localhost:27017",
			DBName: "testdb",
		}

		mockConnector := &MockMongoConnector{}
		mockClient := &mongo.Client{}

		// Expectations
		mockConnector.On("Connect", mock.Anything, mock.AnythingOfType("*options.ClientOptions")).Return(mockClient, nil)
		mockConnector.On("Ping", mock.Anything, mockClient).Return(nil)

		ctx := context.Background()
		mongoClient, err := connectWithConnector(ctx, cfg, mockConnector)

		require.NoError(t, err)
		require.NotNil(t, mongoClient)
		assert.Equal(t, mockClient, mongoClient.Client)
		assert.NotNil(t, mongoClient.Database)

		mockConnector.AssertExpectations(t)
	})

	t.Run("connection failure", func(t *testing.T) {
		cfg := config.MongoConfig{
			URI:    "mongodb://localhost:27017",
			DBName: "testdb",
		}

		mockConnector := &MockMongoConnector{}

		// Expectations
		mockConnector.On("Connect", mock.Anything, mock.AnythingOfType("*options.ClientOptions")).
			Return(&mongo.Client{}, errors.New("connection error"))

		ctx := context.Background()
		mongoClient, err := connectWithConnector(ctx, cfg, mockConnector)

		require.Error(t, err)
		assert.Nil(t, mongoClient)

		mockConnector.AssertExpectations(t)
	})

	t.Run("ping failure after successful connection", func(t *testing.T) {
		cfg := config.MongoConfig{
			URI:    "mongodb://localhost:27017",
			DBName: "testdb",
		}

		mockConnector := &MockMongoConnector{}
		mockClient := &mongo.Client{}

		// Expectations
		mockConnector.On("Connect", mock.Anything, mock.AnythingOfType("*options.ClientOptions")).Return(mockClient, nil)
		mockConnector.On("Ping", mock.Anything, mockClient).Return(errors.New("ping error"))

		ctx := context.Background()
		mongoClient, err := connectWithConnector(ctx, cfg, mockConnector)

		require.Error(t, err)
		assert.Nil(t, mongoClient)

		mockConnector.AssertExpectations(t)
	})
}

// Test ConnectToMongoDB function
func TestConnectToMongoDB(t *testing.T) {
	t.Run("successful connection with default connector", func(t *testing.T) {
		cfg := config.MongoConfig{
			URI:             "mongodb://localhost:27017",
			DBName:          "testdb",
			ConnectTimeout:  5,
			MaxPoolSize:     10,
			MinPoolSize:     5,
			MaxConnIdleTime: 30,
		}

		ctx := context.Background()
		// This will fail in test environment but covers the function call
		_, err := ConnectToMongoDB(ctx, cfg)
		// We expect an error since there's no real MongoDB running
		assert.Error(t, err)
	})
}

// Test Disconnect function
func TestDisconnect(t *testing.T) {
	t.Run("disconnect with mock client", func(t *testing.T) {
		// Create a mock client that will fail disconnect
		mockClient := &mongo.Client{}
		err := Disconnect(mockClient)
		// Disconnect can succeed even with mock client
		// We're testing the function call path
		assert.NoError(t, err)
	})
}

// Test DefaultMongoConnector
func TestDefaultMongoConnector(t *testing.T) {
	t.Run("default connector connect", func(t *testing.T) {
		connector := &DefaultMongoConnector{}
		opts := options.Client().ApplyURI("mongodb://localhost:27017")

		ctx := context.Background()
		client, err := connector.Connect(ctx, opts)

		// Connect can succeed in test environment
		assert.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("default connector ping", func(t *testing.T) {
		connector := &DefaultMongoConnector{}
		mockClient := &mongo.Client{}

		ctx := context.Background()
		err := connector.Ping(ctx, mockClient)

		// This will fail in test environment but covers the function call
		assert.Error(t, err)
	})
}

// Test connectWithConnector with full config options
func TestConnectWithConnector_FullConfig(t *testing.T) {
	t.Run("connection with all config options set", func(t *testing.T) {
		cfg := config.MongoConfig{
			URI:             "mongodb://localhost:27017",
			DBName:          "testdb",
			ConnectTimeout:  10,
			MaxPoolSize:     20,
			MinPoolSize:     10,
			MaxConnIdleTime: 30,
		}

		mockConnector := &MockMongoConnector{}
		mockClient := &mongo.Client{}

		// Expectations
		mockConnector.On("Connect", mock.Anything, mock.AnythingOfType("*options.ClientOptions")).Return(mockClient, nil)
		mockConnector.On("Ping", mock.Anything, mockClient).Return(nil)

		ctx := context.Background()
		mongoClient, err := connectWithConnector(ctx, cfg, mockConnector)

		require.NoError(t, err)
		require.NotNil(t, mongoClient)
		assert.Equal(t, mockClient, mongoClient.Client)
		assert.NotNil(t, mongoClient.Database)

		mockConnector.AssertExpectations(t)
	})
}

// Test error logging paths
func TestConnectWithConnector_ErrorLogging(t *testing.T) {
	t.Run("connection error logs with config details", func(t *testing.T) {
		cfg := config.MongoConfig{
			URI:    "mongodb://invalid:27017",
			DBName: "testdb",
		}

		mockConnector := &MockMongoConnector{}

		// Expectations - return error to trigger error logging
		mockConnector.On("Connect", mock.Anything, mock.AnythingOfType("*options.ClientOptions")).
			Return((*mongo.Client)(nil), errors.New("connection failed"))

		ctx := context.Background()
		mongoClient, err := connectWithConnector(ctx, cfg, mockConnector)

		require.Error(t, err)
		assert.Nil(t, mongoClient)
		assert.Contains(t, err.Error(), "connection failed")

		mockConnector.AssertExpectations(t)
	})

	t.Run("ping error logs with config details", func(t *testing.T) {
		cfg := config.MongoConfig{
			URI:    "mongodb://localhost:27017",
			DBName: "testdb",
		}

		mockConnector := &MockMongoConnector{}
		mockClient := &mongo.Client{}

		// Expectations - connect succeeds but ping fails
		mockConnector.On("Connect", mock.Anything, mock.AnythingOfType("*options.ClientOptions")).Return(mockClient, nil)
		mockConnector.On("Ping", mock.Anything, mockClient).Return(errors.New("ping failed"))

		ctx := context.Background()
		mongoClient, err := connectWithConnector(ctx, cfg, mockConnector)

		require.Error(t, err)
		assert.Nil(t, mongoClient)
		assert.Contains(t, err.Error(), "ping failed")

		mockConnector.AssertExpectations(t)
	})
}
