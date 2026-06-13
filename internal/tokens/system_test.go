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
