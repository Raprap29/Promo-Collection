package kafka

import (
	"context"
	"fmt"
	"testing"

	"promo-collection-worker/internal/pkg/config"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testServer         = "localhost:9092"
	testTopic          = "test-topic"
	testClientID       = "test-client"
	testMessageContent = "test message"
)

// MockProducer is a mock implementation of ProducerInterface for testing.
type MockProducer struct {
	ProduceFunc func(msg *kafka.Message, deliveryChan chan kafka.Event) error
	FlushFunc   func(timeoutMs int) int
	CloseFunc   func()
}

func (m *MockProducer) Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error {
	if m.ProduceFunc != nil {
		return m.ProduceFunc(msg, deliveryChan)
	}
	return nil
}

func (m *MockProducer) Flush(timeoutMs int) int {
	if m.FlushFunc != nil {
		return m.FlushFunc(timeoutMs)
	}
	return 0
}

func (m *MockProducer) Close() {
	if m.CloseFunc != nil {
		m.CloseFunc()
	}
}

func TestNewKafkaProducer(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := config.KafkaConfig{
			Server:           testServer,
			PromoTopic:       testTopic,
			SecurityProtocol: "SASL_SSL",
			SASLMechanism:    "PLAIN",
			SASLUsername:     "testuser",
			SASLPassword:     "testpassword",
			SessionTimeoutMs: 10000,
			ClientID:         testClientID,
		}

		producer, err := NewKafkaProducer(cfg)
		require.NoError(t, err)
		require.NotNil(t, producer)
		defer producer.Close()
		assert.Equal(t, cfg.PromoTopic, producer.topic)
	})

	t.Run("invalid config", func(t *testing.T) {
		cfg := config.KafkaConfig{
			Server:           testServer,
			PromoTopic:       testTopic,
			SecurityProtocol: "INVALID_PROTOCOL", // invalid security protocol
			SASLMechanism:    "PLAIN",
			SASLUsername:     "testuser",
			SASLPassword:     "testpassword",
			ClientID:         testClientID,
		}

		producer, err := NewKafkaProducer(cfg)
		assert.Error(t, err)
		assert.Nil(t, producer)
	})
}

func TestKafkaProducerPublish(t *testing.T) {
	t.Run("timeout case", func(t *testing.T) {
		// Mock producer that does not send delivery event
		mockProducer := &MockProducer{
			ProduceFunc: func(msg *kafka.Message, deliveryChan chan kafka.Event) error {
				// Do not send event to simulate timeout
				return nil
			},
		}

		producer := &KafkaProducer{
			producer: mockProducer,
			topic:    testTopic,
		}

		ctx := context.Background()
		msg := []byte(testMessageContent)

		err := producer.Publish(ctx, msg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout")
	})

	t.Run("successful delivery", func(t *testing.T) {
		mockProducer := &MockProducer{
			ProduceFunc: func(msg *kafka.Message, deliveryChan chan kafka.Event) error {
				// Simulate successful delivery
				go func() {
					deliveryChan <- &kafka.Message{
						TopicPartition: kafka.TopicPartition{
							Topic:     msg.TopicPartition.Topic,
							Partition: msg.TopicPartition.Partition,
							Error:     nil,
						},
						Value: msg.Value,
					}
				}()
				return nil
			},
		}

		producer := &KafkaProducer{
			producer: mockProducer,
			topic:    testTopic,
		}

		ctx := context.Background()
		msg := []byte(testMessageContent)

		err := producer.Publish(ctx, msg)
		assert.NoError(t, err)
	})

	t.Run("delivery failure", func(t *testing.T) {
		mockProducer := &MockProducer{
			ProduceFunc: func(msg *kafka.Message, deliveryChan chan kafka.Event) error {
				// Simulate delivery failure
				go func() {
					deliveryChan <- &kafka.Message{
						TopicPartition: kafka.TopicPartition{
							Topic:     msg.TopicPartition.Topic,
							Partition: msg.TopicPartition.Partition,
							Error:     kafka.NewError(kafka.ErrMsgTimedOut, "delivery failed", false),
						},
						Value: msg.Value,
					}
				}()
				return nil
			},
		}

		producer := &KafkaProducer{
			producer: mockProducer,
			topic:    testTopic,
		}

		ctx := context.Background()
		msg := []byte(testMessageContent)

		err := producer.Publish(ctx, msg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "delivery failed")
	})

	t.Run("produce error", func(t *testing.T) {
		mockProducer := &MockProducer{
			ProduceFunc: func(msg *kafka.Message, deliveryChan chan kafka.Event) error {
				return fmt.Errorf("produce failed")
			},
		}

		producer := &KafkaProducer{
			producer: mockProducer,
			topic:    testTopic,
		}

		ctx := context.Background()
		msg := []byte(testMessageContent)

		err := producer.Publish(ctx, msg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "produce failed")
	})

	t.Run("unexpected event type", func(t *testing.T) {
		mockProducer := &MockProducer{
			ProduceFunc: func(msg *kafka.Message, deliveryChan chan kafka.Event) error {
				// Send unexpected event type (kafka.Error instead of *kafka.Message)
				go func() {
					deliveryChan <- kafka.NewError(kafka.ErrUnknown, "unexpected event", false)
				}()
				return nil
			},
		}

		producer := &KafkaProducer{
			producer: mockProducer,
			topic:    testTopic,
		}

		ctx := context.Background()
		msg := []byte(testMessageContent)

		err := producer.Publish(ctx, msg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected event type")
	})
}

func TestKafkaProducerClose(t *testing.T) {
	cfg := config.KafkaConfig{
		Server:           testServer,
		PromoTopic:       testTopic,
		SecurityProtocol: "SASL_SSL",
		SASLMechanism:    "PLAIN",
		SASLUsername:     "testuser",
		SASLPassword:     "testpassword",
		SessionTimeoutMs: 10000,
		ClientID:         testClientID,
	}

	producer, err := NewKafkaProducer(cfg)
	if err != nil {
		t.Skip("Skipping close test as producer creation failed")
	}

	// Close should not panic
	assert.NotPanics(t, func() {
		producer.Close()
	})
}

func TestKafkaProducerPublishDeliveryFailure(t *testing.T) {
	// This test simulates delivery failure by mocking the delivery event.
	// Since we cannot inject the delivery channel, we skip this for now.
	// In a real scenario, this would require mocking the kafka.Producer.
	t.Skip("Delivery failure test requires mocking kafka.Producer, which is not implemented")
}

func TestKafkaProducerConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		cfg         config.KafkaConfig
		expectError bool
	}{
		{
			name: "valid config",
			cfg: config.KafkaConfig{
				Server:           testServer,
				PromoTopic:       testTopic,
				SecurityProtocol: "SASL_SSL",
				SASLMechanism:    "PLAIN",
				SASLUsername:     "testuser",
				SASLPassword:     "testpassword",
				ClientID:         testClientID,
			},
			expectError: false,
		},
		{
			name: "empty server",
			cfg: config.KafkaConfig{
				Server:           "",
				PromoTopic:       testTopic,
				SecurityProtocol: "SASL_SSL",
				SASLMechanism:    "PLAIN",
				SASLUsername:     "testuser",
				SASLPassword:     "testpassword",
				ClientID:         testClientID,
			},
			expectError: false, // Producer can be created, but connection fails later
		},
		{
			name: "empty topic",
			cfg: config.KafkaConfig{
				Server:           testServer,
				PromoTopic:       "",
				SecurityProtocol: "SASL_SSL",
				SASLMechanism:    "PLAIN",
				SASLUsername:     "testuser",
				SASLPassword:     "testpassword",
				ClientID:         testClientID,
			},
			expectError: false, // Topic can be empty, but Produce may fail
		},
		{
			name: "invalid security protocol",
			cfg: config.KafkaConfig{
				Server:           testServer,
				PromoTopic:       testTopic,
				SecurityProtocol: "INVALID",
				SASLMechanism:    "PLAIN",
				SASLUsername:     "testuser",
				SASLPassword:     "testpassword",
				ClientID:         testClientID,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			producer, err := NewKafkaProducer(tt.cfg)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, producer)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, producer)
				if producer != nil {
					producer.Close()
				}
			}
		})
	}
}
