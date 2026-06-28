package cmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFileTracker(t *testing.T) {
	t.Parallel()

	t.Run("touch reports already-tracked on second call", func(t *testing.T) {
		t.Parallel()
		tr := newFileTracker()
		assert.False(t, tr.touch("/a", time.Now()))
		assert.True(t, tr.touch("/a", time.Now()))
	})

	t.Run("due returns stable files oldest first and leaves the rest", func(t *testing.T) {
		t.Parallel()
		tr := newFileTracker()
		now := time.Now()
		tr.touch("/recent", now.Add(-1*time.Second))
		tr.touch("/old2", now.Add(-8*time.Second))
		tr.touch("/old1", now.Add(-10*time.Second))

		assert.Equal(t, []string{"/old1", "/old2"}, tr.due(now, 5*time.Second))
		// the not-yet-stable file stays queued for a later tick
		assert.Nil(t, tr.due(now, 5*time.Second))
	})

	t.Run("a fresh event defers a previously-due file", func(t *testing.T) {
		t.Parallel()
		tr := newFileTracker()
		now := time.Now()
		tr.touch("/f", now.Add(-10*time.Second)) // old enough to move
		tr.touch("/f", now)                      // new event pushes it forward
		assert.Nil(t, tr.due(now, 5*time.Second))
	})

	t.Run("due removes the files it returns", func(t *testing.T) {
		t.Parallel()
		tr := newFileTracker()
		now := time.Now()
		tr.touch("/f", now.Add(-10*time.Second))
		assert.Equal(t, []string{"/f"}, tr.due(now, 5*time.Second))
		assert.Nil(t, tr.due(now, 5*time.Second))
	})
}
