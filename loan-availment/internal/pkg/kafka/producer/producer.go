package producer

// to package should be kafka change while refactor
// package kafka

import (
	"context"
	"globe/dodrio_loan_availment/configs"
	"globe/dodrio_loan_availment/internal/pkg/logger"
	"strings"

	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	// "loancollection/internal/pkg/models"
)

type Producer struct {
	producer *kafka.Producer
	topic    string
}

var KafkaProducer *Producer

func NewKafkaProducer(broker, topic string) (*Producer, error) {
	p, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": configs.KAFKA_SERVER,
		"security.protocol":  configs.KAFKA_SECURITY_PROTOCOL,
		"sasl.mechanisms":    configs.KAFKA_SASL_MECHANISM,
		"sasl.username":      configs.KAFKA_SASL_USERNAME,
		"sasl.password":      configs.KAFKA_SASL_PASSWORD,
		"session.timeout.ms": configs.KAFKA_SESSION_TIMEOUT_MS,
		"client.id":          configs.KAFKA_CLIENT_ID,
		"log_level":          0})
	if err != nil {
		return nil, err
	}

	return &Producer{
		producer: p,
		topic:    topic,
	}, nil
}

func SendMessageBatch(ctx context.Context, kafkaProducer *Producer, messages [][]string, topic string, retryCount int) ([]string, []string, error) {

	var successIDs []string
	var failedIDs []string

	// prepare kafka message
	// Convert Availment to Kafka Messages
	kafkaMessages := make([]*kafka.Message, len(messages))
	for i, msg := range messages {
		//replaced with 0 as it is containtaing transection Id (ObjectId of record in db)
		transactionId := msg[0]
		stringValue := strings.Join(msg, ",")
		// if err != nil {
		// 	logger.Error(ctx, "Failed to marshal CollectionMessage to JSON: %v", err)
		// 	failedIDs = append(failedIDs, msg[0])
		// 	continue
		// }
		kafkaMessages[i] = &kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Value:          []byte(stringValue),
			Key:            []byte(transactionId),
		}
	}
	// retry
	for _, kafkaMsg := range kafkaMessages {
		success := false
		for attempt := 0; attempt <= retryCount; attempt++ {
			err := kafkaProducer.producer.Produce(kafkaMsg, nil)
			if err == nil {
				logger.Info(ctx, "kafka message sent successfully")
				success = true
				break
			}
			logger.Error(ctx, "Failed to send Kafka message on attempt %d: %v", attempt+1, err)
			// Backoff before retrying
			time.Sleep(time.Second * time.Duration(attempt+1))
		}
		if success {
			successIDs = append(successIDs, string(kafkaMsg.Key))
		}
		if !success {
			// Add the original CollectionMessage to the failed list
			failedIDs = append(successIDs, string(kafkaMsg.Key))
		}
	}
	// Wait for all messages to be delivered
	kafkaProducer.producer.Flush(15 * 1000)
	return successIDs, failedIDs, nil
}

func (p *Producer) Close() {
	p.producer.Close()
}
