package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"promo-collection-worker/internal/pkg/config"
	"promo-collection-worker/internal/pkg/consts"
	"promo-collection-worker/internal/pkg/kafka"
	"promo-collection-worker/internal/pkg/log_messages"
	"promo-collection-worker/internal/pkg/logger"
	"promo-collection-worker/internal/pkg/store/impl/collections"
	"promo-collection-worker/internal/pkg/store/models"
	"promo-collection-worker/internal/service/interfaces"
	"sync"
	"time"

	mongodb "promo-collection-worker/internal/pkg/db/mongo"

	"go.mongodb.org/mongo-driver/mongo"
)

type KafkaRetryServiceInterface interface {
	RetryKafkaCollectionMessages(ctx context.Context) *KafkaRetryResponse
}

type KafkaRetryService struct {
	CollectionRepo interfaces.CollectionsRepoInterface
	kafkaProducer  kafka.KafkaProducerInterface
	workerConfig   config.KafkaRetryServiceConfig
	cursorHandler  *DefaultCursorHandler
}

type DefaultCursorHandler struct{}

func (h *DefaultCursorHandler) StreamDocuments(
	ctx context.Context,
	cursor *mongo.Cursor,
	docChan chan<- models.CollectionWithLoanGUID,
	errorChan chan<- error,
) {
	defer close(docChan)

	hasDocuments := false

	for cursor.Next(ctx) {
		hasDocuments = true
		var doc models.CollectionWithLoanGUID
		if err := cursor.Decode(&doc); err != nil {
			logger.CtxError(ctx, log_messages.ErrorDecodingDocument, err)
			continue
		}

		select {
		case docChan <- doc:
		case <-ctx.Done():
			return
		}
	}

	if !hasDocuments {
		logger.CtxInfo(ctx, log_messages.NoCollectionInDuration)
	}

	if err := cursor.Err(); err != nil {
		select {
		case errorChan <- fmt.Errorf(log_messages.CursorError, err):
		case <-ctx.Done():
		default:
			logger.CtxError(ctx, log_messages.ErrorChannelFullLoggingCursorError, err)
		}
	}
}

func (h *DefaultCursorHandler) DecodeDocument(cursor *mongo.Cursor) (models.Collections, error) {
	var doc models.Collections
	if err := cursor.Decode(&doc); err != nil {
		return models.Collections{}, fmt.Errorf(log_messages.ErrorDecodingDocument, err)
	}
	return doc, nil
}

func NewKafkaRetryService(client *mongodb.MongoClient,
	kafkaProducer kafka.KafkaProducerInterface, cfg config.KafkaRetryServiceConfig) *KafkaRetryService {
	return &KafkaRetryService{
		CollectionRepo: collections.NewCollectionsRepository(client),
		kafkaProducer:  kafkaProducer,
		cursorHandler:  &DefaultCursorHandler{},
		workerConfig:   cfg,
	}
}

func NewKafkaRetryServiceWithDeps(
	collectionRepo interfaces.CollectionsRepoInterface,
	kafkaProducer kafka.KafkaProducerInterface,
	workerConfig config.KafkaRetryServiceConfig,
) *KafkaRetryService {
	return &KafkaRetryService{
		CollectionRepo: collectionRepo,
		kafkaProducer:  kafkaProducer,
		workerConfig:   workerConfig,
		cursorHandler:  &DefaultCursorHandler{},
	}
}

func (ks *KafkaRetryService) setupChannelsAndWorkers(
	ctx context.Context,
	response *KafkaRetryResponse,
) (chan models.CollectionWithLoanGUID, chan []string,
	chan []string, chan error, chan struct{}, *sync.WaitGroup, *[]error, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)

	workerCount := ks.workerConfig.WorkerCount
	bufferSize := ks.workerConfig.BufferSize

	docChan := make(chan models.CollectionWithLoanGUID, bufferSize)

	successChan := make(chan []string, bufferSize)
	failureChan := make(chan []string, bufferSize)
	errorChan := make(chan error, bufferSize)

	var wg sync.WaitGroup

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ks.processDocumentsWorker(ctx, docChan, successChan, failureChan, errorChan)
		}()
	}

	resultsDone := make(chan struct{})
	var resultErrors []error

	go ks.collectResults(ctx, response, successChan, failureChan, errorChan, resultsDone, &resultErrors)

	return docChan, successChan, failureChan, errorChan, resultsDone, &wg, &resultErrors, cancel
}

func (ks *KafkaRetryService) collectResults(
	ctx context.Context,
	response *KafkaRetryResponse,
	successChan chan []string,
	failureChan chan []string,
	errorChan chan error,
	resultsDone chan struct{},
	resultErrors *[]error,
) {
	defer close(resultsDone)

	closedCount := 0
	const totalChannels = 3

	for closedCount < totalChannels {
		select {
		case successIDs, ok := <-successChan:
			if !ok {
				successChan = nil
				closedCount++
				continue
			}
			response.SuccessIDs = append(response.SuccessIDs, successIDs...)

		case failedIDs, ok := <-failureChan:
			if !ok {
				failureChan = nil
				closedCount++
				continue
			}
			response.FailedIDs = append(response.FailedIDs, failedIDs...)

		case err, ok := <-errorChan:
			if !ok {
				errorChan = nil
				closedCount++
				continue
			}
			ks.handleErrorResult(ctx, response, resultErrors, err)

		case <-ctx.Done():
			return
		}
	}
}

func (ks *KafkaRetryService) handleErrorResult(
	ctx context.Context,
	response *KafkaRetryResponse,
	resultErrors *[]error,
	err error,
) {
	*resultErrors = append(*resultErrors, err)
	if response.ErrorMsg == "" {
		response.SetError(err)
	}
	logger.CtxError(ctx, log_messages.ErrorProcessingDocumentBatch, err)
}

func (ks *KafkaRetryService) RetryKafkaCollectionMessages(ctx context.Context) *KafkaRetryResponse {
	response := &KafkaRetryResponse{
		SuccessIDs: []string{},
		FailedIDs:  []string{},
	}
	if ks.workerConfig.WorkerCount <= 0 {
		logger.CtxError(ctx, log_messages.NoWorkerConfigured, errors.New(log_messages.NoWorkerConfigured))
		response.SetError(errors.New(log_messages.NoWorkerConfigured))
		return response
	}

	cursor, err := ks.CollectionRepo.GetFailedKafkaEntriesCursor(ctx, ks.workerConfig.RetryStartDate,
		ks.workerConfig.MongoBatchSize)
	if err != nil {
		response.SetError(err)
		return response
	}
	if cursor == nil {
		logger.CtxInfo(ctx, log_messages.CursorIsNilNoDocumentsToProcess)
		return response
	}

	defer func() {
		if cursor != nil {
			if err := cursor.Close(ctx); err != nil {
				logger.CtxError(ctx, log_messages.ErrorClosingCursor, err)
			}
		}
	}()

	docChan, successChan, failureChan, errorChan, resultsDone, wg, resultErrors, cancel :=
		ks.setupChannelsAndWorkers(ctx, response)
	defer cancel()

	streamWg := &sync.WaitGroup{}
	streamWg.Add(1)
	go func() {
		defer streamWg.Done()
		ks.cursorHandler.StreamDocuments(ctx, cursor, docChan, errorChan)
	}()
	streamWg.Wait()
	wg.Wait()
	close(successChan)
	close(failureChan)
	close(errorChan)
	<-resultsDone
	logger.CtxInfo(ctx, log_messages.KafkaRetryProcessingCompleted,
		slog.Int("successCount", len(response.SuccessIDs)),
		slog.Int("failureCount", len(response.FailedIDs)),
		slog.Int("errorCount", len(*resultErrors)))

	if len(*resultErrors) > 0 {
		logger.CtxWarn(ctx, log_messages.MultipleErrorsOccurredDuringProcessing,
			slog.Int("errorCount", len(*resultErrors)))
		for i, err := range *resultErrors {
			logger.CtxError(ctx, fmt.Sprintf("Error %d", i+1), err)
		}
	}

	return response
}

func (ks *KafkaRetryService) processAndPublishBatch(
	ctx context.Context,
	batch []models.CollectionWithLoanGUID,
	successChan chan<- []string,
	failureChan chan<- []string,
	errorChan chan<- error,
) {
	if len(batch) == 0 {
		return
	}

	successIDs := make([]string, 0, len(batch))
	failedIDs := make([]string, 0, len(batch))

	for _, doc := range batch {
		messageValue, failedID := ks.prepareKafkaMessages(ctx, doc)
		if failedID != "" {
			failedIDs = append(failedIDs, doc.ID.Hex())
			continue
		}

		err := ks.kafkaProducer.Publish(ctx, messageValue)
		if err != nil {
			failedIDs = append(failedIDs, doc.ID.Hex())
			select {
			case errorChan <- err:
			case <-ctx.Done():
				return
			default:
				logger.CtxError(ctx, log_messages.ErrorChannelFullLoggingErrorInstead, err)
			}
			continue
		}

		successIDs = append(successIDs, doc.ID.Hex())

	}

	ks.handleSuccessIDs(ctx, successIDs, failedIDs, successChan, failureChan, errorChan)
}

func (ks *KafkaRetryService) handleSuccessIDs(
	ctx context.Context,
	successIDs []string,
	failedIDs []string,
	successChan chan<- []string,
	failureChan chan<- []string,
	errorChan chan<- error,
) {

	if len(successIDs) > 0 {

		logger.CtxInfo(ctx, log_messages.IDsWhichPublishedToKafkaSuccessfully, slog.Any("successIDs", successIDs))

		select {
		case successChan <- successIDs:
		case <-ctx.Done():
			return
		}
		failedUpdateIDs, err := ks.CollectionRepo.UpdatePublishedToKafkaInBulk(ctx, successIDs)

		if err != nil {
			logger.CtxError(ctx,
				log_messages.ErrorUpdatingKafkaFlag,
				err,
				slog.Any("successIDs", successIDs),
			)
			select {
			case errorChan <- fmt.Errorf("%s: %w", log_messages.ErrorUpdatingKafkaFlag, err):
			case <-ctx.Done():
				return
			}
		} else if len(failedUpdateIDs) > 0 {
			logger.CtxWarn(ctx, log_messages.SomeRecordsFailedToUpdateKafkaFlag,
				slog.Any("failedUpdateIDs", failedUpdateIDs),
				slog.Int("totalFailed", len(failedUpdateIDs)),
				slog.Int("totalSuccess", len(successIDs)-len(failedUpdateIDs)),
			)
		}

	}

	if len(failedIDs) > 0 {
		logger.CtxWarn(ctx, log_messages.IDsWhichFailedToPublishToKafka, slog.Any("failedIDs", failedIDs))

		select {
		case failureChan <- failedIDs:
		case <-ctx.Done():
			return
		}
	}

}

func (ks *KafkaRetryService) processDocumentsWorker(
	ctx context.Context,
	docChan <-chan models.CollectionWithLoanGUID,
	successChan chan<- []string,
	failureChan chan<- []string,
	errorChan chan<- error,
) {
	maxBatchSize := ks.workerConfig.MaxBatchSize
	batch := make([]models.CollectionWithLoanGUID, 0, maxBatchSize)

	ticker := time.NewTicker(ks.workerConfig.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case doc, ok := <-docChan:
			if !ok {
				ks.processAndPublishBatch(ctx, batch, successChan, failureChan, errorChan)
				return
			}

			batch = append(batch, doc)

			if len(batch) >= maxBatchSize {
				ks.processAndPublishBatch(ctx, batch, successChan, failureChan, errorChan)
				batch = batch[:0]
			}

		case <-ticker.C:
			if len(batch) > 0 {
				ks.processAndPublishBatch(ctx, batch, successChan, failureChan, errorChan)
				batch = batch[:0]
			}

		case <-ctx.Done():
			if len(batch) > 0 {
				ks.processAndPublishBatch(ctx, batch, successChan, failureChan, errorChan)
			}
			return
		}
	}
}

func (ks *KafkaRetryService) prepareKafkaMessages(c context.Context,
	messages models.CollectionWithLoanGUID) ([]byte, string) {
	var failedID string

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	record := ks.kafkaPayload(messages)
	if err := writer.Write(record); err != nil {
		logger.CtxError(c, log_messages.FailedToWriteRecordToCSV, err)
		failedID = messages.ID.Hex()
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		logger.CtxError(c, log_messages.ErrorFlushingCSVWriter, err)
		failedID = messages.ID.Hex()
	}
	messageValue := bytes.TrimSuffix(buf.Bytes(), []byte("\n"))
	return messageValue, failedID
}

func (ks *KafkaRetryService) kafkaPayload(messages models.CollectionWithLoanGUID) []string {
	result := consts.ResultTrue
	if !messages.Result {
		result = consts.ResultFalse
	}
	var collectedServicFee, unpaidLoanAmount int
	var transactionId, tokenPaymentId string
	if messages.CollectionCategory == consts.CollectionCategoryData {
		collectedServicFee = int(math.Round(messages.CollectedServiceFee))
		unpaidLoanAmount = int(math.Round(messages.UnpaidLoanAmount))
		transactionId = messages.TransactionId
		tokenPaymentId = messages.TokenPaymentId
	} else {
		collectedServicFee = int(messages.ServiceFee) - int(messages.UnpaidServiceFee)
		unpaidLoanAmount = int(messages.TotalUnpaid) - (int(messages.ServiceFee) - collectedServicFee)
		transactionId = messages.KafkaTransactionId
		tokenPaymentId = ""
		if messages.CollectionCategory == "" {
			messages.CollectionCategory = consts.CollectionCategoryLoad
		}
	}
	payload := []string{
		transactionId,
		messages.LoanGUID,
		messages.CreatedAt.Format(consts.DateTimeFormat),
		messages.Method,
		fmt.Sprintf("%d", messages.Ageing),
		fmt.Sprintf("%d", int(math.Round(messages.TotalCollectedAmount))),
		fmt.Sprintf("%d", collectedServicFee),
		fmt.Sprintf("%d", int(math.Round(messages.TotalUnpaid))),
		fmt.Sprintf("%d", int(math.Round(messages.UnpaidServiceFee))),
		result,
		messages.ErrorText,
		messages.MSISDN,
		messages.CollectionType,
		messages.PaymentChannel,
		messages.CollectionCategory,
		fmt.Sprintf("%d", int(math.Round(messages.CollectedAmount))),
		fmt.Sprintf("%d", unpaidLoanAmount),
		tokenPaymentId,
	}
	if messages.CollectionCategory == consts.CollectionCategoryData {
		payload = append(payload, fmt.Sprintf("%d", int(math.Round(messages.DataCollected))))
	} else {
		payload = append(payload, "")
	}
	return payload
}
