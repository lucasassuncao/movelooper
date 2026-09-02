package config

import (
	"time"

	"github.com/knadh/koanf/v2"
	"github.com/lucasassuncao/movelooper/internal/models"
)

const defaultHistoryLimit = 100
const defaultWatchDelay = 5 * time.Minute
const defaultPollInterval = 5 * time.Second

// LoadConfig reads the application-level settings from k and returns a
// fully populated Configuration. It must be called after InitConfig has
// successfully loaded the file.
func LoadConfig(k *koanf.Koanf) models.Configuration {
	var cfg models.Configuration
	// The mapstructure tags on models.Configuration carry the key names, and
	// koanf's decoder turns a duration string ("5m") into a time.Duration. The
	// error is deliberately dropped: decoding is per field, so a malformed entry
	// leaves that one field zeroed without touching the rest, and the fallbacks
	// below already treat a zero as "not set".
	_ = k.UnmarshalWithConf("configuration", &cfg, koanf.UnmarshalConf{Tag: "mapstructure"})

	// The unmarshal cannot tell an absent history.enabled from an explicit false,
	// and an absent one means true.
	cfg.History.Enabled = historyEnabled(k)

	// Absent and negative are both replaced by the default. Negative matters on
	// its own: validate rejects it, but watch does not run validate, and a
	// hand-edited poll-interval reached time.NewTicker and panicked the process.
	if cfg.Watch.Delay <= 0 {
		cfg.Watch.Delay = defaultWatchDelay
	}
	if cfg.Watch.PollInterval <= 0 {
		cfg.Watch.PollInterval = defaultPollInterval
	}
	if cfg.History.Limit <= 0 {
		cfg.History.Limit = defaultHistoryLimit
	}

	return cfg
}

// historyEnabled reports whether undo history tracking is on. It defaults to
// true and is only disabled when the key is explicitly set to false.
func historyEnabled(k *koanf.Koanf) bool {
	if !k.Exists("configuration.history.enabled") {
		return true
	}
	return k.Bool("configuration.history.enabled")
}
