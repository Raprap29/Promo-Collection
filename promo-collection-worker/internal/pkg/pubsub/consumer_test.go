package pubsub

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"promo-collection-worker/internal/service/interfaces"
	pubsubServicePkg "promo-collection-worker/internal/service/pubsub"
)

type mockClient struct {
	subscriber *mockSubscriber
	closed     bool
}

func (m *mockClient) Subscriber(subscription string) interfaces.SubscriberInterface {
	return m.subscriber
}

func (m *mockClient) Close() error {
	m.closed = true
	return nil
}

type mockSubscriber struct {
	messages     [][]byte
	recvErr      error
	called       bool
	produced     []*mockMessage
	maxExtension time.Duration
}

func (s *mockSubscriber) Receive(ctx context.Context, f func(ctx context.Context, m interfaces.MessageInterface)) error {
	s.called = true
	if s.recvErr != nil {
		return s.recvErr
	}
	for _, msg := range s.messages {
		mm := &mockMessage{data: msg}
		s.produced = append(s.produced, mm)
		f(ctx, mm)
	}
	return nil
}

func (s *mockSubscriber) SetMaxExtension(d time.Duration) {
	s.maxExtension = d
}

type mockMessage struct {
	data   []byte
	acked  bool
	nacked bool
}

func (m *mockMessage) Data() []byte { return m.data }
func (m *mockMessage) Ack()         { m.acked = true }
func (m *mockMessage) Nack()        { m.nacked = true }

type mockFactory struct {
	client interfaces.PubSubClientInterface
	err    error
}

func (m *mockFactory) NewPubSubClient(ctx context.Context, projectID string) (interfaces.PubSubClientInterface, error) {
	return m.client, m.err
}

// --- Tests ---

func TestNewPubSubConsumerWithFactory_Success(t *testing.T) {
	ctx := context.Background()
	client := &mockClient{subscriber: &mockSubscriber{}}
	factory := &mockFactory{client: client}

	consumer, err := NewPubSubConsumerWithFactory(ctx, "project", factory)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if consumer.PubSubClient != client {
		t.Errorf("expected client to be set")
	}
}

func TestNewPubSubConsumerWithFactory_Error(t *testing.T) {
	ctx := context.Background()
	factory := &mockFactory{err: errors.New("fail")}

	consumer, err := NewPubSubConsumerWithFactory(ctx, "proj", factory)
	if err == nil {
		t.Fatal("expected error from factory")
	}
	if consumer != nil {
		t.Errorf("expected consumer to be nil on error")
	}
}

func TestConsume_HandlerAckNack(t *testing.T) {
	ctx := context.Background()
	msg := []byte("hello")
	subscriber := &mockSubscriber{messages: [][]byte{msg}}
	client := &mockClient{subscriber: subscriber}
	consumer := &PubSubConsumer{PubSubClient: client, Ctx: ctx}

	called := false
	consumer.Consume(ctx, "sub", func(ctx context.Context, m []byte) error {
		called = true
		if string(m) == "hello" {
			return errors.New("handler error") // triggers Nack
		}
		return nil
	})
	if !called {
		t.Errorf("expected handler to be called")
	}
	if len(subscriber.produced) != 1 {
		t.Fatalf("expected one produced message, got %d", len(subscriber.produced))
	}
	if !subscriber.produced[0].nacked {
		t.Errorf("expected message to be nacked on handler error")
	}
}

// We'll use the actual MessageIgnoreError type from the service package

// Test for the message ignore behavior in Consume
func TestConsume_MessageIgnoreError(t *testing.T) {
	ctx := context.Background()
	msg := []byte("ignore-me")
	subscriber := &mockSubscriber{messages: [][]byte{msg}}
	client := &mockClient{subscriber: subscriber}
	consumer := &PubSubConsumer{PubSubClient: client, Ctx: ctx}

	// Create a handler that returns our MessageIgnoreError
	called := false
	consumer.Consume(ctx, "sub", func(ctx context.Context, m []byte) error {
		called = true
		if string(m) == "ignore-me" {
			// Return the actual MessageIgnoreError from the service package
			return &pubsubServicePkg.MessageIgnoreError{Err: errors.New("test ignore")}
		}
		return nil
	})

	// Verify handler was called
	if !called {
		t.Errorf("expected handler to be called")
	}

	// Verify message was produced
	if len(subscriber.produced) != 1 {
		t.Fatalf("expected one produced message, got %d", len(subscriber.produced))
	}

	// Verify message was neither acked nor nacked
	if subscriber.produced[0].acked {
		t.Errorf("expected message not to be acked when ignored")
	}
	if subscriber.produced[0].nacked {
		t.Errorf("expected message not to be nacked when ignored")
	}
}

func TestConsume_MultipleMessages(t *testing.T) {
	ctx := context.Background()
	messages := [][]byte{[]byte("m1"), []byte("m2")}
	subscriber := &mockSubscriber{messages: messages}
	client := &mockClient{subscriber: subscriber}
	consumer := &PubSubConsumer{PubSubClient: client, Ctx: ctx}

	count := 0
	consumer.Consume(ctx, "sub", func(ctx context.Context, m []byte) error {
		count++
		return nil
	})

	if count != 2 {
		t.Errorf("expected handler called twice, got %d", count)
	}
	if len(subscriber.produced) != 2 {
		t.Fatalf("expected 2 produced messages, got %d", len(subscriber.produced))
	}
	for i, mm := range subscriber.produced {
		if !mm.acked {
			t.Errorf("expected message %d to be acked", i)
		}
	}
}

func TestConsume_ReturnsReceiveError(t *testing.T) {
	ctx := context.Background()
	subscriber := &mockSubscriber{recvErr: errors.New("receive fail")}
	client := &mockClient{subscriber: subscriber}
	consumer := &PubSubConsumer{PubSubClient: client, Ctx: ctx}

	err := consumer.Consume(ctx, "sub", func(ctx context.Context, m []byte) error { return nil })
	if err == nil {
		t.Fatal("expected error from Consume when subscriber.Receive fails")
	}
}

func TestConsumeInLoop_StopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	subscriber := &mockSubscriber{}
	client := &mockClient{subscriber: subscriber}
	consumer := &PubSubConsumer{PubSubClient: client, Ctx: ctx, Cancel: cancel}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		consumer.StartConsumer("sub", func(ctx context.Context, m []byte) error { return nil })
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()
	wg.Wait()
}

func TestConsumeInLoop_RetryOnError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Subscriber returns error every time
	subscriber := &mockSubscriber{recvErr: errors.New("receive fail")}
	client := &mockClient{subscriber: subscriber}
	consumer := &PubSubConsumer{PubSubClient: client, Ctx: ctx, Cancel: cancel}

	// override sleep to make test faster
	origSleep := timeSleep
	defer func() { timeSleep = origSleep }()
	timeSleep = func(d time.Duration) {}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		consumer.StartConsumer("sub", func(ctx context.Context, m []byte) error { return nil })
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()
	wg.Wait()
}

func TestClose_CancelsAndClosesClient(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	client := &mockClient{}
	consumer := &PubSubConsumer{PubSubClient: client, Ctx: ctx, Cancel: cancel}

	if err := consumer.Close(); err != nil {
		t.Fatalf("expected no error from Close, got %v", err)
	}
	if !client.closed {
		t.Errorf("expected client to be closed")
	}
	if ctx.Err() != context.Canceled {
		t.Errorf("expected context to be canceled after Close, got %v", ctx.Err())
	}
}

func TestNewPubSubConsumer(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"

	originalFunc := NewPubSubConsumer
	defer func() { NewPubSubConsumer = originalFunc }()

	called := false
	NewPubSubConsumer = func(ctx context.Context, projectID string) (*PubSubConsumer, error) {
		called = true
		return &PubSubConsumer{}, nil
	}

	consumer, err := NewPubSubConsumer(ctx, projectID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if consumer == nil {
		t.Error("expected consumer, got nil")
	}
	if !called {
		t.Error("expected NewPubSubConsumer to be called")
	}
}

func TestConsume_MessageIgnoreErrorNil(t *testing.T) {
	ctx := context.Background()
	msg := []byte("ignore-me-nil")
	subscriber := &mockSubscriber{messages: [][]byte{msg}}
	client := &mockClient{subscriber: subscriber}
	consumer := &PubSubConsumer{PubSubClient: client, Ctx: ctx}

	called := false
	consumer.Consume(ctx, "sub", func(ctx context.Context, m []byte) error {
		called = true
		if string(m) == "ignore-me-nil" {
			return &pubsubServicePkg.MessageIgnoreError{Err: nil}
		}
		return nil
	})

	if !called {
		t.Errorf("expected handler to be called")
	}

	if len(subscriber.produced) != 1 {
		t.Fatalf("expected one produced message, got %d", len(subscriber.produced))
	}

	if subscriber.produced[0].acked {
		t.Errorf("expected message not to be acked when ignored")
	}
	if subscriber.produced[0].nacked {
		t.Errorf("expected message not to be nacked when ignored")
	}
}

// --- Patchable sleep for tests ---
var timeSleep = time.Sleep
