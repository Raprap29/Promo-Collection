package cleanup

import (
	"context"
	"errors"
	"log"
	"net/http"
	"testing"
	"time"

	mongopkg "promo-collection-worker/internal/pkg/db/mongo"
	redispkg "promo-collection-worker/internal/pkg/db/redis"
	"promo-collection-worker/internal/pkg/models"

	redisv9 "github.com/redis/go-redis/v9"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestCleanupResources(t *testing.T) {
	ctx := context.Background()
	CleanupResources(ctx, nil, nil, nil, nil, nil, nil, nil)
}

func TestCleanupResourcesWithConsumer(t *testing.T) {
	ctx := context.Background()
	mockConsumer := &mockConsumer{}
	CleanupResources(ctx, mockConsumer, nil, nil, nil, nil, nil, nil)
}

func TestCleanupResourcesWithServer(t *testing.T) {
	ctx := context.Background()
	testServer := &http.Server{
		Addr: ":0",
	}
	CleanupResources(ctx, nil, nil, nil, nil, nil, testServer, nil)
}

func TestCleanupResourcesWithBoth(t *testing.T) {
	ctx := context.Background()
	mockConsumer := &mockConsumer{}
	testServer := &http.Server{
		Addr: ":0",
	}
	CleanupResources(ctx, mockConsumer, nil, nil, nil, nil, testServer, nil)
}

func TestCleanupResourcesWithUnsubscribeConsumer(t *testing.T) {
	ctx := context.Background()
	mockConsumer := &mockConsumerWithUnsubscribe{}
	CleanupResources(ctx, mockConsumer, nil, nil, nil, nil, nil, nil)
}

func TestCleanupResourcesWithUnsubscribeError(t *testing.T) {
	ctx := context.Background()
	mockConsumer := &mockConsumerWithUnsubscribeError{}
	CleanupResources(ctx, mockConsumer, nil, nil, nil, nil, nil, nil)
}

func TestCleanupResourcesWithCloseError(t *testing.T) {
	ctx := context.Background()
	mockConsumer := &mockConsumerWithCloseError{}
	CleanupResources(ctx, mockConsumer, nil, nil, nil, nil, nil, nil)
}

func TestCleanupResourcesWithPublisherCloseError(t *testing.T) {
	ctx := context.Background()
	mockPublisher := &mockConsumerWithCloseError{}
	CleanupResources(ctx, nil, mockPublisher, nil, nil, nil, nil, nil)
}

func TestCleanupResourcesWithPublisherUnsubscribeError(t *testing.T) {
	ctx := context.Background()
	mockPublisher := &mockConsumerWithUnsubscribeError{}
	CleanupResources(ctx, nil, mockPublisher, nil, nil, nil, nil, nil)
}
func TestCleanupResourcesWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	testServer := &http.Server{
		Addr: ":0",
	}
	CleanupResources(ctx, nil, nil, nil, nil, nil, testServer, nil)
}

func TestCleanupResourcesConcurrent(t *testing.T) {
	ctx := context.Background()

	done := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		go func() {
			CleanupResources(ctx, nil, nil, nil, nil, nil, nil, nil)
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

	CleanupResources(ctx, nil, nil, nil, nil, nil, server, nil)
}

func TestCleanupResourcesWithRealConsumer(t *testing.T) {
	ctx := context.Background()
	consumer := &realConsumer{}
	CleanupResources(ctx, consumer, nil, nil, nil, nil, nil, nil)
}

func TestCleanupResourcesWithCancelledContextAndResources(t *testing.T) {
	// Connect with background ctx to get clients
	mClient, err := mongodriver.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Printf("mongo.Connect error: %v", err)
	}

	rClient := redisv9.NewClient(&redisv9.Options{Addr: "127.0.0.1:0"})

	// Wrap
	mongoWrap := &mongopkg.MongoClient{Client: mClient}
	redisWrap := &redispkg.RedisClient{Client: rClient}

	server := &http.Server{Addr: ":0"}

	// Now cancel ctx for Cleanup
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	CleanupResources(ctx, nil, nil, nil, mongoWrap, redisWrap, server, nil)
}

func TestCleanupResourcesWithMongoRedis(t *testing.T) {
	ctx := context.Background()

	// Mongo client created without connecting; Disconnect may error, which is fine.
	mClient, err := mongodriver.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Printf("mongo.Connect error: %v", err)
	}

	// Redis client with dummy addr; Close does not require a live server
	rClient := redisv9.NewClient(&redisv9.Options{Addr: "127.0.0.1:0"})

	// Wrap into our resource structs
	mongoWrap := &mongopkg.MongoClient{Client: mClient}
	redisWrap := &redispkg.RedisClient{Client: rClient}

	server := &http.Server{Addr: ":0"}

	CleanupResources(ctx, nil, nil, nil, mongoWrap, redisWrap, server, nil)
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

type mockGCSClient struct {
	CloseCalled  bool
	UploadCalled bool
}

func (m *mockGCSClient) Upload(ctx context.Context, msg *models.PromoCollectionPublishedMessage) error {
	m.UploadCalled = true
	return nil
}

func (m *mockGCSClient) Close(ctx context.Context) {
	m.CloseCalled = true
}
func TestCleanupGCSResource(t *testing.T) {
	ctx := context.Background()
	mockClient := &mockGCSClient{}

	cleanupGCSResource(mockClient, ctx)

	if !mockClient.CloseCalled {
		t.Error("Close was not called on the GCS client")
	}
}

func TestCleanupGCSResourceNilClient(t *testing.T) {
	ctx := context.Background()

	cleanupGCSResource(nil, ctx)
}

func TestCleanupResourcesWithGCSClient(t *testing.T) {
	ctx := context.Background()
	mockClient := &mockGCSClient{}

	CleanupResources(ctx, nil, nil, nil, nil, nil, nil, mockClient)

	if !mockClient.CloseCalled {
		t.Error("Close was not called on the GCS client")
	}
}
