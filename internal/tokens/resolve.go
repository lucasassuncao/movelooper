package tokens

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// buildStaticPairs returns key-value pairs for strings.NewReplacer covering all static tokens.
func buildStaticPairs(ctx TokenContext) []string {
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

// ResolveGroupBy resolves a group-by template string into a relative subdirectory
// path that should be appended to the category destination.
func ResolveGroupBy(template string, ctx TokenContext) string {
	if template == "" {
		return ""
	}
	name := strings.TrimSuffix(ctx.Info.Name(), filepath.Ext(ctx.Info.Name()))
	template = preProcessNameTrunc(template, name)
	r := strings.NewReplacer(buildStaticPairs(ctx)...)
	return filepath.FromSlash(r.Replace(template))
}

// ResolveRename applies a rename template to produce a destination filename.
// It supports the same tokens as ResolveGroupBy, plus {seq}, {seq:N}, {seq-alpha},
// {seq-roman}, {md5}, {md5:N}, and {sha256:N}.
// When template is empty, the original filename is returned unchanged.
// Path separators are stripped from the result so the output is always a plain filename.
func ResolveRename(template string, ctx TokenContext) string {
	if template == "" {
		return ctx.Info.Name()
	}

	template = preProcessHash(template, ctx.SourcePath)
	if ctx.DestDir != "" && hasSeqToken(template) {
		unlock := acquireSeqLock(ctx.DestDir)
		template = preProcessSeqAlpha(template, ctx.DestDir)
		template = preProcessSeqRoman(template, ctx.DestDir)
		template = preProcessSeq(template, ctx.DestDir)
		unlock()
	} else {
		template = preProcessSeqAlpha(template, ctx.DestDir)
		template = preProcessSeqRoman(template, ctx.DestDir)
		template = preProcessSeq(template, ctx.DestDir)
	}

	resolved := ResolveGroupBy(template, ctx)
	resolved = strings.ReplaceAll(resolved, string(os.PathSeparator), "_")
	resolved = strings.ReplaceAll(resolved, "/", "_")
	return resolved
}
