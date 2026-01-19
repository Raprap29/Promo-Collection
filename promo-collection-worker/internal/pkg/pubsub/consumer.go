package pubsub

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"promo-collection-worker/internal/pkg/logger"
	"promo-collection-worker/internal/pkg/models"
	"promo-collection-worker/internal/service/interfaces"
	pubsubServicePkg "promo-collection-worker/internal/service/pubsub"

	"cloud.google.com/go/pubsub/v2"
)

// PubSubClientFactory makes new clients (mockable in tests).
type PubSubClientFactory interface {
	NewPubSubClient(ctx context.Context, projectID string) (interfaces.PubSubClientInterface, error)
}

// defaultPubSubClientFactory creates Google Pub/Sub clients.
type defaultPubSubClientFactory struct{}

func (f *defaultPubSubClientFactory) NewPubSubClient(ctx context.Context,
	projectID string) (interfaces.PubSubClientInterface, error) {
	sdkClient, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return &pubSubClientAdapter{client: sdkClient}, nil
}

// pubSubClientAdapter wraps *pubsub.Client
type pubSubClientAdapter struct {
	client *pubsub.Client
}

func (c *pubSubClientAdapter) Subscriber(subscription string) interfaces.SubscriberInterface {
	return &subscriberAdapter{sub: c.client.Subscriber(subscription)}
}

func (c *pubSubClientAdapter) Close() error {
	return c.client.Close()
}

type subscriberAdapter struct {
	sub *pubsub.Subscriber
}

func (s *subscriberAdapter) Receive(ctx context.Context, f func(context.Context, interfaces.MessageInterface)) error {
	return s.sub.Receive(ctx, func(ctx context.Context, m *pubsub.Message) {
		// Add publish time to context using shared key from models
		ctx = context.WithValue(ctx, models.PublishTimeKey, m.PublishTime)
		f(ctx, &messageAdapter{msg: m})
	})
}

type messageAdapter struct {
	msg *pubsub.Message
}

func (s *subscriberAdapter) SetMaxExtension(d time.Duration) {
	s.sub.ReceiveSettings.MaxExtension = d
}

func (m *messageAdapter) Data() []byte {
	return m.msg.Data
}

func (m *messageAdapter) Ack() {
	m.msg.Ack()
}

func (m *messageAdapter) Nack() {
	m.msg.Nack()
}

// PubSubConsumer manages subscription consuming with lifecycle.
type PubSubConsumer struct {
	PubSubClient interfaces.PubSubClientInterface
	Ctx          context.Context
	Cancel       context.CancelFunc
}

// NewPubSubConsumer is the default constructor for production use.
// Declared as a variable so tests can replace it.
var NewPubSubConsumer = func(ctx context.Context, projectID string) (*PubSubConsumer, error) {
	factory := &defaultPubSubClientFactory{}
	return NewPubSubConsumerWithFactory(ctx, projectID, factory)
}

// Construct with a factory (testable).
func NewPubSubConsumerWithFactory(ctx context.Context, projectID string,
	factory PubSubClientFactory) (*PubSubConsumer, error) {
	client, err := factory.NewPubSubClient(ctx, projectID)
	if err != nil {
		logger.CtxError(ctx, "Failed creating PubSub client", err)
		return nil, err
	}

	consumerCtx, cancel := context.WithCancel(ctx)
	return &PubSubConsumer{
		PubSubClient: client,
		Ctx:          consumerCtx,
		Cancel:       cancel,
	}, nil
}

// Consume one subscription once.
func (c *PubSubConsumer) Consume(ctx context.Context, subscription string,
	handler func(ctx context.Context, msg []byte) error) error {
	sub := c.PubSubClient.Subscriber(subscription)
	sub.SetMaxExtension(-1)
	return sub.Receive(ctx, func(ctx context.Context, m interfaces.MessageInterface) {
		if err := handler(ctx, m.Data()); err != nil {
			// Import the MessageIgnoreError type from the service package
			var ignoreErr *pubsubServicePkg.MessageIgnoreError
			if errors.As(err, &ignoreErr) {
				if ignoreErr.Err != nil {
					logger.CtxInfo(ctx, "Message processing ignored, will be redelivered",
						slog.String("reason", ignoreErr.Err.Error()))
					// Don't ack or nack - let it be redelivered after timeout
					return
				}
				logger.CtxInfo(ctx, "Message processing ignored, will be redelivered")
				// Don't ack or nack - let it be redelivered after timeout
				return
			}
			// Regular error - NACK for immediate retry
			logger.CtxInfo(ctx, "Nacked the message for immediate retry")
			m.Nack()
			return
		}
		logger.CtxInfo(ctx, "Acked the message")
		m.Ack()
	})
}

// StartConsumer continuously listens for messages until the context is cancelled.
// It automatically restarts consumption whether Consume exits with or without error.
func (c *PubSubConsumer) StartConsumer(subscription string, handler func(ctx context.Context, msg []byte) error) {
	go func() {
		logger.CtxInfo(c.Ctx, fmt.Sprintf("PubSub consumer starting for subscription: %s", subscription))

		for {
			// Exit if context is done
			if c.Ctx.Err() != nil {
				logger.CtxInfo(c.Ctx, "PubSub consumer loop exiting due to context cancellation")
				return
			}

			// Try to consume
			err := c.Consume(c.Ctx, subscription, handler)
			if err != nil {
				logger.CtxError(c.Ctx, "Error consuming messages, retrying in 5s", err)
			} else {
				logger.CtxInfo(c.Ctx, "PubSub consumer stopped without error, restarting in 5s...")
			}

			time.Sleep(5 * time.Second)
		}
	}()
}

func (c *PubSubConsumer) Close() error {
	if c.Cancel != nil {
		c.Cancel()
	}
	return c.PubSubClient.Close()
}
