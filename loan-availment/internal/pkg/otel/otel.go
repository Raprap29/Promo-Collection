package otel

import (
	"context"
	"globe/dodrio_loan_availment/internal/pkg/logger"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

var (
	tracer           trace.Tracer
	connectionFailed bool
	connectionMutex  sync.Mutex
)

func Setup(ctx context.Context, serviceName, collectorURL string) (func(context.Context) error, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, err
	}

	// Set up a context with a timeout for establishing OTLP connections
	connectionCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Set up tracing exporter
	traceExporter, err := otlptracehttp.New(connectionCtx,
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithEndpoint(collectorURL),
	)
	if err != nil {
		handleConnectionError(err)
		return func(ctx context.Context) error { return nil }, nil
	}

	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tracerProvider)
	tracer = tracerProvider.Tracer(serviceName)

	// Set global propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	// Return shutdown function
	return func(ctx context.Context) error {
		cxt, cancel := context.WithTimeout(ctx, time.Second*5)
		defer cancel()
		if err := tracerProvider.Shutdown(cxt); err != nil {
			return err
		}
		return nil
	}, nil
}

func GetTracer() trace.Tracer {
	if tracer == nil {
		return noop.NewTracerProvider().Tracer("")
	}
	return tracer
}

func handleConnectionError(err error) {
	connectionMutex.Lock()
	defer connectionMutex.Unlock()
	if !connectionFailed {
		logger.Error("OTLP connection error:", err.Error())
		connectionFailed = true
	}
}
