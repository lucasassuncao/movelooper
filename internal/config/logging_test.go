package config

import (
	"path/filepath"
	"testing"

	"github.com/knadh/koanf/v2"
	"github.com/pterm/pterm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testParseLogLevel defines the structure for test cases of the parseLogLevel function,
// containing the input string and expected pterm.LogLevel.
type testParseLogLevel struct {
	name  string
	input string
	want  pterm.LogLevel
}

// testParseLogLevelTestCases defines a set of test cases for the parseLogLevel function,
// covering all named levels, empty string, unknown string, and case-sensitive mismatch.
var testParseLogLevelTestCases = []testParseLogLevel{
	{"trace", "trace", pterm.LogLevelTrace},
	{"debug", "debug", pterm.LogLevelDebug},
	{"info", "info", pterm.LogLevelInfo},
	{"warn", "warn", pterm.LogLevelWarn},
	{"warning", "warning", pterm.LogLevelWarn},
	{"error", "error", pterm.LogLevelError},
	{"fatal", "fatal", pterm.LogLevelFatal},
	{"empty defaults to info", "", pterm.LogLevelInfo},
	{"unknown defaults to info", "unknown", pterm.LogLevelInfo},
	{"INFO case sensitive falls to default", "INFO", pterm.LogLevelInfo},
}

// TestParseLogLevel tests the parseLogLevel function to ensure it correctly maps
// log level strings to pterm.LogLevel values.
func TestParseLogLevel(t *testing.T) {
	t.Parallel()
	for _, tt := range testParseLogLevelTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, parseLogLevel(tt.input))
		})
	}
}

// testLogWriterFactory defines the structure for test cases of the logWriterFactory function,
// containing the output type and whether the result must be a consoleStrategy.
type testLogWriterFactory struct {
	name        string
	output      string
	wantConsole bool
}

// testLogWriterFactoryTestCases defines a set of test cases for the logWriterFactory function,
// covering console, file, log, both, unknown, and empty output types.
var testLogWriterFactoryTestCases = []testLogWriterFactory{
	{"console", "console", false},
	{"file", "file", false},
	{"log", "log", false},
	{"both", "both", false},
	{"unknown falls to console", "unknown", true},
	{"empty falls to console", "", true},
}

// TestLogWriterFactory tests the logWriterFactory function to ensure it returns
// the correct strategy for each output type.
func TestLogWriterFactory(t *testing.T) {
	t.Parallel()
	for _, tt := range testLogWriterFactoryTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := logWriterFactory(tt.output)
			assert.NotNil(t, s)
			if tt.wantConsole {
				_, ok := s.(consoleStrategy)
				assert.True(t, ok, "expected consoleStrategy for output=%q", tt.output)
			}
		})
	}
}

// testConsoleStrategyWriter defines the structure for test cases of the consoleStrategy.Writer method,
// containing the expected presence of a closer.
type testConsoleStrategyWriter struct {
	name       string
	wantCloser bool
}

// testConsoleStrategyWriterTestCases defines a set of test cases for the consoleStrategy.Writer method.
var testConsoleStrategyWriterTestCases = []testConsoleStrategyWriter{
	{"returns stdout writer with no closer", false},
}

// TestConsoleStrategyWriter tests the consoleStrategy.Writer method to ensure it returns
// a non-nil writer without an error or closer.
func TestConsoleStrategyWriter(t *testing.T) {
	t.Parallel()
	for _, tt := range testConsoleStrategyWriterTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			k := koanf.New(".")
			w, closer, err := consoleStrategy{}.Writer(k)
			require.NoError(t, err)
			assert.NotNil(t, w)
			if tt.wantCloser {
				assert.NotNil(t, closer)
			} else {
				assert.Nil(t, closer)
			}
		})
	}
}

// testFileStrategy defines the structure for test cases of the fileStrategy.Writer method,
// containing a yaml builder function and an expected error substring.
type testFileStrategy struct {
	name    string
	yaml    func(*testing.T) string
	wantErr string
}

// testFileStrategyTestCases defines a set of test cases for the fileStrategy.Writer method,
// covering successful log file creation and missing log-file configuration.
var testFileStrategyTestCases = []testFileStrategy{
	{
		name: "creates log file",
		yaml: func(t *testing.T) string {
			return "configuration:\n  log-file: " + filepath.Join(t.TempDir(), "logs", "app.log") + "\n"
		},
	},
	{
		name:    "error when log-file not set",
		wantErr: "log-file is required",
	},
}

// TestFileStrategy tests the fileStrategy.Writer method to ensure it correctly
// creates a log file or returns an error when log-file is not configured.
func TestFileStrategy(t *testing.T) {
	t.Parallel()
	for _, tt := range testFileStrategyTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var k *koanf.Koanf
			if tt.yaml == nil {
				k = koanf.New(".")
			} else {
				k = loadKoanfFromYAML(t, tt.yaml(t))
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

// testMultiStrategy defines the structure for test cases of the multiStrategy.Writer method,
// containing a yaml builder function and an error expectation flag.
type testMultiStrategy struct {
	name    string
	yaml    func(*testing.T) string
	wantErr bool
}

// testMultiStrategyTestCases defines a set of test cases for the multiStrategy.Writer method,
// covering successful multi-writer creation and missing log-file configuration.
var testMultiStrategyTestCases = []testMultiStrategy{
	{
		name: "creates file and multi-writer",
		yaml: func(t *testing.T) string {
			return "configuration:\n  log-file: " + filepath.Join(t.TempDir(), "app.log") + "\n"
		},
	},
	{
		name:    "error when log-file not set",
		wantErr: true,
	},
}

// TestMultiStrategy tests the multiStrategy.Writer method to ensure it correctly
// creates a multi-writer or returns an error when log-file is not configured.
func TestMultiStrategy(t *testing.T) {
	t.Parallel()
	for _, tt := range testMultiStrategyTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var k *koanf.Koanf
			if tt.yaml == nil {
				k = koanf.New(".")
			} else {
				k = loadKoanfFromYAML(t, tt.yaml(t))
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

// testConfigureLogger defines the structure for test cases of the ConfigureLogger function,
// containing a yaml builder function, closer expectation flag, and caller expectation flag.
type testConfigureLogger struct {
	name       string
	yaml       func(*testing.T) string
	wantCloser bool
	wantCaller bool
}

// testConfigureLoggerTestCases defines a set of test cases for the ConfigureLogger function,
// covering console, file, both, and unknown output types, and show-caller configuration.
var testConfigureLoggerTestCases = []testConfigureLogger{
	{
		name: "console output",
		yaml: func(*testing.T) string { return "configuration:\n  output: console\n  log-level: debug\n" },
	},
	{
		name: "file output creates closer",
		yaml: func(t *testing.T) string {
			return "configuration:\n  output: file\n  log-file: " + filepath.Join(t.TempDir(), "app.log") + "\n"
		},
		wantCloser: true,
	},
	{
		name: "both output creates closer",
		yaml: func(t *testing.T) string {
			return "configuration:\n  output: both\n  log-file: " + filepath.Join(t.TempDir(), "app.log") + "\n"
		},
		wantCloser: true,
	},
	{
		name: "unknown output defaults to console",
		yaml: func(*testing.T) string { return "configuration:\n  output: syslog\n" },
	},
	{
		name:       "show-caller enabled",
		yaml:       func(*testing.T) string { return "configuration:\n  output: console\n  show-caller: true\n" },
		wantCaller: true,
	},
}

// TestConfigureLogger tests the ConfigureLogger function to ensure it correctly
// configures the logger with the specified output type, log level, and caller settings.
func TestConfigureLogger(t *testing.T) {
	t.Parallel()
	for _, tt := range testConfigureLoggerTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			k := loadKoanfFromYAML(t, tt.yaml(t))
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

// loadKoanfFromYAML is a helper that writes yaml content to a temp file and loads it into koanf.
func loadKoanfFromYAML(t *testing.T, content string) *koanf.Koanf {
	t.Helper()
	dir := t.TempDir()
	path := writeYAML(t, dir, "cfg.yaml", content)
	k := koanf.New(".")
	require.NoError(t, InitConfig(k, path))
	return k
}
