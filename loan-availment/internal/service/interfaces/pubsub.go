package interfaces

import (
	"context"
)
type PublisherInterface interface {
	Publisher(topic string) TopicPublisherInterface
	Close() error
}

type TopicPublisherInterface interface {
	Publish(ctx context.Context, data []byte, attributes map[string]string) (string, error)
}
