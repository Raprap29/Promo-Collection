package interfaces

import (
	"context"
	sm "promocollection/internal/pkg/models"

	kafkaclient "github.com/confluentinc/confluent-kafka-go/kafka"
)

type KafkaConsumerInterface interface {
	Subscribe(topic string) error
	Consume() (*kafkaclient.Message, error)
	Close() error
}

type KafkaConsumerServiceInterface interface {
	StartKafkaConsumer(ctx context.Context,
		consumer KafkaConsumerInterface) (*sm.PromoEventMessage, *kafkaclient.Message, error)
	SerializeKafkaMessage(message []byte) (*sm.PromoEventMessage, error)
	PrintEvent(payload *sm.PromoEventMessage)
}
