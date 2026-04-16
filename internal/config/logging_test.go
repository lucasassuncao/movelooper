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

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
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
		{"INFO", pterm.LogLevelInfo}, // case sensitive - falls to default
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, parseLogLevel(tt.input))
		})
	}
}

// --- logWriterFactory ---

func TestLogWriterFactory(t *testing.T) {
	tests := []struct {
		output      string
		wantConsole bool // true = result must be a consoleStrategy
	}{
		{"console", false},
		{"file", false},
		{"log", false},
		{"both", false},
		{"unknown", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.output, func(t *testing.T) {
			s := logWriterFactory(tt.output)
			assert.NotNil(t, s)
			if tt.wantConsole {
				_, ok := s.(consoleStrategy)
				assert.True(t, ok, "expected consoleStrategy for output=%q", tt.output)
			}
		})
	}
}

// --- consoleStrategy ---

func TestConsoleStrategy_WriterReturnsStdout(t *testing.T) {
	k := koanf.New(".")
	w, closer, err := consoleStrategy{}.Writer(k)
	require.NoError(t, err)
	assert.NotNil(t, w)
	assert.Nil(t, closer)
}

// --- fileStrategy ---

func TestFileStrategy(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name: "creates log file",
			yaml: func() string {
				return "configuration:\n  log-file: " + filepath.Join(t.TempDir(), "logs", "app.log") + "\n"
			}(),
		},
		{
			name:    "error when log-file not set",
			yaml:    "",
			wantErr: "log-file is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var k *koanf.Koanf
			if tt.yaml == "" {
				k = koanf.New(".")
			} else {
				k = loadKoanfFromYAML(t, tt.yaml)
			}

			w, closer, err := fileStrategy{}.Writer(k)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, w)
			require.NotNil(t, closer)
			assert.NoError(t, closer.Close())
		})
	}
}

// --- multiStrategy ---

func TestMultiStrategy(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{
			name: "creates file and multi-writer",
			yaml: func() string {
				return "configuration:\n  log-file: " + filepath.Join(t.TempDir(), "app.log") + "\n"
			}(),
		},
		{
			name:    "error when log-file not set",
			yaml:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var k *koanf.Koanf
			if tt.yaml == "" {
				k = koanf.New(".")
			} else {
				k = loadKoanfFromYAML(t, tt.yaml)
			}

			w, closer, err := multiStrategy{}.Writer(k)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, w)
			require.NotNil(t, closer)
			assert.NoError(t, closer.Close())
		})
	}
}

// --- ConfigureLogger ---

func TestConfigureLogger(t *testing.T) {
	tests := []struct {
		name       string
		yaml       func() string
		wantCloser bool
		wantCaller bool
	}{
		{
			name: "console output",
			yaml: func() string { return "configuration:\n  output: console\n  log-level: debug\n" },
		},
		{
			name: "file output creates closer",
			yaml: func() string {
				return "configuration:\n  output: file\n  log-file: " + filepath.Join(t.TempDir(), "app.log") + "\n"
			},
			wantCloser: true,
		},
		{
			name: "both output creates closer",
			yaml: func() string {
				return "configuration:\n  output: both\n  log-file: " + filepath.Join(t.TempDir(), "app.log") + "\n"
			},
			wantCloser: true,
		},
		{
			name: "unknown output defaults to console",
			yaml: func() string { return "configuration:\n  output: syslog\n" },
		},
		{
			name:       "show-caller enabled",
			yaml:       func() string { return "configuration:\n  output: console\n  show-caller: true\n" },
			wantCaller: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := loadKoanfFromYAML(t, tt.yaml())
			logger, closer, err := ConfigureLogger(k)
			require.NoError(t, err)
			assert.NotNil(t, logger)
			if tt.wantCloser {
				require.NotNil(t, closer)
				assert.NoError(t, closer.Close())
			} else {
				assert.Nil(t, closer)
			}
			if tt.wantCaller {
				assert.True(t, logger.ShowCaller)
			}
		})
	}
}
