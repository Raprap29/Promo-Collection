package log_messages

const (
	FailureInPubsubConsumerCreation = "failed to create Pubsub consumer: %v"
	PubsubErrorConsuming            = "pubsub consumer error in consuming: %v"
	ErrorSerializingPubsubMessage   = "error serializing Pubsub message: %v"
	ErrorUnmarshalingPubsubMessage  = "error unmarshaling Pubsub message: %v"
	ServerStartFailure              = "failed to start server: %v"
	ServerShutdown                  = "Shutting down server..."
	ServerForcedShutdown            = "Server forced to shutdown: %v"
	ServerExiting                   = "Server exiting"
	FailedLoadingConfiguration      = "Failed to load configuration: %v"
	CleanupStarted                  = "Starting cleanup of resources..."
	CleanupCompleted                = "All resources cleaned up successfully"
)
