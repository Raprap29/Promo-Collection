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

func TestGetRequestID(t *testing.T) {
	// valid request ID
	ctxWithID := context.WithValue(context.Background(), traceIDKey, "id123")
	assert.Equal(t, "id123", GetTraceID(ctxWithID))

	// no request ID returns empty string
	assert.Empty(t, GetTraceID(context.Background()))

	// request ID with wrong type returns empty string
	ctxWrongType := context.WithValue(context.Background(), traceIDKey, 42)
	assert.Empty(t, GetTraceID(ctxWrongType))
}

func TestCtxLogging_InjectsRequestID(t *testing.T) {
	var buf bytes.Buffer
	setupTestLogger(&buf)

	ctx := WithTraceID(context.Background(), "req-edge")
	CtxInfo(ctx, "info with reqid")

	log := buf.String()
	assert.Contains(t, log, `"trace_id":"req-edge"`)
	assert.Contains(t, log, `"msg":"info with reqid"`)
}

func TestCtxLogging_NoRequestID(t *testing.T) {
	var buf bytes.Buffer
	setupTestLogger(&buf)

	CtxWarn(context.Background(), "warn without reqid")

	log := buf.String()
	assert.NotContains(t, log, `"trace_id"`)
	assert.Contains(t, log, `"msg":"warn without reqid"`)
}

func TestCtxError_IncludesErrorAndRequestID(t *testing.T) {
	var buf bytes.Buffer
	setupTestLogger(&buf)

	err := errors.New("fatal error")
	ctx := WithTraceID(context.Background(), "req-error")

	CtxError(ctx, "error occurred", err)

	log := buf.String()
	assert.Contains(t, log, `"error":"fatal error"`)
	assert.Contains(t, log, `"trace_id":"req-error"`)
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

func TestWithRequestID_RoundTrip(t *testing.T) {
	ctx := WithTraceID(context.Background(), "roundtrip-id")
	assert.Equal(t, "roundtrip-id", GetTraceID(ctx))
}

func TestCtxDebug_WithAndWithoutRequestID(t *testing.T) {
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

func TestCtxWarn_WithAndWithoutRequestID(t *testing.T) {
	var buf bytes.Buffer
	setupTestLogger(&buf)

	t.Run("warn with request ID", func(t *testing.T) {
		buf.Reset()
		ctx := WithTraceID(context.Background(), "req-warn")
		CtxWarn(ctx, "warning with reqid")

		log := buf.String()
		assert.Contains(t, log, `"trace_id":"req-warn"`)
		assert.Contains(t, log, `"msg":"warning with reqid"`)
	})

	t.Run("warn without request ID", func(t *testing.T) {
		buf.Reset()
		CtxWarn(context.Background(), "warn without reqid")

		log := buf.String()
		assert.NotContains(t, log, `"trace_id"`)
		assert.Contains(t, log, `"msg":"warn without reqid"`)
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
		Init()
	})

	// Verify that slog.Default is set (we can't easily test the exact configuration)
	// but we can test that it doesn't panic
	assert.NotPanics(t, func() {
		slog.Info("test message")
	})
}
