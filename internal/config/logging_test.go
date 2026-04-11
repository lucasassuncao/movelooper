package config

import (
	"path/filepath"
	"testing"

	"github.com/knadh/koanf/v2"
	"github.com/pterm/pterm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loadKoanfFromYAML is a helper that writes yaml content to a temp file and loads it into koanf.
func loadKoanfFromYAML(t *testing.T, content string) *koanf.Koanf {
	t.Helper()
	dir := t.TempDir()
	path := writeYAML(t, dir, "cfg.yaml", content)
	k := koanf.New(".")
	require.NoError(t, InitConfig(k, path))
	return k
}

// --- parseLogLevel ---

func TestParseLogLevel_AllLevels(t *testing.T) {
	cases := []struct {
		input string
		want  pterm.LogLevel
	}{
		{"trace", pterm.LogLevelTrace},
		{"debug", pterm.LogLevelDebug},
		{"info", pterm.LogLevelInfo},
		{"warn", pterm.LogLevelWarn},
		{"warning", pterm.LogLevelWarn},
		{"error", pterm.LogLevelError},
		{"fatal", pterm.LogLevelFatal},
		{"", pterm.LogLevelInfo},
		{"unknown", pterm.LogLevelInfo},
		{"INFO", pterm.LogLevelInfo}, // case sensitive — falls to default
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, parseLogLevel(tc.input), "input=%q", tc.input)
	}
}

// --- logWriterFactory ---

func TestLogWriterFactory_KnownStrategies(t *testing.T) {
	for _, output := range []string{"console", "file", "log", "both"} {
		s := logWriterFactory(output)
		assert.NotNil(t, s, "output=%q", output)
	}
}

func TestLogWriterFactory_UnknownFallsToConsole(t *testing.T) {
	s := logWriterFactory("unknown")
	_, ok := s.(consoleStrategy)
	assert.True(t, ok, "unknown output should fall back to consoleStrategy")
}

func TestLogWriterFactory_EmptyFallsToConsole(t *testing.T) {
	s := logWriterFactory("")
	_, ok := s.(consoleStrategy)
	assert.True(t, ok)
}

// --- consoleStrategy ---

func TestConsoleStrategy_WriterReturnsStdout(t *testing.T) {
	k := koanf.New(".")
	w, closer, err := consoleStrategy{}.Writer(k)
	require.NoError(t, err)
	assert.NotNil(t, w)
	assert.Nil(t, closer)
}

// --- fileStrategy / openLogFile ---

func TestFileStrategy_WriterCreatesFile(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "logs", "app.log")
	k := loadKoanfFromYAML(t, "configuration:\n  log-file: "+logPath+"\n")

	w, closer, err := fileStrategy{}.Writer(k)
	require.NoError(t, err)
	assert.NotNil(t, w)
	require.NotNil(t, closer)
	assert.NoError(t, closer.Close())
}

func TestFileStrategy_WriterErrorWhenNoLogFile(t *testing.T) {
	k := koanf.New(".") // log-file is empty
	_, _, err := fileStrategy{}.Writer(k)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "log-file is required")
}

// --- multiStrategy ---

func TestMultiStrategy_WriterCreatesFileAndMultiWriter(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "app.log")
	k := loadKoanfFromYAML(t, "configuration:\n  log-file: "+logPath+"\n")

	w, closer, err := multiStrategy{}.Writer(k)
	require.NoError(t, err)
	assert.NotNil(t, w)
	require.NotNil(t, closer)
	assert.NoError(t, closer.Close())
}

func TestMultiStrategy_WriterErrorWhenNoLogFile(t *testing.T) {
	k := koanf.New(".")
	_, _, err := multiStrategy{}.Writer(k)
	assert.Error(t, err)
}

// --- ConfigureLogger ---

func TestConfigureLogger_ConsoleOutput(t *testing.T) {
	k := loadKoanfFromYAML(t, `
configuration:
  output: console
  log-level: debug
`)
	logger, closer, err := ConfigureLogger(k)
	require.NoError(t, err)
	assert.NotNil(t, logger)
	assert.Nil(t, closer)
}

func TestConfigureLogger_FileOutput(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "app.log")
	k := loadKoanfFromYAML(t, "configuration:\n  output: file\n  log-file: "+logPath+"\n")

	logger, closer, err := ConfigureLogger(k)
	require.NoError(t, err)
	assert.NotNil(t, logger)
	require.NotNil(t, closer)
	assert.NoError(t, closer.Close())
}

func TestConfigureLogger_BothOutput(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "app.log")
	k := loadKoanfFromYAML(t, "configuration:\n  output: both\n  log-file: "+logPath+"\n")

	logger, closer, err := ConfigureLogger(k)
	require.NoError(t, err)
	assert.NotNil(t, logger)
	require.NotNil(t, closer)
	assert.NoError(t, closer.Close())
}

func TestConfigureLogger_UnknownOutputDefaultsToConsole(t *testing.T) {
	k := loadKoanfFromYAML(t, "configuration:\n  output: syslog\n")
	logger, closer, err := ConfigureLogger(k)
	require.NoError(t, err)
	assert.NotNil(t, logger)
	assert.Nil(t, closer)
}

func TestConfigureLogger_ShowCallerEnabled(t *testing.T) {
	k := loadKoanfFromYAML(t, "configuration:\n  output: console\n  show-caller: true\n")
	logger, _, err := ConfigureLogger(k)
	require.NoError(t, err)
	assert.True(t, logger.ShowCaller)
}
