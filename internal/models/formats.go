package models

import (
	"path/filepath"
	"regexp"

	"github.com/lucasassuncao/movelooper/internal/tokens"
	"github.com/lucasassuncao/yedit/spec"
)

var (
	// FormatGlob validates that the value is a syntactically valid glob pattern.
	FormatGlob = spec.FormatCustom("glob", func(v string) bool {
		_, err := filepath.Match(v, "")
		return err == nil
	})

	// FormatRegex validates that the value is a valid RE2 regular expression.
	FormatRegex = spec.FormatCustom("regex", func(v string) bool {
		_, err := regexp.Compile(v)
		return err == nil
	})

	// FormatOrganizeByPattern validates organize-by token strings using the
	// same token set as the runtime resolver.
	FormatOrganizeByPattern = spec.FormatCustom("organize-by pattern", func(v string) bool {
		return tokens.ValidateTemplate(v) == nil
	})

	// FormatRenamePattern validates rename token strings using the same token
	// set as the runtime resolver.
	FormatRenamePattern = spec.FormatCustom("rename pattern", func(v string) bool {
		return tokens.ValidateTemplate(v) == nil
	})
)
