package tokens

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSystemContext tests the initialization of system context variables to ensure they are populated correctly.
func TestSystemContext(t *testing.T) {
	t.Parallel()
	initSystemContext()
	assert.NotEmpty(t, systemHostname)
	assert.NotEmpty(t, systemUsername)
	assert.NotEmpty(t, systemOS)
}

// testStripDomain defines a structure for test cases of the stripDomain function,
// containing the name, the raw username, and the expected unqualified result.
type testStripDomain struct {
	name  string
	input string
	want  string
}

// testStripDomainTestCases covers Windows domain-qualified usernames and plain ones,
// ensuring no path separator survives into a {username} organize-by subdirectory.
var testStripDomainTestCases = []testStripDomain{
	{"windows domain backslash", `CORP\lucas`, "lucas"},
	{"forward slash", "corp/lucas", "lucas"},
	{"plain username", "lucas", "lucas"},
	{"empty", "", ""},
}

// TestStripDomain tests that stripDomain removes a leading DOMAIN\ or domain/ qualifier.
func TestStripDomain(t *testing.T) {
	t.Parallel()
	for _, tt := range testStripDomainTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, stripDomain(tt.input))
		})
	}
}
