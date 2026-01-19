package interfaces

import (
	"context"

	mongodb "promo-collection-worker/internal/pkg/db/mongo"
	redis "promo-collection-worker/internal/pkg/db/redis"
	"promo-collection-worker/internal/pkg/models"
	servicekafka "promo-collection-worker/internal/service/kafka"
)

// RuntimePubSubPublisher is a small interface representing the methods
// the runtime provides for publishing notifications to Pub/Sub. It's
// intentionally minimal to avoid import cycles with pkg/pubsub.
type RuntimePubSubPublisher interface {
	Publish(ctx context.Context, topic string, msg []byte) error
	Close() error
}

// SuccessHandlerFunc is the callback signature used by the pubsub consumer
// and error handlers to invoke the success handling logic. It uses concrete
// project types to avoid casting in callers.
type SuccessHandlerFunc func(
	mClient *mongodb.MongoClient,
	rClient *redis.RedisClient,
	pubSub RuntimePubSubPublisher,
	kafkaSvc *servicekafka.CollectionWorkerKafkaService,
	msg *models.PromoCollectionPublishedMessage,
	ctx context.Context,
	notificationTopic string,
) error
