package config

import (
	"testing"
	"time"

	"github.com/knadh/koanf/v2"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testLoadConfig defines the structure for test cases of the LoadConfig function,
// containing YAML content and a check function for assertions on the resulting Configuration.
type testLoadConfig struct {
	name  string
	yaml  string
	check func(t *testing.T, cfg models.Configuration)
}

// testLoadConfigTestCases defines a set of test cases for the LoadConfig function,
// covering default values, custom values, and watch.delay fallback.
var testLoadConfigTestCases = []testLoadConfig{
	{
		name: "defaults when not set",
		yaml: "",
		check: func(t *testing.T, cfg models.Configuration) {
			assert.Equal(t, defaultWatchDelay, cfg.Watch.Delay)
			assert.Equal(t, defaultPollInterval, cfg.Watch.PollInterval)
			assert.Equal(t, defaultHistoryLimit, cfg.History.Limit)
			assert.True(t, cfg.History.Enabled, "history enabled by default")
			assert.Nil(t, cfg.Defaults, "no defaults block when absent")
		},
	},
	{
		name: "history disabled and custom poll-interval",
		yaml: `
configuration:
  watch:
    poll-interval: 2s
  history:
    enabled: false
`,
		check: func(t *testing.T, cfg models.Configuration) {
			assert.Equal(t, 2*time.Second, cfg.Watch.PollInterval)
			assert.False(t, cfg.History.Enabled)
		},
	},
	{
		name: "defaults block is read",
		yaml: `
configuration:
  defaults:
    conflict-strategy: skip
    action: copy
    organize-by: "{ext}"
`,
		check: func(t *testing.T, cfg models.Configuration) {
			require.NotNil(t, cfg.Defaults)
			assert.Equal(t, models.ConflictStrategySkip, cfg.Defaults.ConflictStrategy)
			assert.Equal(t, models.ActionCopy, cfg.Defaults.Action)
			assert.Equal(t, "{ext}", cfg.Defaults.OrganizeBy)
		},
	},
	{
		name: "custom values",
		yaml: `
configuration:
  logging:
    output: json
    level: debug
    file: /var/log/movelooper.log
    show-caller: true
  watch:
    delay: 2m
  history:
    limit: 100
    file: /var/lib/movelooper/history.json
`,
		check: func(t *testing.T, cfg models.Configuration) {
			assert.Equal(t, "json", cfg.Logging.Output)
			assert.Equal(t, "debug", cfg.Logging.Level)
			assert.Equal(t, "/var/log/movelooper.log", cfg.Logging.File)
			assert.True(t, cfg.Logging.ShowCaller)
			assert.Equal(t, 2*time.Minute, cfg.Watch.Delay)
			assert.Equal(t, 100, cfg.History.Limit)
			assert.Equal(t, "/var/lib/movelooper/history.json", cfg.History.File)
		},
	},
	{
		name: "watch.delay fallback to default",
		yaml: "configuration:\n  logging:\n    output: text\n",
		check: func(t *testing.T, cfg models.Configuration) {
			assert.Equal(t, defaultWatchDelay, cfg.Watch.Delay)
		},
	},
	{
		// Regression: a negative duration is not zero, so it used to survive
		// LoadConfig and reach time.NewTicker, which panics on anything that is
		// not positive. validate rejects these, but watch does not run validate.
		name: "negative durations and limits fall back to the defaults",
		yaml: `
configuration:
  watch:
    delay: -1m
    poll-interval: -5s
  history:
    limit: -3
`,
		check: func(t *testing.T, cfg models.Configuration) {
			assert.Equal(t, defaultWatchDelay, cfg.Watch.Delay)
			assert.Equal(t, defaultPollInterval, cfg.Watch.PollInterval)
			assert.Equal(t, defaultHistoryLimit, cfg.History.Limit)
		},
	},
	{
		// LoadConfig drops the unmarshal error on purpose. Decoding is per field,
		// so a value koanf cannot read must cost only its own field: the ones
		// around it still load and the bad one falls back like an absent value.
		name: "malformed duration costs only its own field",
		yaml: `
configuration:
  logging:
    output: console
    level: debug
  watch:
    delay: abacaxi
  history:
    limit: 7
`,
		check: func(t *testing.T, cfg models.Configuration) {
			assert.Equal(t, "console", cfg.Logging.Output)
			assert.Equal(t, "debug", cfg.Logging.Level)
			assert.Equal(t, 7, cfg.History.Limit)
			assert.Equal(t, defaultWatchDelay, cfg.Watch.Delay)
		},
	},
}

// TestLoadConfig tests the LoadConfig function to ensure it correctly applies
// defaults and parses custom configuration values.
func TestLoadConfig(t *testing.T) {
	t.Parallel()
	for _, tt := range testLoadConfigTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := writeYAML(t, dir, "cfg.yaml", tt.yaml)
			k := koanf.New(".")
			if tt.yaml != "" {
				require.NoError(t, InitConfig(k, path))
			}
			tt.check(t, LoadConfig(k))
		})
	}
}
