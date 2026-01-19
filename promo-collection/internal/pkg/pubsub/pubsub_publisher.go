package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"promocollection/internal/pkg/log_messages"
	"promocollection/internal/pkg/logger"

	"cloud.google.com/go/pubsub"
	"google.golang.org/api/option"
)

type PubSubResult interface {
	Get(ctx context.Context) (string, error)
}

type PubSubTopic interface {
	Publish(ctx context.Context, msg *pubsub.Message) PubSubResult
}

type PubSubClient struct {
	Client *pubsub.Client
	Topic  PubSubTopic
}

type GCPTopicAdapter struct {
	topic *pubsub.Topic
}

func (g *GCPTopicAdapter) Publish(ctx context.Context, msg *pubsub.Message) PubSubResult {
	return &GCPPublishResultAdapter{g.topic.Publish(ctx, msg)}
}

type GCPPublishResultAdapter struct {
	res *pubsub.PublishResult
}

func (g *GCPPublishResultAdapter) Get(ctx context.Context) (string, error) {
	return g.res.Get(ctx)
}

type GCPClientFactory func(ctx context.Context, projectID string, opts ...option.ClientOption) (*pubsub.Client, error)

func NewPubSubClient(ctx context.Context, projectID, topicID string, factory GCPClientFactory) (*PubSubClient, error) {
	client, err := factory(ctx, projectID)
	if err != nil {
		return nil, err
	}

	topic := client.Topic(topicID)
	if topic == nil {
		return nil, fmt.Errorf(log_messages.TopicDoesNotExists, topicID)
	}

	adapter := &GCPTopicAdapter{topic: topic}

	return &PubSubClient{Client: client, Topic: adapter}, nil
}

func (p *PubSubClient) Close() {
	if err := p.Client.Close(); err != nil {
		logger.Error("failed to close pubsub client: %v", err)
	}
}

func (p *PubSubClient) PublishMessage(ctx context.Context,
	message any) (string, error) {

	inputData, err := json.Marshal(message)
	if err != nil {
		logger.Error(log_messages.ErrorMarshallingMessage, err)
		return "", fmt.Errorf(log_messages.ErrorMarshallingMessage, err)
	}

	res := p.Topic.Publish(ctx, &pubsub.Message{Data: inputData})

	messageID, err := res.Get(ctx)
	if err != nil {
		logger.Error(log_messages.ErrorInMessagePublishing, err)
		return "", fmt.Errorf(log_messages.ErrorInMessagePublishing, err)
	}

	return messageID, nil
}
