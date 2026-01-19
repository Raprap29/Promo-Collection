package producer

import (
	"context"
	"fmt"
	"globe/dodrio_loan_availment/configs"
	"globe/dodrio_loan_availment/internal/pkg/common"
	"globe/dodrio_loan_availment/internal/pkg/logger"

	kafkaservice "github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type KafkaService struct {
}

func NewKafkaService() *KafkaService {
	return &KafkaService{}
}

func KafkaPublisher(ctx context.Context, payload string) error {

	// err := configs.LoadEnv()
	// if err != nil {
	// 	logger.Error(ctx, "error loading .env file: %v", err)
	// }

	KafkaTopic := configs.KAFKA_TOPIC

	config := &kafkaservice.ConfigMap{
		"bootstrap.servers":  configs.KAFKA_SERVER,
		"security.protocol":  configs.KAFKA_SECURITY_PROTOCOL,
		"sasl.mechanisms":    configs.KAFKA_SASL_MECHANISM,
		"sasl.username":      configs.KAFKA_SASL_USERNAME,
		"sasl.password":      configs.KAFKA_SASL_PASSWORD,
		"session.timeout.ms": configs.KAFKA_SESSION_TIMEOUT_MS,
		"client.id":          configs.KAFKA_CLIENT_ID,
		"log_level":          0,
	}
	logger.Info(ctx, "kafka configuration %w", config)
	producer, err := kafkaservice.NewProducer(config)
	if err != nil {
		return fmt.Errorf("failed to create producer: %w", err)
	}
	defer producer.Close()

	// Serialize the payload to JSON
	payloadBytes := []byte(payload)
	// if err != nil {
	// 	return fmt.Errorf("failed to marshal payload: %w", err)
	// }

	// Publish the message
	deliveryChan := make(chan kafkaservice.Event, 1)
	err = producer.Produce(&kafkaservice.Message{
		TopicPartition: kafkaservice.TopicPartition{Topic: &KafkaTopic, Partition: kafkaservice.PartitionAny},
		Value:          payloadBytes,
	}, deliveryChan)
	if err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}

	// Wait for message delivery
	event := <-deliveryChan
	msg := event.(*kafkaservice.Message)
	if msg.TopicPartition.Error != nil {
		return fmt.Errorf("delivery failed: %w", msg.TopicPartition.Error)
	}

	// log.Printf("Message delivered to %v", msg.TopicPartition)
	// Print the delivered message
	logger.Info(ctx, "Message delivered to topic: %s, partition: %d, offset: %v, Message content: %s",
		*msg.TopicPartition.Topic, msg.TopicPartition.Partition, msg.TopicPartition.Offset, string(payloadBytes))

	return nil
}

func (k *KafkaService) PublishAvailmentStatusToKafka(ctx context.Context, transactionID string, availmentChannel string, msisdn string, brandType string, loanType string, loanProductName string, loanProductKeyword string, servicingPartner string, loanAmount float64, serviceFeeAmount float64, availmentResult string, availmentErrorCode string, loanAgeing int32, statusOfProvisionedLoan string) error {

	// log.Println("I am inside PublishAvailmentStatusToKafka")

	payload := common.SerializeAvailmentKafkaMessage(transactionID, availmentChannel, msisdn, brandType, loanType, loanProductName, loanProductKeyword, servicingPartner, loanAmount, serviceFeeAmount, availmentResult, availmentErrorCode, loanAgeing, statusOfProvisionedLoan)

	err := KafkaPublisher(ctx, payload)

	if err != nil {
		return err
	}

	return nil
}
