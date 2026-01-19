package interfaces

import "context"

// MessageInterface defines the methods we need from pubsub.Message

type MessageInterface interface {
	Data() []byte
	Ack()
	Nack()
}

// SubscriberInterface defines the methods we need from pubsub.Subscriber
type SubscriberInterface interface {
	Receive(ctx context.Context, f func(context.Context, MessageInterface)) error
}

// PubSubClientInterface defines the methods we need from pubsub.Client
type PubSubClientInterface interface {
	Subscriber(subscription string) SubscriberInterface
	Close() error
}
