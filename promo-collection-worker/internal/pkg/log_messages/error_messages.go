package log_messages

const (
	FailureInPubsubConsumerCreation       = "failed to create Pubsub consumer: %v"
	PubsubErrorConsuming                  = "pubsub consumer error in consuming: %v"
	ErrorSerializingPubsubMessage         = "error serializing Pubsub message: %v"
	ErrorUnmarshalingPubsubMessage        = "error unmarshaling Pubsub message: %v"
	ServerStartFailure                    = "failed to start server: %v"
	ServerShutdown                        = "Shutting down server..."
	ServerForcedShutdown                  = "Server forced to shutdown: %v"
	ServerExiting                         = "Server exiting"
	FailedLoadingConfiguration            = "Failed to load configuration: %v"
	CleanupStarted                        = "Starting cleanup of resources..."
	CleanupCompleted                      = "All resources cleaned up successfully"
	ErrorDecrementWalletSubscription      = "DecrementWalletSubscription failure response"
	ErrorNonAPIErrorDuringWalletDecrement = "Non-API error during wallet decrement - code error: %v"

	// Error messages for retrieve subscriber usage
	ErrorNonAPIErrorDuringRetrieveSubscriberUsage = "Non-API error during retrieve subscriber usage - code error: %v"
	ErrorRetrieveSubscriberUsage                  = "RetrieveSubscriberUsage error"

	// Error codes and messages for wallet bucket issues
	ErrorCodeMissingSubscriberWalletBucket    = "MISSING_SUBSCRIBER_WALLET_BUCKET"
	ErrorMessageMissingSubscriberWalletBucket = "Subscriber does not have matching data wallet to deduct from " +
		"(wallet may have expired or exhausted)"

	// Database operation errors
	ErrorFailedToCreateCollectionEntry       = "Failed to create collection entry: %v"
	ErrorFailedToDeleteTransactionInProgress = "Failed to delete transaction in progress: %v"

	// Subscriber usage response errors
	ErrorNoBucketsFoundInSubscriberUsageResponse = "no buckets found in subscriber usage response"
	ErrorNoMatchingBucketFoundForWalletKeyword   = "no matching bucket found for wallet keyword: %s"
	ErrorFailedToParseWalletAmount               = "Failed to parse wallet amount: %v"
	ErrorInsufficientBalanceForDeduction         = "Insufficient balance for deduction"
	ErrorDeductionDidntHappenInsufficientBalance = "deduction didn't happen: insufficient balance"
	ErrorDecrementWalletSubscriptionError        = "DecrementWalletSubscription error: %v"
	OutboundConnError                            = "OUTBOUND_CONN_ERROR"

	// Error handling messages
	ErrorHandlingBusinessOrNonOutboundConnError = "Handling business error or non outbound conn error of " +
		"decrement wallet subscription"
	ErrorHandlingNonOutboundConnError = "Handling non outbound conn error of decrement wallet subscription"
	ErrorHandlingOutboundConnError    = "Handling outbound conn error of decrement wallet subscription"
	ErrorHandlingSystemErrorCode      = "Handling system error of decrement wallet subscription"

	// Request/response errors
	ErrorFailedToBuildURLWithQueryParams                  = "failed to build URL with query parameters: %v"
	ErrorPreparingToSendRetrieveSubscriberUsageRequest    = "Preparing to send RetrieveSubscriberUsage request"
	ErrorFailedToBuildRetrieveSubscriberUsageRequest      = "failed to build RetrieveSubscriberUsage request: %v"
	ErrorFailedToSendRetrieveSubscriberUsageRequest       = "failed to send request to RetrieveSubscriberUsage: %v"
	ErrorFailedToCloseRetrieveSubscriberUsageResponseBody = "failed to close RetrieveSubscriberUsage response body: %v"
	ReceivedRetrieveSubscriberUsageResponse               = "Received RetrieveSubscriberUsage response"

	// Decode errors
	ErrorFailedToDecodeRetrieveSubscriberUsageSuccessResponse = "failed to decode RetrieveSubscriberUsage " +
		"success response: %v"
	ErrorFailedToDecodeRetrieveSubscriberUsageErrorResponse = "failed to decode RetrieveSubscriberUsage " +
		"error response: %v"

	// Unknown response errors from retrieve subscriber usage
	ErrorUnknownResponseStructFormat                  = "UNKNOWN_RESPONSE_STRUCT_FORMAT_OR_NON_API_ERROR"
	ErrorTextUnknownResponseOrNonAPIErrorStructFormat = "Unknown response struct format/ non -api error from " +
		"retrieve subscriber usage"

	// Unknown response errors from decrement wallet subscription
	// Error codes for unknown response format
	ErrorUnknownResponseStructFormatDWS = "UNKNOWN_RESPONSE_STRUCT_FORMAT_OR_NON_API_ERROR"
	// Error text for unknown response format from decrement wallet subscription
	ErrorTextUnknownResponseOrNonAPIErrorStructFormatDWS = "Unknown response struct format/ non -api error from " +
		"decrement wallet subscription"

	ErrorFetchingLoansMongoDbDocument             = "Error fetching loans mongodb document: %v"
	ErrorFetchingLatestUnpaidLoansMongoDbDocument = "Error fetching latest unpaid loans mongodb document: %v"
	ErrorFetchingSLRMongoDbDocument               = "Error fetching system level rules mongodb document: %v"

	PublishingToGCSBucketForExternalProcessing = "Publishing to gcs bucket for external processing"
	ErrorUnknownFormatError                    = "API returned unknown format error"
	ErrorApiReturnedError                      = "API returned error: %s"
	ErrorUpdatingLoansMongoDbDocument          = "Error updating loans document: %v"
	ErrorCreatingUnpaidLoansMongoDbDocument    = "Error inserting unpaid loans document: %v"
	ErrorCreatingClosedLoansMongoDbDocument    = "Error inserting closed loans document: %v"
	ErrorUpdatingCollectionsMongoDbDocument    = "Error updating the collections document: %v"
	ErrorUpdatingUnpaidLoansMongoDbDocument    = "Error updating the unpaid loans document: %v"
	InvalidDurationFormat                      = "Invalid duration format: %s"
	InvalidObjectID                            = "Invalid ObjectID: %s"
	SettingKafkaFlagForCollectionIDs           = "Setting Kafka flag for collection IDs"
	NoCollectionInDuration                     = "no collections with publishedToKafka flag false found in this duration"
	ErrorUpdatingKafkaFlag                     = "error updating Kafka flag in database for transactions with error"
	FailedToUpdateKafkaSuccessfully            = "failed to update for these ids after sending to kafka successfully"
	FailedToGetFailedKafkaEntries              = "Failed to get failed kafka entries: %v"
	FailedToWriteRecordToCSV                   = "Failed to write record to CSV"
	ErrorFlushingCSVWriter                     = "Error flushing CSV writer"
	KafkaPublishingMessage                     = "Kafka Publishing message"
	TimeoutWaitingForDeliveryEventForKey       = "Timeout waiting for delivery event for key"
	InvalidEvent                               = "Invalid event"
	DeliveryFailed                             = "Delivery failed"
	MessageDeliveredToTopic                    = "Message delivered to topic, partition, offset"
	UnexpectedEventType                        = "Unexpected event type"
	FailedToProduceKafkaMessage                = "Failed to produce Kafka message"
	FailedToUpdateKafkaFlag                    = "Failed to update Kafka flag in database for transactions with error"
	ErrorDecodingDocument                      = "error decoding document: %w"
	CursorError                                = "cursor error: %w"
	ErrorChannelFullLoggingCursorError         = "Error channel full, logging cursor error"
	ErrorProcessingDocumentBatch               = "Error processing document batch"
	CursorIsNilNoDocumentsToProcess            = "Cursor is nil, no documents to process"
	ErrorClosingCursor                         = "Error closing cursor"
	ErrorChannelFullLoggingErrorInstead        = "Error channel full, logging error instead"
	IDsWhichPublishedToKafkaSuccessfully       = "IDs which published to kafka successfully"
	IDsWhichFailedToPublishToKafka             = "IDs which failed to publish to kafka"
	MultipleErrorsOccurredDuringProcessing     = "Multiple errors occurred during processing"
	SomeRecordsFailedToUpdateKafkaFlag         = "Some records failed to update Kafka flag"
	NoWorkerConfigured                         = "no worker has been configured. " +
		"please check the worker count in environment variable"
	ErrorUploadingToGCSBucket = "Failed to upload to GCS bucket"
	ErrorClosingGCSClient     = "Failed to close GCS client"
	ErrorClosingGCSWriter     = "Failed to close GCS writer"
	ErrorMarshallingJSON      = "Failed to marshal JSON"
)
