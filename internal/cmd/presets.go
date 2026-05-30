package cmd

import (
	"sort"
	"time"

	"github.com/lucasassuncao/movelooper/internal/models"
)

func configPresetsMap() map[string]*models.Config {
	downloads := "~/Downloads"
	defaultCfg := models.Configuration{
		Output:       "console",
		LogLevel:     "info",
		ShowCaller:   false,
		WatchDelay:   5 * time.Minute,
		HistoryLimit: 50,
	}

	return map[string]*models.Config{
		"basic": {
			Configuration: defaultCfg,
			Categories: []models.Category{
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
		},
		"images": {
			Configuration: defaultCfg,
			Categories: []models.Category{
				{
					Name: "images",
					Source: models.CategorySource{
						Path:       downloads,
						Extensions: []string{"jpg", "jpeg", "png", "gif", "bmp", "webp", "svg"},
					},
					Destination: models.CategoryDestination{
						Path:             downloads + "/images",
						ConflictStrategy: "rename",
						OrganizeBy:       "{ext}",
					},
				},
			},
		},
		"music": {
			Configuration: defaultCfg,
			Categories: []models.Category{
				{
					Name: "music",
					Source: models.CategorySource{
						Path:       downloads,
						Extensions: []string{"mp3", "wav", "flac", "aac"},
					},
					Destination: models.CategoryDestination{
						Path:             downloads + "/music",
						ConflictStrategy: "rename",
						OrganizeBy:       "{ext}",
					},
				},
			},
		},
		"video": {
			Configuration: defaultCfg,
			Categories: []models.Category{
				{
					Name: "videos",
					Source: models.CategorySource{
						Path:       downloads,
						Extensions: []string{"mp4", "avi", "mkv", "mov", "wmv"},
					},
					Destination: models.CategoryDestination{
						Path:             downloads + "/videos",
						ConflictStrategy: "rename",
						OrganizeBy:       "{ext}",
					},
				},
			},
		},
		"books": {
			Configuration: defaultCfg,
			Categories: []models.Category{
				{
					Name: "books",
					Source: models.CategorySource{
						Path:       downloads,
						Extensions: []string{"pdf", "epub", "mobi", "azw3", "doc", "docx"},
					},
					Destination: models.CategoryDestination{
						Path:             downloads + "/books",
						ConflictStrategy: "rename",
						OrganizeBy:       "{ext}",
					},
				},
			},
		},
		"archives": {
			Configuration: defaultCfg,
			Categories: []models.Category{
				{
					Name: "archives",
					Source: models.CategorySource{
						Path:       downloads,
						Extensions: []string{"zip", "tar", "gz", "bz2", "rar", "7z"},
					},
					Destination: models.CategoryDestination{
						Path:             downloads + "/archives",
						ConflictStrategy: "rename",
						OrganizeBy:       "{ext}",
					},
				},
			},
		},
		"installers": {
			Configuration: defaultCfg,
			Categories: []models.Category{
				{
					Name: "installers",
					Source: models.CategorySource{
						Path:       downloads,
						Extensions: []string{"exe", "msi", "apk"},
					},
					Destination: models.CategoryDestination{
						Path:             downloads + "/installers",
						ConflictStrategy: "rename",
						OrganizeBy:       "{ext}",
					},
				},
			},
		},
		"regex": {
			Configuration: defaultCfg,
			Categories: []models.Category{
				{
					Name: "dated-docs",
					Source: models.CategorySource{
						Path:       downloads,
						Extensions: []string{"pdf", "txt", "log"},
						Filter: models.CategoryFilter{
							Regex: `^\d{4}-\d{2}-\d{2}_.*`,
						},
					},
					Destination: models.CategoryDestination{
						Path:             downloads + "/dated-docs",
						ConflictStrategy: "rename",
						OrganizeBy:       "{ext}",
					},
				},
			},
		},
		"full": {
			Configuration: defaultCfg,
			Categories: []models.Category{
				{
					Name: "images",
					Source: models.CategorySource{
						Path:       downloads,
						Extensions: []string{"jpg", "jpeg", "png", "gif", "bmp", "webp", "svg"},
						Filter: models.CategoryFilter{
							Ignore: []string{"screenshot_*", "*_temp.*"},
						},
					},
					Destination: models.CategoryDestination{
						Path:             downloads + "/images",
						ConflictStrategy: "rename",
						OrganizeBy:       "{ext}",
					},
				},
				{
					Name: "videos",
					Source: models.CategorySource{
						Path:       downloads,
						Extensions: []string{"mp4", "avi", "mkv", "mov", "wmv"},
						Filter: models.CategoryFilter{
							Ignore:  []string{"*_preview.*", "*_draft.*"},
							MinSize: "100MB",
						},
					},
					Destination: models.CategoryDestination{
						Path:             downloads + "/videos",
						ConflictStrategy: "overwrite",
						OrganizeBy:       "{ext}",
					},
				},
				{
					Name: "music",
					Source: models.CategorySource{
						Path:       downloads,
						Extensions: []string{"mp3", "wav", "flac", "aac"},
					},
					Destination: models.CategoryDestination{
						Path:             downloads + "/music",
						ConflictStrategy: "skip",
						OrganizeBy:       "{ext}",
					},
				},
				{
					Name: "books",
					Source: models.CategorySource{
						Path:       downloads,
						Extensions: []string{"pdf", "epub", "mobi", "azw3", "doc", "docx"},
						Filter: models.CategoryFilter{
							MinSize: "1MB",
						},
					},
					Destination: models.CategoryDestination{
						Path:             downloads + "/books",
						ConflictStrategy: "hash_check",
						OrganizeBy:       "{ext}",
					},
				},
				{
					Name: "archives",
					Source: models.CategorySource{
						Path:       downloads,
						Extensions: []string{"zip", "tar", "gz", "bz2", "rar", "7z"},
					},
					Destination: models.CategoryDestination{
						Path:             downloads + "/archives",
						ConflictStrategy: "hash_check",
						OrganizeBy:       "{ext}",
					},
				},
				{
					Name: "installers",
					Source: models.CategorySource{
						Path:       downloads,
						Extensions: []string{"exe", "msi", "apk"},
					},
					Destination: models.CategoryDestination{
						Path:             downloads + "/installers",
						ConflictStrategy: "hash_check",
						OrganizeBy:       "{ext}",
					},
				},
			},
		},
	}
}

// ConfigPreset returns the Config for the named preset, or nil if not found.
func ConfigPreset(name string) *models.Config {
	return configPresetsMap()[name]
}

// ListOfConfigPresets returns the sorted list of available preset names.
func ListOfConfigPresets() []string {
	presets := configPresetsMap()
	keys := make([]string, 0, len(presets))
	for key := range presets {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
