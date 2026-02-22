package log

import (
	"log/slog"
	"time"
)

type Attr = slog.Attr

func Any(key string, value any) Attr {
	return slog.Any(key, value)
}

func Bool(key string, value bool) Attr {
	return slog.Bool(key, value)
}

func Duration(key string, value time.Duration) Attr {
	return slog.Duration(key, value)
}

func Float64(key string, value float64) Attr {
	return slog.Float64(key, value)
}

func Group(key string, args ...any) Attr {
	return slog.Group(key, args...)
}

func GroupAttrs(key string, attrs ...Attr) Attr {
	return slog.GroupAttrs(key, attrs...)
}

func Int(key string, value int) Attr {
	return slog.Int(key, value)
}

func Int64(key string, value int64) Attr {
	return slog.Int64(key, value)
}

func String(key, value string) Attr {
	return slog.String(key, value)
}

func Time(key string, value time.Time) Attr {
	return slog.Time(key, value)
}

func Uint64(key string, value uint64) Attr {
	return slog.Uint64(key, value)
}
