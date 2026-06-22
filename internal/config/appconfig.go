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
	cfg := models.Configuration{
		Logging: models.Logging{
			Output:     k.String("configuration.logging.output"),
			File:       k.String("configuration.logging.file"),
			Level:      k.String("configuration.logging.level"),
			ShowCaller: k.Bool("configuration.logging.show-caller"),
			Format:     k.String("configuration.logging.format"),
			Color:      k.String("configuration.logging.color"),
			MaxWidth:   k.Int("configuration.logging.max-width"),
		},
		Watch: models.Watch{
			Delay:        k.Duration("configuration.watch.delay"),
			PollInterval: k.Duration("configuration.watch.poll-interval"),
		},
		History: models.History{
			Limit:   k.Int("configuration.history.limit"),
			File:    k.String("configuration.history.file"),
			Enabled: historyEnabled(k),
		},
		Defaults: loadDefaults(k),
	}

	if cfg.Watch.Delay == 0 {
		cfg.Watch.Delay = defaultWatchDelay
	}
	if cfg.Watch.PollInterval == 0 {
		cfg.Watch.PollInterval = defaultPollInterval
	}
	if cfg.History.Limit == 0 {
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

// loadDefaults reads the optional defaults sub-section, returning nil when it
// is absent so categories fall back to their own built-in defaults.
func loadDefaults(k *koanf.Koanf) *models.Defaults {
	if !k.Exists("configuration.defaults") {
		return nil
	}
	return &models.Defaults{
		ConflictStrategy: models.ConflictStrategy(k.String("configuration.defaults.conflict-strategy")),
		Action:           models.Action(k.String("configuration.defaults.action")),
		OrganizeBy:       k.String("configuration.defaults.organize-by"),
	}
}
