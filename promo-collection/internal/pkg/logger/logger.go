package logger

import (
	"context"
	"log/slog"
	"os"
	"time"
)

// Define context key for request ID
type contextKey string

const traceIDKey contextKey = "trace_id"

// getTraceID retrieves trace_id from context, returns empty string if missing
func GetTraceID(ctx context.Context) string {
	if v := ctx.Value(traceIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// WithTraceID returns a new context with the given trace ID
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// Init sets up the global slog JSON logger with file:line info
func Init() {
	// Configure slog with JSON output
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Change default time format
			if a.Key == slog.TimeKey && a.Value.Kind() == slog.KindTime {
				return slog.String(slog.TimeKey, a.Value.Time().Format(time.RFC3339))
			}
			return a
		},
	})
	slog.SetDefault(slog.New(handler))
}

// CONTEXT-AWARE LOGGING //

// CtxInfo logs an info message with request ID
func CtxInfo(ctx context.Context, msg string, args ...slog.Attr) {
	if traceID := GetTraceID(ctx); traceID != "" {
		args = append(args, slog.String("trace_id", traceID))
	}
	slog.LogAttrs(ctx, slog.LevelInfo, msg, args...)
}

func CtxError(ctx context.Context, msg string, err error, args ...slog.Attr) {
	if traceID := GetTraceID(ctx); traceID != "" {
		args = append(args, slog.String("trace_id", traceID))
	}
	args = append(args, slog.Any("error", err))
	slog.LogAttrs(ctx, slog.LevelError, msg, args...)
}

// CtxDebug logs debug messages
func CtxDebug(ctx context.Context, msg string, args ...slog.Attr) {
	if traceID := GetTraceID(ctx); traceID != "" {
		args = append(args, slog.String("trace_id", traceID))
	}
	slog.LogAttrs(ctx, slog.LevelDebug, msg, args...)
}

// CtxWarn logs warnings
func CtxWarn(ctx context.Context, msg string, args ...slog.Attr) {
	if traceID := GetTraceID(ctx); traceID != "" {
		args = append(args, slog.String("trace_id", traceID))
	}
	slog.LogAttrs(ctx, slog.LevelWarn, msg, args...)
}

// NON-CONTEXT LOGGING //

func Info(msg string, args ...slog.Attr) {
	slog.LogAttrs(context.Background(), slog.LevelInfo, msg, args...)
}

func Debug(msg string, args ...slog.Attr) {
	slog.LogAttrs(context.Background(), slog.LevelDebug, msg, args...)
}

func Warn(msg string, args ...slog.Attr) {
	slog.LogAttrs(context.Background(), slog.LevelWarn, msg, args...)
}

func Error(msg string, err error, args ...slog.Attr) {
	args = append(args, slog.Any("error", err))
	slog.LogAttrs(context.Background(), slog.LevelError, msg, args...)
}
