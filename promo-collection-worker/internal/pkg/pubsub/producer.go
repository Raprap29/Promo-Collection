package pubsub

import (
	"context"

	"promo-collection-worker/internal/pkg/log_messages"
	"promo-collection-worker/internal/pkg/logger"
	"promo-collection-worker/internal/service/interfaces"

	"cloud.google.com/go/pubsub/v2"
)

// PubSubPublisher manages topic publishing with lifecycle.
type PubSubPublisher struct {
	PubSubClient interfaces.PubSubPublisherClientInterface
	Ctx          context.Context
	Cancel       context.CancelFunc
}

// PubSubPublisherClientFactory makes new clients (mockable in tests).
type PubSubPublisherClientFactory interface {
	NewPubSubPublisherClient(ctx context.Context, projectID string) (interfaces.PubSubPublisherClientInterface, error)
}

// defaultPubSubPublisherClientFactory creates Google Pub/Sub clients for publishing.
type defaultPubSubPublisherClientFactory struct{}

func (f *defaultPubSubPublisherClientFactory) NewPubSubPublisherClient(ctx context.Context,
	projectID string) (interfaces.PubSubPublisherClientInterface, error) {
	sdkClient, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return &pubSubPublisherClientAdapter{client: sdkClient}, nil
}

// pubSubPublisherClientAdapter wraps *pubsub.Client for publishing
type pubSubPublisherClientAdapter struct {
	client *pubsub.Client
}

func (c *pubSubPublisherClientAdapter) Publisher(topic string) interfaces.PublisherInterface {
	return &publisherAdapter{publisher: c.client.Publisher(topic)}
}

func (c *pubSubPublisherClientAdapter) Close() error {
	return c.client.Close()
}

type publisherAdapter struct {
	publisher *pubsub.Publisher
}

func (p *publisherAdapter) Publish(ctx context.Context, msg []byte) error {
	result := p.publisher.Publish(ctx, &pubsub.Message{
		Data: msg,
	})
	// Wait for the result
	_, err := result.Get(ctx)
	return err
}

// NewPubSubPublisher is the default constructor for production use.
// Declared as a variable so tests can replace it.
var NewPubSubPublisher = func(ctx context.Context, projectID string) (*PubSubPublisher, error) {
	factory := &defaultPubSubPublisherClientFactory{}
	return NewPubSubPublisherWithFactory(ctx, projectID, factory)
}

// Construct with a factory (testable).
func NewPubSubPublisherWithFactory(ctx context.Context, projectID string,
	factory PubSubPublisherClientFactory) (*PubSubPublisher, error) {
	client, err := factory.NewPubSubPublisherClient(ctx, projectID)
	if err != nil {
		logger.CtxError(ctx, "Failed creating PubSub client", err)
		return nil, err
	}
	logger.CtxInfo(ctx, log_messages.PubsubPublisherCreated)

	publisherCtx, cancel := context.WithCancel(ctx)
	return &PubSubPublisher{
		PubSubClient: client,
		Ctx:          publisherCtx,
		Cancel:       cancel,
	}, nil
}

// Publish publishes a message to a topic.
func (p *PubSubPublisher) Publish(ctx context.Context, topic string, msg []byte) error {
	publisher := p.PubSubClient.Publisher(topic)
	return publisher.Publish(ctx, msg)
}

func (p *PubSubPublisher) Close() error {
	if p.Cancel != nil {
		p.Cancel()
	}
	return p.PubSubClient.Close()
}
