package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"promocollection/internal/pkg/log_messages"
	"promocollection/internal/pkg/logger"
	"promocollection/internal/pkg/models"
	"promocollection/internal/service/interfaces"

	kafkaclient "github.com/confluentinc/confluent-kafka-go/kafka"
)

type KafkaConsumerService struct{}

func (k *KafkaConsumerService) StartKafkaConsumer(ctx context.Context,
	 consumer interfaces.KafkaConsumerInterface) (*models.PromoEventMessage, 
		*kafkaclient.Message, error) {
    return StartPromoKafkaConsumer(ctx, consumer)
}

func (k *KafkaConsumerService) SerializeKafkaMessage(message []byte) (*models.PromoEventMessage, error) {
    return SerializePromoKafkaMessage(message)
}

func (k *KafkaConsumerService) PrintEvent(payload *models.PromoEventMessage) {
    printEvent(payload)
}


func StartPromoKafkaConsumer(ctx context.Context, 
	consumer interfaces.KafkaConsumerInterface) (*models.PromoEventMessage,
	 *kafkaclient.Message, error) {

	logger.Debug("Kafka Consumer Created")

	for {
	
		msg, err := consumer.Consume()
		if err != nil {
			logger.CtxError(ctx, log_messages.KafkaErrorConsuming, err)
			continue
		}
		
		payload, err := SerializePromoKafkaMessage(msg.Value)
		printEvent(payload)
		
		if err != nil {
			logger.CtxError(ctx, log_messages.ErrorSerializingKafkaMessage, err)
			continue
		}

		return payload,msg, nil
	}
}



func SerializePromoKafkaMessage(message []byte) (*models.PromoEventMessage, error) {

	var pem = models.PromoEventMessage{}

	err := json.Unmarshal(message, &pem)
	if err != nil {
		logger.Error("Failed to unmarshal config", err)
	}
	return &pem,nil
}



func printEvent(payload *models.PromoEventMessage) {
	logger.Info(fmt.Sprintf("ID: %v", payload.ID))
	logger.Info(fmt.Sprintf("MSISDN: %v", payload.MSISDN))
	logger.Info(fmt.Sprintf("MESSAGE: %v", payload.MESSAGE))
	logger.Info(fmt.Sprintf("SVC_ID: %v", payload.SVCID))
	logger.Info(fmt.Sprintf("OP_ID: %v", payload.OPID))
	logger.Info(fmt.Sprintf("DENOM: %v", payload.DENOM))
	logger.Info(fmt.Sprintf("Charged Amount: %v", payload.CHARGEDAMT))
	logger.Info(fmt.Sprintf("Operation Datetime: %v", payload.OPDATETIME))
	logger.Info(fmt.Sprintf("Expiry Date: %v", payload.EXPIRYDATE))
	logger.Info(fmt.Sprintf("Status: %v", payload.STATUS))
	logger.Info(fmt.Sprintf("Error Code: %v", payload.ERRCODE))
	logger.Info(fmt.Sprintf("Origin: %v", payload.ORIGIN))
	logger.Info(fmt.Sprintf("Start Time: %v", payload.STARTTIME))
	logger.Info(fmt.Sprintf("HPLMN: %v", payload.HPLMN))
	logger.Info(fmt.Sprintf("Original Expiry: %v", payload.ORIGEXPIRY))
	logger.Info(fmt.Sprintf("Expiry: %v", payload.EXPIRY))
	logger.Info(fmt.Sprintf("Wallet Amount: %v", payload.WALLETAMOUNT))
	logger.Info(fmt.Sprintf("Wallet Keyword: %v", payload.WALLETKEYWORD))
}