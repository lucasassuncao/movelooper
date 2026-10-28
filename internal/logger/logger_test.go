package logger

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/lucasassuncao/bezel/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewJSON_EmitsArgs(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	NewJSON(&buf, "info", false).Info("moved file", slog.String("file", "a.jpg"), slog.Int("count", 3))

	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	assert.Equal(t, "moved file", entry["msg"])
	assert.Equal(t, "INFO", entry["level"])
	assert.Equal(t, "a.jpg", entry["file"])
	assert.Equal(t, float64(3), entry["count"])
}

func TestNewJSON_FiltersByLevel(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	l := NewJSON(&buf, "warn", false)
	l.Info("dropped")
	l.Warn("kept")
	assert.NotContains(t, buf.String(), "dropped")
	assert.Contains(t, buf.String(), "kept")
}

func TestNewJSON_NamesTraceAndFatal(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	l := NewJSON(&buf, "trace", false)
	l.Log(t.Context(), LevelTrace, "low")
	l.Log(t.Context(), levelFatal, "high")
	assert.Contains(t, buf.String(), `"level":"TRACE"`)
	assert.Contains(t, buf.String(), `"level":"FATAL"`)
}

func TestNewJSON_AddsSource(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	NewJSON(&buf, "info", true).Info("with caller")
	assert.Contains(t, buf.String(), `"source"`)
}

func TestNewPretty_HangsArgsUnderTheMessage(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	l := NewPretty(&buf, theme.Resolve(theme.Default, true), "info", false, "never", 0)
	l.Info("moved", slog.String("file", "a.jpg"))
	assert.Contains(t, buf.String(), "INFO  moved\n")
	assert.Contains(t, buf.String(), "└─ file=a.jpg\n")
	assert.True(t, Pretty(l))
	assert.False(t, Pretty(NewJSON(&buf, "info", false)))
}

func TestNewPretty_ColoursOnlyWhenAsked(t *testing.T) {
	t.Parallel()
	th := theme.Resolve(theme.Default, true)
	var on, off bytes.Buffer
	NewPretty(&on, th, "info", false, "always", 0).Info("m")
	NewPretty(&off, th, "info", false, "auto", 0).Info("m")
	assert.Contains(t, on.String(), "\x1b[")
	assert.NotContains(t, off.String(), "\x1b[", "a buffer is no terminal")
}

func TestLevel(t *testing.T) {
	t.Parallel()
	for name, want := range map[string]slog.Level{
		"trace": LevelTrace, "debug": slog.LevelDebug, "info": slog.LevelInfo,
		"warn": slog.LevelWarn, "warning": slog.LevelWarn, "error": slog.LevelError,
		"fatal": levelFatal, "": slog.LevelInfo, "INFO": slog.LevelInfo,
	} {
		assert.Equal(t, want, Level(name), name)
	}
}

func TestDiscard(t *testing.T) {
	t.Parallel()
	assert.False(t, Discard().Enabled(t.Context(), slog.LevelError))
}
