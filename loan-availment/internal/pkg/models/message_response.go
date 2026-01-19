package models

// Message contains information about the message ID and required parameters
type MessageResponse struct {
	MessageID  int32
	Parameters []string // Required parameters for the message
}
