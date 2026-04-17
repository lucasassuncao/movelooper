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

func TestResolveSeq(t *testing.T) {
	tests := []struct {
		name     string
		existing []string
		nonExist bool
		want     int
	}{
		{
			name:     "empty directory returns 1",
			existing: nil,
			want:     1,
		},
		{
			name:     "directory does not exist returns 1",
			nonExist: true,
			want:     1,
		},
		{
			name:     "single file with leading number",
			existing: []string{"0001_photo.jpg"},
			want:     2,
		},
		{
			name:     "multiple files picks max",
			existing: []string{"0001_a.jpg", "0005_b.jpg", "0003_c.jpg"},
			want:     6,
		},
		{
			name:     "files without leading number are ignored",
			existing: []string{"photo.jpg", "banner.png"},
			want:     1,
		},
		{
			name:     "mixed: some with numbers some without",
			existing: []string{"0002_x.jpg", "logo.png"},
			want:     3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dir string
			if tt.nonExist {
				dir = filepath.Join(t.TempDir(), "nonexistent")
			} else {
				dir = t.TempDir()
				for _, name := range tt.existing {
					require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("x"), 0644))
				}
			}
			got := ResolveSeq(dir)
			assert.Equal(t, tt.want, got)
		})
	}
}

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

func TestValidateTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		wantErr  bool
	}{
		{"empty template", "", false},
		{"valid single token", "{ext}", false},
		{"valid composite", "{mod-date}_{name}.{ext}", false},
		{"all known tokens", "{name}{ext}{ext-upper}{mod-year}{mod-month}{mod-day}{mod-date}{mod-weekday}{created-year}{created-month}{created-day}{created-date}{year}{month}{day}{date}{weekday}{size-range}{category}", false},
		{"unknown token", "{unknown}", true},
		{"mixed valid and unknown", "{name}_{foo}", true},
		{"partial brace no token", "hello world", false},
		// seq token cases
		{"seq no padding", "{seq}", false},
		{"seq with padding 4", "{seq:4}", false},
		{"seq with padding 1", "{seq:1}", false},
		{"seq with padding 20", "{seq:20}", false},
		{"seq zero padding invalid", "{seq:0}", true},
		{"seq padding too large", "{seq:21}", true},
		{"seq non-numeric padding", "{seq:abc}", true},
		{"seq empty padding", "{seq:}", true},
		{"seq combined with other tokens", "{seq:4}_{name}.{ext}", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTemplate(tt.template)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestResolveRename(t *testing.T) {
	now := time.Date(2025, 4, 16, 0, 0, 0, 0, time.UTC)

	tmp := t.TempDir()
	path := filepath.Join(tmp, "photo.JPG")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0644))
	modTime := time.Date(2024, 3, 5, 12, 0, 0, 0, time.UTC)
	require.NoError(t, os.Chtimes(path, modTime, modTime))
	info, err := os.Stat(path)
	require.NoError(t, err)

	// empty destDir — seq starts at 1
	destDir := t.TempDir()

	tests := []struct {
		name     string
		template string
		want     string
	}{
		{"empty template returns original name", "", "photo.JPG"},
		{"ext token lowercase", "{name}.{ext}", "photo.jpg"},
		{"ext-upper token", "{name}.{ext-upper}", "photo.JPG"},
		{"mod-date prefix", "{mod-date}_{name}.{ext}", "2024-03-05_photo.jpg"},
		{"category token", "{category}_{name}.{ext}", "images_photo.jpg"},
		{"run date", "{date}_{name}.{ext}", "2025-04-16_photo.jpg"},
		{"seq no padding", "{seq}_{name}.{ext}", "1_photo.jpg"},
		{"seq with padding 3", "{seq:3}_{name}.{ext}", "001_photo.jpg"},
		{"seq with padding 4", "{seq:4}_{name}.{ext}", "0001_photo.jpg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveRename(tt.template, info, "images", now, destDir)
			assert.Equal(t, tt.want, got)
		})
	}
}
