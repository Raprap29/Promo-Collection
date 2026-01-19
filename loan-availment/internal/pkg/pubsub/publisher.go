package pubsub

import (
	"context"
	"globe/dodrio_loan_availment/internal/pkg/logger"
	"globe/dodrio_loan_availment/internal/service/interfaces"

	"cloud.google.com/go/pubsub/v2"
)

// PubSubPublisherInterface defines the interface for PubSub publishing operations
type PubSubPublisherInterface interface {
	Publish(ctx context.Context, topic string, data []byte, attributes map[string]string) (string, error)
	Stop(ctx context.Context) error
	Close() error
}

// PublisherFactory type for creating a PubSub publisher
type PublisherFactory interface {
	NewPublisher(ctx context.Context, projectID string) (interfaces.PublisherInterface, error)
}

// Default publisher factory implementation
type defaultPublisherFactory struct{}

func (f *defaultPublisherFactory) NewPublisher(ctx context.Context, projectID string) (interfaces.PublisherInterface, error) {
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return &defaultPublisher{client: client}, nil
}

// defaultPublisher wraps the real pubsub.Client
type defaultPublisher struct {
	client *pubsub.Client
}

func (p *defaultPublisher) Publisher(topic string) interfaces.TopicPublisherInterface {
	return &defaultTopicPublisher{
		topic:  topic,
		client: p.client,
	}
}

func (p *defaultPublisher) Close() error {
	return p.client.Close()
}

// defaultTopicPublisher wraps the real pubsub.Topic
type defaultTopicPublisher struct {
	topic  string
	client *pubsub.Client
}

func (tp *defaultTopicPublisher) Publish(ctx context.Context, data []byte, attributes map[string]string) (string, error) {
	publisher := tp.client.Publisher(tp.topic)

	// Publish message
	res := publisher.Publish(ctx, &pubsub.Message{
		Data:       data,
		Attributes: attributes,
	})
	
	// Wait for the publish to complete
	messageID, err := res.Get(ctx)
	if err != nil {
		return "", err
	}

	return messageID, nil
}

// PubSubPublisher manages publishing to Google Cloud Pub/Sub
type PubSubPublisher struct {
	client interfaces.PublisherInterface
	ctx    context.Context
	cancel context.CancelFunc
}

// NewPubSubPublisher uses the default factory
func NewPubSubPublisher(ctx context.Context, projectID string) (*PubSubPublisher, error) {
	factory := &defaultPublisherFactory{}
	client, err := factory.NewPublisher(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Create a cancellable context for this publisher
	publisherCtx, cancel := context.WithCancel(ctx)

	return &PubSubPublisher{
		client: client,
		ctx:    publisherCtx,
		cancel: cancel,
	}, nil
}

func NewPubSubPublisherWithFactory(
	ctx context.Context,
	projectID string,
	factory PublisherFactory,
) (*PubSubPublisher, error) {
	client, err := factory.NewPublisher(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Create a cancellable context for this publisher
	publisherCtx, cancel := context.WithCancel(ctx)

	return &PubSubPublisher{
		client: client,
		ctx:    publisherCtx,
		cancel: cancel,
	}, nil
}

// Publish sends a single message to the specified topic
func (p *PubSubPublisher) Publish(ctx context.Context, topic string, data []byte, attributes map[string]string) (string, error) {
	topicPublisher := p.client.Publisher(topic)

	messageID, err := topicPublisher.Publish(context.Background(), data, attributes)
	if err != nil {
		logger.Error(ctx, "Failed to publish message to topic %s: %v", topic, err)
		return "", err
	}

	logger.Info(ctx, "Successfully published message to topic %s with ID: %s", topic, messageID)
	return messageID, nil
}

// Stop gracefully stops the publisher
func (p *PubSubPublisher) Stop(ctx context.Context) error {
	if p.cancel != nil {
		p.cancel() // This will cancel any ongoing operations
		logger.Info(ctx, "PubSub publisher stopped gracefully")
	}
	return nil
}

// Close closes the publisher and releases resources
func (p *PubSubPublisher) Close() error {
	// First stop if not already done
	if p.cancel != nil {
		p.cancel()
	}

	// Then close the client
	return p.client.Close()
}

// GetPublisher returns the underlying publisher interface
func (p *PubSubPublisher) GetPublisher() interfaces.PublisherInterface {
	return p.client
}
