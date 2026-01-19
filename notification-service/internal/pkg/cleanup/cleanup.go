package cleanup

import (
	"context"
	"net/http"
	"notificationservice/internal/pkg/log_messages"
	"notificationservice/internal/pkg/logger"
	"time"
)

func CleanupResources(ctx context.Context, consumer interface{ Close() error }, server *http.Server) {
	logger.CtxInfo(ctx, log_messages.CleanupStarted)

	if consumer != nil {
		if unsubscribeFunc, ok := consumer.(interface{ Unsubscribe(context.Context) error }); ok {
			logger.CtxInfo(ctx, "Gracefully unsubscribing from PubSub...")
			if err := unsubscribeFunc.Unsubscribe(ctx); err != nil {
				logger.CtxError(ctx, "Failed to unsubscribe from PubSub", err)
			} else {
				logger.CtxInfo(ctx, "Successfully unsubscribed from PubSub")
			}
		}
		if err := consumer.Close(); err != nil {
			logger.CtxError(ctx, "Failed to close PubSub consumer", err)
		} else {
			logger.CtxInfo(ctx, "PubSub consumer closed successfully")
		}
	}
	if server != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.CtxError(ctx, "Failed to shutdown HTTP server", err)
		} else {
			logger.CtxInfo(ctx, "HTTP server shutdown successfully")
		}
	}

	logger.CtxInfo(ctx, log_messages.CleanupCompleted)
}
