package main

import (
	"context"

	"promo-collection-worker/internal/app/runtime"
	"promo-collection-worker/internal/pkg/logger"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	app, err := runtime.New(ctx)
	if err != nil {
		logger.CtxError(ctx, "failed to initialize app", err)
		return
	}

	if err := app.Run(ctx); err != nil {
		logger.CtxError(ctx, "app stopped with error", err)
		return
	}
}
