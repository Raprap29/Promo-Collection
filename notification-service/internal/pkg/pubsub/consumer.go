package pubsub

import (
	"context"
	"notificationservice/internal/pkg/logger"
	"notificationservice/internal/service/interfaces"

	"cloud.google.com/go/pubsub/v2"
)

// Factory type for creating a PubSub client
type PubSubClientFactory interface {
	NewClient(ctx context.Context, projectID string) (interfaces.PubSubClientInterface, error)
}

// Default factory implementation
type defaultPubSubClientFactory struct{}

func (f *defaultPubSubClientFactory) NewClient(ctx context.Context,
	projectID string) (interfaces.PubSubClientInterface, error) {
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return &defaultPubSubClient{client: client}, nil
}

// defaultPubSubClient wraps the real pubsub.Client
type defaultPubSubClient struct {
	client *pubsub.Client
}

func (c *defaultPubSubClient) Subscriber(subscription string) interfaces.SubscriberInterface {
	return &defaultSubscriber{sub: c.client.Subscriber(subscription)}
}

func (c *defaultPubSubClient) Close() error {
	return c.client.Close()
}

type defaultSubscriber struct {
	sub *pubsub.Subscriber
}

func (s *defaultSubscriber) Receive(ctx context.Context, f func(context.Context, interfaces.MessageInterface)) error {
	return s.sub.Receive(ctx, func(ctx context.Context, m *pubsub.Message) {
		// Wrap the real message
		msg := &defaultMessage{msg: m}
		f(ctx, msg)
	})
}

// defaultMessage wraps the real pubsub.Message
type defaultMessage struct {
	msg *pubsub.Message
}

func (m *defaultMessage) Data() []byte {
	return m.msg.Data
}

func (m *defaultMessage) Ack() {
	m.msg.Ack()
}

func (m *defaultMessage) Nack() {
	m.msg.Nack()
}

type PubSubConsumer struct {
	client interfaces.PubSubClientInterface
	sub    interfaces.SubscriberInterface
	ctx    context.Context
	cancel context.CancelFunc
}

// NewPubSubConsumer uses the default factory
func NewPubSubConsumer(ctx context.Context, projectID string) (*PubSubConsumer, error) {
	factory := &defaultPubSubClientFactory{}
	client, err := factory.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Create a cancellable context for this consumer
	consumerCtx, cancel := context.WithCancel(ctx)

	return &PubSubConsumer{
		client: client,
		ctx:    consumerCtx,
		cancel: cancel,
	}, nil
}

func NewPubSubConsumerWithFactory(
	ctx context.Context,
	projectID string,
	factory PubSubClientFactory,
) (*PubSubConsumer, error) {
	client, err := factory.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Create a cancellable context for this consumer
	consumerCtx, cancel := context.WithCancel(ctx)

	return &PubSubConsumer{
		client: client,
		ctx:    consumerCtx,
		cancel: cancel,
	}, nil
}

func (g *PubSubConsumer) Consume(ctx context.Context, subscription string,
	handler func(ctx context.Context, msg []byte) error) error {
	g.sub = g.client.Subscriber(subscription)
	return g.sub.Receive(g.ctx, func(ctx context.Context, m interfaces.MessageInterface) {
		if err := handler(ctx, m.Data()); err != nil {
			m.Nack()
		} else {
			m.Ack()
		}
	})
}

// Unsubscribe gracefully stops receiving messages
func (g *PubSubConsumer) Unsubscribe(ctx context.Context) error {
	if g.cancel != nil {
		g.cancel() // This will stop the Receive loop
		logger.CtxInfo(ctx, "PubSub consumer unsubscribed gracefully")
	}
	return nil
}

func (g *PubSubConsumer) Close() error {
	// First unsubscribe if not already done
	if g.cancel != nil {
		g.cancel()
	}

	// Then close the client
	return g.client.Close()
}
