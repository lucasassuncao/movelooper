package tokens

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// knownTokens is the complete set of fixed tokens recognised by both organize-by and rename templates.
var knownTokens = map[string]bool{
	// identification
	"{name}":        true,
	"{ext}":         true,
	"{ext-upper}":   true,
	"{ext-lower}":   true,
	"{ext-reverse}": true,
	// name transforms
	"{name-slug}":     true,
	"{name-snake}":    true,
	"{name-upper}":    true,
	"{name-lower}":    true,
	"{name-alpha}":    true,
	"{name-ascii}":    true,
	"{name-initials}": true,
	"{name-reverse}":  true,
	// modification date
	"{mod-year}":    true,
	"{mod-month}":   true,
	"{mod-day}":     true,
	"{mod-date}":    true,
	"{mod-weekday}": true,
	// creation date
	"{created-year}":  true,
	"{created-month}": true,
	"{created-day}":   true,
	"{created-date}":  true,
	// run date
	"{year}":    true,
	"{month}":   true,
	"{day}":     true,
	"{date}":    true,
	"{weekday}": true,
	// run time
	"{hour}":      true,
	"{minute}":    true,
	"{second}":    true,
	"{timestamp}": true,
	// size
	"{size-range}": true,
	// category
	"{category}": true,
	// system context
	"{hostname}": true,
	"{username}": true,
	"{os}":       true,
	// advanced sequence (rename only)
	"{seq-alpha}": true,
	"{seq-roman}": true,
	// hash (rename only)
	"{md5}": true,
}

var tokenPattern = regexp.MustCompile(`\{[^}]+\}`)
var paramPattern = regexp.MustCompile(`^\d+$`)

// ValidateTemplate returns an error if the template contains any unrecognised
// or malformed {token}.
func ValidateTemplate(template string) error {
	for _, tok := range tokenPattern.FindAllString(template, -1) {
		switch {
		case knownTokens[tok]:
			// known fixed token
		case tok == "{seq}":
			// ok
		case strings.HasPrefix(tok, "{seq:") && strings.HasSuffix(tok, "}"):
			if err := validateIntParam(tok, "seq", 20); err != nil {
				return err
			}
		case strings.HasPrefix(tok, "{name-trunc:") && strings.HasSuffix(tok, "}"):
			if err := validateIntParam(tok, "name-trunc", 255); err != nil {
				return err
			}
		case strings.HasPrefix(tok, "{md5:") && strings.HasSuffix(tok, "}"):
			if err := validateIntParam(tok, "md5", 32); err != nil {
				return err
			}
		case strings.HasPrefix(tok, "{sha256:") && strings.HasSuffix(tok, "}"):
			if err := validateIntParam(tok, "sha256", 64); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown token %q in template", tok)
		}
	}
	return nil
}

// validateIntParam checks that a parametric token like {family:N} has a valid integer N.
func validateIntParam(tok, family string, max int) error {
	prefix := "{" + family + ":"
	param := tok[len(prefix) : len(tok)-1]
	if param == "" {
		return fmt.Errorf("token %q: missing N value", tok)
	}
	if !paramPattern.MatchString(param) {
		return fmt.Errorf("token %q: N must be a positive integer", tok)
	}
	n, _ := strconv.Atoi(param)
	if n < 1 || n > max {
		return fmt.Errorf("token %q: N must be between 1 and %d", tok, max)
	}
	return nil
}
