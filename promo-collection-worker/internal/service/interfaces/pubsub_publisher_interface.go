package interfaces

import "context"

// PublisherInterface defines the methods we need from pubsub.Publisher
type PublisherInterface interface {
	Publish(ctx context.Context, msg []byte) error
}

// PubSubPublisherClientInterface defines the methods we need from pubsub.Client for publishing
type PubSubPublisherClientInterface interface {
	Publisher(topic string) PublisherInterface
	Close() error
}
