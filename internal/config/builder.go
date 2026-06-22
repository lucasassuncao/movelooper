package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/knadh/koanf/v2"
	"github.com/lucasassuncao/movelooper/internal/history"
	"github.com/lucasassuncao/movelooper/internal/models"
)

// Option configures which initialization steps NewApp will run.
type Option func(*options)

type options struct {
	configureLogger bool
	loadConfig      bool
	loadCategories  bool
	initHistory     bool
	validateDirs    bool
}

func WithLogger() Option {
	return func(o *options) { o.configureLogger = true }
}

func WithConfig() Option {
	return func(o *options) { o.loadConfig = true }
}

func WithCategories() Option {
	return func(o *options) { o.loadCategories = true }
}

func WithHistory() Option {
	return func(o *options) { o.initHistory = true }
}

func WithValidateDirs() Option {
	return func(o *options) { o.validateDirs = true }
}

// NewApp resolves the config file and runs the requested initialization steps in order.
func NewApp(m *models.Movelooper, configPath string, opts ...Option) (retErr error) {
	o := options{}
	for _, opt := range opts {
		opt(&o)
	}

	defer func() {
		if retErr != nil && m.LogCloser != nil {
			m.LogCloser.Close()
			m.LogCloser = nil
		}
	}()

	k := koanf.New(".")

	resolved, err := ResolveConfigPath(configPath)
	if err != nil {
		return wrapConfigNotFound(configPath, err)
	}
	if err := InitConfig(k, resolved); err != nil {
		return wrapConfigNotFound(configPath, err)
	}

	if o.configureLogger {
		logger, closer, err := ConfigureLogger(k)
		if err != nil {
			return fmt.Errorf("failed to configure logger: %w", err)
		}
		m.Logger = logger
		m.LogCloser = closer
	}

	if o.loadConfig {
		m.Config = LoadConfig(k)
	}

	if o.loadCategories {
		cats, err := UnmarshalConfig(k)
		if err != nil {
			return err
		}
		if err := applyCategoryDefaults(cats, m.Config.Defaults); err != nil {
			return err
		}
		m.Categories = cats
	}

	// History.Enabled is populated by LoadConfig (default true); preRunHandler
	// always loads the config before initializing history.
	if o.initHistory && m.Config.History.Enabled {
		histPath := m.Config.History.File
		if histPath == "" {
			histPath = defaultHistoryFilePath()
		}
		if hist, err := history.NewHistory(histPath, m.Config.History.Limit); err != nil {
			m.Logger.Warn("failed to initialize history tracking", m.Logger.Args("error", err.Error()))
		} else {
			m.History = hist
		}
	}

	if o.validateDirs {
		validateSourceDirs(m)
	}

	return nil
}

func wrapConfigNotFound(configPath string, err error) error {
	if !errors.Is(err, ErrConfigNotFound) {
		return err
	}
	if configPath != "" {
		return fmt.Errorf("configuration file not found at %q: %w", configPath, err)
	}
	return fmt.Errorf("configuration file not found\n\nPlease run 'movelooper init' to create a configuration file: %w", err)
}

func defaultHistoryFilePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "movelooper", "history", "movelooper.json")
	}
	return filepath.Join(homeDir, ".movelooper", "history", "movelooper.json")
}

func validateSourceDirs(m *models.Movelooper) {
	for _, cat := range m.Categories {
		if !cat.IsEnabled() {
			continue
		}
		if _, err := os.Stat(cat.Source.Path); os.IsNotExist(err) {
			m.Logger.Warn("source directory does not exist",
				m.Logger.Args("category", cat.Name, "path", cat.Source.Path))
		}
		if cat.Source.Recursive {
			if cat.Source.MaxDepth < 0 {
				m.Logger.Error("max-depth must be >= 0 (0 = unlimited)",
					m.Logger.Args("category", cat.Name, "max-depth", cat.Source.MaxDepth))
			}
			for _, p := range cat.Source.ExcludePaths {
				info, err := os.Stat(p)
				if err != nil || !info.IsDir() {
					m.Logger.Warn("exclude-path does not exist or is not a directory",
						m.Logger.Args("category", cat.Name, "path", p))
				}
			}
		}
	}
}
