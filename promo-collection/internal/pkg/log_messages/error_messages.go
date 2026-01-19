package log_messages

const (
	FailureInKafkaConsumerCreation          = "failed to create Kafka consumer %v"
	KafkaErrorConsuming                     = "kafka consumer error in consuming %v"
	ErrorSerializingKafkaMessage            = "error serializing Kafka message %v"
	ErrorUnmarshalingKafkaMessage           = "error unmarshaling Kafka message: %v"
	TopicDoesNotExists                      = "pubsub topic does not exist: %v"
	ErrorMarshallingMessage                 = "failed to marshal message: %v"
	ErrorInMessagePublishing                = "failed to publish message: %v"
	ErrorInBuildingPubSubMsg                = "failure in building pubsub message: %v"
	ErrorPubSubClientCreation               = "error creating pubsub client: %v"
	ErrorFetchingSystemLevelRulesMongoDBDoc = "error fetching document from systemlevelrules mongoDB: %v"
	ErrorFetchingLoansMongoDBDoc            = "error fetching document from loans mongoDB: %v"
	EmptyDocumentFoundFromDb                = "no associated mongodb document found: %v"
	NoUnpaidBalanceFound                    = "no unpaid balance found for loan: %s"
	NoConversionRateInLoans                 = "no conversion rate found in loans"
	NoConversionRateInSystemLevelRules      = "no conversion rate found in systemlevelrules"
)
