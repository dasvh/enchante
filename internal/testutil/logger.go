package testutil

import (
	"bytes"
	"context"
	"log/slog"
	"sync"
)

var logBuffer bytes.Buffer
var logMu sync.Mutex

var Logger = slog.New(&CustomLogHandler{level: slog.LevelDebug})

// GetLogs returns the logs safely (prevent race conditions)
func GetLogs() string {
	logMu.Lock()
	defer logMu.Unlock()
	return logBuffer.String()
}

// WriteLog writes a log message to the buffer safely (prevent race conditions)
func WriteLog(msg string) {
	logMu.Lock()
	defer logMu.Unlock()
	logBuffer.WriteString(msg + "\n")
}

type CustomLogHandler struct{ level slog.Level }

// Handle implements the slog.Handler interface
func (h *CustomLogHandler) Handle(ctx context.Context, r slog.Record) error {
	WriteLog(r.Message)
	return nil
}

func (h *CustomLogHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *CustomLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler { return h }

func (h *CustomLogHandler) WithGroup(name string) slog.Handler { return h }
