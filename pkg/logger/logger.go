package logger

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type contextKey string

const (
	ContextKeyTraceID  contextKey = "trace_id"
	ContextKeyUploadID contextKey = "upload_id"
)

type Logger struct {
	zap *zap.Logger
}

func New(level string) *Logger {
	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Parse log level
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}
	config.Level = zap.NewAtomicLevelAt(zapLevel)

	zapLogger, _ := config.Build()
	return &Logger{zap: zapLogger}
}

func NewNop() *Logger {
	return &Logger{zap: zap.NewNop()}
}

func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, ContextKeyTraceID, traceID)
}

func WithUploadID(ctx context.Context, uploadID string) context.Context {
	return context.WithValue(ctx, ContextKeyUploadID, uploadID)
}

func GetTraceID(ctx context.Context) string {
	if v := ctx.Value(ContextKeyTraceID); v != nil {
		if traceID, ok := v.(string); ok {
			return traceID
		}
	}
	return ""
}

func GetUploadID(ctx context.Context) string {
	if v := ctx.Value(ContextKeyUploadID); v != nil {
		if uploadID, ok := v.(string); ok {
			return uploadID
		}
	}
	return ""
}

func (l *Logger) buildFields(ctx context.Context, fields ...interface{}) []zap.Field {
	zapFields := []zap.Field{}

	if traceID := GetTraceID(ctx); traceID != "" {
		zapFields = append(zapFields, zap.String("trace_id", traceID))
	}

	if uploadID := GetUploadID(ctx); uploadID != "" {
		zapFields = append(zapFields, zap.String("upload_id", uploadID))
	}

	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			key, ok := fields[i].(string)
			if !ok {
				continue
			}
			value := fields[i+1]
			zapFields = append(zapFields, zap.Any(key, value))
		}
	}

	return zapFields
}

func (l *Logger) Debug(ctx context.Context, msg string, fields ...interface{}) {
	zapFields := l.buildFields(ctx, fields...)
	l.zap.Debug(msg, zapFields...)
}

func (l *Logger) Info(ctx context.Context, msg string, fields ...interface{}) {
	zapFields := l.buildFields(ctx, fields...)
	l.zap.Info(msg, zapFields...)
}

func (l *Logger) Warn(ctx context.Context, msg string, fields ...interface{}) {
	zapFields := l.buildFields(ctx, fields...)
	l.zap.Warn(msg, zapFields...)
}

func (l *Logger) Error(ctx context.Context, msg string, fields ...interface{}) {
	zapFields := l.buildFields(ctx, fields...)
	l.zap.Error(msg, zapFields...)
}

func (l *Logger) Fatal(ctx context.Context, msg string, fields ...interface{}) {
	zapFields := l.buildFields(ctx, fields...)
	l.zap.Fatal(msg, zapFields...)
}

func (l *Logger) Sync() error {
	return l.zap.Sync()
}
