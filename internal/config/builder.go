package config

import (
	"fmt"
	"os"

	"github.com/knadh/koanf/v2"
	"github.com/lucasassuncao/movelooper/internal/history"
	"github.com/lucasassuncao/movelooper/internal/models"
)

// AppBuilder constructs a Movelooper instance step-by-step.
// Each method is a no-op when a previous step has already set an error.
type AppBuilder struct {
	m          *models.Movelooper
	k          *koanf.Koanf
	configPath string
	err        error
}

// NewAppBuilder creates a builder for m using the given config file path.
func NewAppBuilder(m *models.Movelooper, configPath string) *AppBuilder {
	return &AppBuilder{m: m, k: koanf.New("."), configPath: configPath}
}

// ResolveConfig resolves the config file path and loads the YAML into the builder's koanf instance.
func (b *AppBuilder) ResolveConfig() *AppBuilder {
	if b.err != nil {
		return b
	}
	resolved, err := ResolveConfigPath(b.configPath)
	if err != nil {
		b.err = err
		return b
	}
	if err := InitConfig(b.k, resolved); err != nil {
		b.err = err
		return b
	}
	return b
}

// ConfigureLogger reads logging settings from the loaded config and sets m.Logger and m.LogCloser.
func (b *AppBuilder) ConfigureLogger() *AppBuilder {
	if b.err != nil {
		return b
	}
	logger, closer, err := ConfigureLogger(b.k)
	if err != nil {
		b.err = fmt.Errorf("failed to configure logger: %w", err)
		return b
	}
	b.m.Logger = logger
	b.m.LogCloser = closer
	return b
}

// LoadConfig populates m.Config from the loaded koanf instance.
func (b *AppBuilder) LoadConfig() *AppBuilder {
	if b.err != nil {
		return b
	}
	b.m.Config = LoadConfig(b.k)
	return b
}

// LoadCategories unmarshals, validates, and pre-compiles the categories from config.
func (b *AppBuilder) LoadCategories() *AppBuilder {
	if b.err != nil {
		return b
	}
	cats, err := UnmarshalConfig(b.k)
	if err != nil {
		b.err = err
		return b
	}
	b.m.Categories = cats
	return b
}

// InitHistory initialises the move history store using m.Config.HistoryLimit.
// A failure here is non-fatal: it logs a warning and leaves m.History nil.
func (b *AppBuilder) InitHistory() *AppBuilder {
	if b.err != nil {
		return b
	}
	hist, err := history.NewHistory(b.m.Config.HistoryLimit)
	if err != nil {
		b.m.Logger.Warn("failed to initialize history tracking", b.m.Logger.Args("error", err.Error()))
		return b
	}
	b.m.History = hist
	return b
}

// ValidateDirectories warns about source or destination directories that do not exist.
// It does not abort startup — missing directories are reported and skipped at runtime.
func (b *AppBuilder) ValidateDirectories() *AppBuilder {
	if b.err != nil {
		return b
	}
	validateDirectoriesFromBuilder(b.m)
	return b
}

// Build returns the first error encountered during the chain, or nil on success.
func (b *AppBuilder) Build() error {
	return b.err
}

// validateDirectoriesFromBuilder warns about source or destination directories that do not exist.
func validateDirectoriesFromBuilder(m *models.Movelooper) {
	for _, cat := range m.Categories {
		if !cat.IsEnabled() {
			continue
		}
		if _, err := os.Stat(cat.Source.Path); os.IsNotExist(err) {
			m.Logger.Warn("source directory does not exist",
				m.Logger.Args("category", cat.Name, "path", cat.Source.Path))
		}
		if _, err := os.Stat(cat.Destination.Path); os.IsNotExist(err) {
			m.Logger.Warn("destination directory does not exist",
				m.Logger.Args("category", cat.Name, "path", cat.Destination.Path))
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
