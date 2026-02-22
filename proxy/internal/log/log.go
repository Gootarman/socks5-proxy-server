package log

import (
	"context"
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
	case OutputText:
		h = slog.NewTextHandler(os.Stdout, hOpts)
	default:
		panic(fmt.Sprintf("unknown output format %q", outputFormat))
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

func Debug(ctx context.Context, msg string, attrs ...Attr) {
	slog.LogAttrs(ctx, slog.LevelDebug, msg, attrs...)
}

func Info(ctx context.Context, msg string, attrs ...Attr) {
	slog.LogAttrs(ctx, slog.LevelInfo, msg, attrs...)
}

func Warn(ctx context.Context, msg string, attrs ...Attr) {
	slog.LogAttrs(ctx, slog.LevelWarn, msg, attrs...)
}

func Error(ctx context.Context, msg string, attrs ...Attr) {
	slog.LogAttrs(ctx, slog.LevelError, msg, attrs...)
}
