package consumer

import (
	"time"

	"promocollection/internal/pkg/config"
	"promocollection/internal/pkg/log_messages"
	"promocollection/internal/pkg/logger"

	kafka "github.com/confluentinc/confluent-kafka-go/kafka"
)

// ---- Interface wrapping the real consumer for testability ----
type lowLevelConsumer interface {
	SubscribeTopics(topics []string, rebalanceCb kafka.RebalanceCb) error
	ReadMessage(timeout time.Duration) (*kafka.Message, error)
	Close() error
}

type KafkaConsumerInterface interface {
	Subscribe(topic string) error
	Consume() (*kafka.Message, error)
	Close() error
}

// KafkaConsumer implements KafkaConsumerInterface
type KafkaConsumer struct {
	Consumer lowLevelConsumer
}

// ---- Factory type and default factory (no globals) ----
type consumerFactory func(cfg *kafka.ConfigMap) (lowLevelConsumer, error)

func defaultKafkaFactory(cfg *kafka.ConfigMap) (lowLevelConsumer, error) {
	return kafka.NewConsumer(cfg)
}

// NewKafkaConsumerWithFactory allows injecting a mock factory in tests.
func NewKafkaConsumerWithFactory(kcfg config.KafkaConfig, factory consumerFactory) (*KafkaConsumer, error) {
	kafkaCfg := &kafka.ConfigMap{
		"bootstrap.servers":  kcfg.Server,
		"security.protocol":  kcfg.SecurityProtocol,
		"sasl.mechanisms":    kcfg.SASLMechanism,
		"sasl.username":      kcfg.SASLUsername,
		"sasl.password":      kcfg.SASLPassword,
		"session.timeout.ms": kcfg.SessionTimeoutMs,
		"client.id":          kcfg.ClientID,
		"group.id":           kcfg.GroupID,
		"log_level":          0,
		"auto.offset.reset":  "earliest",
	}

	consumer, err := factory(kafkaCfg)
	if err != nil {
		return nil, err
	}
	logger.Info(log_messages.KafkaConsumerCreated)
	return &KafkaConsumer{Consumer: consumer}, nil
}

// Production constructor uses the default factory.
func NewKafkaConsumer(kcfg config.KafkaConfig) (*KafkaConsumer, error) {
	return NewKafkaConsumerWithFactory(kcfg, defaultKafkaFactory)
}

func (kc *KafkaConsumer) Subscribe(topic string) error {
	return kc.Consumer.SubscribeTopics([]string{topic}, nil)
}

func (kc *KafkaConsumer) Consume() (*kafka.Message, error) {
	return kc.Consumer.ReadMessage(time.Duration(-1))
}

func (kc *KafkaConsumer) Close() error {
	logger.Info(log_messages.KafkaConsumerClosed)
	return kc.Consumer.Close()
}
