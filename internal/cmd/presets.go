package cmd

import (
	"fmt"
	"sort"
	"time"

	"github.com/lucasassuncao/movelooper/internal/models"
	"gopkg.in/yaml.v3"
)

func configurationPresetsMap() map[string]*models.Configuration {
	logFile := "~/movelooper.log"

	return map[string]*models.Configuration{
		"base": {
			Output:       "console",
			LogFile:      logFile,
			LogLevel:     "info",
			ShowCaller:   false,
			WatchDelay:   5 * time.Minute,
			HistoryLimit: 100,
		},
		"output-console": {
			Output:       "console",
			LogFile:      logFile,
			LogLevel:     "debug",
			ShowCaller:   false,
			WatchDelay:   5 * time.Minute,
			HistoryLimit: 100,
		},
		"output-file": {
			Output:       "file",
			LogFile:      logFile,
			LogLevel:     "warn",
			ShowCaller:   false,
			WatchDelay:   5 * time.Minute,
			HistoryLimit: 100,
		},
		"output-console-and-file": {
			Output:       "both",
			LogFile:      logFile,
			LogLevel:     "error",
			ShowCaller:   false,
			WatchDelay:   5 * time.Minute,
			HistoryLimit: 100,
		},
		"loglevel-trace": {
			Output:       "console",
			LogFile:      logFile,
			LogLevel:     "trace",
			ShowCaller:   true,
			WatchDelay:   5 * time.Minute,
			HistoryLimit: 100,
		},
		"debug": {
			Output:       "console",
			LogFile:      logFile,
			LogLevel:     "debug",
			ShowCaller:   true,
			WatchDelay:   5 * time.Minute,
			HistoryLimit: 100,
		},
		"loglevel-info": {
			Output:       "console",
			LogFile:      logFile,
			LogLevel:     "info",
			ShowCaller:   false,
			WatchDelay:   5 * time.Minute,
			HistoryLimit: 100,
		},
		"loglevel-warn": {
			Output:       "file",
			LogFile:      logFile,
			LogLevel:     "warn",
			ShowCaller:   false,
			WatchDelay:   5 * time.Minute,
			HistoryLimit: 100,
		},
		"loglevel-error": {
			Output:       "file",
			LogFile:      logFile,
			LogLevel:     "error",
			ShowCaller:   true,
			WatchDelay:   5 * time.Minute,
			HistoryLimit: 100,
		},
		"loglevel-fatal": {
			Output:       "file",
			LogFile:      logFile,
			LogLevel:     "fatal",
			ShowCaller:   true,
			WatchDelay:   5 * time.Minute,
			HistoryLimit: 100,
		},
	}
}

func ConfigurationPreset(name string) *models.Configuration {
	return configurationPresetsMap()[name]
}

func ListOfConfigurationPresets() []string {
	presets := configurationPresetsMap()
	keys := make([]string, 0, len(presets))
	for key := range presets {
		keys = append(keys, key)
	}
	return keys
}

func categoriesPresetsMap() map[string][]models.Category {
	downloads := "~/Downloads"
	enabled := true
	return map[string][]models.Category{
		"base": {
			{
				Name: "images",
				Source: models.CategorySource{
					Path:       downloads,
					Extensions: []string{"jpg", "jpeg", "png", "gif", "bmp", "webp"},
				},
				Destination: models.CategoryDestination{
					Path:             downloads + "/images",
					ConflictStrategy: "rename",
					OrganizeBy:       "{ext}",
				},
			},
		},
		"regex": {
			// match files whose names start with a date prefix (YYYY-MM-DD_)
			{
				Name: "dated-reports",
				Source: models.CategorySource{
					Path:       downloads,
					Extensions: []string{"pdf", "csv", "xlsx"},
					Filter: models.CategoryFilter{
						Regex:         `^\d{4}-\d{2}-\d{2}_.*`,
						CaseSensitive: false,
					},
				},
				Destination: models.CategoryDestination{
					Path:             downloads + "/reports",
					ConflictStrategy: "rename",
					OrganizeBy:       "{year}/{month}",
				},
			},
			// match invoice files regardless of case, skip anything in draft state
			{
				Name: "invoices",
				Source: models.CategorySource{
					Path:       downloads,
					Extensions: []string{"pdf", "xml"},
					Filter: models.CategoryFilter{
						Regex:         `(?i)^invoice[-_]`,
						CaseSensitive: false,
						Ignore:        []string{"*_draft.*"},
					},
				},
				Destination: models.CategoryDestination{
					Path:             downloads + "/invoices",
					ConflictStrategy: "hash_check",
					OrganizeBy:       "{year}",
				},
			},
		},
		"glob": {
			// include only files matching a specific naming convention
			{
				Name: "screenshots",
				Source: models.CategorySource{
					Path:       downloads,
					Extensions: []string{"png", "jpg"},
					Filter: models.CategoryFilter{
						Glob:    "screenshot_*",
						Ignore:  []string{"*_edited.*", "*_crop.*"},
						Include: []string{"screenshot_????-??-??_*"},
					},
				},
				Destination: models.CategoryDestination{
					Path:             downloads + "/screenshots",
					ConflictStrategy: "rename",
					OrganizeBy:       "{year}/{month}",
				},
			},
			// exclude temp and backup files, only keep final versions
			{
				Name: "documents",
				Source: models.CategorySource{
					Path:       downloads,
					Extensions: []string{"doc", "docx", "odt"},
					Filter: models.CategoryFilter{
						Include: []string{"*_final.*", "*_v[0-9]*"},
						Ignore:  []string{"*_temp.*", "*_backup.*", "~*"},
					},
				},
				Destination: models.CategoryDestination{
					Path:             downloads + "/documents",
					ConflictStrategy: "rename",
					OrganizeBy:       "{ext}",
				},
			},
		},
		"conflict": {
			// rename: appends a counter when destination file already exists
			{
				Name: "photos-rename",
				Source: models.CategorySource{
					Path:       downloads,
					Extensions: []string{"jpg", "jpeg", "heic"},
				},
				Destination: models.CategoryDestination{
					Path:             downloads + "/photos",
					ConflictStrategy: "rename",
					OrganizeBy:       "{year}/{month}",
				},
			},
			// overwrite: replaces the destination file unconditionally
			{
				Name: "config-sync",
				Source: models.CategorySource{
					Path:       downloads,
					Extensions: []string{"yaml", "json", "toml"},
				},
				Destination: models.CategoryDestination{
					Path:             downloads + "/config",
					ConflictStrategy: "overwrite",
					OrganizeBy:       "{ext}",
				},
			},
			// skip: leaves the destination untouched when a conflict occurs
			{
				Name: "music-skip",
				Source: models.CategorySource{
					Path:       downloads,
					Extensions: []string{"mp3", "flac", "wav"},
				},
				Destination: models.CategoryDestination{
					Path:             downloads + "/music",
					ConflictStrategy: "skip",
					OrganizeBy:       "{ext}",
				},
			},
			// hash_check: skips only if source and destination are identical
			{
				Name: "archives-dedup",
				Source: models.CategorySource{
					Path:       downloads,
					Extensions: []string{"zip", "tar", "gz", "rar", "7z"},
				},
				Destination: models.CategoryDestination{
					Path:             downloads + "/archives",
					ConflictStrategy: "hash_check",
					OrganizeBy:       "{ext}",
				},
			},
		},
		"filters": {
			// size bounds: only move files within a specific size range
			{
				Name: "large-videos",
				Source: models.CategorySource{
					Path:       downloads,
					Extensions: []string{"mp4", "mkv", "avi"},
					Filter: models.CategoryFilter{
						MinSize: "500MB",
						MaxSize: "50GB",
					},
				},
				Destination: models.CategoryDestination{
					Path:             downloads + "/videos/large",
					ConflictStrategy: "hash_check",
					OrganizeBy:       "{ext}",
				},
			},
			// age bounds: archive files older than 30 days, no older than 1 year
			{
				Name: "old-downloads",
				Source: models.CategorySource{
					Path:       downloads,
					Extensions: []string{"pdf", "zip", "exe", "dmg"},
					Filter: models.CategoryFilter{
						MinAge: 30 * 24 * time.Hour,
						MaxAge: 365 * 24 * time.Hour,
					},
				},
				Destination: models.CategoryDestination{
					Path:             downloads + "/old",
					ConflictStrategy: "skip",
					OrganizeBy:       "{year}",
				},
			},
			// any: match files that satisfy at least one sub-filter
			{
				Name: "reports-any",
				Source: models.CategorySource{
					Path:       downloads,
					Extensions: []string{"pdf", "xlsx", "csv"},
					Filter: models.CategoryFilter{
						Any: []models.CategoryFilter{
							{Regex: `^report_`},
							{Glob: "summary_*"},
							{MinSize: "1MB"},
						},
					},
				},
				Destination: models.CategoryDestination{
					Path:             downloads + "/reports",
					ConflictStrategy: "rename",
					OrganizeBy:       "{ext}",
				},
			},
			// all: match files that satisfy every sub-filter simultaneously
			{
				Name: "large-recent-docs",
				Source: models.CategorySource{
					Path:       downloads,
					Extensions: []string{"pdf", "docx"},
					Filter: models.CategoryFilter{
						All: []models.CategoryFilter{
							{MinSize: "100KB"},
							{MaxAge: 7 * 24 * time.Hour},
							{Ignore: []string{"*_draft.*"}},
						},
					},
				},
				Destination: models.CategoryDestination{
					Path:             downloads + "/recent-docs",
					ConflictStrategy: "rename",
					OrganizeBy:       "{year}/{month}",
				},
			},
		},
		"hooks": {
			// before hook: validate or prepare before moving; after hook: notify or post-process
			{
				Name: "processed-videos",
				Source: models.CategorySource{
					Path:       downloads,
					Extensions: []string{"mp4", "mkv"},
					Filter: models.CategoryFilter{
						MinSize: "100MB",
					},
				},
				Destination: models.CategoryDestination{
					Path:             downloads + "/videos",
					ConflictStrategy: "hash_check",
					OrganizeBy:       "{year}/{month}",
				},
				Hooks: &models.CategoryHooks{
					Before: &models.CategoryHook{
						Shell:     "bash",
						OnFailure: "skip", // skip this file if the hook fails
						Run: []string{
							"mkdir -p ~/Downloads/videos",
							"echo 'moving {file}'",
						},
					},
					After: &models.CategoryHook{
						Shell:     "bash",
						OnFailure: "log", // log the error but continue
						Run: []string{
							"echo '{file} moved to {dest}'",
						},
					},
				},
			},
		},
		"full": {
			{
				Name:    "documents",
				Enabled: &enabled,
				Source: models.CategorySource{
					Path:         downloads,
					Extensions:   []string{"pdf", "doc", "docx", "txt"},
					Recursive:    true,
					MaxDepth:     3,
					ExcludePaths: []string{downloads + "/archives", downloads + "/temp"},
					Filter: models.CategoryFilter{
						Regex:         `^\d{4}-\d{2}-\d{2}_.*`,
						Glob:          "report_*",
						Include:       []string{"*_final.*"},
						Ignore:        []string{"*_draft.*", "*_temp.*"},
						CaseSensitive: true,
						MinAge:        7 * 24 * time.Hour,
						MaxAge:        365 * 24 * time.Hour,
						MinSize:       "10KB",
						MaxSize:       "500MB",
						Any: []models.CategoryFilter{
							{Regex: `^invoice_.*`},
							{Glob: "contract_*"},
						},
						All: []models.CategoryFilter{
							{MinSize: "1KB"},
							{MaxAge: 180 * 24 * time.Hour},
						},
					},
				},
				Destination: models.CategoryDestination{
					Path:             downloads + "/documents",
					OrganizeBy:       "{year}/{month}",
					ConflictStrategy: "hash_check",
					Action:           "move",
					Rename:           "{year}-{month}-{day}_{name}",
				},
				Hooks: &models.CategoryHooks{
					Before: &models.CategoryHook{
						Shell:     "bash",
						OnFailure: "skip",
						Run:       []string{"echo 'starting move'", "mkdir -p ~/Downloads/documents"},
					},
					After: &models.CategoryHook{
						Shell:     "bash",
						OnFailure: "log",
						Run:       []string{"echo 'move complete: {file}'"},
					},
				},
			},
		},
	}
}

func CategoriesPreset(name string) []models.Category {
	return categoriesPresetsMap()[name]
}

func ListOfCategoriesPresets() []string {
	presets := categoriesPresetsMap()
	keys := make([]string, 0, len(presets))
	for key := range presets {
		keys = append(keys, key)
	}
	return keys
}

// configPresetSource implements presets.Source backed by the Go-struct presets.
type configPresetSource struct{}

func (configPresetSource) ListFields() []string {
	return []string{"categories", "configuration"}
}

func (configPresetSource) ListPresets(field string) []string {
	var keys []string
	if field == "configuration" {
		keys = ListOfConfigurationPresets()
	} else {
		keys = ListOfCategoriesPresets()
	}
	sort.Strings(keys)
	return keys
}

func (configPresetSource) PresetYAML(field, name string) (string, error) {
	var val any
	switch field {
	case "configuration":
		cfg, ok := configurationPresetsMap()[name]
		if !ok {
			return "", fmt.Errorf("configuration preset %q not found", name)
		}
		val = cfg
	default:
		cats, ok := categoriesPresetsMap()[name]
		if !ok {
			return "", fmt.Errorf("categories preset %q not found", name)
		}
		val = cats
	}

	out, err := yaml.Marshal(map[string]any{field: val})
	if err != nil {
		return "", err
	}
	return string(out), nil
}
