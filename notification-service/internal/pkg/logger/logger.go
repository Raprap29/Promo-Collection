package logger

import (
	"context"
	"log/slog"
	"os"
	"time"
)

// Define context key for request ID
type contextKey string

const requestIDKey contextKey = "request_id"

// getRequestID retrieves request_id from context, returns empty string if missing
func getRequestID(ctx context.Context) string {
	if v := ctx.Value(requestIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// WithRequestID returns a new context with the given request ID
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// Init sets up the global slog JSON logger with file:line info
func Init(logLevel string) {
	var level slog.Level
	switch logLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo // fallback
	}
	// Configure slog with JSON output
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
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
	if reqID := getRequestID(ctx); reqID != "" {
		args = append(args, slog.String("request_id", reqID))
	}
	slog.LogAttrs(ctx, slog.LevelInfo, msg, args...)
}

// CtxError logs an error with request ID and error detail
func CtxError(ctx context.Context, msg string, err error, args ...slog.Attr) {
	if reqID := getRequestID(ctx); reqID != "" {
		args = append(args, slog.String("request_id", reqID))
	}
	args = append(args, slog.Any("error", err))
	slog.LogAttrs(ctx, slog.LevelError, msg, args...)
}

// CtxDebug logs debug messages
func CtxDebug(ctx context.Context, msg string, args ...slog.Attr) {
	if reqID := getRequestID(ctx); reqID != "" {
		args = append(args, slog.String("request_id", reqID))
	}
	slog.LogAttrs(ctx, slog.LevelDebug, msg, args...)
}

// CtxWarn logs warnings
func CtxWarn(ctx context.Context, msg string, args ...slog.Attr) {
	if reqID := getRequestID(ctx); reqID != "" {
		args = append(args, slog.String("request_id", reqID))
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
