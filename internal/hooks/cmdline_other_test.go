//go:build !windows

package hooks

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRawCmdLineLeavesTheCommandAlone(t *testing.T) {
	t.Parallel()
	c := exec.Command("sh", "-c", "true")
	rawCmdLine(c, "true")
	assert.Nil(t, c.SysProcAttr)
}
