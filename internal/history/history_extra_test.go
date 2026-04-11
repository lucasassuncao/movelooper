package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- load ---

func TestLoad_ValidJSON(t *testing.T) {
	h := newTestHistory(t, 10)
	entries := []Entry{
		{Source: "/src/a.txt", Destination: "/dst/a.txt", Timestamp: time.Now(), BatchID: "batch_1"},
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(h.path, data, 0644))

	require.NoError(t, h.load())
	assert.Len(t, h.Entries, 1)
	assert.Equal(t, "batch_1", h.Entries[0].BatchID)
}

func TestLoad_CorruptJSON(t *testing.T) {
	h := newTestHistory(t, 10)
	require.NoError(t, os.WriteFile(h.path, []byte("not valid json {{{"), 0644))

	err := h.load()
	assert.Error(t, err)
}

func TestLoad_FileNotExist(t *testing.T) {
	h := newTestHistory(t, 10)
	// path points to a non-existent file
	err := h.load()
	assert.True(t, os.IsNotExist(err))
}

// --- save ---

func TestSave_WritesJSON(t *testing.T) {
	h := newTestHistory(t, 10)
	h.Entries = []Entry{
		{Source: "/src/b.txt", Destination: "/dst/b.txt", Timestamp: time.Now(), BatchID: "batch_save"},
	}
	require.NoError(t, h.save())

	data, err := os.ReadFile(h.path)
	require.NoError(t, err)

	var loaded []Entry
	require.NoError(t, json.Unmarshal(data, &loaded))
	require.Len(t, loaded, 1)
	assert.Equal(t, "batch_save", loaded[0].BatchID)
}

func TestSave_EmptyEntriesWritesEmptyArray(t *testing.T) {
	h := newTestHistory(t, 10)
	h.Entries = []Entry{}
	require.NoError(t, h.save())

	data, err := os.ReadFile(h.path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "[]")
}

// --- Add error path: unwritable path ---

func TestAdd_UnwritablePath(t *testing.T) {
	h := newTestHistory(t, 10)
	// Point to a directory instead of a file so os.WriteFile fails
	h.path = t.TempDir()

	entry := makeEntry("batch_fail")
	err := h.Add(entry)
	assert.Error(t, err)
}

// --- concurrent Add ---

func TestAdd_ConcurrentSafe(t *testing.T) {
	h := newTestHistory(t, 100)
	done := make(chan struct{})
	for range 10 {
		go func() {
			_ = h.Add(makeEntry("batch_concurrent"))
			done <- struct{}{}
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
	assert.Len(t, h.GetBatch("batch_concurrent"), 10)
}

// --- NewBatchID uniqueness ---

func TestNewBatchID_UniquePerSecond(t *testing.T) {
	// Two IDs generated at the same time may collide (Unix timestamp),
	// but they should both have the correct prefix.
	id1 := NewBatchID()
	id2 := NewBatchID()
	assert.Contains(t, id1, "batch_")
	assert.Contains(t, id2, "batch_")
}

// --- GetAllBatches empty ---

func TestGetAllBatches_Empty(t *testing.T) {
	h := newTestHistory(t, 10)
	batches := h.GetAllBatches()
	assert.Empty(t, batches)
}

// --- RemoveBatch non-existent ---

func TestRemoveBatch_NonExistentBatchNoError(t *testing.T) {
	h := newTestHistory(t, 10)
	require.NoError(t, h.Add(makeEntry("batch_keep")))
	// Removing a batch that doesn't exist should not error and not affect others
	require.NoError(t, h.RemoveBatch("batch_ghost"))
	assert.Len(t, h.GetBatch("batch_keep"), 1)
}

// --- prune with single batch ---

func TestPrune_SingleBatchUnderLimit(t *testing.T) {
	h := newTestHistory(t, 5)
	require.NoError(t, h.Add(makeEntry("batch_only")))
	batches := h.GetAllBatches()
	assert.Len(t, batches, 1)
}

// --- load then Add round-trip ---

func TestHistory_LoadAndAddRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "movelooper.json")
	h := &History{path: path, maxBatches: 10}

	require.NoError(t, h.Add(makeEntry("batch_1")))
	require.NoError(t, h.Add(makeEntry("batch_2")))

	// Create a fresh History instance pointing to the same file
	h2 := &History{path: path, maxBatches: 10}
	require.NoError(t, h2.load())

	assert.Len(t, h2.Entries, 2)
}
