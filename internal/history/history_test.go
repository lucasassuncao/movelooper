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

// newTestHistory creates a History backed by a temp directory, bypassing
// os.Executable() so tests are hermetic.
func newTestHistory(t *testing.T, maxBatches int) *History {
	t.Helper()
	if maxBatches < 1 {
		maxBatches = defaultMaxBatches
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "movelooper.json")
	h := &History{
		path:       path,
		maxBatches: maxBatches,
	}
	return h
}

func makeEntry(batchID string) Entry {
	return Entry{
		Source:      "/src/file.txt",
		Destination: "/dst/file.txt",
		Timestamp:   time.Now(),
		BatchID:     batchID,
	}
}

// --- Add / GetBatch ---

func TestAdd_PersistsToDisk(t *testing.T) {
	h := newTestHistory(t, 10)
	entry := makeEntry("batch_1")

	require.NoError(t, h.Add(entry))

	data, err := os.ReadFile(h.path)
	require.NoError(t, err)

	var loaded []Entry
	require.NoError(t, json.Unmarshal(data, &loaded))
	require.Len(t, loaded, 1)
	assert.Equal(t, entry.BatchID, loaded[0].BatchID)
}

func TestGetBatch_ReturnsCorrectEntries(t *testing.T) {
	h := newTestHistory(t, 10)
	require.NoError(t, h.Add(makeEntry("batch_A")))
	require.NoError(t, h.Add(makeEntry("batch_A")))
	require.NoError(t, h.Add(makeEntry("batch_B")))

	batchA := h.GetBatch("batch_A")
	assert.Len(t, batchA, 2)

	batchB := h.GetBatch("batch_B")
	assert.Len(t, batchB, 1)
}

func TestGetBatch_UnknownIDReturnsEmpty(t *testing.T) {
	h := newTestHistory(t, 10)
	assert.Empty(t, h.GetBatch("nonexistent"))
}

// --- GetLastBatchID ---

func TestGetLastBatchID_ReturnsLast(t *testing.T) {
	h := newTestHistory(t, 10)
	require.NoError(t, h.Add(makeEntry("batch_1")))
	require.NoError(t, h.Add(makeEntry("batch_2")))

	id, err := h.GetLastBatchID()
	require.NoError(t, err)
	assert.Equal(t, "batch_2", id)
}

func TestGetLastBatchID_EmptyHistoryErrors(t *testing.T) {
	h := newTestHistory(t, 10)
	_, err := h.GetLastBatchID()
	assert.Error(t, err)
}

// --- GetAllBatches ---

func TestGetAllBatches_OrderedOldestFirst(t *testing.T) {
	h := newTestHistory(t, 10)
	require.NoError(t, h.Add(makeEntry("batch_1")))
	require.NoError(t, h.Add(makeEntry("batch_2")))
	require.NoError(t, h.Add(makeEntry("batch_1"))) // second entry for batch_1

	batches := h.GetAllBatches()
	require.Len(t, batches, 2)
	assert.Equal(t, "batch_1", batches[0].BatchID)
	assert.Equal(t, 2, batches[0].Count)
	assert.Equal(t, "batch_2", batches[1].BatchID)
	assert.Equal(t, 1, batches[1].Count)
}

// --- RemoveBatch ---

func TestRemoveBatch_RemovesEntries(t *testing.T) {
	h := newTestHistory(t, 10)
	require.NoError(t, h.Add(makeEntry("batch_1")))
	require.NoError(t, h.Add(makeEntry("batch_2")))

	require.NoError(t, h.RemoveBatch("batch_1"))
	assert.Empty(t, h.GetBatch("batch_1"))
	assert.Len(t, h.GetBatch("batch_2"), 1)
}

func TestRemoveBatch_PersistsRemoval(t *testing.T) {
	h := newTestHistory(t, 10)
	require.NoError(t, h.Add(makeEntry("batch_del")))
	require.NoError(t, h.RemoveBatch("batch_del"))

	data, err := os.ReadFile(h.path)
	require.NoError(t, err)
	var loaded []Entry
	require.NoError(t, json.Unmarshal(data, &loaded))
	for _, e := range loaded {
		assert.NotEqual(t, "batch_del", e.BatchID)
	}
}

// --- prune ---

func TestPrune_KeepsMaxBatches(t *testing.T) {
	h := newTestHistory(t, 2)

	require.NoError(t, h.Add(makeEntry("batch_1")))
	require.NoError(t, h.Add(makeEntry("batch_2")))
	require.NoError(t, h.Add(makeEntry("batch_3")))

	batches := h.GetAllBatches()
	assert.Len(t, batches, 2)
	ids := make([]string, len(batches))
	for i, b := range batches {
		ids[i] = b.BatchID
	}
	assert.NotContains(t, ids, "batch_1")
	assert.Contains(t, ids, "batch_2")
	assert.Contains(t, ids, "batch_3")
}

// --- NewBatchID / NewWatchBatchID ---

func TestNewBatchID_HasPrefix(t *testing.T) {
	id := NewBatchID()
	assert.Contains(t, id, "batch_")
}

func TestNewWatchBatchID_HasPrefix(t *testing.T) {
	id := NewWatchBatchID()
	assert.Contains(t, id, "watch_")
}

func TestNewWatchBatchID_UniquePerCall(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 50; i++ {
		id := NewWatchBatchID()
		assert.False(t, ids[id], "collision on id %s", id)
		ids[id] = true
	}
}
