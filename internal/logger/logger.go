package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	timestampStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	infoStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // green
	debugStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("13")) // purple
	warnStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("11")) // yellow
	errorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))  // red
	msgStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("15")) // white
	attrStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))  // grey
	sourceStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))  // dark grey
)

// CustomHandler is a custom slog.Handler that applies colors using lipgloss
type CustomHandler struct {
	slog.Handler
	writer     io.Writer
	showSource bool
}

// NewCustomHandler creates a new CustomHandler for colored logs
func NewCustomHandler(out io.Writer, opts slog.HandlerOptions, showSource bool) *CustomHandler {
	return &CustomHandler{
		Handler:    slog.NewTextHandler(out, &opts),
		writer:     out,
		showSource: showSource,
	}
}

// Handle formats and prints log messages with colors
func (h *CustomHandler) Handle(ctx context.Context, r slog.Record) error {
	timestamp := timestampStyle.Render(r.Time.Format("15:04:05.000"))
	levelStr := r.Level.String()
	coloredLevel := formatLevel(levelStr)
	message := msgStyle.Render(r.Message)
	source := ""
	if h.showSource {
		source = sourceStyle.Render(getCallerInfo())
	}

	var attrStr string
	r.Attrs(func(a slog.Attr) bool {
		attrStr += fmt.Sprintf(" %s=%v", attrStyle.Render(a.Key), a.Value.Any())
		return true
	})

	if h.showSource {
		fmt.Fprintf(h.writer, "%s %s %s %s%s\n", timestamp, coloredLevel, source, message, attrStr)
	} else {
		fmt.Fprintf(h.writer, "%s %s %s%s\n", timestamp, coloredLevel, message, attrStr)
	}

	return nil
}

func formatLevel(level string) string {
	switch level {
	case "DEBUG":
		return debugStyle.Render("[DEBUG]")
	case "INFO":
		return infoStyle.Render("[INFO]")
	case "WARN":
		return warnStyle.Render("[WARN]")
	case "ERROR":
		return errorStyle.Render("[ERROR]")
	default:
		return level
	}
}

// trimPath removes the base directory
func trimPath(fullPath string) string {
	parts := strings.Split(fullPath, "/")
	if len(parts) > 3 {
		return strings.Join(parts[len(parts)-3:], "/")
	}
	return fullPath
}

// getCallerInfo retrieves the file and line number of the actual caller
func getCallerInfo() string {
	const maxDepth = 15
	for i := 3; i < maxDepth; i++ {
		_, file, line, ok := runtime.Caller(i)
		if !ok {
			return "unknown"
		}
		// skip stdlib and slog package files
		if !strings.Contains(file, "log/slog") && !strings.Contains(file, "/src/runtime/") {
			return fmt.Sprintf("%s:%d", trimPath(file), line)
		}
	}
	return "unknown"
}

// NewLogger initializes the logger with optional debug mode
func NewLogger(debug bool) *slog.Logger {
	var level slog.Level
	if debug {
		level = slog.LevelDebug
	} else {
		level = slog.LevelInfo
	}

	handler := NewCustomHandler(os.Stdout, slog.HandlerOptions{
		Level: level,
	}, debug)

	return slog.New(handler)
}
