package testutil

import (
	"bytes"
	"log/slog"
)

var logBuffer bytes.Buffer

var Logger = slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{Level: slog.LevelDebug}))

func GetLogs() string {
	return logBuffer.String()
}
