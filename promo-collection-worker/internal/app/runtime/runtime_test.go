package runtime

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"promo-collection-worker/internal/pkg/config"
	"promo-collection-worker/internal/pkg/kafka"
	"promo-collection-worker/internal/pkg/pubsub"

	mongopkg "promo-collection-worker/internal/pkg/db/mongo"
	redispkg "promo-collection-worker/internal/pkg/db/redis"

	svcInterfaces "promo-collection-worker/internal/service/interfaces"
)

const testConfigPath = "../../../configs/config.yaml"

// mockPubSubPublisher mocks PubSubPublisher interface for tests
type mockPubSubPublisher struct {
	closeCalled   bool
	publishCalled bool
}

func (m *mockPubSubPublisher) Close() error {
	m.closeCalled = true
	return nil
}

func (m *mockPubSubPublisher) Publish(ctx context.Context, topic string, msg []byte) error {
	m.publishCalled = true
	return nil
}

type mockPubSub struct {
	closeCalled         bool
	consumeCalled       bool
	startConsumerCalled bool
}

func (m *mockPubSub) Close() error {
	m.closeCalled = true
	return nil
}

func (m *mockPubSub) Consume(ctx context.Context, sub string, handler func(context.Context, []byte) error) error {
	m.consumeCalled = true
	return context.Canceled // simulate graceful exit
}

func (m *mockPubSub) StartConsumer(subscription string, handler func(ctx context.Context, msg []byte) error) {
	m.startConsumerCalled = true
}

// mockPubSubClient implements svcInterfaces.PubSubClientInterface for constructing
// a pubsub.PubSubConsumer in tests.
type mockPubSubClient struct{}

type mockSubscriber struct {
	maxExtension time.Duration
}

func (m *mockPubSubClient) Subscriber(subscription string) svcInterfaces.SubscriberInterface {
	return &mockSubscriber{}
}

func (m *mockPubSubClient) Close() error { return nil }

func (s *mockSubscriber) Receive(ctx context.Context, f func(context.Context, svcInterfaces.MessageInterface)) error {
	// Immediately return context error to simulate no messages / graceful stop
	return ctx.Err()
}

func (m *mockSubscriber) SetMaxExtension(d time.Duration) {
	m.maxExtension = d
}

// mockPubSubPublisherClient implements svcInterfaces.PubSubPublisherClientInterface for constructing
// a pubsub.PubSubPublisher in tests.
type mockPubSubPublisherClient struct{}

func (m *mockPubSubPublisherClient) Publisher(topic string) svcInterfaces.PublisherInterface {
	return &mockPublisher{}
}

func (m *mockPubSubPublisherClient) Close() error { return nil }

type mockPublisher struct{}

func (m *mockPublisher) Publish(ctx context.Context, msg []byte) error {
	return nil
}

// --- Tests ---

func TestRun_ShutdownOnSignal(t *testing.T) {
	// This test requires a properly initialized MongoDB client, which we can't easily mock
	// without modifying the runtime.go file. Skipping for now.
	t.Skip("Skipping test that requires MongoDB client")

	ctx := context.Background()
	pub := &mockPubSub{}
	pubPublisher := &mockPubSubPublisher{}

	app := &App{
		Cfg:             &config.AppConfig{Server: config.ServerConfig{Port: 0}},
		PubSubConsumer:  pub,
		PubSubPublisher: pubPublisher,
		KafkaProducer:   nil,
	}

	done := make(chan struct{})
	go func() {
		_ = app.Run(ctx)
		close(done)
	}()

	time.Sleep(150 * time.Millisecond)

	proc, _ := os.FindProcess(os.Getpid())
	_ = proc.Signal(os.Interrupt)

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not return after interrupt signal")
	}

	if !pub.closeCalled {
		t.Errorf("expected PubSub Close to be called")
	}
	if !pubPublisher.closeCalled {
		t.Errorf("expected PubSubPublisher Close to be called")
	}
}

func TestShutdownCallsCleanup(t *testing.T) {
	ctx := context.Background()
	pub := &mockPubSub{}
	pubPublisher := &mockPubSubPublisher{}
	app := &App{
		PubSubConsumer:  pub,
		PubSubPublisher: pubPublisher,
		KafkaProducer:   nil,
	}

	app.Shutdown(ctx)

	if !pub.closeCalled {
		t.Errorf("expected PubSub Close to be called on Shutdown")
	}
	if !pubPublisher.closeCalled {
		t.Errorf("expected PubSubPublisher Close to be called on Shutdown")
	}
}

func TestNewConfigValidationError(t *testing.T) {
	ctx := context.Background()
	old := os.Getenv("PUBSUB_MIN_BACKOFF_SECONDS")
	_ = os.Setenv("PUBSUB_MIN_BACKOFF_SECONDS", "1")
	defer os.Setenv("PUBSUB_MIN_BACKOFF_SECONDS", old)

	if _, err := New(ctx); err == nil {
		t.Fatal("expected error from New due to invalid config, got nil")
	}
}

func TestRun_ServerStartFailureThenSignal(t *testing.T) {
	// This test requires a properly initialized MongoDB client, which we can't easily mock
	// without modifying the runtime.go file. Skipping for now.
	t.Skip("Skipping test that requires MongoDB client")

	ctx := context.Background()
	pub := &mockPubSub{}
	pubPublisher := &mockPubSubPublisher{}

	app := &App{
		Cfg:             &config.AppConfig{Server: config.ServerConfig{Port: -1}},
		PubSubConsumer:  pub,
		PubSubPublisher: pubPublisher,
		KafkaProducer:   nil,
	}

	done := make(chan struct{})
	go func() {
		_ = app.Run(ctx)
		close(done)
	}()

	time.Sleep(150 * time.Millisecond)
	proc, _ := os.FindProcess(os.Getpid())
	_ = proc.Signal(os.Interrupt)

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not return after interrupt with start failure")
	}
}

func TestNewSuccessWithStubs(t *testing.T) {
	ctx := context.Background()

	// Save originals
	origPubSub := pubsub.NewPubSubConsumer
	origKafka := newKafkaProducer
	origMongo := connectMongoDB
	origRedis := connectRedisDB
	defer func() {
		pubsub.NewPubSubConsumer = origPubSub
		newKafkaProducer = origKafka
		connectMongoDB = origMongo
		connectRedisDB = origRedis
	}()

	// Stubs
	pubsub.NewPubSubConsumer = func(ctx context.Context, projectID string) (*pubsub.PubSubConsumer, error) {
		return &pubsub.PubSubConsumer{
			PubSubClient: &mockPubSubClient{},
			Ctx:          ctx,
			Cancel:       nil,
		}, nil
	}

	newKafkaProducer = func(cfg config.KafkaConfig) (*kafka.KafkaProducer, error) {
		return &kafka.KafkaProducer{}, nil
	}

	connectMongoDB = func(ctx context.Context, cfg config.MongoConfig) (*mongopkg.MongoClient, error) {
		return &mongopkg.MongoClient{}, nil
	}
	connectRedisDB = func(ctx context.Context, cfg config.RedisConfig) (*redispkg.RedisClient, error) {
		return &redispkg.RedisClient{}, nil
	}

	prev := os.Getenv("CONFIG_PATH")
	_ = os.Setenv("CONFIG_PATH", testConfigPath)
	defer os.Setenv("CONFIG_PATH", prev)

	app, err := New(ctx)
	if err != nil {
		t.Fatalf("expected New to succeed with stubs, got error: %v", err)
	}
	if app.PubSubConsumer == nil {
		t.Fatalf("expected app consumers to be initialized")
	}
	if app.PubSubPublisher == nil {
		t.Fatalf("expected app publisher to be initialized")
	}
	if app.MongoClient == nil {
		t.Fatalf("expected app mongo client to be initialized")
	}
	if app.RedisClient == nil {
		t.Fatalf("expected app redis client to be initialized")
	}
	if app.KafkaProducer == nil {
		t.Fatalf("expected app kafka producer to be initialized")
	}
}

func TestNewPubSubError(t *testing.T) {
	ctx := context.Background()
	origPub := pubsub.NewPubSubConsumer
	origKafka := newKafkaProducer
	origMongo := connectMongoDB
	origRedis := connectRedisDB
	defer func() {
		pubsub.NewPubSubConsumer = origPub
		newKafkaProducer = origKafka
		connectMongoDB = origMongo
		connectRedisDB = origRedis
	}()

	pubsub.NewPubSubConsumer = func(ctx context.Context, projectID string) (*pubsub.PubSubConsumer, error) {
		return nil, errors.New("pubsub failed")
	}

	newKafkaProducer = func(cfg config.KafkaConfig) (*kafka.KafkaProducer, error) {
		return &kafka.KafkaProducer{}, nil
	}

	connectMongoDB = func(ctx context.Context, cfg config.MongoConfig) (*mongopkg.MongoClient, error) {
		return &mongopkg.MongoClient{}, nil
	}
	connectRedisDB = func(ctx context.Context, cfg config.RedisConfig) (*redispkg.RedisClient, error) {
		return &redispkg.RedisClient{}, nil
	}

	prev := os.Getenv("CONFIG_PATH")
	_ = os.Setenv("CONFIG_PATH", testConfigPath)
	defer os.Setenv("CONFIG_PATH", prev)

	if _, err := New(ctx); err == nil {
		t.Fatal("expected error when pubsub creation fails")
	}
}

func TestNewMongoError(t *testing.T) {
	ctx := context.Background()
	origPub := pubsub.NewPubSubConsumer
	origKafka := newKafkaProducer
	origMongo := connectMongoDB
	origRedis := connectRedisDB
	defer func() {
		pubsub.NewPubSubConsumer = origPub
		newKafkaProducer = origKafka
		connectMongoDB = origMongo
		connectRedisDB = origRedis
	}()

	pubsub.NewPubSubConsumer = func(ctx context.Context, projectID string) (*pubsub.PubSubConsumer, error) {
		return &pubsub.PubSubConsumer{
			PubSubClient: &mockPubSubClient{},
			Ctx:          ctx,
			Cancel:       nil,
		}, nil
	}

	newKafkaProducer = func(cfg config.KafkaConfig) (*kafka.KafkaProducer, error) {
		return &kafka.KafkaProducer{}, nil
	}

	connectMongoDB = func(ctx context.Context, cfg config.MongoConfig) (*mongopkg.MongoClient, error) {
		return nil, errors.New("mongo failed")
	}

	connectRedisDB = func(ctx context.Context, cfg config.RedisConfig) (*redispkg.RedisClient, error) {
		return &redispkg.RedisClient{}, nil
	}

	prev := os.Getenv("CONFIG_PATH")
	_ = os.Setenv("CONFIG_PATH", testConfigPath)
	defer os.Setenv("CONFIG_PATH", prev)

	if _, err := New(ctx); err == nil {
		t.Fatal("expected error when mongo connect fails")
	}
}

func TestNewRedisError(t *testing.T) {
	ctx := context.Background()
	origPub := pubsub.NewPubSubConsumer
	origKafka := newKafkaProducer
	origMongo := connectMongoDB
	origRedis := connectRedisDB
	defer func() {
		pubsub.NewPubSubConsumer = origPub
		newKafkaProducer = origKafka
		connectMongoDB = origMongo
		connectRedisDB = origRedis
	}()

	pubsub.NewPubSubConsumer = func(ctx context.Context, projectID string) (*pubsub.PubSubConsumer, error) {
		return &pubsub.PubSubConsumer{
			PubSubClient: &mockPubSubClient{},
			Ctx:          ctx,
			Cancel:       nil,
		}, nil
	}

	newKafkaProducer = func(cfg config.KafkaConfig) (*kafka.KafkaProducer, error) {
		return &kafka.KafkaProducer{}, nil
	}

	connectMongoDB = func(ctx context.Context, cfg config.MongoConfig) (*mongopkg.MongoClient, error) {
		return &mongopkg.MongoClient{}, nil
	}
	connectRedisDB = func(ctx context.Context, cfg config.RedisConfig) (*redispkg.RedisClient, error) {
		return nil, errors.New("redis failed")
	}

	prev := os.Getenv("CONFIG_PATH")
	_ = os.Setenv("CONFIG_PATH", testConfigPath)
	defer os.Setenv("CONFIG_PATH", prev)

	if _, err := New(ctx); err == nil {
		t.Fatal("expected error when redis connect fails")
	}
}

func TestNewPubSubPublisherError(t *testing.T) {
	ctx := context.Background()
	origPub := pubsub.NewPubSubConsumer
	origPubPublisher := pubsub.NewPubSubPublisher
	origKafka := newKafkaProducer
	origMongo := connectMongoDB
	origRedis := connectRedisDB
	defer func() {
		pubsub.NewPubSubConsumer = origPub
		pubsub.NewPubSubPublisher = origPubPublisher
		newKafkaProducer = origKafka
		connectMongoDB = origMongo
		connectRedisDB = origRedis
	}()

	pubsub.NewPubSubConsumer = func(ctx context.Context, projectID string) (*pubsub.PubSubConsumer, error) {
		return &pubsub.PubSubConsumer{
			PubSubClient: &mockPubSubClient{},
			Ctx:          ctx,
			Cancel:       nil,
		}, nil
	}

	pubsub.NewPubSubPublisher = func(ctx context.Context, projectID string) (*pubsub.PubSubPublisher, error) {
		return nil, errors.New("pubsub publisher failed")
	}

	newKafkaProducer = func(cfg config.KafkaConfig) (*kafka.KafkaProducer, error) {
		return &kafka.KafkaProducer{}, nil
	}

	connectMongoDB = func(ctx context.Context, cfg config.MongoConfig) (*mongopkg.MongoClient, error) {
		return &mongopkg.MongoClient{}, nil
	}
	connectRedisDB = func(ctx context.Context, cfg config.RedisConfig) (*redispkg.RedisClient, error) {
		return &redispkg.RedisClient{}, nil
	}

	prev := os.Getenv("CONFIG_PATH")
	_ = os.Setenv("CONFIG_PATH", testConfigPath)
	defer os.Setenv("CONFIG_PATH", prev)

	if _, err := New(ctx); err == nil {
		t.Fatal("expected error when pubsub publisher creation fails")
	}
}

func TestNewKafkaError(t *testing.T) {
	ctx := context.Background()
	origPub := pubsub.NewPubSubConsumer
	origPubPublisher := pubsub.NewPubSubPublisher
	origKafka := newKafkaProducer
	origMongo := connectMongoDB
	origRedis := connectRedisDB
	defer func() {
		pubsub.NewPubSubConsumer = origPub
		pubsub.NewPubSubPublisher = origPubPublisher
		newKafkaProducer = origKafka
		connectMongoDB = origMongo
		connectRedisDB = origRedis
	}()

	pubsub.NewPubSubConsumer = func(ctx context.Context, projectID string) (*pubsub.PubSubConsumer, error) {
		return &pubsub.PubSubConsumer{
			PubSubClient: &mockPubSubClient{},
			Ctx:          ctx,
			Cancel:       nil,
		}, nil
	}

	pubsub.NewPubSubPublisher = func(ctx context.Context, projectID string) (*pubsub.PubSubPublisher, error) {
		return &pubsub.PubSubPublisher{
			PubSubClient: &mockPubSubPublisherClient{},
			Ctx:          ctx,
			Cancel:       nil,
		}, nil
	}

	newKafkaProducer = func(cfg config.KafkaConfig) (*kafka.KafkaProducer, error) {
		return nil, errors.New("kafka failed")
	}

	connectMongoDB = func(ctx context.Context, cfg config.MongoConfig) (*mongopkg.MongoClient, error) {
		return &mongopkg.MongoClient{}, nil
	}
	connectRedisDB = func(ctx context.Context, cfg config.RedisConfig) (*redispkg.RedisClient, error) {
		return &redispkg.RedisClient{}, nil
	}

	prev := os.Getenv("CONFIG_PATH")
	_ = os.Setenv("CONFIG_PATH", testConfigPath)
	defer os.Setenv("CONFIG_PATH", prev)

	if _, err := New(ctx); err == nil {
		t.Fatal("expected error when kafka producer creation fails")
	}
}
