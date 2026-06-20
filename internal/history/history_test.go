package history

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestHistory creates a History backed by a temp directory.
func newTestHistory(t *testing.T, maxBatches int) *History {
	t.Helper()
	if maxBatches < 1 {
		maxBatches = defaultMaxBatches
	}
	return &History{
		path:       filepath.Join(t.TempDir(), "movelooper.json"),
		maxBatches: maxBatches,
		batchCount: make(map[string]int),
	}
}

func makeEntry(batchID string) Entry {
	return Entry{
		Source:      "/src/file.txt",
		Destination: "/dst/file.txt",
		Timestamp:   time.Now(),
		BatchID:     batchID,
	}
}

// testAdd defines the structure for test cases of the Add function,
// containing the entries to add, an error expectation flag, expected length, and a bad path flag.
type testAdd struct {
	name    string
	entries []Entry
	wantLen int
	wantErr bool
	badPath bool
}

// testAddTestCases defines a set of test cases for the Add function,
// covering single entry persistence and unwritable path errors.
var testAddTestCases = []testAdd{
	{
		name:    "persists single entry to disk",
		entries: []Entry{makeEntry("batch_1")},
		wantLen: 1,
	},
	{
		name:    "unwritable path returns error",
		entries: []Entry{makeEntry("batch_fail")},
		wantErr: true,
		badPath: true,
	},
}

// TestAdd tests the Add function to ensure it correctly persists entries and handles errors.
func TestAdd(t *testing.T) {
	t.Parallel()
	for _, tt := range testAddTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := newTestHistory(t, 10)
			if tt.badPath {
				h.path = t.TempDir()
			}

			var lastErr error
			for _, e := range tt.entries {
				lastErr = h.Add(e)
			}

			if tt.wantErr {
				assert.Error(t, lastErr)
				return
			}
			require.NoError(t, lastErr)

			h2 := &History{path: h.path, maxBatches: h.maxBatches}
			require.NoError(t, h2.load())
			assert.Len(t, h2.entries, tt.wantLen)
		})
	}
}

// testGetBatch defines the structure for test cases of the GetBatch function,
// containing batch IDs to add, the query batch ID, and the expected result length.
type testGetBatch struct {
	name    string
	add     []string
	query   string
	wantLen int
}

// testGetBatchTestCases defines a set of test cases for the GetBatch function,
// covering known batch, single entry batch, and unknown batch scenarios.
var testGetBatchTestCases = []testGetBatch{
	{"known batch returns entries", []string{"A", "A", "B"}, "A", 2},
	{"single entry batch", []string{"A", "A", "B"}, "B", 1},
	{"unknown batch returns empty", []string{"A"}, "nonexistent", 0},
}

// TestGetBatch tests the GetBatch function to ensure it correctly retrieves entries by batch ID.
func TestGetBatch(t *testing.T) {
	t.Parallel()
	for _, tt := range testGetBatchTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := newTestHistory(t, 10)
			for _, id := range tt.add {
				require.NoError(t, h.Add(makeEntry(id)))
			}
			assert.Len(t, h.GetBatch(tt.query), tt.wantLen)
		})
	}
}

// testGetLastBatchID defines the structure for test cases of the GetLastBatchID function,
// containing batch IDs to add, the expected batch ID, and an error expectation flag.
type testGetLastBatchID struct {
	name    string
	add     []string
	want    string
	wantErr bool
}

// testGetLastBatchIDTestCases defines a set of test cases for the GetLastBatchID function,
// covering populated history and empty history scenarios.
var testGetLastBatchIDTestCases = []testGetLastBatchID{
	{"returns last added batch", []string{"batch_1", "batch_2"}, "batch_2", false},
	{"empty history errors", nil, "", true},
}

// TestGetLastBatchID tests the GetLastBatchID function to ensure it correctly identifies the last batch.
func TestGetLastBatchID(t *testing.T) {
	t.Parallel()
	for _, tt := range testGetLastBatchIDTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := newTestHistory(t, 10)
			for _, id := range tt.add {
				require.NoError(t, h.Add(makeEntry(id)))
			}
			id, err := h.GetLastBatchID()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, id)
		})
	}
}

// testGetAllBatches defines the structure for test cases of the GetAllBatches function,
// containing batch IDs to add and a check function for assertions.
type testGetAllBatches struct {
	name  string
	add   []string
	check func(t *testing.T, batches []BatchSummary)
}

// testGetAllBatchesTestCases defines a set of test cases for the GetAllBatches function,
// covering empty history and ordered batches with counts.
var testGetAllBatchesTestCases = []testGetAllBatches{
	{
		name:  "empty history",
		check: func(t *testing.T, batches []BatchSummary) { assert.Empty(t, batches) },
	},
	{
		name: "ordered oldest first with counts",
		add:  []string{"batch_1", "batch_2", "batch_1"},
		check: func(t *testing.T, batches []BatchSummary) {
			require.Len(t, batches, 2)
			assert.Equal(t, "batch_1", batches[0].BatchID)
			assert.Equal(t, 2, batches[0].Count)
			assert.Equal(t, "batch_2", batches[1].BatchID)
			assert.Equal(t, 1, batches[1].Count)
		},
	},
}

// TestGetAllBatches tests the GetAllBatches function to ensure it returns batches in the correct order with counts.
func TestGetAllBatches(t *testing.T) {
	t.Parallel()
	for _, tt := range testGetAllBatchesTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := newTestHistory(t, 10)
			for _, id := range tt.add {
				require.NoError(t, h.Add(makeEntry(id)))
			}
			tt.check(t, h.GetAllBatches())
		})
	}
}

// testRemoveBatch defines the structure for test cases of the RemoveBatch function,
// containing batch IDs to add, the batch to remove, and a check function for assertions.
type testRemoveBatch struct {
	name   string
	add    []string
	remove string
	check  func(t *testing.T, h *History)
}

// testRemoveBatchTestCases defines a set of test cases for the RemoveBatch function,
// covering successful removal and non-existent batch scenarios.
var testRemoveBatchTestCases = []testRemoveBatch{
	{
		name:   "removes entries and persists to disk",
		add:    []string{"batch_1", "batch_2"},
		remove: "batch_1",
		check: func(t *testing.T, h *History) {
			assert.Empty(t, h.GetBatch("batch_1"))
			assert.Len(t, h.GetBatch("batch_2"), 1)

			h2 := &History{path: h.path, maxBatches: h.maxBatches}
			require.NoError(t, h2.load())
			for _, e := range h2.entries {
				assert.NotEqual(t, "batch_1", e.BatchID)
			}
		},
	},
	{
		name:   "non-existent batch no error",
		add:    []string{"batch_keep"},
		remove: "batch_ghost",
		check: func(t *testing.T, h *History) {
			assert.Len(t, h.GetBatch("batch_keep"), 1)
		},
	},
}

// TestRemoveBatch tests the RemoveBatch function to ensure it correctly removes entries and persists changes.
func TestRemoveBatch(t *testing.T) {
	t.Parallel()
	for _, tt := range testRemoveBatchTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := newTestHistory(t, 10)
			for _, id := range tt.add {
				require.NoError(t, h.Add(makeEntry(id)))
			}
			require.NoError(t, h.RemoveBatch(tt.remove))
			tt.check(t, h)
		})
	}
}

// testPrune defines the structure for test cases of the prune function,
// containing the max batch limit, batch IDs to add, and expected present/absent IDs.
type testPrune struct {
	name       string
	maxBatches int
	add        []string
	wantIDs    []string
	notWantIDs []string
}

// testPruneTestCases defines a set of test cases for the prune function,
// covering eviction of oldest batches and single batch under limit.
var testPruneTestCases = []testPrune{
	{
		name:       "evicts oldest when over limit",
		maxBatches: 2,
		add:        []string{"batch_1", "batch_2", "batch_3"},
		wantIDs:    []string{"batch_2", "batch_3"},
		notWantIDs: []string{"batch_1"},
	},
	{
		name:       "single batch under limit kept",
		maxBatches: 5,
		add:        []string{"batch_only"},
		wantIDs:    []string{"batch_only"},
	},
}

// TestPrune tests the prune function to ensure it correctly evicts oldest batches when the limit is exceeded.
func TestPrune(t *testing.T) {
	t.Parallel()
	for _, tt := range testPruneTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := newTestHistory(t, tt.maxBatches)
			for _, id := range tt.add {
				require.NoError(t, h.Add(makeEntry(id)))
			}
			batches := h.GetAllBatches()
			ids := make([]string, len(batches))
			for i, b := range batches {
				ids[i] = b.BatchID
			}
			for _, want := range tt.wantIDs {
				assert.Contains(t, ids, want)
			}
			for _, notWant := range tt.notWantIDs {
				assert.NotContains(t, ids, notWant)
			}
		})
	}
}

// testBatchID defines the structure for test cases of the NewBatchID and NewWatchBatchID functions,
// containing the function under test, expected prefix, and a uniqueness check flag.
type testBatchID struct {
	name      string
	fn        func() string
	prefix    string
	checkUniq bool
}

// testBatchIDTestCases defines a set of test cases for the batch ID generator functions,
// covering prefix validation and uniqueness across multiple calls.
var testBatchIDTestCases = []testBatchID{
	{"NewBatchID has prefix", NewBatchID, "batch_", false},
	{"NewWatchBatchID has prefix", NewWatchBatchID, "watch_", false},
	{"NewWatchBatchID unique per call", NewWatchBatchID, "watch_", true},
}

// TestBatchIDs tests the NewBatchID and NewWatchBatchID functions to ensure they produce correct prefixes and unique IDs.
func TestBatchIDs(t *testing.T) {
	t.Parallel()
	for _, tt := range testBatchIDTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.checkUniq {
				ids := make(map[string]bool)
				for range 50 {
					id := tt.fn()
					assert.False(t, ids[id], "collision: %s", id)
					ids[id] = true
				}
				return
			}
			assert.Contains(t, tt.fn(), tt.prefix)
		})
	}
}

// testLoad defines the structure for test cases of the load function,
// containing setup logic, expected entry count, error expectation, and a non-existent path flag.
type testLoad struct {
	name     string
	setup    func(t *testing.T, h *History)
	wantLen  int
	wantErr  bool
	notExist bool
}

// testLoadTestCases defines a set of test cases for the load function,
// covering valid JSON, corrupt JSON, and non-existent file scenarios.
var testLoadTestCases = []testLoad{
	{
		name: "valid json",
		setup: func(t *testing.T, h *History) {
			entries := []Entry{{Source: "/src/a.txt", Destination: "/dst/a.txt", Timestamp: time.Now(), BatchID: "batch_1"}}
			data, err := json.MarshalIndent(entries, "", "  ")
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(h.path, data, 0o644))
		},
		wantLen: 1,
	},
	{
		name: "corrupt ndjson lines are skipped gracefully",
		setup: func(t *testing.T, h *History) {
			require.NoError(t, os.WriteFile(h.path, []byte("not valid json {{{"), 0o644))
		},
		wantLen: 0,
	},
	{
		name:     "file not exist",
		notExist: true,
		wantErr:  true,
	},
}

// TestLoad tests the load function to ensure it correctly reads and parses history files.
func TestLoad(t *testing.T) {
	t.Parallel()
	for _, tt := range testLoadTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := newTestHistory(t, 10)
			if tt.notExist {
				h.path = filepath.Join(t.TempDir(), "nonexistent.json")
			} else if tt.setup != nil {
				tt.setup(t, h)
			}

			err := h.load()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, h.entries, tt.wantLen)
		})
	}
}

// testSave defines the structure for test cases of the save function,
// containing entries to persist and a check function for assertions on the written file.
type testSave struct {
	name    string
	entries []Entry
	check   func(t *testing.T, data []byte)
}

// testSaveTestCases defines a set of test cases for the save function,
// covering entries with data and empty entry arrays.
var testSaveTestCases = []testSave{
	{
		name:    "writes ndjson with entries",
		entries: []Entry{{Source: "/src/b.txt", Destination: "/dst/b.txt", Timestamp: time.Now(), BatchID: "batch_save"}},
		check: func(t *testing.T, data []byte) {
			dec := json.NewDecoder(bytes.NewReader(data))
			var loaded []Entry
			for dec.More() {
				var e Entry
				require.NoError(t, dec.Decode(&e))
				loaded = append(loaded, e)
			}
			require.Len(t, loaded, 1)
			assert.Equal(t, "batch_save", loaded[0].BatchID)
		},
	},
	{
		name:    "empty entries writes empty file",
		entries: []Entry{},
		check: func(t *testing.T, data []byte) {
			assert.Empty(t, bytes.TrimSpace(data))
		},
	},
}

// TestSave tests the save function to ensure it correctly serializes and writes history entries.
func TestSave(t *testing.T) {
	t.Parallel()
	for _, tt := range testSaveTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := newTestHistory(t, 10)
			h.entries = tt.entries
			require.NoError(t, h.save())

			data, err := os.ReadFile(h.path)
			require.NoError(t, err)
			tt.check(t, data)
		})
	}
}

// TestAdd_ConcurrentSafe tests that concurrent Add calls are safe and all entries are persisted.
func TestAdd_ConcurrentSafe(t *testing.T) {
	t.Parallel()
	h := newTestHistory(t, 100)
	done := make(chan struct{})
	for range 10 {
		go func() {
			_ = h.Add(makeEntry("batch_concurrent"))
			done <- struct{}{}
		}()
	}
	for range 10 {
		<-done
	}
	assert.Len(t, h.GetBatch("batch_concurrent"), 10)
}

// TestHistory_LoadAndAddRoundTrip tests that entries added to one History instance
// are correctly loaded by a second instance reading the same file.
func TestHistory_LoadAndAddRoundTrip(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "movelooper.json")
	h := &History{path: path, maxBatches: 10}

	require.NoError(t, h.Add(makeEntry("batch_1")))
	require.NoError(t, h.Add(makeEntry("batch_2")))

	h2 := &History{path: path, maxBatches: 10}
	require.NoError(t, h2.load())
	assert.Len(t, h2.entries, 2)
}

// testRemoveCategoryFromBatch defines the structure for test cases of the RemoveCategoryFromBatch function,
// containing setup logic, the target batch ID, categories to remove, and expected state assertions.
type testRemoveCategoryFromBatch struct {
	name       string
	setup      func(h *History)
	batchID    string
	categories []string
	wantLen    int
	wantBatch  bool
}

// testRemoveCategoryFromBatchTestCases defines a set of test cases for the RemoveCategoryFromBatch function,
// covering partial removal, full removal, empty category, and unknown batch ID scenarios.
var testRemoveCategoryFromBatchTestCases = []testRemoveCategoryFromBatch{
	{
		name: "removes matching entries keeps others",
		setup: func(h *History) {
			_ = h.Add(Entry{Source: "/a", Destination: "/b", BatchID: "b1", Category: "images"})
			_ = h.Add(Entry{Source: "/c", Destination: "/d", BatchID: "b1", Category: "docs"})
		},
		batchID:    "b1",
		categories: []string{"images"},
		wantLen:    1,
		wantBatch:  true,
	},
	{
		name: "batch becomes empty no entries remain",
		setup: func(h *History) {
			_ = h.Add(Entry{Source: "/a", Destination: "/b", BatchID: "b1", Category: "images"})
		},
		batchID:    "b1",
		categories: []string{"images"},
		wantLen:    0,
		wantBatch:  false,
	},
	{
		name: "entry with empty category is not removed",
		setup: func(h *History) {
			_ = h.Add(Entry{Source: "/a", Destination: "/b", BatchID: "b1", Category: ""})
		},
		batchID:    "b1",
		categories: []string{"images"},
		wantLen:    1,
		wantBatch:  true,
	},
	{
		name: "unknown batchID no-op no error",
		setup: func(h *History) {
			_ = h.Add(Entry{Source: "/a", Destination: "/b", BatchID: "b1", Category: "images"})
		},
		batchID:    "b99",
		categories: []string{"images"},
		wantLen:    1,
		wantBatch:  false,
	},
}

// TestRemoveCategoryFromBatch tests the RemoveCategoryFromBatch function to ensure it correctly
// removes entries by category while leaving other entries intact.
func TestRemoveCategoryFromBatch(t *testing.T) {
	t.Parallel()
	for _, tt := range testRemoveCategoryFromBatchTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := newTestHistory(t, 10)
			tt.setup(h)
			_, err := h.RemoveCategoryFromBatch(tt.batchID, tt.categories)
			require.NoError(t, err)
			assert.Len(t, h.entries, tt.wantLen)
			hasBatch := false
			for _, e := range h.entries {
				if e.BatchID == tt.batchID {
					hasBatch = true
					break
				}
			}
			assert.Equal(t, tt.wantBatch, hasBatch)
		})
	}
}
