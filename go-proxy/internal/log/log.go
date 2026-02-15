package log

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

type Output string

const (
	OutputJSON Output = "json"
	OutputText Output = "text"
)

func SetDefaultWithParams(outputFormat Output, logLevel slog.Level) {
	hOpts := &slog.HandlerOptions{Level: logLevel}

	var h slog.Handler

	switch outputFormat {
	case OutputJSON:
		h = slog.NewJSONHandler(os.Stdout, hOpts)
	default:
		h = slog.NewTextHandler(os.Stdout, hOpts)
	}

	slog.SetDefault(slog.New(h))
}

func ParseStringLogLevel(logLevel string) slog.Level {
	switch strings.ToLower(logLevel) {
	case "info":
		return slog.LevelInfo
	case "debug":
		return slog.LevelDebug
	case "error":
		return slog.LevelError
	case "warn", "warning":
		return slog.LevelWarn
	default:
		panic(fmt.Sprintf("unknown log level %s, must be one of: info, debug, warn or error", logLevel))
	}
}
