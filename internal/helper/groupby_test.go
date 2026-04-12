package helper

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileSizeRange(t *testing.T) {
	tests := []struct {
		size int64
		want string
	}{
		{0, "tiny"},
		{500 * 1024, "tiny"},
		{sizeThresholdTiny, "small"},
		{50 * 1024 * 1024, "small"},
		{sizeThresholdSmall, "medium"},
		{500 * 1024 * 1024, "medium"},
		{sizeThresholdMedium, "large"},
		{2 * 1024 * 1024 * 1024, "large"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.size), func(t *testing.T) {
			assert.Equal(t, tt.want, fileSizeRange(tt.size))
		})
	}
}

func TestResolveGroupBy(t *testing.T) {
	// Fixed reference times used across cases.
	now := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC) // Friday
	modTime := time.Date(2023, 7, 4, 12, 0, 0, 0, time.Local)

	// Create files once and reuse across cases.
	dir := t.TempDir()

	newFile := func(name string, size int, mt time.Time) os.FileInfo {
		p := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(p, make([]byte, size), 0644))
		if !mt.IsZero() {
			require.NoError(t, os.Chtimes(p, mt, mt))
		}
		info, err := os.Stat(p)
		require.NoError(t, err)
		return info
	}

	plain := newFile("my-report.PDF", 1, modTime)
	tiny := newFile("tiny.bin", 500, modTime)

	// For created tokens: platform-agnostic expected value via getBirthTime.
	createdTime := getBirthTime(plain)

	tests := []struct {
		name     string
		template string
		info     os.FileInfo
		category string
		now      time.Time
		want     string
	}{
		// empty
		{"empty template", "", plain, "docs", now, ""},

		// identification
		{"name", "{name}", plain, "docs", now, "my-report"},
		{"ext lowercase", "{ext}", plain, "docs", now, "pdf"},
		{"ext uppercase", "{ext-upper}", plain, "docs", now, "PDF"},

		// modification date
		{"mod-year", "{mod-year}", plain, "cat", now, "2023"},
		{"mod-month", "{mod-month}", plain, "cat", now, "07"},
		{"mod-day", "{mod-day}", plain, "cat", now, "04"},
		{"mod-date", "{mod-date}", plain, "cat", now, "2023-07-04"},
		{"mod-weekday", "{mod-weekday}", plain, "cat", now, modTime.Weekday().String()},

		// creation date (platform-agnostic)
		{"created-year", "{created-year}", plain, "cat", now, createdTime.Format("2006")},
		{"created-month", "{created-month}", plain, "cat", now, createdTime.Format("01")},
		{"created-day", "{created-day}", plain, "cat", now, createdTime.Format("02")},
		{"created-date", "{created-date}", plain, "cat", now, createdTime.Format("2006-01-02")},

		// run date
		{"year", "{year}", plain, "cat", now, "2024"},
		{"month", "{month}", plain, "cat", now, "03"},
		{"day", "{day}", plain, "cat", now, "15"},
		{"date", "{date}", plain, "cat", now, "2024-03-15"},
		{"weekday", "{weekday}", plain, "cat", now, "Friday"},

		// size-range (boundary cases covered by TestFileSizeRange)
		{"size-range tiny", "{size-range}", tiny, "cat", now, "tiny"},

		// category
		{"category", "{category}", plain, "MyCategory", now, "MyCategory"},

		// combined
		{"combined", "{category}/{year}/{ext}", plain, "docs", now, filepath.FromSlash("docs/2024/pdf")},
		{"combined created path", "{created-year}/{created-month}/{created-day}", plain, "cat", now, filepath.FromSlash(createdTime.Format("2006/01/02"))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveGroupBy(tt.template, tt.info, tt.category, tt.now)
			assert.Equal(t, tt.want, got)
		})
	}
}
