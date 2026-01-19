package service

type KafkaRetryResponse struct {
	SuccessIDs []string `json:"success_ids,omitempty"` // IDs of collections that were published to Kafka successfully
	FailedIDs  []string `json:"failed_ids,omitempty"`  // IDs of collections that failed to publish to Kafka
	ErrorMsg   string   `json:"error,omitempty"`       // Error message from the Kafka retry service
	Message    string   `json:"message,omitempty"`     // Message from the Kafka retry service
}

func (r *KafkaRetryResponse) SetError(err error) {
	if err != nil {
		r.ErrorMsg = err.Error()
	}
}
