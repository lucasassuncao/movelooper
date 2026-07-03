package tokens

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/lucasassuncao/movelooper/internal/content"
)

// buildStaticPairs returns key-value pairs for strings.NewReplacer covering all static tokens.
func buildStaticPairs(ctx *TokenContext) []string {
	initSystemContext()

	modTime := ctx.Info.ModTime()
	createdTime := getBirthTime(ctx.Info)
	rawExt := strings.TrimPrefix(filepath.Ext(ctx.Info.Name()), ".")
	name := strings.TrimSuffix(ctx.Info.Name(), filepath.Ext(ctx.Info.Name()))

	return []string{
		// identification
		"{name}", name,
		"{ext}", strings.ToLower(rawExt),
		"{ext-upper}", strings.ToUpper(rawExt),
		"{ext-lower}", strings.ToLower(rawExt),
		"{ext-reverse}", reverseString(strings.ToLower(rawExt)),
		// name transforms
		"{name-slug}", nameSlug(name),
		"{name-snake}", nameSnake(name),
		"{name-upper}", strings.ToUpper(name),
		"{name-lower}", strings.ToLower(name),
		"{name-alpha}", nameAlpha(name),
		"{name-ascii}", nameASCII(name),
		"{name-initials}", nameInitials(name),
		"{name-reverse}", nameReverse(name),
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
		"{year}", ctx.Now.Format("2006"),
		"{month}", ctx.Now.Format("01"),
		"{day}", ctx.Now.Format("02"),
		"{date}", ctx.Now.Format("2006-01-02"),
		"{weekday}", ctx.Now.Weekday().String(),
		// run time
		"{hour}", ctx.Now.Format("15"),
		"{minute}", ctx.Now.Format("04"),
		"{second}", ctx.Now.Format("05"),
		"{timestamp}", ctx.Now.Format("20060102-150405"),
		// size
		"{size-range}", fileSizeRange(ctx.Info.Size()),
		// category
		"{category}", ctx.CategoryName,
		// system context
		"{hostname}", systemHostname,
		"{username}", systemUsername,
		"{os}", systemOS,
	}
}

var nameTruncToken = regexp.MustCompile(`\{name-trunc:(\d+)\}`)

func preProcessNameTrunc(template, name string) string {
	return nameTruncToken.ReplaceAllStringFunc(template, func(tok string) string {
		m := nameTruncToken.FindStringSubmatch(tok)
		n, _ := strconv.Atoi(m[1])
		rr := []rune(name)
		if n >= len(rr) {
			return name
		}
		return string(rr[:n])
	})
}

// staticReplacer lazily builds and caches the strings.Replacer for static tokens.
func (ctx *TokenContext) staticReplacer() *strings.Replacer {
	if ctx.replacer == nil {
		ctx.replacer = strings.NewReplacer(buildStaticPairs(ctx)...)
	}
	return ctx.replacer
}

// ResolveGroupBy resolves a group-by template string into a relative subdirectory
// path that should be appended to the category destination.
func ResolveGroupBy(template string, ctx *TokenContext) string {
	if template == "" {
		return ""
	}
	name := strings.TrimSuffix(ctx.Info.Name(), filepath.Ext(ctx.Info.Name()))
	template = preProcessNameTrunc(template, name)
	template = preProcessMime(template, ctx.SourcePath)
	return filepath.FromSlash(ctx.staticReplacer().Replace(template))
}

// preProcessMime resolves the {mime}, {mime-type}, and {mime-ext} tokens by
// detecting the file's real type. It is a no-op unless the template references a
// mime token, so files are only read when MIME is actually used. Detection
// errors fall back to application/octet-stream. Unlike seq/hash, MIME resolves
// in dry-run too: it is read-only and the preview value is showing the real
// destination.
func preProcessMime(template, sourcePath string) string {
	if !strings.Contains(template, "{mime") {
		return template
	}
	full, top, ext := "application/octet-stream", "application", "bin"
	if info, err := content.Detect(sourcePath); err == nil {
		full = info.Full
		if info.Type != "" {
			top = info.Type
		}
		if info.Ext != "" {
			ext = info.Ext
		}
	}
	// {mime-type} and {mime-ext} first: {mime} is their common prefix.
	return strings.NewReplacer(
		"{mime-type}", top,
		"{mime-ext}", ext,
		"{mime}", full,
	).Replace(template)
}

// ResolveRename applies a rename template to produce a destination filename.
// It supports the same tokens as ResolveGroupBy, plus {seq}, {seq:N}, {seq-alpha},
// {seq-roman}, {md5}, {md5:N}, and {sha256:N}.
// When template is empty, the original filename is returned unchanged.
// Path separators are stripped from the result so the output is always a plain filename.
func ResolveRename(template string, ctx *TokenContext) string {
	if template == "" {
		return ctx.Info.Name()
	}

	// In dry-run the seq and hash tokens are left literal as placeholders: they
	// must not read the source file (hashing) or scan the destination directory
	// (which does not exist yet), keeping the preview strictly non-mutating.
	if !ctx.DryRun {
		template = preProcessHash(template, ctx.SourcePath)
		if ctx.DestDir != "" && hasSeqToken(template) {
			unlock := acquireSeqLock(ctx.DestDir)
			template = preProcessSeqAlpha(template, ctx.DestDir, ctx.SeqAlloc)
			template = preProcessSeqRoman(template, ctx.DestDir, ctx.SeqAlloc)
			template = preProcessSeq(template, ctx.DestDir, ctx.SeqAlloc)
			unlock()
		} else {
			template = preProcessSeqAlpha(template, ctx.DestDir, ctx.SeqAlloc)
			template = preProcessSeqRoman(template, ctx.DestDir, ctx.SeqAlloc)
			template = preProcessSeq(template, ctx.DestDir, ctx.SeqAlloc)
		}
	}

	resolved := ResolveGroupBy(template, ctx)
	resolved = strings.ReplaceAll(resolved, string(os.PathSeparator), "_")
	resolved = strings.ReplaceAll(resolved, "/", "_")
	return resolved
}

// ResolveArchiveName resolves an archive filename template using only tokens
// that do not depend on a specific file: category, run date/time, and system
// context. It cannot use file tokens ({name}, {ext}, {mod-*}), sequence, or hash
// tokens, which need a concrete file or destination directory. Unknown tokens are
// left as-is; path separators in the result are replaced with underscores so the
// output is always a plain filename. An empty template returns the category name.
func ResolveArchiveName(template, category string, now time.Time) string {
	if template == "" {
		return category
	}
	initSystemContext()
	resolved := strings.NewReplacer(archiveNamePairs(category, now)...).Replace(template)
	resolved = strings.ReplaceAll(resolved, string(os.PathSeparator), "_")
	resolved = strings.ReplaceAll(resolved, "/", "_")
	return resolved
}

func archiveNamePairs(category string, now time.Time) []string {
	return []string{
		"{category}", category,
		"{year}", now.Format("2006"),
		"{month}", now.Format("01"),
		"{day}", now.Format("02"),
		"{date}", now.Format("2006-01-02"),
		"{weekday}", now.Weekday().String(),
		"{hour}", now.Format("15"),
		"{minute}", now.Format("04"),
		"{second}", now.Format("05"),
		"{timestamp}", now.Format("20060102-150405"),
		"{hostname}", systemHostname,
		"{username}", systemUsername,
		"{os}", systemOS,
	}
}
