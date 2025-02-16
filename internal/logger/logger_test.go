package logger_test

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/dasvh/enchante/internal/logger"
	"github.com/stretchr/testify/assert"
)

func TestLoggerLevels(t *testing.T) {
	tests := []struct {
		level   slog.Level
		message string
		want    string
	}{
		{slog.LevelInfo, "Info message", "[INFO]"},
		{slog.LevelDebug, "Debug message", "[DEBUG]"},
		{slog.LevelWarn, "Warn message", "[WARN]"},
		{slog.LevelError, "Error message", "[ERROR]"},
	}

	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			var buf bytes.Buffer
			l := logger.NewCustomHandler(&buf, slog.HandlerOptions{Level: tt.level}, true)
			log := slog.New(l)

			log.LogAttrs(t.Context(), tt.level, tt.message)

			got := buf.String()
			assert.Contains(t, got, tt.want, "Expected level %s in log output", tt.want)
			assert.Contains(t, got, tt.message, "Expected message %s in log output", tt.message)
		})
	}
}

func TestLoggerWithAttributes(t *testing.T) {
	var buf bytes.Buffer
	l := logger.NewCustomHandler(&buf, slog.HandlerOptions{Level: slog.LevelInfo}, true)
	log := slog.New(l)

	log.Info("User logged in", slog.String("user", "admin"), slog.Int("id", 42))

	got := buf.String()
	assert.Contains(t, got, "user=admin", "Expected 'user=admin' in log output")
	assert.Contains(t, got, "id=42", "Expected 'id=42' in log output")
}

func TestLoggerWithSource(t *testing.T) {
	var buf bytes.Buffer
	l := logger.NewCustomHandler(&buf, slog.HandlerOptions{Level: slog.LevelInfo}, true)
	log := slog.New(l)

	log.Info("Source test")

	got := buf.String()
	assert.Contains(t, got, ".go:", "Expected source file info in log output")
}

func TestLoggerWithoutSource(t *testing.T) {
	var buf bytes.Buffer
	l := logger.NewCustomHandler(&buf, slog.HandlerOptions{Level: slog.LevelInfo}, false)
	log := slog.New(l)

	log.Info("No source test")

	got := buf.String()
	assert.NotContains(t, got, ".go:", "Did not expect source file info in log output")
}
