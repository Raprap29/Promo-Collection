package kafka

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"math"

	"promo-collection-worker/internal/pkg/kafka"
	"promo-collection-worker/internal/pkg/logger"
	"promo-collection-worker/internal/pkg/models"
)

// CollectionWorkerKafkaService handles publishing success and failure events to Kafka.
type CollectionWorkerKafkaService struct {
	KafkaProducer kafka.KafkaProducerInterface
}

// NewCollectionWorkerKafkaService creates a new instance of CollectionWorkerKafkaService.
func NewCollectionWorkerKafkaService(producer kafka.KafkaProducerInterface) *CollectionWorkerKafkaService {
	return &CollectionWorkerKafkaService{
		KafkaProducer: producer,
	}
}

// publishMessage is a helper function to publish a message to Kafka.
func (s *CollectionWorkerKafkaService) publishMessage(
	ctx context.Context,
	msg models.KafkaMessageForPublishing,
	messageType string,
) error {
	data, err := s.formatPayloadAsCsvBytes(msg)
	if err != nil {
		logger.CtxError(ctx, "Failed to format "+messageType+" message as CSV", err)
		return err
	}
	err = s.KafkaProducer.Publish(ctx, data)
	if err != nil {
		logger.CtxError(ctx, "Failed to publish "+messageType+" message to Kafka", err)
		return err
	}
	logger.CtxInfo(ctx, "Successfully published "+messageType+" message to Kafka")
	return nil
}

// PublishSuccess publishes a success message to Kafka.
func (s *CollectionWorkerKafkaService) PublishSuccess(ctx context.Context, msg models.KafkaMessageForPublishing) error {
	return s.publishMessage(ctx, msg, "success")
}

// PublishFailure publishes a failure message to Kafka.
func (s *CollectionWorkerKafkaService) PublishFailure(ctx context.Context, msg models.KafkaMessageForPublishing) error {
	return s.publishMessage(ctx, msg, "failure")
}

func (s *CollectionWorkerKafkaService) formatPayloadAsCsvBytes(msg models.KafkaMessageForPublishing) ([]byte, error) {
	record := []string{
		msg.TransactionId,
		msg.AvailmentId,
		msg.CollectionDateTime.Format("01/02/2006 15:04"),
		msg.Method,
		fmt.Sprintf("%d", msg.LoanAgeing),
		fmt.Sprintf("%d", int(math.Round(msg.CollectedLoanAmount))),
		fmt.Sprintf("%d", int(math.Round(msg.CollectedServiceFee))),
		fmt.Sprintf("%d", int(math.Round(msg.UnpaidLoanAmount))),
		fmt.Sprintf("%d", int(math.Round(msg.UnpaidServiceFee))),
		msg.Result,
		msg.CollectionErrorText,
		msg.MSISDN,
		msg.CollectionCategory,
		msg.PaymentChannel,
		msg.CollectionType,
		fmt.Sprintf("%d", int(math.Round(msg.CollectedAmount))),
		fmt.Sprintf("%d", int(math.Round(msg.TotalUnpaid))),
		msg.TokenPaymentId,
		fmt.Sprintf("%d", int(math.Round(msg.DataCollected))),
	}

	buffer := &bytes.Buffer{}
	writer := csv.NewWriter(buffer)

	if err := writer.Write(record); err != nil {
		return nil, fmt.Errorf("error writing CSV record: %w", err)
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("error flushing CSV writer: %w", err)
	}

	csvBytes := bytes.TrimRight(buffer.Bytes(), "\n")
	return csvBytes, nil
}
