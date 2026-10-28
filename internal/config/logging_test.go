package config

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/knadh/koanf/v2"
	"github.com/lucasassuncao/bezel/theme"
	"github.com/lucasassuncao/movelooper/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
// covering successful log file creation and the default path fallback.
var testFileStrategyTestCases = []testFileStrategy{
	{
		name: "creates log file",
		yaml: func(t *testing.T) string {
			return "configuration:\n  logging:\n    file: " + filepath.Join(t.TempDir(), "logs", "app.log") + "\n"
		},
	},
	{
		name: "uses default log path when log-file not set",
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
// covering successful multi-writer creation and the default path fallback.
var testMultiStrategyTestCases = []testMultiStrategy{
	{
		name: "creates file and multi-writer",
		yaml: func(t *testing.T) string {
			return "configuration:\n  logging:\n    file: " + filepath.Join(t.TempDir(), "app.log") + "\n"
		},
	},
	{
		name: "uses default log path when log-file not set",
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
// containing a yaml builder function, the --show-caller flag, and what to expect.
type testConfigureLogger struct {
	name       string
	yaml       func(*testing.T) string
	showCaller bool
	wantCloser bool
	wantCaller bool
}

// testConfigureLoggerTestCases defines a set of test cases for the ConfigureLogger function,
// covering console, file, both, and unknown output types, and the --show-caller flag.
var testConfigureLoggerTestCases = []testConfigureLogger{
	{
		name: "console output",
		yaml: func(*testing.T) string { return "configuration:\n  logging:\n    output: console\n    level: debug\n" },
	},
	{
		name: "file output creates closer",
		yaml: func(t *testing.T) string {
			return "configuration:\n  logging:\n    output: file\n    file: " + filepath.Join(t.TempDir(), "app.log") + "\n"
		},
		wantCloser: true,
	},
	{
		name: "both output creates closer",
		yaml: func(t *testing.T) string {
			return "configuration:\n  logging:\n    output: both\n    file: " + filepath.Join(t.TempDir(), "app.log") + "\n"
		},
		wantCloser: true,
	},
	{
		name: "unknown output defaults to console",
		yaml: func(*testing.T) string { return "configuration:\n  logging:\n    output: syslog\n" },
	},
	{
		name: "show-caller from the flag",
		yaml: func(t *testing.T) string {
			return "configuration:\n  logging:\n    output: file\n    file: " + filepath.Join(t.TempDir(), "app.log") + "\n"
		},
		showCaller: true,
		wantCaller: true,
		wantCloser: true,
	},
	{
		name: "the config key no longer turns it on",
		yaml: func(t *testing.T) string {
			return "configuration:\n  logging:\n    output: file\n    show-caller: true\n    file: " + filepath.Join(t.TempDir(), "app.log") + "\n"
		},
		wantCloser: true,
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
			log, closer, err := ConfigureLogger(k, "", tt.showCaller, testTheme)
			require.NoError(t, err)
			require.NotNil(t, log)
			assert.True(t, logger.Pretty(log), "the default format is the pretty logger")
			log.Info("hello")
			if !tt.wantCloser {
				assert.Nil(t, closer)
				return
			}
			require.NotNil(t, closer)
			require.NoError(t, closer.Close())
			data, err := os.ReadFile(k.String("configuration.logging.file"))
			require.NoError(t, err)
			assert.Equal(t, tt.wantCaller, strings.Contains(string(data), "logging_test.go:"), string(data))
		})
	}
}

// TestConfigureLogger_JSONFormat verifies that logging.format=json selects the
// structured logger and emits valid JSON lines instead of pretty output.
func TestConfigureLogger_JSONFormat(t *testing.T) {
	t.Parallel()
	logPath := filepath.Join(t.TempDir(), "app.log")
	yaml := "configuration:\n  logging:\n    output: file\n    level: info\n    format: json\n    file: " + logPath + "\n"
	k := loadKoanfFromYAML(t, yaml)

	log, closer, err := ConfigureLogger(k, "", false, testTheme)
	require.NoError(t, err)
	require.NotNil(t, closer)

	assert.False(t, logger.Pretty(log), "json format should not use the pretty logger")

	log.Info("hello", slog.String("k", "v"))
	require.NoError(t, closer.Close())

	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	var entry map[string]any
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(data), &entry), "output should be a JSON object")
	assert.Equal(t, "hello", entry["msg"])
	assert.Equal(t, "v", entry["k"])
}

// TestConfigureLogger_FormatOverride verifies that a non-empty format override
// (from the --format flag) wins over configuration.logging.format.
func TestConfigureLogger_FormatOverride(t *testing.T) {
	t.Parallel()
	logPath := filepath.Join(t.TempDir(), "app.log")
	yaml := "configuration:\n  logging:\n    output: file\n    format: pretty\n    file: " + logPath + "\n"
	k := loadKoanfFromYAML(t, yaml)

	log, closer, err := ConfigureLogger(k, "json", false, testTheme)
	require.NoError(t, err)
	require.NotNil(t, closer)
	defer func() { require.NoError(t, closer.Close()) }()

	isPretty := logger.Pretty(log)
	assert.False(t, isPretty, "override to json should win over the configured pretty format")
}

// TestConfigureLogger_JSONShowCaller verifies that --show-caller reaches the
// structured logger too, as slog's source field.
func TestConfigureLogger_JSONShowCaller(t *testing.T) {
	t.Parallel()
	logPath := filepath.Join(t.TempDir(), "app.log")
	yaml := "configuration:\n  logging:\n    output: file\n    file: " + logPath + "\n"

	log, closer, err := ConfigureLogger(loadKoanfFromYAML(t, yaml), "json", true, testTheme)
	require.NoError(t, err)
	log.Info("hello")
	require.NoError(t, closer.Close())

	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	var entry map[string]any
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(data), &entry))
	assert.Contains(t, entry, "source")
}

// TestConfigureLogger_MaxWidth verifies that max-width wraps a long value,
// and that unset leaves a file's lines whole: a file has no terminal width.
func TestConfigureLogger_MaxWidth(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("x/", 60)
	for name, tc := range map[string]struct {
		line  string
		width int
	}{
		"unset keeps lines whole": {"", 0},
		"zero keeps lines whole":  {"    max-width: 0\n", 0},
		"a value wraps the line":  {"    max-width: 60\n", 60},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			logPath := filepath.Join(t.TempDir(), "app.log")
			yaml := "configuration:\n  logging:\n    output: file\n    file: " + logPath + "\n" + tc.line
			log, closer, err := ConfigureLogger(loadKoanfFromYAML(t, yaml), "", false, testTheme)
			require.NoError(t, err)
			log.Info("m", slog.String("path", long))
			require.NoError(t, closer.Close())

			data, err := os.ReadFile(logPath)
			require.NoError(t, err)
			lines := strings.Split(strings.TrimSpace(string(data)), "\n")
			if tc.width == 0 {
				assert.Len(t, lines, 2, "the message and one whole value line")
				return
			}
			assert.Greater(t, len(lines), 2)
			for _, l := range lines {
				assert.LessOrEqual(t, utf8.RuneCountInString(l), tc.width, l)
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

// The log file only ever grew: past the limit it now starts over, keeping the
// previous one as <file>.1.
func TestRotateLog(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "movelooper.log")
	require.NoError(t, os.WriteFile(path, []byte("0123456789"), 0o600))
	require.NoError(t, os.WriteFile(path+".1", []byte("older"), 0o600))

	rotateLog(path, 100)
	assert.FileExists(t, path, "under the limit nothing moves")

	rotateLog(path, 5)
	assert.NoFileExists(t, path)
	got, err := os.ReadFile(path + ".1")
	require.NoError(t, err)
	assert.Equal(t, "0123456789", string(got), "the full log replaces the older copy")

	rotateLog(filepath.Join(dir, "missing.log"), 5)
}

// TestLogTheme verifies that a bezel theme is found by name, empty is the
// default, and an unknown one fails with the hint to list them.
func TestLogTheme(t *testing.T) {
	t.Parallel()
	got, err := LogTheme("grape")
	require.NoError(t, err)
	assert.Equal(t, theme.Grape, got)

	got, err = LogTheme("")
	require.NoError(t, err)
	assert.Equal(t, theme.Default, got)

	_, err = LogTheme("nope")
	require.ErrorContains(t, err, `unknown theme "nope"`)
	require.ErrorContains(t, err, "--list-themes")
}

// testTheme is the theme the pretty logger draws in across these tests.
var testTheme = theme.Resolve(theme.Default, true)
