package config

import (
	"time"

	"github.com/knadh/koanf/v2"
	"github.com/lucasassuncao/movelooper/internal/models"
)

const defaultHistoryLimit = 50
const defaultWatchDelay = 5 * time.Minute

// LoadConfig reads the application-level settings from k and returns a
// fully populated Configuration. It must be called after InitConfig has
// successfully loaded the file.
func LoadConfig(k *koanf.Koanf) models.Configuration {
	cfg := models.Configuration{
		Output:       k.String("configuration.output"),
		LogFile:      k.String("configuration.log-file"),
		LogLevel:     k.String("configuration.log-level"),
		ShowCaller:   k.Bool("configuration.show-caller"),
		WatchDelay:   k.Duration("configuration.watch-delay"),
		HistoryLimit: k.Int("configuration.history-limit"),
	}

	if cfg.WatchDelay == 0 {
		cfg.WatchDelay = defaultWatchDelay
	}
	if cfg.HistoryLimit == 0 {
		cfg.HistoryLimit = defaultHistoryLimit
	}

	return cfg
}
