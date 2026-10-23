package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/knadh/koanf/v2"
	"github.com/lucasassuncao/movelooper/internal/models"
)

const defaultHistoryLimit = 100
const defaultWatchDelay = 5 * time.Minute
const defaultPollInterval = 5 * time.Second

// durationKeys must carry a unit: the decoder reads a bare 10 as 10ns.
var durationKeys = []string{"configuration.watch.delay", "configuration.watch.poll-interval"}

// LoadConfig reads the application-level settings from k, after InitConfig. The
// error names the values it could not read: the Configuration holds their
// defaults instead, so the error is a warning, never a reason to stop.
func LoadConfig(k *koanf.Koanf) (models.Configuration, error) {
	var problems []error
	for _, key := range durationKeys {
		if v := k.Get(key); v != nil {
			if _, ok := v.(string); !ok {
				problems = append(problems, fmt.Errorf("%s: %v needs a unit, such as 10s or 5m", strings.TrimPrefix(key, "configuration."), v))
				k.Delete(key)
			}
		}
	}

	var cfg models.Configuration
	// The mapstructure tags on models.Configuration carry the key names, and
	// koanf's decoder turns a duration string ("5m") into a time.Duration.
	// Decoding is per field: a malformed entry leaves only that field zeroed.
	if err := k.UnmarshalWithConf("configuration", &cfg, koanf.UnmarshalConf{Tag: "mapstructure"}); err != nil {
		problems = append(problems, err)
	}

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

	return cfg, errors.Join(problems...)
}

// historyEnabled reports whether undo history tracking is on. It defaults to
// true and is only disabled when the key is explicitly set to false.
func historyEnabled(k *koanf.Koanf) bool {
	if !k.Exists("configuration.history.enabled") {
		return true
	}
	return k.Bool("configuration.history.enabled")
}
