package logger

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlogLogger_EmitsJSONWithArgs(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	l := NewSlog(&buf, "info", false)

	l.Info("moved file", l.Args("file", "a.jpg", "count", 3))

	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	assert.Equal(t, "moved file", entry["msg"])
	assert.Equal(t, "INFO", entry["level"])
	assert.Equal(t, "a.jpg", entry["file"])
	assert.Equal(t, float64(3), entry["count"])
}

func TestSlogLogger_LevelFiltering(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	l := NewSlog(&buf, "warn", false)

	l.Info("dropped", nil)
	l.Warn("kept", nil)

	out := buf.String()
	assert.NotContains(t, out, "dropped")
	assert.Contains(t, out, "kept")
}

func TestSlogLogger_CustomLevelLabels(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	l := NewSlog(&buf, "trace", false)

	l.Trace("low", nil)

	assert.Contains(t, buf.String(), `"level":"TRACE"`)
}

func TestSlogLogger_ArgsPairsValues(t *testing.T) {
	t.Parallel()
	l := NewSlog(&bytes.Buffer{}, "info", false)
	// trailing key without a value is dropped, not panicked on
	args := l.Args("k1", "v1", "dangling")
	require.Len(t, args, 1)
	assert.Equal(t, "k1", args[0].Key)
	assert.Equal(t, "v1", args[0].Value)
}

func TestSlogLogger_AddSourceEmitsCaller(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	l := NewSlog(&buf, "info", true)

	l.Info("with caller", nil)

	assert.True(t, strings.Contains(buf.String(), `"source"`), "expected source attribute when addSource is true")
}
