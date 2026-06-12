package models

import (
	"path/filepath"
	"regexp"

	"github.com/lucasassuncao/yedit/editor"
)

var (
	// FormatGlob validates that the value is a syntactically valid glob pattern.
	FormatGlob = editor.FormatCustom("glob", func(v string) bool {
		_, err := filepath.Match(v, "")
		return err == nil
	})

	// FormatRegex validates that the value is a valid RE2 regular expression.
	FormatRegex = editor.FormatCustom("regex", func(v string) bool {
		_, err := regexp.Compile(v)
		return err == nil
	})

	// FormatOrganizeByPattern validates organize-by token strings.
	// Valid tokens: {ext}, {year}, {month}, {day}.
	FormatOrganizeByPattern = editor.FormatCustom("organize-by pattern", func(v string) bool {
		return validTokens(v, organizeByTokens)
	})

	// FormatRenamePattern validates rename token strings.
	// Valid tokens: {name}, {ext}, {year}, {month}, {day}, {hour}, {min}, {sec}, {seq}, {hash}.
	FormatRenamePattern = editor.FormatCustom("rename pattern", func(v string) bool {
		return validTokens(v, renameTokens)
	})
)

var (
	organizeByTokens = map[string]bool{
		"ext": true, "year": true, "month": true, "day": true,
	}
	renameTokens = map[string]bool{
		"name": true, "ext": true, "year": true, "month": true, "day": true,
		"hour": true, "min": true, "sec": true, "seq": true, "hash": true,
	}
	tokenRe       = regexp.MustCompile(`\{([^{}]+)\}`)
	unclosedBrace = regexp.MustCompile(`\{[^}]*$`)
)

func validTokens(v string, allowed map[string]bool) bool {
	if unclosedBrace.MatchString(v) {
		return false
	}
	for _, m := range tokenRe.FindAllStringSubmatch(v, -1) {
		if !allowed[m[1]] {
			return false
		}
	}
	return true
}
