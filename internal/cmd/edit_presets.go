package cmd

import (
	"time"

	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/yedit/presets"
)

var MovelooperPresets = presets.Combine(
	presets.ForField("configuration", configurationPresetsMap()),
	presets.ForField("categories", categoriesPresetsMap()),
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
	field := "configuration"
	return presets.ForField(field, configurationPresetsMap()).ListPresets(field)
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
					ConflictStrategy: models.ConflictStrategyRename,
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
						Match: &models.MatchFilter{
							Regex: `^\d{4}-\d{2}-\d{2}_.*`,
						},
					},
				},
				Destination: models.CategoryDestination{
					Path:             downloads + "/reports",
					ConflictStrategy: models.ConflictStrategyRename,
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
						Match: &models.MatchFilter{
							Regex: `(?i)^invoice[-_]`,
						},
						Not: []models.CategoryFilter{
							{Match: &models.MatchFilter{Glob: "*_draft.*"}},
						},
					},
				},
				Destination: models.CategoryDestination{
					Path:             downloads + "/invoices",
					ConflictStrategy: models.ConflictStrategyHashCheck,
					OrganizeBy:       "{year}",
				},
			},
		},
		"glob": {
			// match files following a specific naming convention, excluding edited/cropped versions
			{
				Name: "screenshots",
				Source: models.CategorySource{
					Path:       downloads,
					Extensions: []string{"png", "jpg"},
					Filter: models.CategoryFilter{
						Match: &models.MatchFilter{Glob: "screenshot_????-??-??_*"},
						Not: []models.CategoryFilter{
							{Match: &models.MatchFilter{Glob: "*_edited.*"}},
							{Match: &models.MatchFilter{Glob: "*_crop.*"}},
						},
					},
				},
				Destination: models.CategoryDestination{
					Path:             downloads + "/screenshots",
					ConflictStrategy: models.ConflictStrategyRename,
					OrganizeBy:       "{year}/{month}",
				},
			},
			// exclude temp and backup files
			{
				Name: "documents",
				Source: models.CategorySource{
					Path:       downloads,
					Extensions: []string{"doc", "docx", "odt"},
					Filter: models.CategoryFilter{
						Not: []models.CategoryFilter{
							{Match: &models.MatchFilter{Glob: "*_temp.*"}},
							{Match: &models.MatchFilter{Glob: "*_backup.*"}},
							{Match: &models.MatchFilter{Glob: "~*"}},
						},
					},
				},
				Destination: models.CategoryDestination{
					Path:             downloads + "/documents",
					ConflictStrategy: models.ConflictStrategyRename,
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
					ConflictStrategy: models.ConflictStrategyRename,
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
					ConflictStrategy: models.ConflictStrategyOverwrite,
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
					ConflictStrategy: models.ConflictStrategySkip,
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
					ConflictStrategy: models.ConflictStrategyHashCheck,
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
						Size: &models.SizeFilter{Min: "500MB", Max: "50GB"},
					},
				},
				Destination: models.CategoryDestination{
					Path:             downloads + "/videos/large",
					ConflictStrategy: models.ConflictStrategyHashCheck,
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
						Age: &models.AgeFilter{
							Min: 30 * 24 * time.Hour,
							Max: 365 * 24 * time.Hour,
						},
					},
				},
				Destination: models.CategoryDestination{
					Path:             downloads + "/old",
					ConflictStrategy: models.ConflictStrategySkip,
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
							{Match: &models.MatchFilter{Regex: `^report_`}},
							{Match: &models.MatchFilter{Glob: "summary_*"}},
							{Size: &models.SizeFilter{Min: "1MB"}},
						},
					},
				},
				Destination: models.CategoryDestination{
					Path:             downloads + "/reports",
					ConflictStrategy: models.ConflictStrategyRename,
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
							{Size: &models.SizeFilter{Min: "100KB"}},
							{Age: &models.AgeFilter{Max: 7 * 24 * time.Hour}},
							{Not: []models.CategoryFilter{
								{Match: &models.MatchFilter{Glob: "*_draft.*"}},
							}},
						},
					},
				},
				Destination: models.CategoryDestination{
					Path:             downloads + "/recent-docs",
					ConflictStrategy: models.ConflictStrategyRename,
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
						Size: &models.SizeFilter{Min: "100MB"},
					},
				},
				Destination: models.CategoryDestination{
					Path:             downloads + "/videos",
					ConflictStrategy: models.ConflictStrategyHashCheck,
					OrganizeBy:       "{year}/{month}",
				},
				Hooks: &models.CategoryHooks{
					Before: &models.CategoryHook{
						Shell:     "bash",
						OnFailure: "abort",
						Run: []string{
							"mkdir -p ~/Downloads/videos",
							"echo 'moving {file}'",
						},
					},
					After: &models.CategoryHook{
						Shell:     "bash",
						OnFailure: "warn",
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
						Match: &models.MatchFilter{
							Regex:         `^\d{4}-\d{2}-\d{2}_.*`,
							CaseSensitive: true,
						},
						Age:  &models.AgeFilter{Min: 7 * 24 * time.Hour, Max: 365 * 24 * time.Hour},
						Size: &models.SizeFilter{Min: "10KB", Max: "500MB"},
						Not: []models.CategoryFilter{
							{Match: &models.MatchFilter{Glob: "*_draft.*"}},
							{Match: &models.MatchFilter{Glob: "*_temp.*"}},
						},
						Any: []models.CategoryFilter{
							{Match: &models.MatchFilter{Regex: `^invoice_.*`}},
							{Match: &models.MatchFilter{Glob: "contract_*"}},
						},
					},
				},
				Destination: models.CategoryDestination{
					Path:             downloads + "/documents",
					OrganizeBy:       "{year}/{month}",
					ConflictStrategy: models.ConflictStrategyHashCheck,
					Action:           "move",
					Rename:           "{year}-{month}-{day}_{name}",
				},
				Hooks: &models.CategoryHooks{
					Before: &models.CategoryHook{
						Shell:     "bash",
						OnFailure: "abort",
						Run:       []string{"echo 'starting move'", "mkdir -p ~/Downloads/documents"},
					},
					After: &models.CategoryHook{
						Shell:     "bash",
						OnFailure: "warn",
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
	field := "categories"
	return presets.ForField(field, categoriesPresetsMap()).ListPresets(field)
}
