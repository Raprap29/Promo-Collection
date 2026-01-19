package interfaces

import (
	"context"
	"promo-collection-worker/internal/pkg/models"
)

// KafkaPublisherInterface defines the methods for publishing messages to Kafka.
type KafkaPublisherInterface interface {
	PublishSuccess(ctx context.Context, msg models.KafkaMessageForPublishing) error
	PublishFailure(ctx context.Context, msg models.KafkaMessageForPublishing) error
}
