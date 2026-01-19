package kafka

import (
	"context"
	"fmt"
	"time"

	"promo-collection-worker/internal/pkg/config"
	"promo-collection-worker/internal/pkg/log_messages"
	"promo-collection-worker/internal/pkg/logger"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// ProducerInterface defines the interface for Kafka producer operations.
type ProducerInterface interface {
	Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error
	Flush(timeoutMs int) int
	Close()
}

// KafkaProducerInterface defines the interface for KafkaProducer.
type KafkaProducerInterface interface {
	Publish(ctx context.Context, msg []byte) error
}

// KafkaProducer manages Kafka producer lifecycle and publishing.
type KafkaProducer struct {
	producer ProducerInterface
	topic    string
}

// NewKafkaProducer creates and returns a new KafkaProducer instance.
func NewKafkaProducer(cfg config.KafkaConfig) (*KafkaProducer, error) {
	kafkaConfig := &kafka.ConfigMap{
		"bootstrap.servers": cfg.Server,
		"security.protocol": cfg.SecurityProtocol,
		"sasl.mechanisms":   cfg.SASLMechanism,
		"sasl.username":     cfg.SASLUsername,
		"sasl.password":     cfg.SASLPassword,
		"client.id":         cfg.ClientID,
	}

	producer, err := kafka.NewProducer(kafkaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka producer: %w", err)
	}
	logger.Info(log_messages.KafkaProducerCreated)

	return &KafkaProducer{
		producer: producer,
		topic:    cfg.PromoTopic,
	}, nil
}

// Publish sends a message to the Kafka topic.
func (kp *KafkaProducer) Publish(ctx context.Context, msg []byte) error {
	deliveryChan := make(chan kafka.Event, 1)
	defer close(deliveryChan)

	err := kp.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &kp.topic, Partition: kafka.PartitionAny},
		Value:          msg,
	}, deliveryChan)

	if err != nil {
		logger.CtxError(ctx, "Failed to produce Kafka message", err)
		return err
	}

	select {
	case ev := <-deliveryChan:
		m, ok := ev.(*kafka.Message)
		if !ok {
			return fmt.Errorf("unexpected event type")
		}
		if m.TopicPartition.Error != nil {
			return fmt.Errorf("delivery failed: %w", m.TopicPartition.Error)
		}
	case <-time.After(10 * time.Second):
		return fmt.Errorf("timeout waiting for Kafka delivery report")
	}

	return nil
}

// Close flushes and closes the Kafka producer.
func (kp *KafkaProducer) Close() error {
	kp.producer.Flush(5000)
	kp.producer.Close()
	return nil
}
