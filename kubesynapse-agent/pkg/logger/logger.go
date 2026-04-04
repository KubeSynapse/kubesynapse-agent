package logger

import (
	"context"
	"log/slog"
	"os"
)

// Instance is the global logger.
var Instance *slog.Logger

func init() {
	// Initialize with a default JSON logger (Enterprise Standard)
	Instance = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// Info logs an informational message.
func Info(msg string, args ...any) {
	Instance.Info(msg, args...)
}

// Error logs an error message.
func Error(msg string, args ...any) {
	Instance.Error(msg, args...)
}

// Debug logs a debug message.
func Debug(msg string, args ...any) {
	Instance.Debug(msg, args...)
}

// Warn logs a warning message.
func Warn(msg string, args ...any) {
	Instance.Warn(msg, args...)
}

// With returns a new logger with the given attributes.
func With(args ...any) *slog.Logger {
	return Instance.With(args...)
}

// WithContext returns a logger that is aware of the context (for tracing/correlation ids).
func WithContext(ctx context.Context) *slog.Logger {
	// Future-proofing for context-based attributes
	return Instance
}
