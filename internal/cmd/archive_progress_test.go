package cmd

import (
	"bytes"
	"testing"

	"github.com/lucasassuncao/movelooper/internal/logger"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/pterm/pterm"
	"github.com/stretchr/testify/assert"
)

// A bar drawn into JSON logs or a pipe would corrupt them, so both get nil.
func TestNewArchiveProgress_NilOutsideAnInteractiveTerminal(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	structured := &models.Movelooper{Logger: logger.NewSlog(&buf, "info", false)}
	assert.False(t, isInteractiveTerminal(structured))
	assert.Nil(t, newArchiveProgress(structured))

	// go test captures stdout, so a pretty logger still is not on a terminal
	pretty := &models.Movelooper{Logger: pterm.DefaultLogger.WithWriter(&buf)}
	assert.False(t, isInteractiveTerminal(pretty))
	assert.Nil(t, newArchiveProgress(pretty))
}
