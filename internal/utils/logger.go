// Package utils provides utility functions including logging.
package utils

import (
	"log/slog"
	"os"
)

// Logger is the global structured logger instance.
var Logger *slog.Logger

// InitLogger initializes the structured logger with JSON output.
func InitLogger(env, service string) {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	// Use JSON handler for structured logging
	handler := slog.NewJSONHandler(os.Stdout, opts)
	Logger = slog.New(handler)

	// Set as default logger
	slog.SetDefault(Logger)

	// Log initialization with required fields
	Logger.Info("logger initialized",
		slog.String("level", "info"),
		slog.String("env", env),
		slog.String("service", service),
	)
}

// Info logs an info level message with optional key-value pairs.
func Info(msg string, args ...any) {
	Logger.Info(msg, args...)
}

// Error logs an error level message with optional key-value pairs.
func Error(msg string, args ...any) {
	Logger.Error(msg, args...)
}

// Debug logs a debug level message with optional key-value pairs.
func Debug(msg string, args ...any) {
	Logger.Debug(msg, args...)
}

// Warn logs a warning level message with optional key-value pairs.
func Warn(msg string, args ...any) {
	Logger.Warn(msg, args...)
}
