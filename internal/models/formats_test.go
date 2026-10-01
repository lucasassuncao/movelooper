package models

import (
	"testing"

	"github.com/lucasassuncao/yedit/spec"
	"github.com/stretchr/testify/assert"
)

func TestFormats(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		format spec.Format
		valid  string
		bad    string
	}{
		{"glob", FormatGlob, "*.jpg", "[unclosed"},
		{"regex", FormatRegex, `^IMG_\d+$`, "(unclosed"},
		{"organize-by", FormatOrganizeByPattern, "{category}/{year}", "{unknown-token}"},
		{"rename", FormatRenamePattern, "{name}_{seq}", "{unknown-token}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.True(t, tt.format.Matches(tt.valid), "%q should be valid", tt.valid)
			assert.False(t, tt.format.Matches(tt.bad), "%q should be rejected", tt.bad)
		})
	}
}
