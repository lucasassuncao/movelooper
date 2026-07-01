package cmd

import (
	"fmt"
	"sort"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/yedit/presets"
)

var MovelooperBlockPresets = presets.Combine(
	presets.ForField("configuration", configurationPresetsMap()),
	presets.ForField("categories", categoriesPresetsMap()),
)

// MovelooperDocPresets is a whole-document preset source for the root template
// picker (ctrl+p). Each entry combines the base configuration with one of the
// available category presets.
var MovelooperDocPresets presets.Source = buildDocPresets()

// docPresetSource implements presets.Source for whole-document templates.
// PresetYAML("", name) returns the full YAML for the named template.
type docPresetSource struct {
	names []string
	yamls map[string]string
}

func (s *docPresetSource) ListFields() []string { return []string{""} }
func (s *docPresetSource) ListPresets(field string) []string {
	if field != "" {
		return nil
	}
	return s.names
}
func (s *docPresetSource) PresetYAML(field, name string) (string, error) {
	if field != "" {
		return "", fmt.Errorf("docPresetSource: unknown field %q", field)
	}
	y, ok := s.yamls[name]
	if !ok {
		return "", fmt.Errorf("docPresetSource: unknown preset %q", name)
	}
	return y, nil
}

func buildDocPresets() *docPresetSource {
	docs := docPresetsMap()

	names := make([]string, 0, len(docs))
	for name := range docs {
		names = append(names, name)
	}
	sort.Strings(names)

	yamls := make(map[string]string, len(docs))
	for _, name := range names {
		raw, err := yaml.Marshal(docs[name])
		if err != nil {
			continue
		}
		yamls[name] = string(raw)
	}
	return &docPresetSource{names: names, yamls: yamls}
}

func docPresetsMap() map[string]*models.Config {
	downloads := "~/Downloads"
	enabled := true
	logFile := "~/.movelooper/logs/movelooper.log"
	histFile := "~/.movelooper/history/movelooper.json"

	consoleCfg := models.Configuration{
		Logging: models.Logging{
			Output: "console",
			Level:  "info",
			File:   logFile,
			Format: "pretty",
			Color:  "auto",
		},
		Watch:   models.Watch{Delay: 5 * time.Minute},
		History: models.History{Limit: 100, File: histFile},
	}

	fileCfg := models.Configuration{
		Logging: models.Logging{
			Output: "file",
			Level:  "info",
			File:   logFile,
			Format: "pretty",
		},
		Watch:   models.Watch{Delay: 5 * time.Minute},
		History: models.History{Limit: 100, File: histFile},
	}

	return map[string]*models.Config{
		// filter.any: files that qualify if they satisfy at least one sub-filter (OR)
		"full-filter-any": {
			Configuration: consoleCfg,
			Categories: []models.Category{
				{
					Name:    "reports",
					Enabled: &enabled,
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
						OrganizeBy:       "{year}/{month}",
					},
				},
				{
					Name:    "media",
					Enabled: &enabled,
					Source: models.CategorySource{
						Path:       downloads,
						Extensions: []string{"mp4", "mkv", "mp3", "flac"},
						Filter: models.CategoryFilter{
							Any: []models.CategoryFilter{
								{Size: &models.SizeFilter{Min: "100MB"}},
								{Match: &models.MatchFilter{Regex: `^\d{4}-\d{2}-\d{2}`}},
							},
						},
					},
					Destination: models.CategoryDestination{
						Path:             downloads + "/media",
						ConflictStrategy: models.ConflictStrategyHashCheck,
						OrganizeBy:       "{year}",
					},
				},
			},
		},
		// filter.all: files that qualify only when every sub-filter passes (AND)
		"full-filter-all": {
			Configuration: consoleCfg,
			Categories: []models.Category{
				{
					Name:    "invoices",
					Enabled: &enabled,
					Source: models.CategorySource{
						Path:       downloads,
						Extensions: []string{"pdf", "xml"},
						Filter: models.CategoryFilter{
							All: []models.CategoryFilter{
								{Size: &models.SizeFilter{Min: "50KB"}},
								{Age: &models.AgeFilter{Max: 90 * 24 * time.Hour}},
								{Match: &models.MatchFilter{Regex: `^invoice_`}},
							},
						},
					},
					Destination: models.CategoryDestination{
						Path:             downloads + "/invoices",
						ConflictStrategy: models.ConflictStrategyHashCheck,
						OrganizeBy:       "{year}/{month}",
					},
				},
				{
					Name:    "recent-photos",
					Enabled: &enabled,
					Source: models.CategorySource{
						Path:       downloads,
						Extensions: []string{"jpg", "jpeg", "heic"},
						Filter: models.CategoryFilter{
							All: []models.CategoryFilter{
								{Size: &models.SizeFilter{Min: "500KB"}},
								{Age: &models.AgeFilter{Max: 30 * 24 * time.Hour}},
							},
						},
					},
					Destination: models.CategoryDestination{
						Path:             downloads + "/photos",
						ConflictStrategy: models.ConflictStrategyRename,
						OrganizeBy:       "{year}/{month}",
					},
				},
			},
		},
		// flat filters: match + size + age used directly (implicit AND, mutually exclusive with any/all)
		"full-filter-flat": {
			Configuration: consoleCfg,
			Categories: []models.Category{
				{
					Name:    "dated-docs",
					Enabled: &enabled,
					Source: models.CategorySource{
						Path:       downloads,
						Extensions: []string{"pdf", "docx"},
						Filter: models.CategoryFilter{
							Match: &models.MatchFilter{Regex: `^\d{4}-\d{2}-\d{2}_`},
							Age:   &models.AgeFilter{Min: 7 * 24 * time.Hour},
							Size:  &models.SizeFilter{Min: "10KB"},
						},
					},
					Destination: models.CategoryDestination{
						Path:             downloads + "/documents",
						ConflictStrategy: models.ConflictStrategyRename,
						OrganizeBy:       "{year}",
					},
				},
				{
					Name:    "screenshots",
					Enabled: &enabled,
					Source: models.CategorySource{
						Path:       downloads,
						Extensions: []string{"png", "jpg"},
						Filter: models.CategoryFilter{
							Match: &models.MatchFilter{Glob: "screenshot_*"},
							Size:  &models.SizeFilter{Max: "5MB"},
						},
					},
					Destination: models.CategoryDestination{
						Path:             downloads + "/screenshots",
						ConflictStrategy: models.ConflictStrategyRename,
						OrganizeBy:       "{year}/{month}",
					},
				},
			},
		},
		// classic downloads organizer: no filters, sort by extension group
		"downloads-organizer": {
			Configuration: consoleCfg,
			Categories: []models.Category{
				{
					Name:    "images",
					Enabled: &enabled,
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
				{
					Name:    "documents",
					Enabled: &enabled,
					Source: models.CategorySource{
						Path:       downloads,
						Extensions: []string{"pdf", "doc", "docx", "txt", "odt"},
					},
					Destination: models.CategoryDestination{
						Path:             downloads + "/documents",
						ConflictStrategy: models.ConflictStrategyRename,
						OrganizeBy:       "{year}",
					},
				},
				{
					Name:    "videos",
					Enabled: &enabled,
					Source: models.CategorySource{
						Path:       downloads,
						Extensions: []string{"mp4", "mkv", "avi"},
					},
					Destination: models.CategoryDestination{
						Path:             downloads + "/videos",
						ConflictStrategy: models.ConflictStrategyHashCheck,
						OrganizeBy:       "{year}/{month}",
					},
				},
				{
					Name:    "music",
					Enabled: &enabled,
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
				{
					Name:    "archives",
					Enabled: &enabled,
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
		},
		// non-destructive photographer workflow: copy to separate RAW/JPEG trees, log to file
		"photographer": {
			Configuration: fileCfg,
			Categories: []models.Category{
				{
					Name:    "raw",
					Enabled: &enabled,
					Source: models.CategorySource{
						Path:       downloads,
						Extensions: []string{"raw", "cr2", "nef", "arw"},
					},
					Destination: models.CategoryDestination{
						Path:             "~/Pictures/RAW",
						ConflictStrategy: models.ConflictStrategyHashCheck,
						Action:           models.ActionCopy,
						OrganizeBy:       "{year}/{month}",
					},
				},
				{
					Name:    "jpeg",
					Enabled: &enabled,
					Source: models.CategorySource{
						Path:       downloads,
						Extensions: []string{"jpg", "jpeg", "heic"},
					},
					Destination: models.CategoryDestination{
						Path:             "~/Pictures/JPEG",
						ConflictStrategy: models.ConflictStrategyRename,
						Action:           models.ActionCopy,
						OrganizeBy:       "{year}/{month}",
						Rename:           "{year}-{month}-{day}_{name}",
					},
				},
			},
		},
	}
}

func configurationPresetsMap() map[string]*models.Configuration {
	logFile := "~/.movelooper/logs/movelooper.log"
	histFile := "~/.movelooper/history/movelooper.json"

	cfg := func(output, level string, showCaller bool, format string) *models.Configuration {
		return &models.Configuration{
			Logging: models.Logging{
				Output:     output,
				Level:      level,
				File:       logFile,
				ShowCaller: showCaller,
				Format:     format,
				Color:      "auto",
			},
			Watch:   models.Watch{Delay: 5 * time.Minute},
			History: models.History{Limit: 100, File: histFile},
		}
	}

	return map[string]*models.Configuration{
		"console-trace": cfg("console", "trace", true, "pretty"),
		"console-debug": cfg("console", "debug", true, "pretty"),
		"console-info":  cfg("console", "info", false, "pretty"),
		"console-warn":  cfg("console", "warn", false, "pretty"),
		"console-error": cfg("console", "error", false, "pretty"),
		"console-fatal": cfg("console", "fatal", false, "pretty"),
		"file":          cfg("file", "warn", false, "pretty"),
		"both":          cfg("both", "info", false, "pretty"),
		"json":          cfg("file", "info", false, "json"),
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
		// archive: pack a whole category into one compressed file
		"archive-old-downloads": {
			{
				Name:    "archive-old-downloads",
				Enabled: &enabled,
				Source: models.CategorySource{
					Path:       downloads,
					Extensions: []string{"all"},
					Filter:     models.CategoryFilter{Age: &models.AgeFilter{Min: 30 * 24 * time.Hour}},
				},
				Destination: models.CategoryDestination{
					Path:   downloads + "/archives",
					Action: models.ActionArchive,
					Archive: &models.ArchiveConfig{
						Format:      "zip",
						Name:        "{category}_{date}",
						Compression: "best",
					},
				},
			},
		},
		// rename: appends a counter when destination file already exists
		"conflict-rename": {
			{
				Name:    "photos",
				Enabled: &enabled,
				Source: models.CategorySource{
					Path:       downloads,
					Extensions: []string{"jpg", "jpeg", "png"},
				},
				Destination: models.CategoryDestination{
					Path:             downloads + "/photos",
					ConflictStrategy: models.ConflictStrategyRename,
					OrganizeBy:       "{year}/{month}",
				},
			},
		},
		// hash_check: skips the move if source and destination are byte-identical
		"conflict-hash-check": {
			{
				Name:    "archives",
				Enabled: &enabled,
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
		// overwrite: replaces the destination file unconditionally
		"conflict-overwrite": {
			{
				Name:    "config-sync",
				Enabled: &enabled,
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
		},
		// skip: leaves the destination file untouched on conflict
		"conflict-skip": {
			{
				Name:    "music",
				Enabled: &enabled,
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
		},
		// newest: keeps whichever file has the most recent modification time
		"conflict-newest": {
			{
				Name:    "videos",
				Enabled: &enabled,
				Source: models.CategorySource{
					Path:       downloads,
					Extensions: []string{"mp4", "mkv", "avi"},
				},
				Destination: models.CategoryDestination{
					Path:             downloads + "/videos",
					ConflictStrategy: models.ConflictStrategyNewest,
					OrganizeBy:       "{year}",
				},
			},
		},
		// oldest: keeps whichever file has the earliest modification time
		"conflict-oldest": {
			{
				Name:    "documents",
				Enabled: &enabled,
				Source: models.CategorySource{
					Path:       downloads,
					Extensions: []string{"pdf", "doc", "docx"},
				},
				Destination: models.CategoryDestination{
					Path:             downloads + "/documents",
					ConflictStrategy: models.ConflictStrategyOldest,
					OrganizeBy:       "{year}",
				},
			},
		},
		// larger: keeps whichever file is larger in bytes
		"conflict-larger": {
			{
				Name:    "archives-large",
				Enabled: &enabled,
				Source: models.CategorySource{
					Path:       downloads,
					Extensions: []string{"zip", "7z"},
				},
				Destination: models.CategoryDestination{
					Path:             downloads + "/archives/large",
					ConflictStrategy: models.ConflictStrategyLarger,
					OrganizeBy:       "{ext}",
				},
			},
		},
		// smaller: keeps whichever file is smaller in bytes
		"conflict-smaller": {
			{
				Name:    "photos-small",
				Enabled: &enabled,
				Source: models.CategorySource{
					Path:       downloads,
					Extensions: []string{"jpg", "jpeg"},
				},
				Destination: models.CategoryDestination{
					Path:             downloads + "/photos/small",
					ConflictStrategy: models.ConflictStrategySmaller,
					OrganizeBy:       "{year}",
				},
			},
		},
		// filter.size: match files within a byte-size range
		"filter-size": {
			{
				Name:    "large-videos",
				Enabled: &enabled,
				Source: models.CategorySource{
					Path:       downloads,
					Extensions: []string{"mp4", "mkv"},
					Filter: models.CategoryFilter{
						Size: &models.SizeFilter{Min: "100MB", Max: "10GB"},
					},
				},
				Destination: models.CategoryDestination{
					Path:             downloads + "/videos",
					ConflictStrategy: models.ConflictStrategyHashCheck,
					OrganizeBy:       "{year}/{month}",
				},
			},
		},
		// filter.age: match files within a modification-time window
		"filter-age": {
			{
				Name:    "old-downloads",
				Enabled: &enabled,
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
		},
		// filter.any: OR — match files satisfying at least one sub-filter
		"filter-any": {
			{
				Name:    "reports",
				Enabled: &enabled,
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
		},
		// filter.all: AND — match files satisfying every sub-filter simultaneously
		"filter-all": {
			{
				Name:    "recent-docs",
				Enabled: &enabled,
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
		// filter.not: exclude files matching any of the sub-filters
		"filter-not": {
			{
				Name:    "documents",
				Enabled: &enabled,
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
		// match.glob: wildcard pattern against the filename
		"filter-match-glob": {
			{
				Name:    "screenshots",
				Enabled: &enabled,
				Source: models.CategorySource{
					Path:       downloads,
					Extensions: []string{"png", "jpg"},
					Filter: models.CategoryFilter{
						Match: &models.MatchFilter{Glob: "screenshot_????-??-??_*"},
					},
				},
				Destination: models.CategoryDestination{
					Path:             downloads + "/screenshots",
					ConflictStrategy: models.ConflictStrategyRename,
					OrganizeBy:       "{year}/{month}",
				},
			},
		},
		// match.regex: RE2 regular expression against the filename
		"filter-match-regex": {
			{
				Name:    "dated-reports",
				Enabled: &enabled,
				Source: models.CategorySource{
					Path:       downloads,
					Extensions: []string{"pdf", "csv", "xlsx"},
					Filter: models.CategoryFilter{
						Match: &models.MatchFilter{Regex: `^\d{4}-\d{2}-\d{2}_.*`},
					},
				},
				Destination: models.CategoryDestination{
					Path:             downloads + "/reports",
					ConflictStrategy: models.ConflictStrategyRename,
					OrganizeBy:       "{year}/{month}",
				},
			},
		},
		// match.literal: exact filename match (whole name including extension)
		"filter-match-literal": {
			{
				Name:    "annas-archive",
				Enabled: &enabled,
				Source: models.CategorySource{
					Path:       downloads,
					Extensions: []string{"pdf"},
					Filter: models.CategoryFilter{
						Match: &models.MatchFilter{Literal: "Anna's Archive.pdf"},
					},
				},
				Destination: models.CategoryDestination{
					Path:             downloads + "/books",
					ConflictStrategy: models.ConflictStrategySkip,
				},
			},
		},
		// hooks.before: shell commands run before each file operation; on-failure: abort cancels the move
		"hooks-before": {
			{
				Name:    "videos-before",
				Enabled: &enabled,
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
							`echo "moving $ML_SOURCE_PATH"`,
						},
					},
				},
			},
		},
		// hooks.after: shell commands run after each file operation; on-failure: warn logs but continues
		"hooks-after": {
			{
				Name:    "videos-after",
				Enabled: &enabled,
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
					After: &models.CategoryHook{
						Shell:     "bash",
						OnFailure: "warn",
						Run: []string{
							`echo "$ML_FILES_MOVED files moved to $ML_DEST_PATH"`,
						},
					},
				},
			},
		},
		// recursive: scan sub-directories up to max-depth, skipping exclude-paths
		"recursive": {
			{
				Name:    "documents-recursive",
				Enabled: &enabled,
				Source: models.CategorySource{
					Path:         downloads,
					Extensions:   []string{"pdf", "doc", "docx", "txt"},
					Recursive:    true,
					MaxDepth:     3,
					ExcludePaths: []string{downloads + "/archives", downloads + "/temp"},
				},
				Destination: models.CategoryDestination{
					Path:             downloads + "/documents",
					ConflictStrategy: models.ConflictStrategyHashCheck,
					OrganizeBy:       "{year}/{month}",
				},
			},
		},
		// action.copy: keeps the source file, places a copy at the destination
		"action-copy": {
			{
				Name:    "photos-backup",
				Enabled: &enabled,
				Source: models.CategorySource{
					Path:       downloads,
					Extensions: []string{"jpg", "jpeg", "heic"},
				},
				Destination: models.CategoryDestination{
					Path:             downloads + "/Pictures/backup",
					ConflictStrategy: models.ConflictStrategySkip,
					Action:           models.ActionCopy,
					OrganizeBy:       "{year}/{month}",
				},
			},
		},
		// action.symlink: creates a symbolic link at the destination pointing to the source
		"action-symlink": {
			{
				Name:    "media-links",
				Enabled: &enabled,
				Source: models.CategorySource{
					Path:       downloads,
					Extensions: []string{"mp4", "mkv", "avi"},
				},
				Destination: models.CategoryDestination{
					Path:             downloads + "/Media/links",
					ConflictStrategy: models.ConflictStrategySkip,
					Action:           models.ActionSymlink,
					OrganizeBy:       "{year}",
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
