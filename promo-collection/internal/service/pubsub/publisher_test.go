package publisher

import (
	"context"
	"errors"
	"testing"

	"promocollection/internal/pkg/models"
)

type mockPublisher struct {
	publishFunc func(ctx context.Context, msg any) (string, error)
}

func (m *mockPublisher) Close() {
	// no-op for tests
}

func (m *mockPublisher) PublishMessage(ctx context.Context, msg any) (string, error) {
	return m.publishFunc(ctx, msg)
}

func TestPubSubPublisherServiceSuccess(t *testing.T) {
	ctx := context.Background()
	mock := &mockPublisher{
		publishFunc: func(ctx context.Context, msg any) (string, error) {
			return "mock-message-id", nil
		},
	}

	service := NewPubSubPublisherService(mock)
	msg := models.PublishtoWorkerMsgFormat{}

	id, err := service.PubSubPublisher(ctx, msg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if id != "mock-message-id" {
		t.Fatalf("expected 'mock-message-id', got %s", id)
	}
}

func TestPubSubPublisherServiceError(t *testing.T) {
	ctx := context.Background()
	mock := &mockPublisher{
		publishFunc: func(ctx context.Context, msg any) (string, error) {
			return "", errors.New("publish failed")
		},
	}

	service := NewPubSubPublisherService(mock)
	msg := models.PublishtoWorkerMsgFormat{}

	id, err := service.PubSubPublisher(ctx, msg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if id != "" {
		t.Fatalf("expected empty id, got %s", id)
	}
}

func TestPubSubPublisherHelperFunction(t *testing.T) {
	ctx := context.Background()

	// Success case
	mockSuccess := &mockPublisher{
		publishFunc: func(ctx context.Context, msg any) (string, error) {
			return "helper-msg-id", nil
		},
	}
	id, err := PubSubPublisher(ctx, mockSuccess, models.PublishtoWorkerMsgFormat{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if id != "helper-msg-id" {
		t.Fatalf("expected 'helper-msg-id', got %s", id)
	}

	// Error case
	mockError := &mockPublisher{
		publishFunc: func(ctx context.Context, msg any) (string, error) {
			return "", errors.New("helper publish failed")
		},
	}
	id, err = PubSubPublisher(ctx, mockError, models.PublishtoWorkerMsgFormat{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if id != "" {
		t.Fatalf("expected empty id, got %s", id)
	}
}
