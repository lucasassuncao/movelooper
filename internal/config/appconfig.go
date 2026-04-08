package config

import (
	"time"

	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/spf13/viper"
)

const defaultHistoryLimit = 50
const defaultWatchDelay = 5 * time.Minute

// LoadConfig reads the application-level settings from v and returns a
// fully populated Configuration. It must be called after InitConfig has
// successfully loaded the file.
func LoadConfig(v *viper.Viper) models.Configuration {
	cfg := models.Configuration{
		Output:       v.GetString("configuration.output"),
		LogFile:      v.GetString("configuration.log-file"),
		LogLevel:     v.GetString("configuration.log-level"),
		ShowCaller:   v.GetBool("configuration.show-caller"),
		WatchDelay:   v.GetDuration("configuration.watch-delay"),
		HistoryLimit: v.GetInt("configuration.history-limit"),
	}

	if cfg.WatchDelay == 0 {
		cfg.WatchDelay = defaultWatchDelay
	}
	if cfg.HistoryLimit == 0 {
		cfg.HistoryLimit = defaultHistoryLimit
	}

	return cfg
}
