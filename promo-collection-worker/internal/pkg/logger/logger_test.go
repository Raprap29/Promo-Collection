package logger

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

// helper to set test logger writing JSON to buffer
func setupTestLogger(buf *bytes.Buffer) {
	handler := slog.NewJSONHandler(buf, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: false,
	})
	slog.SetDefault(slog.New(handler))
}

func TestGetTraceID(t *testing.T) {
	// valid trace ID
	ctxWithID := context.WithValue(context.Background(), traceIDKey, "id123")
	assert.Equal(t, "id123", getTraceID(ctxWithID))

	// no trace ID returns empty string
	assert.Empty(t, getTraceID(context.Background()))

	// trace ID with wrong type returns empty string
	ctxWrongType := context.WithValue(context.Background(), traceIDKey, 42)
	assert.Empty(t, getTraceID(ctxWrongType))
}

func TestCtxLogging_InjectsTraceID(t *testing.T) {
	var buf bytes.Buffer
	setupTestLogger(&buf)

	ctx := WithTraceID(context.Background(), "trace-edge")
	CtxInfo(ctx, "info with trace ID")

	log := buf.String()
	assert.Contains(t, log, `"trace_id":"trace-edge"`)
	assert.Contains(t, log, `"msg":"info with trace ID"`)
}

func TestCtxLogging_NoTraceID(t *testing.T) {
	var buf bytes.Buffer
	setupTestLogger(&buf)

	CtxWarn(context.Background(), "warn without trace ID")

	log := buf.String()
	assert.NotContains(t, log, `"trace_id"`)
	assert.Contains(t, log, `"msg":"warn without trace ID"`)
}

func TestCtxError_IncludesErrorAndTraceID(t *testing.T) {
	var buf bytes.Buffer
	setupTestLogger(&buf)

	err := errors.New("fatal error")
	ctx := WithTraceID(context.Background(), "trace-error")

	CtxError(ctx, "error occurred", err)

	log := buf.String()
	assert.Contains(t, log, `"error":"fatal error"`)
	assert.Contains(t, log, `"trace_id":"trace-error"`)
	assert.Contains(t, log, `"msg":"error occurred"`)
}

func TestNonContextError_IncludesErrorField(t *testing.T) {
	var buf bytes.Buffer
	setupTestLogger(&buf)

	err := errors.New("fail")
	Error("error message", err)

	log := buf.String()
	assert.Contains(t, log, `"error":"fail"`)
	assert.Contains(t, log, `"msg":"error message"`)
	assert.NotContains(t, log, `"trace_id"`)
}

func TestLogLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))

	Debug("debug should not show")
	Info("info should show")

	out := buf.String()
	assert.NotContains(t, out, "debug should not show")
	assert.Contains(t, out, "info should show")
}

func TestWithTraceID_RoundTrip(t *testing.T) {
	ctx := WithTraceID(context.Background(), "roundtrip-id")
	assert.Equal(t, "roundtrip-id", getTraceID(ctx))
}

func TestCtxDebug_WithAndWithoutTraceID(t *testing.T) {
	var buf bytes.Buffer
	setupTestLogger(&buf)

	// With trace_id
	ctx := WithTraceID(context.Background(), "rid-debug")
	CtxDebug(ctx, "debug message with id")
	out := buf.String()
	assert.Contains(t, out, `"trace_id":"rid-debug"`)
	assert.Contains(t, out, `"msg":"debug message with id"`)

	buf.Reset()

	// Without trace_id
	CtxDebug(context.Background(), "debug no id")
	out = buf.String()
	assert.NotContains(t, out, `"trace_id"`)
	assert.Contains(t, out, `"msg":"debug no id"`)
}

func TestCtxWarn_WithAndWithoutTraceID(t *testing.T) {
	var buf bytes.Buffer
	setupTestLogger(&buf)

	t.Run("warn with trace ID", func(t *testing.T) {
		buf.Reset()
		ctx := WithTraceID(context.Background(), "trace-warn")
		CtxWarn(ctx, "warning with trace ID")

		log := buf.String()
		assert.Contains(t, log, `"trace_id":"trace-warn"`)
		assert.Contains(t, log, `"msg":"warning with trace ID"`)
	})

	t.Run("warn without trace ID", func(t *testing.T) {
		buf.Reset()
		CtxWarn(context.Background(), "warn without trace ID")

		log := buf.String()
		assert.NotContains(t, log, `"trace_id"`)
		assert.Contains(t, log, `"msg":"warn without trace ID"`)
	})
}

func TestWarn_NoPanic(t *testing.T) {
	var buf bytes.Buffer
	setupTestLogger(&buf)

	// Test that Warn doesn't panic
	assert.NotPanics(t, func() {
		Warn("test warning message")
	})

	log := buf.String()
	assert.Contains(t, log, `"msg":"test warning message"`)
}

func TestInit_SetsGlobalLogger(t *testing.T) {
	// Test that Init doesn't panic and sets up the logger
	assert.NotPanics(t, func() {
		Init("info")
	})

	// Verify that slog.Default is set (we can't easily test the exact configuration)
	// but we can test that it doesn't panic
	assert.NotPanics(t, func() {
		slog.Info("test message")
	})
}
