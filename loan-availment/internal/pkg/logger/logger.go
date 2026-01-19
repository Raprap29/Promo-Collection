package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"globe/dodrio_loan_availment/configs"
	"globe/dodrio_loan_availment/internal/app/middleware"
	"globe/dodrio_loan_availment/internal/pkg/downstreams/models"
	"time"

	"runtime"
	"strings"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	log         *zap.Logger
	serviceName string
)

func init() {

	err := configs.LoadEnv()
	if err != nil {
		fmt.Printf("error loading .env file : %v", err)
	}
	serviceName = configs.SERVICE_NAME
	if serviceName == "" {
		serviceName = "loanavailment"
	}

	logLevelStr := configs.LOG_LEVEL
	var logLevel zapcore.Level

	// Map the LOG_LEVEL string to zapcore.Level
	switch strings.ToLower(logLevelStr) {
	case "debug":
		logLevel = zap.DebugLevel
	case "info":
		logLevel = zap.InfoLevel
	case "warn":
		logLevel = zap.WarnLevel
	case "error":
		logLevel = zap.ErrorLevel
	case "panic":
		logLevel = zap.PanicLevel
	default:
		logLevel = zap.InfoLevel
	}

	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(logLevel)
	config.Encoding = "json"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.LevelKey = "log_level"
	config.EncoderConfig.MessageKey = "message"
	config.EncoderConfig.TimeKey = "timestamp"
	config.OutputPaths = []string{"stdout"} // Logging as stdout
	config.EncoderConfig.CallerKey = ""
	config.EncoderConfig.StacktraceKey = ""
	config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	log, _ = config.Build(zap.AddCallerSkip(1))
}

func Info(args ...interface{}) {
	logMessage(zap.InfoLevel, args...)
}

func Debug(args ...interface{}) {
	logMessage(zap.DebugLevel, args...)
}

func Error(args ...interface{}) {
	logMessage(zap.ErrorLevel, args...)
}

func Warn(args ...interface{}) {
	logMessage(zap.WarnLevel, args...)
}

func Panic(args ...interface{}) {
	logMessage(zap.PanicLevel, args...)
}

func logMessage(level zapcore.Level, args ...interface{}) {
	var msg string
	var fields []zapcore.Field
	var ctx context.Context
	seenFields := make(map[string]struct{}) // Track added field keys

	if len(args) > 0 {
		if firstArg, ok := args[0].(context.Context); ok {
			ctx = firstArg
			if len(args) > 1 {
				switch secondArg := args[1].(type) {
				case string:
					msg = formatMessage(args[1:]...)
				default:
					jsonFields := extractFieldsFromJSON(secondArg)
					msg = formatMessage(args[2:]...)
					fields = append(fields, jsonFields...)
				}
			}
		} else {
			msg = formatMessage(args...)
		}
	}

	essentialFields := extractEssentialFields(ctx)
	for _, field := range essentialFields {
		if _, exists := seenFields[field.Key]; !exists {
			fields = append(fields, field)
			seenFields[field.Key] = struct{}{}
		}
	}

	globalLogLevel := log.Core().Enabled(zap.DebugLevel)
	if globalLogLevel || level == zap.DebugLevel {
		callerFields := getCallerInfo(3)
		for _, field := range callerFields {
			if _, exists := seenFields[field.Key]; !exists {
				fields = append(fields, field)
				seenFields[field.Key] = struct{}{}
			}
		}
	}

	switch level {
	case zap.DebugLevel:
		log.Debug(msg, fields...)
	case zap.InfoLevel:
		log.Info(msg, fields...)
	case zap.WarnLevel:
		log.Warn(msg, fields...)
	case zap.ErrorLevel:
		log.Error(msg, fields...)
	case zap.PanicLevel:
		log.Panic(msg, fields...)
	}
}
func extractFieldsFromJSON(data interface{}) []zapcore.Field {
	var fields []zapcore.Field

	// Add a timestamp field
	timestamp := time.Now().Format(time.RFC3339)
	fields = append(fields, zap.String("timestamp", timestamp))

	// Convert the struct to JSON
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		fields = append(fields, zap.String("error", fmt.Sprintf("failed to serialize struct: %v", err)))
		return fields
	}

	// Unmarshal the JSON into a map
	var jsonMap map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &jsonMap); err != nil {
		fields = append(fields, zap.String("error", fmt.Sprintf("failed to parse JSON: %v", err)))
		return fields
	}

	// Convert the map to zapcore.Field values, ensuring all values are strings
	for key, value := range jsonMap {
		fields = append(fields, zap.String(key, fmt.Sprintf("%v", value)))
	}

	return fields
}

func formatMessage(args ...interface{}) string {
	if len(args) == 0 {
		return ""
	}
	msg, ok := args[0].(string)
	if !ok {
		msg = fmt.Sprintf("%v", args[0])
	}

	if len(args) > 1 {
		msg = fmt.Sprintf(msg, args[1:]...)
	}
	return msg
}

func extractEssentialFields(ctx context.Context) []zapcore.Field {
	var fields []zapcore.Field

	if ctx != nil {
		if details, ok := ctx.Value(middleware.LogDetailsKey).(models.RequestDetails); ok {
			fields = append(fields, zap.String("request_id", details.RequestID))
		}
		span := trace.SpanFromContext(ctx)
		if span != nil {
			spanContext := span.SpanContext()
			if spanContext.HasTraceID() {
				fields = append(fields, zap.String("trace_id", spanContext.TraceID().String()))
			}
		}

		fields = append(fields, zap.String("service_name", serviceName))
	}

	return fields
}

func getCallerInfo(skip int) []zapcore.Field {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return nil
	}

	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return nil
	}

	fileParts := strings.Split(file, "/")
	fileName := fileParts[len(fileParts)-1]

	return []zapcore.Field{
		zap.String("file_name", fileName),
		zap.Int("line_number", line),
		zap.String("function_name", extractFunctionName(fn.Name())),
	}
}

func extractFunctionName(fullFuncName string) string {
	parts := strings.Split(fullFuncName, ".")
	return parts[len(parts)-1]
}
