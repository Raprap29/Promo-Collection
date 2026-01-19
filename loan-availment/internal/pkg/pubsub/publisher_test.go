package pubsub

import (
	"context"
	"errors"
	"testing"

	"globe/dodrio_loan_availment/internal/service/interfaces"

	"cloud.google.com/go/pubsub/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockPubSubClient struct {
	mock.Mock
}

func (m *MockPubSubClient) Publisher(topicName string) interfaces.TopicPublisherInterface {
	args := m.Called(topicName)
	return args.Get(0).(interfaces.TopicPublisherInterface)
}

func (m *MockPubSubClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

type MockTopicPublisher struct {
	mock.Mock
}

func (m *MockTopicPublisher) Publish(ctx context.Context, data []byte, attributes map[string]string) (string, error) {
	args := m.Called(ctx, data, attributes)
	return args.String(0), args.Error(1)
}

type MockFactory struct {
	mock.Mock
}

func (m *MockFactory) NewPublisher(ctx context.Context, projectID string) (interfaces.PublisherInterface, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(interfaces.PublisherInterface), args.Error(1)
}

func TestNewPubSubPublisher(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		ctx := context.Background()
		publisher, err := NewPubSubPublisher(ctx, "test-project")
		if err != nil {
			t.Skip("Skipping - GCP credentials not available")
		}
		assert.NoError(t, err)
		assert.NotNil(t, publisher)
		defer func() {
			if publisher != nil {
				publisher.Close()
			}
		}()
	})

	t.Run("empty project ID", func(t *testing.T) {
		ctx := context.Background()
		_, err := NewPubSubPublisher(ctx, "")
		assert.Error(t, err)
	})
}

func TestNewPubSubPublisherWithFactory(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		mockFactory := new(MockFactory)
		mockClient := new(MockPubSubClient)

		mockFactory.On("NewPublisher", mock.Anything, "test-project").Return(mockClient, nil)

		ctx := context.Background()
		publisher, err := NewPubSubPublisherWithFactory(ctx, "test-project", mockFactory)

		assert.NoError(t, err)
		assert.NotNil(t, publisher)

		mockFactory.AssertExpectations(t)
	})

	t.Run("factory error", func(t *testing.T) {
		mockFactory := new(MockFactory)
		mockFactory.On("NewPublisher", mock.Anything, "test-project").Return(nil, errors.New("factory error"))

		ctx := context.Background()
		publisher, err := NewPubSubPublisherWithFactory(ctx, "test-project", mockFactory)

		assert.Error(t, err)
		assert.Nil(t, publisher)
		mockFactory.AssertExpectations(t)
	})
}

func TestPubSubPublisher_Publish(t *testing.T) {
	mockFactory := new(MockFactory)
	mockClient := new(MockPubSubClient)
	mockTopicPublisher := new(MockTopicPublisher)

	mockFactory.On("NewPublisher", mock.Anything, "test-project").Return(mockClient, nil)

	ctx := context.Background()
	publisher, err := NewPubSubPublisherWithFactory(ctx, "test-project", mockFactory)
	assert.NoError(t, err)

	t.Run("successful publish", func(t *testing.T) {
		mockClient.On("Publisher", "test-topic").Return(mockTopicPublisher)
		mockTopicPublisher.On("Publish", mock.Anything, []byte("test message"), map[string]string{"key": "value"}).Return("test-message-id", nil)

		messageID, err := publisher.Publish(ctx, "test-topic", []byte("test message"), map[string]string{"key": "value"})

		assert.NoError(t, err)
		assert.Equal(t, "test-message-id", messageID)
	})

	t.Run("publish error", func(t *testing.T) {
		mockClient.On("Publisher", "error-topic").Return(mockTopicPublisher)
		mockTopicPublisher.On("Publish", mock.Anything, []byte("test"), mock.Anything).Return("", errors.New("publish failed"))

		messageID, err := publisher.Publish(ctx, "error-topic", []byte("test"), nil)

		assert.Error(t, err)
		assert.Empty(t, messageID)
	})
}

func TestPubSubPublisher_Stop(t *testing.T) {
	mockFactory := new(MockFactory)
	mockClient := new(MockPubSubClient)

	mockFactory.On("NewPublisher", mock.Anything, "test-project").Return(mockClient, nil)

	ctx := context.Background()
	publisher, err := NewPubSubPublisherWithFactory(ctx, "test-project", mockFactory)
	assert.NoError(t, err)

	err = publisher.Stop(context.Background())
	assert.NoError(t, err)
}

func TestPubSubPublisher_Close(t *testing.T) {
	mockFactory := new(MockFactory)
	mockClient := new(MockPubSubClient)

	mockFactory.On("NewPublisher", mock.Anything, "test-project").Return(mockClient, nil)
	mockClient.On("Close").Return(nil)

	ctx := context.Background()
	publisher, err := NewPubSubPublisherWithFactory(ctx, "test-project", mockFactory)
	assert.NoError(t, err)

	err = publisher.Close()
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestDefaultPublisherFactory(t *testing.T) {
	factory := &defaultPublisherFactory{}

	t.Run("creation attempt", func(t *testing.T) {
		ctx := context.Background()
		client, err := factory.NewPublisher(ctx, "test-project")
		if err != nil {
			t.Skip("Skipping - GCP credentials not available")
		}
		assert.NoError(t, err)
		assert.NotNil(t, client)
		defer func() {
			if client != nil {
				client.Close()
			}
		}()
	})

	t.Run("empty project", func(t *testing.T) {
		ctx := context.Background()
		_, err := factory.NewPublisher(ctx, "")
		assert.Error(t, err)
	})
}

func TestDefaultPublisher(t *testing.T) {
	ctx := context.Background()
	client, err := pubsub.NewClient(ctx, "test-project")
	if err != nil {
		t.Skip("Skipping - GCP credentials not available")
	}
	defer func() {
		if client != nil {
			client.Close()
		}
	}()

	publisher := &defaultPublisher{client: client}

	t.Run("get topic publisher", func(t *testing.T) {
		topicPublisher := publisher.Publisher("test-topic")
		assert.NotNil(t, topicPublisher)
	})

	t.Run("close publisher", func(t *testing.T) {
		err := publisher.Close()
		assert.NoError(t, err)
	})
}

func TestPubSubPublisher_EdgeCases(t *testing.T) {
	mockFactory := new(MockFactory)
	mockClient := new(MockPubSubClient)
	mockTopicPublisher := new(MockTopicPublisher)

	mockFactory.On("NewPublisher", mock.Anything, "test-project").Return(mockClient, nil)

	ctx := context.Background()
	publisher, err := NewPubSubPublisherWithFactory(ctx, "test-project", mockFactory)
	assert.NoError(t, err)

	t.Run("empty data", func(t *testing.T) {
		mockClient.On("Publisher", "test-topic").Return(mockTopicPublisher)
		mockTopicPublisher.On("Publish", mock.Anything, []byte{}, mock.Anything).Return("test-id", nil)

		messageID, err := publisher.Publish(ctx, "test-topic", []byte{}, nil)
		assert.NoError(t, err)
		assert.Equal(t, "test-id", messageID)
	})

	t.Run("large payload", func(t *testing.T) {
		largeData := make([]byte, 1024*100) // 100KB
		mockClient.On("Publisher", "test-topic").Return(mockTopicPublisher)
		mockTopicPublisher.On("Publish", mock.Anything, largeData, mock.Anything).Return("large-id", nil)

		messageID, err := publisher.Publish(ctx, "test-topic", largeData, nil)
		assert.NoError(t, err)
		assert.Equal(t, "large-id", messageID)
	})
}

func TestPubSubPublisher_ConcurrentAccess(t *testing.T) {
	mockFactory := new(MockFactory)
	mockClient := new(MockPubSubClient)
	mockTopicPublisher := new(MockTopicPublisher)

	mockFactory.On("NewPublisher", mock.Anything, "test-project").Return(mockClient, nil)
	mockClient.On("Publisher", "test-topic").Return(mockTopicPublisher)
	mockTopicPublisher.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return("concurrent-id", nil)

	ctx := context.Background()
	publisher, err := NewPubSubPublisherWithFactory(ctx, "test-project", mockFactory)
	assert.NoError(t, err)

	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func() {
			messageID, err := publisher.Publish(ctx, "test-topic", []byte("concurrent test"), nil)
			assert.NoError(t, err)
			assert.Equal(t, "concurrent-id", messageID)
			done <- true
		}()
	}

	for i := 0; i < 5; i++ {
		<-done
	}
}
