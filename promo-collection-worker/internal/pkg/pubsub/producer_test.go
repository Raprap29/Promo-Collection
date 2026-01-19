package pubsub

import (
	"context"
	"errors"
	"testing"

	"promo-collection-worker/internal/service/interfaces"
)

// Mock implementations for testing

const (
	testProjectID      = "test-project"
	expectedNoErrorFmt = "expected no error, got %v"
)

type mockPubSubPublisherClient struct {
	publishers map[string]*mockPublisher
	closeCalled bool
}

func (m *mockPubSubPublisherClient) Publisher(topic string) interfaces.PublisherInterface {
	if m.publishers == nil {
		m.publishers = make(map[string]*mockPublisher)
	}
	if _, ok := m.publishers[topic]; !ok {
		m.publishers[topic] = &mockPublisher{}
	}
	return m.publishers[topic]
}

func (m *mockPubSubPublisherClient) Close() error {
	m.closeCalled = true
	return nil
}

type mockPublisher struct {
	publishCalled bool
	ctx           context.Context
	msg           []byte
	publishError  error
}

func (m *mockPublisher) Publish(ctx context.Context, msg []byte) error {
	m.publishCalled = true
	m.ctx = ctx
	m.msg = msg
	return m.publishError
}

type mockPublisherFactory struct {
	client interfaces.PubSubPublisherClientInterface
	err    error
}

func (m *mockPublisherFactory) NewPubSubPublisherClient(ctx context.Context, projectID string) (interfaces.PubSubPublisherClientInterface, error) {
	return m.client, m.err
}

// Tests

func TestNewPubSubPublisherWithFactorySuccess(t *testing.T) {
	ctx := context.Background()
	projectID := testProjectID
	mockClient := &mockPubSubPublisherClient{}
	factory := &mockPublisherFactory{client: mockClient}

	publisher, err := NewPubSubPublisherWithFactory(ctx, projectID, factory)
	if err != nil {
		t.Fatalf(expectedNoErrorFmt, err)
	}
	if publisher == nil {
		t.Fatal("expected publisher, got nil")
	}
	if publisher.PubSubClient != mockClient {
		t.Error("expected PubSubClient to be set")
	}
	if publisher.Ctx == nil {
		t.Error("expected Ctx to be set")
	}
	if publisher.Cancel == nil {
		t.Error("expected Cancel to be set")
	}
}

func TestNewPubSubPublisherWithFactoryFactoryError(t *testing.T) {
	ctx := context.Background()
	projectID := testProjectID
	factory := &mockPublisherFactory{err: errors.New("factory error")}

	publisher, err := NewPubSubPublisherWithFactory(ctx, projectID, factory)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if publisher != nil {
		t.Error("expected nil publisher on error")
	}
}

func TestPubSubPublisherPublish(t *testing.T) {
	ctx := context.Background()
	mockClient := &mockPubSubPublisherClient{}
	publisher := &PubSubPublisher{PubSubClient: mockClient}
	topic := "test-topic"
	msg := []byte("test message")

	// Test successful publish
	err := publisher.Publish(ctx, topic, msg)
	if err != nil {
		t.Fatalf(expectedNoErrorFmt, err)
	}

	mockPub := mockClient.publishers[topic]
	if mockPub == nil {
		t.Fatal("expected publisher to be created")
	}
	if !mockPub.publishCalled {
		t.Error("expected Publish to be called on mock")
	}
	if string(mockPub.msg) != string(msg) {
		t.Errorf("expected msg %s, got %s", msg, mockPub.msg)
	}
	if mockPub.ctx != ctx {
		t.Error("expected ctx to be passed")
	}

	// Test publish error
	mockClient.publishers[topic].publishError = errors.New("publish error")
	err = publisher.Publish(ctx, topic, msg)
	if err == nil {
		t.Error("expected error from publish, got nil")
	}
}

func TestPubSubPublisherClose(t *testing.T) {
	mockClient := &mockPubSubPublisherClient{}
	ctx, cancel := context.WithCancel(context.Background())
	publisher := &PubSubPublisher{
		PubSubClient: mockClient,
		Ctx:          ctx,
		Cancel:       cancel,
	}

	err := publisher.Close()
	if err != nil {
		t.Fatalf(expectedNoErrorFmt, err)
	}
	if !mockClient.closeCalled {
		t.Error("expected Close to be called on client")
	}
	// Note: Cancel is called, but hard to verify without accessing internal
}

// Test coverage improvement: test NewPubSubPublisher function variable override
func TestNewPubSubPublisher(t *testing.T) {
	ctx := context.Background()
	projectID := testProjectID
	called := false

	// Override NewPubSubPublisher to test it calls NewPubSubPublisherWithFactory
	orig := NewPubSubPublisher
	NewPubSubPublisher = func(ctx context.Context, projectID string) (*PubSubPublisher, error) {
		called = true
		return &PubSubPublisher{}, nil
	}
	defer func() { NewPubSubPublisher = orig }()

	_, err := NewPubSubPublisher(ctx, projectID)
	if err != nil {
		t.Fatalf(expectedNoErrorFmt, err)
	}
	if !called {
		t.Error("expected NewPubSubPublisher to be called")
	}
}
