package app

import (
	"context"
)

type KafkaRetryService interface {
	RetryKafkaAvailmentMessage(context.Context) ([]string, []string, error)
}
