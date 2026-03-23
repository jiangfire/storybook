package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

func New(level, format string, out io.Writer) (*slog.Logger, error) {
	if out == nil {
		out = os.Stdout
	}

	logLevel, err := parseLevel(level)
	if err != nil {
		return nil, err
	}

	options := &slog.HandlerOptions{Level: logLevel}
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "text":
		return slog.New(slog.NewTextHandler(out, options)), nil
	case "json":
		return slog.New(slog.NewJSONHandler(out, options)), nil
	default:
		return nil, fmt.Errorf("unsupported LOG_FORMAT: %s", format)
	}
}

func parseLevel(raw string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("unsupported LOG_LEVEL: %s", raw)
	}
}
