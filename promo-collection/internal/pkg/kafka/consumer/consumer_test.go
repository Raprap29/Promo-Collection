package consumer

import (
	"errors"
	"testing"
	"time"

	"promocollection/internal/pkg/config"

	kafka "github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock low level consumer
type MockLowLevelConsumer struct {
	mock.Mock
}

func (m *MockLowLevelConsumer) SubscribeTopics(topics []string, rebalanceCb kafka.RebalanceCb) error {
	args := m.Called(topics, rebalanceCb)
	return args.Error(0)
}

func (m *MockLowLevelConsumer) ReadMessage(timeout time.Duration) (*kafka.Message, error) {
	args := m.Called(timeout)
	msg := args.Get(0)
	if msg == nil {
		return nil, args.Error(1)
	}
	return msg.(*kafka.Message), args.Error(1)
}

func (m *MockLowLevelConsumer) Close() error {
	args := m.Called()
	return args.Error(0)
}

// Mock factory function
func mockFactory(cfg *kafka.ConfigMap) (lowLevelConsumer, error) {
	return &MockLowLevelConsumer{}, nil
}

func mockFactoryError(cfg *kafka.ConfigMap) (lowLevelConsumer, error) {
	return nil, errors.New("factory error")
}

func TestNewKafkaConsumerWithFactory(t *testing.T) {
	t.Run("successful creation with mock factory", func(t *testing.T) {
		kcfg := config.KafkaConfig{
			Server:           "localhost:9092",
			SecurityProtocol: "PLAINTEXT",
			SASLMechanism:    "PLAIN",
			SASLUsername:     "user",
			SASLPassword:     "pass",
			SessionTimeoutMs: 12000,
			ClientID:         "test-client",
			GroupID:          "test-group",
		}

		consumer, err := NewKafkaConsumerWithFactory(kcfg, mockFactory)

		assert.NoError(t, err)
		assert.NotNil(t, consumer)
		assert.NotNil(t, consumer.Consumer)
	})

	t.Run("factory error", func(t *testing.T) {
		kcfg := config.KafkaConfig{
			Server:           "localhost:9092",
			SecurityProtocol: "PLAINTEXT",
			SASLMechanism:    "PLAIN",
			SASLUsername:     "user",
			SASLPassword:     "pass",
			SessionTimeoutMs: 12000,
			ClientID:         "test-client",
			GroupID:          "test-group",
		}

		consumer, err := NewKafkaConsumerWithFactory(kcfg, mockFactoryError)

		assert.Error(t, err)
		assert.Nil(t, consumer)
		assert.Contains(t, err.Error(), "factory error")
	})
}

func TestNewKafkaConsumer(t *testing.T) {
	t.Run("production constructor", func(t *testing.T) {
		kcfg := config.KafkaConfig{
			Server:           "localhost:9092",
			SecurityProtocol: "PLAINTEXT",
			SASLMechanism:    "PLAIN",
			SASLUsername:     "user",
			SASLPassword:     "pass",
			SessionTimeoutMs: 12000,
			ClientID:         "test-client",
			GroupID:          "test-group",
		}

		// This will use the real kafka.NewConsumer which may fail in test environment
		// but we can test that the function doesn't panic
		assert.NotPanics(t, func() {
			NewKafkaConsumer(kcfg)
		})
	})
}

func TestKafkaConsumer_Subscribe(t *testing.T) {
	t.Run("successful subscription", func(t *testing.T) {
		mockConsumer := &MockLowLevelConsumer{}
		mockConsumer.On("SubscribeTopics", []string{"test-topic"}, mock.AnythingOfType("kafka.RebalanceCb")).Return(nil)

		kc := &KafkaConsumer{Consumer: mockConsumer}
		err := kc.Subscribe("test-topic")

		assert.NoError(t, err)
		mockConsumer.AssertExpectations(t)
	})

	t.Run("subscription error", func(t *testing.T) {
		mockConsumer := &MockLowLevelConsumer{}
		mockConsumer.On("SubscribeTopics", []string{"test-topic"}, mock.AnythingOfType("kafka.RebalanceCb")).Return(errors.New("subscribe error"))

		kc := &KafkaConsumer{Consumer: mockConsumer}
		err := kc.Subscribe("test-topic")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "subscribe error")
		mockConsumer.AssertExpectations(t)
	})
}

func TestKafkaConsumer_Consume(t *testing.T) {
	t.Run("successful message consumption", func(t *testing.T) {
		mockConsumer := &MockLowLevelConsumer{}
		mockMessage := &kafka.Message{Value: []byte("test message")}
		mockConsumer.On("ReadMessage", time.Duration(-1)).Return(mockMessage, nil)

		kc := &KafkaConsumer{Consumer: mockConsumer}
		msg, err := kc.Consume()

		assert.NoError(t, err)
		assert.Equal(t, mockMessage, msg)
		mockConsumer.AssertExpectations(t)
	})

	t.Run("consumption error", func(t *testing.T) {
		mockConsumer := &MockLowLevelConsumer{}
		mockConsumer.On("ReadMessage", time.Duration(-1)).Return(nil, errors.New("consume error"))

		kc := &KafkaConsumer{Consumer: mockConsumer}
		msg, err := kc.Consume()

		assert.Error(t, err)
		assert.Nil(t, msg)
		assert.Contains(t, err.Error(), "consume error")
		mockConsumer.AssertExpectations(t)
	})
}

func TestKafkaConsumer_Close(t *testing.T) {
	t.Run("successful close", func(t *testing.T) {
		mockConsumer := &MockLowLevelConsumer{}
		mockConsumer.On("Close").Return(nil)

		kc := &KafkaConsumer{Consumer: mockConsumer}
		err := kc.Close()

		assert.NoError(t, err)
		mockConsumer.AssertExpectations(t)
	})

	t.Run("close error", func(t *testing.T) {
		mockConsumer := &MockLowLevelConsumer{}
		mockConsumer.On("Close").Return(errors.New("close error"))

		kc := &KafkaConsumer{Consumer: mockConsumer}
		err := kc.Close()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "close error")
		mockConsumer.AssertExpectations(t)
	})
}

func TestKafkaConsumer_Interface(t *testing.T) {
	t.Run("implements interface", func(t *testing.T) {
		var _ KafkaConsumerInterface = &KafkaConsumer{}
	})

	t.Run("implements low level interface", func(t *testing.T) {
		var _ lowLevelConsumer = &MockLowLevelConsumer{}
	})
}

func TestConsumerFactory(t *testing.T) {
	t.Run("default factory type", func(t *testing.T) {
		// Test that defaultKafkaFactory is of the correct type
		var factory consumerFactory = defaultKafkaFactory
		assert.NotNil(t, factory)
	})

	t.Run("mock factory type", func(t *testing.T) {
		// Test that mockFactory is of the correct type
		var factory consumerFactory = mockFactory
		assert.NotNil(t, factory)
	})
}
