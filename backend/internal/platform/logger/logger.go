// Package logger builds the structured logger used by the MOVO binaries.
package logger

import (
	"io"
	"log/slog"
	"strings"
)

// New returns a JSON slog.Logger writing to w that drops records below level.
//
// level accepts the values produced by config.Load (case-insensitive); any
// other value falls back to info, which is also the config default.
func New(level string, w io.Writer) *slog.Logger {
	if w == nil {
		w = io.Discard
	}

	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: parseLevel(level)})
	return slog.New(handler)
}

// parseLevel maps a textual level to its slog severity.
func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
