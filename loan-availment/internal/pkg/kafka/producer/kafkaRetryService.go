package producer

import (
	"context"
	"fmt"
	"globe/dodrio_loan_availment/configs"
	"globe/dodrio_loan_availment/internal/pkg/common"
	"globe/dodrio_loan_availment/internal/pkg/logger"
	"globe/dodrio_loan_availment/internal/pkg/store"
	storeModel "globe/dodrio_loan_availment/internal/pkg/store/models"
)

type AvailmentStoreInterface interface {
	GetFailedKafkaEntries(context.Context, int32) ([]storeModel.AvailmentTransaction, error)
	SetKafkaFlag(context.Context, []string) ([]string, error)
}

type KafkaRetryService struct {
	availementStore  AvailmentStoreInterface
	walletRepository *store.WalletTypesRepository
}

func NewKafkaRetryService(availementStore AvailmentStoreInterface, walletRepository *store.WalletTypesRepository) *KafkaRetryService {
	return &KafkaRetryService{availementStore: availementStore, walletRepository: walletRepository}
}
func (ks *KafkaRetryService) RetryKafkaAvailmentMessage(ctx context.Context) ([]string, []string, error) {
	topic := configs.KAFKA_TOPIC
	server := configs.KAFKA_SERVER
	if KafkaProducer != nil {
		producer, err := NewKafkaProducer(server, topic)
		if err != nil {
			logger.Error(ctx, "failed to create Kafka Producer error: %v", err)
		}
		logger.Info(ctx, "Kafka Producer Created")
		defer producer.Close()
		KafkaProducer = producer
	}

	// get  the transactionIDs
	availments, err := ks.availementStore.GetFailedKafkaEntries(ctx, int32(configs.KAFKA_RETRY_DURATION))
	if err != nil {
		return nil, nil, err
	}
	if len(availments) <= 0 {
		return nil, nil, fmt.Errorf("no availments found in the duration")
	}
	filter := map[string]interface{}{}
	wallet, _ := ks.walletRepository.WalletTypesByFilter(ctx, filter)

	availmentMessages, err := common.PrepareAvailmentMessage(wallet, availments)
	if err != nil {
		return nil, nil, err
	}
	if len(availmentMessages) <= 0 {
		return nil, nil, fmt.Errorf("no availments found in the duration")
	}
	// try sending to kafka
	successMessagesId, failedMessagesId, err := SendMessageBatch(ctx, KafkaProducer, availmentMessages, topic, 2)
	if err != nil {
		return nil, nil, err
	}
	// update flag
	failedList, err := ks.availementStore.SetKafkaFlag(ctx, successMessagesId)
	if err != nil {
		return successMessagesId, failedMessagesId, fmt.Errorf("error updating Kafka flag in database for transactions %v with error %v", failedList, err)
	}
	return successMessagesId, failedMessagesId, nil
}
