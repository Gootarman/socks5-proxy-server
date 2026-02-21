package log

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetDefaultWithParams(t *testing.T) {
	t.Parallel()

	assert.NotPanics(t, func() {
		SetDefaultWithParams(OutputJSON, slog.LevelDebug)
	})
	assert.NotPanics(t, func() {
		SetDefaultWithParams(OutputText, slog.LevelInfo)
	})
	assert.Panics(t, func() {
		SetDefaultWithParams(Output("bad"), slog.LevelInfo)
	})
}

func TestParseStringLogLevel(t *testing.T) {
	t.Parallel()

	assert.Equal(t, slog.LevelInfo, ParseStringLogLevel("info"))
	assert.Equal(t, slog.LevelDebug, ParseStringLogLevel("DEBUG"))
	assert.Equal(t, slog.LevelError, ParseStringLogLevel("error"))
	assert.Equal(t, slog.LevelWarn, ParseStringLogLevel("warn"))
	assert.Equal(t, slog.LevelWarn, ParseStringLogLevel("warning"))
	assert.Panics(t, func() {
		ParseStringLogLevel("bad")
	})
}
