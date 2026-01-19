package kafka_test

import (
	"context"
	"encoding/json"
	"errors"
	"promocollection/internal/pkg/models"
	"promocollection/internal/service/kafka"
	"testing"

	kafkaclient "github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockKafkaConsumer implements interfaces.KafkaConsumerInterface
type mockKafkaConsumer struct {
	mock.Mock
}

func (m *mockKafkaConsumer) Subscribe(topic string) error {
	args := m.Called(topic)
	return args.Error(0)
}

func (m *mockKafkaConsumer) Consume() (*kafkaclient.Message, error) {
	args := m.Called()
	msg, _ := args.Get(0).(*kafkaclient.Message)
	return msg, args.Error(1)
}

func (m *mockKafkaConsumer) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestStartKafkaConsumer_Sucess(t *testing.T) {
	mockConsumer := new(mockKafkaConsumer)

	payload := models.PromoEventMessage{
		ID:       123,
		MSISDN:   9876543210,
	}
	payloadBytes, _ := json.Marshal(payload)

	msg := &kafkaclient.Message{Value: payloadBytes}

	// First call returns message with no error
	mockConsumer.On("Consume").Return(msg, nil).Once()

	svc := kafka.KafkaConsumerService{}
	gotPayload, gotMsg, err := svc.StartKafkaConsumer(context.Background(), mockConsumer)

	assert.NoError(t, err)
	assert.Equal(t, &payload, gotPayload)
	assert.Equal(t, msg, gotMsg)
	mockConsumer.AssertExpectations(t)
}

func TestStartKafkaConsumer_ErrorThenSuccess(t *testing.T) {
	mockConsumer := new(mockKafkaConsumer)

	payload := models.PromoEventMessage{ID: 123}
	payloadBytes, _ := json.Marshal(payload)
	msg := &kafkaclient.Message{Value: payloadBytes}

	// First call returns error, second call returns success
	mockConsumer.On("Consume").Return(nil, errors.New("consume error")).Once()
	mockConsumer.On("Consume").Return(msg, nil).Once()

	svc := kafka.KafkaConsumerService{}
	gotPayload, gotMsg, err := svc.StartKafkaConsumer(context.Background(), mockConsumer)

	assert.NoError(t, err)
	assert.Equal(t, &payload, gotPayload)
	assert.Equal(t, msg, gotMsg)
	mockConsumer.AssertExpectations(t)
}

func TestSerializeKafkaMessage_ValidJSON(t *testing.T) {
	svc := kafka.KafkaConsumerService{}
	payload := models.PromoEventMessage{ID:1}
	data, _ := json.Marshal(payload)

	got, err := svc.SerializeKafkaMessage(data)

	assert.NoError(t, err)
	assert.Equal(t, &payload, got)
}

func TestPrintEvent_NoPanic(t *testing.T) {
	svc := kafka.KafkaConsumerService{}
	payload := &models.PromoEventMessage{ID: 1}
	// Just verify it runs without panic
	svc.PrintEvent(payload)
}
