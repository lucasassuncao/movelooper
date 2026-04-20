package tokens

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- validate ---

func TestValidateTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		wantErr  bool
	}{
		{"empty template", "", false},
		{"valid single token", "{ext}", false},
		{"valid composite", "{mod-date}_{name}.{ext}", false},
		{
			"all known static tokens",
			"{name}{ext}{ext-upper}{ext-lower}{ext-reverse}{name-slug}{name-snake}{name-upper}" +
				"{name-lower}{name-alpha}{name-ascii}{name-initials}{name-reverse}{mod-year}{mod-month}" +
				"{mod-day}{mod-date}{mod-weekday}{created-year}{created-month}{created-day}{created-date}" +
				"{year}{month}{day}{date}{weekday}{hour}{minute}{second}{timestamp}{size-range}{category}" +
				"{hostname}{username}{os}{seq-alpha}{seq-roman}{md5}",
			false,
		},
		{"unknown token", "{unknown}", true},
		{"mixed valid and unknown", "{name}_{foo}", true},
		{"partial brace no token", "hello world", false},
		// seq
		{"seq no padding", "{seq}", false},
		{"seq with padding 4", "{seq:4}", false},
		{"seq with padding 1", "{seq:1}", false},
		{"seq with padding 20", "{seq:20}", false},
		{"seq zero padding invalid", "{seq:0}", true},
		{"seq padding too large", "{seq:21}", true},
		{"seq non-numeric padding", "{seq:abc}", true},
		{"seq empty padding", "{seq:}", true},
		{"seq combined with other tokens", "{seq:4}_{name}.{ext}", false},
		// name-trunc
		{"name-trunc valid", "{name-trunc:10}", false},
		{"name-trunc min", "{name-trunc:1}", false},
		{"name-trunc max", "{name-trunc:255}", false},
		{"name-trunc zero", "{name-trunc:0}", true},
		{"name-trunc over max", "{name-trunc:256}", true},
		{"name-trunc non-numeric", "{name-trunc:abc}", true},
		{"name-trunc empty param", "{name-trunc:}", true},
		// md5
		{"md5 default", "{md5}", false},
		{"md5:N valid", "{md5:8}", false},
		{"md5:N min", "{md5:1}", false},
		{"md5:N max", "{md5:32}", false},
		{"md5:N zero", "{md5:0}", true},
		{"md5:N over max", "{md5:33}", true},
		{"md5:N non-numeric", "{md5:abc}", true},
		{"md5:N empty param", "{md5:}", true},
		// sha256
		{"sha256:N valid", "{sha256:16}", false},
		{"sha256:N min", "{sha256:1}", false},
		{"sha256:N max", "{sha256:64}", false},
		{"sha256:N zero", "{sha256:0}", true},
		{"sha256:N over max", "{sha256:65}", true},
		{"sha256:N non-numeric", "{sha256:abc}", true},
		{"sha256:N empty param", "{sha256:}", true},
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

// --- name transforms ---

func TestNameTransforms(t *testing.T) {
	tests := []struct {
		fn   func(string) string
		name string
		in   string
		want string
	}{
		{nameSlug, "slug basic", "My File Name", "my-file-name"},
		{nameSlug, "slug specials", "report (final)!", "report-final"},
		{nameSlug, "slug collapse hyphens", "foo--bar", "foo-bar"},
		{nameSlug, "slug trim hyphens", "-hello-", "hello"},
		{nameSnake, "snake basic", "My File Name", "my_file_name"},
		{nameSnake, "snake specials", "report (final)!", "report_final"},
		{nameAlpha, "alpha removes specials", "report 2025 (final)!", "report2025final"},
		{nameAlpha, "alpha keeps alnum", "abc123", "abc123"},
		{nameASCII, "ascii accents", "Ação_résumé", "Acao_resume"},
		{nameASCII, "ascii plain", "hello", "hello"},
		{nameInitials, "initials spaces", "my vacation photos", "mvp"},
		{nameInitials, "initials mixed sep", "my-vacation_photos", "mvp"},
		{nameInitials, "initials single word", "report", "r"},
		{nameReverse, "reverse", "photo", "otohp"},
		{nameReverse, "reverse unicode", "café", "éfac"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.fn(tt.in))
		})
	}
}

// --- system context ---

func TestSystemContext(t *testing.T) {
	initSystemContext()
	assert.NotEmpty(t, systemHostname)
	assert.NotEmpty(t, systemUsername)
	assert.NotEmpty(t, systemOS)
}

// --- size ---

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

// --- name-trunc ---

func TestPreProcessNameTrunc(t *testing.T) {
	tests := []struct {
		template string
		name     string
		want     string
	}{
		{"{name-trunc:4}", "very-long-name", "very"},
		{"{name-trunc:8}", "very-long-name", "very-lon"},
		{"{name-trunc:20}", "short", "short"},
		{"{name-trunc:1}", "abc", "a"},
		{"prefix_{name-trunc:3}.txt", "report", "prefix_rep.txt"},
		{"no-token", "anything", "no-token"},
	}
	for _, tt := range tests {
		t.Run(tt.template, func(t *testing.T) {
			assert.Equal(t, tt.want, preProcessNameTrunc(tt.template, tt.name))
		})
	}
}

// --- hash ---

func TestPreProcessHash(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "file.txt")
	require.NoError(t, os.WriteFile(path, []byte("hello"), 0644))

	t.Run("md5 default 8 chars", func(t *testing.T) {
		// MD5 of "hello" = 5d41402abc4b2a76b9719d911017c592
		got := preProcessHash("{md5}_{name}.{ext}", path)
		assert.Equal(t, "5d41402a_{name}.{ext}", got)
	})
	t.Run("md5:N custom length", func(t *testing.T) {
		got := preProcessHash("{md5:4}_{name}", path)
		assert.Equal(t, "5d41_{name}", got)
	})
	t.Run("sha256:N", func(t *testing.T) {
		// SHA-256 of "hello" = 2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824
		got := preProcessHash("{sha256:6}_{name}", path)
		assert.Equal(t, "2cf24d_{name}", got)
	})
	t.Run("no hash token passthrough", func(t *testing.T) {
		got := preProcessHash("{name}.{ext}", path)
		assert.Equal(t, "{name}.{ext}", got)
	})
	t.Run("unreadable file returns unknown", func(t *testing.T) {
		got := preProcessHash("{md5}_{name}", "/nonexistent/file.txt")
		assert.Equal(t, "unknown_{name}", got)
	})
}

// --- seq numeric ---

func TestResolveSeq(t *testing.T) {
	tests := []struct {
		name     string
		existing []string
		nonExist bool
		want     int
	}{
		{"empty directory returns 1", nil, false, 1},
		{"directory does not exist returns 1", nil, true, 1},
		{"single file with leading number", []string{"0001_photo.jpg"}, false, 2},
		{"multiple files picks max", []string{"0001_a.jpg", "0005_b.jpg", "0003_c.jpg"}, false, 6},
		{"files without leading number are ignored", []string{"photo.jpg", "banner.png"}, false, 1},
		{"mixed: some with numbers some without", []string{"0002_x.jpg", "logo.png"}, false, 3},
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
			assert.Equal(t, tt.want, ResolveSeq(dir))
		})
	}
}

// --- seq alpha ---

func TestAlphaConversion(t *testing.T) {
	cases := []struct {
		n    int
		want string
	}{
		{1, "a"}, {2, "b"}, {26, "z"},
		{27, "aa"}, {28, "ab"}, {52, "az"},
		{53, "ba"}, {702, "zz"}, {703, "aaa"},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%d=%s", c.n, c.want), func(t *testing.T) {
			assert.Equal(t, c.want, intToAlpha(c.n))
			assert.Equal(t, c.n, alphaToInt(c.want))
		})
	}
}

func TestResolveSeqAlpha(t *testing.T) {
	tests := []struct {
		name     string
		existing []string
		want     string
	}{
		{"empty dir returns a", nil, "a"},
		{"after a returns b", []string{"a_doc.pdf"}, "b"},
		{"after z returns aa", []string{"z_doc.pdf"}, "aa"},
		{"after aa returns ab", []string{"aa_doc.pdf"}, "ab"},
		{"picks max", []string{"a_x.pdf", "c_x.pdf", "b_x.pdf"}, "d"},
		{"ignores non-alpha prefix", []string{"1_x.pdf"}, "a"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, f := range tt.existing {
				require.NoError(t, os.WriteFile(filepath.Join(dir, f), []byte("x"), 0644))
			}
			assert.Equal(t, tt.want, ResolveSeqAlpha(dir))
		})
	}
}

// --- seq roman ---

func TestRomanConversion(t *testing.T) {
	cases := []struct {
		n    int
		want string
	}{
		{1, "i"}, {4, "iv"}, {5, "v"}, {9, "ix"},
		{10, "x"}, {14, "xiv"}, {40, "xl"}, {90, "xc"},
		{100, "c"}, {400, "cd"}, {500, "d"}, {900, "cm"},
		{1000, "m"}, {1999, "mcmxcix"}, {2024, "mmxxiv"},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%d=%s", c.n, c.want), func(t *testing.T) {
			assert.Equal(t, c.want, intToRoman(c.n))
			assert.Equal(t, c.n, romanToInt(c.want))
		})
	}
}

func TestResolveSeqRoman(t *testing.T) {
	tests := []struct {
		name     string
		existing []string
		want     string
	}{
		{"empty dir returns i", nil, "i"},
		{"after i returns ii", []string{"i_doc.pdf"}, "ii"},
		{"after iv returns v", []string{"iv_doc.pdf"}, "v"},
		{"picks max", []string{"i_x.pdf", "iii_x.pdf", "ii_x.pdf"}, "iv"},
		{"ignores non-roman prefix", []string{"1_x.pdf", "a_x.pdf"}, "i"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, f := range tt.existing {
				require.NoError(t, os.WriteFile(filepath.Join(dir, f), []byte("x"), 0644))
			}
			assert.Equal(t, tt.want, ResolveSeqRoman(dir))
		})
	}
}

// --- ResolveGroupBy ---

func TestResolveGroupBy(t *testing.T) {
	now := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC) // Friday
	modTime := time.Date(2023, 7, 4, 12, 0, 0, 0, time.Local)

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
	createdTime := getBirthTime(plain)

	initSystemContext()

	tests := []struct {
		name     string
		template string
		info     os.FileInfo
		category string
		now      time.Time
		want     string
	}{
		{"empty template", "", plain, "docs", now, ""},
		// identification
		{"name", "{name}", plain, "docs", now, "my-report"},
		{"ext lowercase", "{ext}", plain, "docs", now, "pdf"},
		{"ext uppercase", "{ext-upper}", plain, "docs", now, "PDF"},
		{"ext-lower", "{ext-lower}", plain, "docs", now, "pdf"},
		{"ext-reverse", "{ext-reverse}", plain, "docs", now, "fdp"},
		// name transforms
		{"name-slug", "{name-slug}", plain, "docs", now, "my-report"},
		{"name-snake", "{name-snake}", plain, "docs", now, "my_report"},
		{"name-upper", "{name-upper}", plain, "docs", now, "MY-REPORT"},
		{"name-lower", "{name-lower}", plain, "docs", now, "my-report"},
		{"name-alpha", "{name-alpha}", plain, "docs", now, "myreport"},
		{"name-initials", "{name-initials}", plain, "docs", now, "mr"},
		{"name-reverse", "{name-reverse}", plain, "docs", now, "troper-ym"},
		// modification date
		{"mod-year", "{mod-year}", plain, "cat", now, "2023"},
		{"mod-month", "{mod-month}", plain, "cat", now, "07"},
		{"mod-day", "{mod-day}", plain, "cat", now, "04"},
		{"mod-date", "{mod-date}", plain, "cat", now, "2023-07-04"},
		{"mod-weekday", "{mod-weekday}", plain, "cat", now, modTime.Weekday().String()},
		// creation date
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
		// run time
		{"hour", "{hour}", plain, "cat", now, "00"},
		{"minute", "{minute}", plain, "cat", now, "00"},
		{"second", "{second}", plain, "cat", now, "00"},
		{"timestamp", "{timestamp}", plain, "cat", now, "20240315-000000"},
		// size
		{"size-range tiny", "{size-range}", tiny, "cat", now, "tiny"},
		// category
		{"category", "{category}", plain, "MyCategory", now, "MyCategory"},
		// system context
		{"hostname non-empty", "{hostname}", plain, "cat", now, systemHostname},
		{"username non-empty", "{username}", plain, "cat", now, systemUsername},
		{"os non-empty", "{os}", plain, "cat", now, systemOS},
		// name-trunc
		{"name-trunc:4", "{name-trunc:4}", plain, "docs", now, "my-r"},
		{"name-trunc:100 longer than name", "{name-trunc:100}", plain, "docs", now, "my-report"},
		// combined
		{"combined", "{category}/{year}/{ext}", plain, "docs", now, filepath.FromSlash("docs/2024/pdf")},
		{"combined created path", "{created-year}/{created-month}/{created-day}", plain, "cat", now, filepath.FromSlash(createdTime.Format("2006/01/02"))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveGroupBy(tt.template, TokenContext{
				Info:         tt.info,
				CategoryName: tt.category,
				Now:          tt.now,
			})
			assert.Equal(t, tt.want, got)
		})
	}
}

// --- ResolveRename ---

func TestResolveRename(t *testing.T) {
	now := time.Date(2025, 4, 16, 0, 0, 0, 0, time.UTC)

	tmp := t.TempDir()
	path := filepath.Join(tmp, "photo.JPG")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0644))
	modTime := time.Date(2024, 3, 5, 12, 0, 0, 0, time.UTC)
	require.NoError(t, os.Chtimes(path, modTime, modTime))
	info, err := os.Stat(path)
	require.NoError(t, err)

	destDir := t.TempDir()

	ctx := func() TokenContext {
		return TokenContext{
			Info:         info,
			CategoryName: "images",
			Now:          now,
			DestDir:      destDir,
			SourcePath:   path,
		}
	}

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
		{"seq-alpha empty dir", "{seq-alpha}_{name}.{ext}", "a_photo.jpg"},
		{"seq-roman empty dir", "{seq-roman}_{name}.{ext}", "i_photo.jpg"},
		// md5 of "x" = 9dd4e461268c8034f5c8564e155c67a6
		{"md5 default 8 chars", "{md5}_{name}.{ext}", "9dd4e461_photo.jpg"},
		{"md5:4", "{md5:4}_{name}.{ext}", "9dd4_photo.jpg"},
		// sha256 of "x" = 2d711642b726b04401627ca9fbac32f5c8530fb1903cc4db02258717921a4881
		{"sha256:6", "{sha256:6}_{name}.{ext}", "2d7116_photo.jpg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveRename(tt.template, ctx())
			assert.Equal(t, tt.want, got)
		})
	}
}
