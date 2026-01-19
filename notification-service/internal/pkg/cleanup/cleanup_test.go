package cleanup

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestCleanupResources(t *testing.T) {
	ctx := context.Background()
	CleanupResources(ctx, nil, nil)
}

func TestCleanupResourcesWithConsumer(t *testing.T) {
	ctx := context.Background()
	mockConsumer := &mockConsumer{}
	CleanupResources(ctx, mockConsumer, nil)
}

func TestCleanupResourcesWithServer(t *testing.T) {
	ctx := context.Background()
	testServer := &http.Server{
		Addr: ":0",
	}
	CleanupResources(ctx, nil, testServer)
}

func TestCleanupResourcesWithBoth(t *testing.T) {
	ctx := context.Background()
	mockConsumer := &mockConsumer{}
	testServer := &http.Server{
		Addr: ":0",
	}
	CleanupResources(ctx, mockConsumer, testServer)
}

func TestCleanupResourcesWithUnsubscribeConsumer(t *testing.T) {
	ctx := context.Background()
	mockConsumer := &mockConsumerWithUnsubscribe{}
	CleanupResources(ctx, mockConsumer, nil)
}

func TestCleanupResourcesWithUnsubscribeError(t *testing.T) {
	ctx := context.Background()
	mockConsumer := &mockConsumerWithUnsubscribeError{}
	CleanupResources(ctx, mockConsumer, nil)
}

func TestCleanupResourcesWithCloseError(t *testing.T) {
	ctx := context.Background()
	mockConsumer := &mockConsumerWithCloseError{}
	CleanupResources(ctx, mockConsumer, nil)
}
func TestCleanupResourcesWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	testServer := &http.Server{
		Addr: ":0",
	}
	CleanupResources(ctx, nil, testServer)
}

func TestCleanupResourcesConcurrent(t *testing.T) {
	ctx := context.Background()

	done := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		go func() {
			CleanupResources(ctx, nil, nil)
			done <- true
		}()
	}

	for i := 0; i < 5; i++ {
		<-done
	}
}

func TestCleanupResourcesWithRealServer(t *testing.T) {
	ctx := context.Background()

	server := &http.Server{
		Addr:         ":0",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	CleanupResources(ctx, nil, server)
}

func TestCleanupResourcesWithRealConsumer(t *testing.T) {
	ctx := context.Background()
	consumer := &realConsumer{}
	CleanupResources(ctx, consumer, nil)
}

type mockConsumer struct{}

func (m *mockConsumer) Close() error {
	return nil
}

type mockConsumerWithUnsubscribe struct{}

func (m *mockConsumerWithUnsubscribe) Close() error {
	return nil
}

func (m *mockConsumerWithUnsubscribe) Unsubscribe(ctx context.Context) error {
	return nil
}

type mockConsumerWithUnsubscribeError struct{}

func (m *mockConsumerWithUnsubscribeError) Close() error {
	return nil
}

func (m *mockConsumerWithUnsubscribeError) Unsubscribe(ctx context.Context) error {
	return errors.New("unsubscribe failed")
}

type mockConsumerWithCloseError struct{}

func (m *mockConsumerWithCloseError) Close() error {
	return errors.New("close failed")
}

func (m *mockConsumerWithCloseError) Unsubscribe(ctx context.Context) error {
	return nil
}

type realConsumer struct{}

func (r *realConsumer) Close() error {
	return nil
}

func (r *realConsumer) Unsubscribe(ctx context.Context) error {
	return nil
}
