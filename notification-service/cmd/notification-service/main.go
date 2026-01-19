package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"notificationservice/internal/app/router"
	"notificationservice/internal/pkg/cleanup"
	"notificationservice/internal/pkg/config"
	"notificationservice/internal/pkg/log_messages"
	"notificationservice/internal/pkg/logger"
	"notificationservice/internal/pkg/pubsub"
	"notificationservice/internal/service"
)

// setupServices initializes configuration, logger, and core services
func setupServices(ctx context.Context) (*config.AppConfig, *pubsub.PubSubConsumer, error) {
	// Load configuration
	cfg, err := config.LoadFromConfig()
	if err != nil {
		logger.CtxError(context.Background(), log_messages.FailedLoadingConfiguration, err)
		return nil, nil, err
	}
	logger.Init(cfg.Logging.LogLevel)

	// Initialize PubSub consumer
	consumer, err := pubsub.NewPubSubConsumer(ctx, cfg.PubSub.ProjectID)
	if err != nil {
		logger.CtxError(ctx, log_messages.FailureInPubsubConsumerCreation, err)
		return nil, nil, err
	}

	return cfg, consumer, nil
}

// startPubSubConsumer starts the PubSub consumer in a goroutine
func startPubSubConsumer(ctx context.Context,
	consumer *pubsub.PubSubConsumer,
	subscription string,
	handler func(ctx context.Context, msg []byte) error,
) {
	go func() {
		err := consumer.Consume(ctx, subscription, handler)
		if err != nil && !errors.Is(err, context.Canceled) {
			logger.CtxError(ctx, log_messages.PubsubErrorConsuming, err)
		}
	}()
}

// startHTTPServer starts the HTTP server in a goroutine
func startHTTPServer(ctx context.Context, port int, engine http.Handler) *http.Server {
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.CtxError(ctx, log_messages.ServerStartFailure, err)
		}
	}()

	return srv
}

// waitForShutdownSignal waits for shutdown signals and returns when received
func waitForShutdownSignal() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}

// gracefulShutdown performs graceful shutdown of all services
func gracefulShutdown(ctx context.Context,
	cancel context.CancelFunc, consumer *pubsub.PubSubConsumer, server *http.Server) {
	logger.CtxInfo(ctx, log_messages.ServerShutdown)

	// Stop receiving new messages immediately
	cancel()

	// Cleanup all resources (handles unsubscribe and close)
	cleanup.CleanupResources(ctx, consumer, server)
}

func main() {
	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup services
	cfg, consumer, err := setupServices(ctx)
	if err != nil {
		return
	}
	defer func() {
		if consumer != nil {
			if err := consumer.Close(); err != nil {
				logger.CtxError(ctx, "Failed to close consumer during defer", err)
			}
		}
	}()

	// Initialize notification service
	notifService := service.NewNotificationService(cfg)

	// Start PubSub consumer
	startPubSubConsumer(ctx, consumer, cfg.PubSub.Subscription, notifService.HandleMessage)

	// Setup and start HTTP server
	engine := router.SetupRouter(ctx)
	server := startHTTPServer(ctx, cfg.Server.Port, engine)

	// Wait for shutdown signal
	waitForShutdownSignal()

	// Perform graceful shutdown
	gracefulShutdown(ctx, cancel, consumer, server)

	logger.CtxInfo(ctx, log_messages.ServerExiting)
}
