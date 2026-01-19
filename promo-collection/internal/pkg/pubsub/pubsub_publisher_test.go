package pubsub

import (
	"context"
	"errors"
	"testing"

	gcppubsub "cloud.google.com/go/pubsub"
	"google.golang.org/api/option"
)

type mockPubSubResult struct {
	msgID string
	err   error
}

func (m *mockPubSubResult) Get(ctx context.Context) (string, error) {
	return m.msgID, m.err
}

type mockPubSubTopic struct {
	result PubSubResult
}

func (m *mockPubSubTopic) Publish(ctx context.Context, msg *gcppubsub.Message) PubSubResult {
	return m.result
}

type TestingPromoCollectionWorkerPubSubMsgFormat struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func TestNewPubSubClient(t *testing.T) {
	ctx := context.Background()

	factoryOK := func(ctx context.Context, projectID string, opts ...option.ClientOption) (*gcppubsub.Client, error) {
		return &gcppubsub.Client{}, nil
	}
	client, err := NewPubSubClient(ctx, "proj", "topic", factoryOK)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if client == nil {
		t.Fatalf("expected client, got nil")
	}

	factoryErr := func(ctx context.Context, projectID string, opts ...option.ClientOption) (*gcppubsub.Client, error) {
		return nil, errors.New("factory failed")
	}
	_, err = NewPubSubClient(ctx, "proj", "topic", factoryErr)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestPublishMessage(t *testing.T) {
	ctx := context.Background()

	ps := &PubSubClient{
		Topic: &mockPubSubTopic{result: &mockPubSubResult{msgID: "123", err: nil}},
	}
	msg := TestingPromoCollectionWorkerPubSubMsgFormat{ID: "1", Name: "Test"}
	got, err := ps.PublishMessage(ctx, msg)
	if err != nil || got != "123" {
		t.Errorf("expected 123, got %v, err %v", got, err)
	}

	badMsg := struct {
		Ch chan int `json:"ch"`
	}{Ch: make(chan int)}
	_, err = ps.PublishMessage(ctx, badMsg)
	if err == nil {
		t.Errorf("expected marshalling error, got nil")
	}

	ps.Topic = &mockPubSubTopic{result: &mockPubSubResult{msgID: "", err: errors.New("publish failed")}}
	_, err = ps.PublishMessage(ctx, msg)
	if err == nil {
		t.Errorf("expected publish error, got nil")
	}
}

func TestClose(t *testing.T) {
	ctx := context.Background()

	client, err := gcppubsub.NewClient(ctx, "dummy-project")
	if err != nil {
		t.Fatalf("failed to create pubsub client: %v", err)
	}

	ps := &PubSubClient{
		Client: client,
	}
	ps.Close()

	ps.Close()
}
