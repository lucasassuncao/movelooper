package helper

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	sizeThresholdTiny   = int64(1 << 20)         // 1 MB
	sizeThresholdSmall  = int64(100 * (1 << 20)) // 100 MB
	sizeThresholdMedium = int64(1 << 30)         // 1 GB
)

// ResolveGroupBy resolves a group-by template string into a relative subdirectory
// path that should be appended to the category destination.
//
// Supported tokens:
//
//	File identification:
//	  {name}          — filename without extension
//	  {ext}           — extension without dot, lowercase
//	  {ext-upper}     — extension without dot, uppercase
//
//	File modification date:
//	  {mod-year}      — year  (2025)
//	  {mod-month}     — month (04)
//	  {mod-day}       — day   (08)
//	  {mod-date}      — 2025-04-08
//	  {mod-weekday}   — Tuesday
//
//	File creation date (falls back to mod time on Linux):
//	  {created-year}  — year
//	  {created-month} — month
//	  {created-day}   — day
//	  {created-date}  — 2025-04-08
//
//	Run date (time.Now()):
//	  {year}          — year
//	  {month}         — month
//	  {day}           — day
//	  {date}          — 2025-04-08
//	  {weekday}       — Tuesday
//
//	File size:
//	  {size-range}    — tiny (<1 MB) | small (1 MB–100 MB) | medium (100 MB–1 GB) | large (≥1 GB)
//
//	Category:
//	  {category}      — category name from config
func ResolveGroupBy(template string, info os.FileInfo, categoryName string, now time.Time) string {
	if template == "" {
		return ""
	}

	modTime := info.ModTime()
	createdTime := getBirthTime(info)

	rawExt := strings.TrimPrefix(filepath.Ext(info.Name()), ".")
	name := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))

	r := strings.NewReplacer(
		// identification
		"{name}", name,
		"{ext}", strings.ToLower(rawExt),
		"{ext-upper}", strings.ToUpper(rawExt),
		// modification date
		"{mod-year}", modTime.Format("2006"),
		"{mod-month}", modTime.Format("01"),
		"{mod-day}", modTime.Format("02"),
		"{mod-date}", modTime.Format("2006-01-02"),
		"{mod-weekday}", modTime.Weekday().String(),
		// creation date
		"{created-year}", createdTime.Format("2006"),
		"{created-month}", createdTime.Format("01"),
		"{created-day}", createdTime.Format("02"),
		"{created-date}", createdTime.Format("2006-01-02"),
		// run date
		"{year}", now.Format("2006"),
		"{month}", now.Format("01"),
		"{day}", now.Format("02"),
		"{date}", now.Format("2006-01-02"),
		"{weekday}", now.Weekday().String(),
		// size
		"{size-range}", fileSizeRange(info.Size()),
		// category
		"{category}", categoryName,
	)

	return filepath.FromSlash(r.Replace(template))
}

func fileSizeRange(size int64) string {
	switch {
	case size < sizeThresholdTiny:
		return "tiny"
	case size < sizeThresholdSmall:
		return "small"
	case size < sizeThresholdMedium:
		return "medium"
	default:
		return "large"
	}
}
