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

// newTestHistory creates a History backed by a temp directory.
func newTestHistory(t *testing.T, maxBatches int) *History {
	t.Helper()
	if maxBatches < 1 {
		maxBatches = defaultMaxBatches
	}
	return &History{
		path:       filepath.Join(t.TempDir(), "movelooper.json"),
		maxBatches: maxBatches,
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

// --- Add ---

func TestAdd(t *testing.T) {
	tests := []struct {
		name    string
		entries []Entry
		wantLen int
		wantErr bool
		badPath bool
	}{
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHistory(t, 10)
			if tt.badPath {
				h.path = t.TempDir() // directory, not a file
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

			data, err := os.ReadFile(h.path)
			require.NoError(t, err)
			var loaded []Entry
			require.NoError(t, json.Unmarshal(data, &loaded))
			assert.Len(t, loaded, tt.wantLen)
		})
	}
}

// --- GetBatch ---

func TestGetBatch(t *testing.T) {
	tests := []struct {
		name    string
		add     []string // batchIDs to add
		query   string
		wantLen int
	}{
		{"known batch returns entries", []string{"A", "A", "B"}, "A", 2},
		{"single entry batch", []string{"A", "A", "B"}, "B", 1},
		{"unknown batch returns empty", []string{"A"}, "nonexistent", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHistory(t, 10)
			for _, id := range tt.add {
				require.NoError(t, h.Add(makeEntry(id)))
			}
			assert.Len(t, h.GetBatch(tt.query), tt.wantLen)
		})
	}
}

// --- GetLastBatchID ---

func TestGetLastBatchID(t *testing.T) {
	tests := []struct {
		name    string
		add     []string
		want    string
		wantErr bool
	}{
		{"returns last added batch", []string{"batch_1", "batch_2"}, "batch_2", false},
		{"empty history errors", nil, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

// --- GetAllBatches ---

func TestGetAllBatches(t *testing.T) {
	tests := []struct {
		name  string
		add   []string
		check func(t *testing.T, batches []BatchSummary)
	}{
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHistory(t, 10)
			for _, id := range tt.add {
				require.NoError(t, h.Add(makeEntry(id)))
			}
			tt.check(t, h.GetAllBatches())
		})
	}
}

// --- RemoveBatch ---

func TestRemoveBatch(t *testing.T) {
	tests := []struct {
		name   string
		add    []string
		remove string
		check  func(t *testing.T, h *History)
	}{
		{
			name:   "removes entries and persists to disk",
			add:    []string{"batch_1", "batch_2"},
			remove: "batch_1",
			check: func(t *testing.T, h *History) {
				assert.Empty(t, h.GetBatch("batch_1"))
				assert.Len(t, h.GetBatch("batch_2"), 1)

				data, err := os.ReadFile(h.path)
				require.NoError(t, err)
				var loaded []Entry
				require.NoError(t, json.Unmarshal(data, &loaded))
				for _, e := range loaded {
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHistory(t, 10)
			for _, id := range tt.add {
				require.NoError(t, h.Add(makeEntry(id)))
			}
			require.NoError(t, h.RemoveBatch(tt.remove))
			tt.check(t, h)
		})
	}
}

// --- prune ---

func TestPrune(t *testing.T) {
	tests := []struct {
		name       string
		maxBatches int
		add        []string
		wantIDs    []string
		notWantIDs []string
	}{
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

// --- NewBatchID / NewWatchBatchID ---

func TestBatchIDs(t *testing.T) {
	tests := []struct {
		name      string
		fn        func() string
		prefix    string
		checkUniq bool
	}{
		{"NewBatchID has prefix", NewBatchID, "batch_", false},
		{"NewWatchBatchID has prefix", NewWatchBatchID, "watch_", false},
		{"NewWatchBatchID unique per call", NewWatchBatchID, "watch_", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

// --- load ---

func TestLoad(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T, h *History)
		wantLen  int
		wantErr  bool
		notExist bool
	}{
		{
			name: "valid json",
			setup: func(t *testing.T, h *History) {
				entries := []Entry{{Source: "/src/a.txt", Destination: "/dst/a.txt", Timestamp: time.Now(), BatchID: "batch_1"}}
				data, err := json.MarshalIndent(entries, "", "  ")
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(h.path, data, 0644))
			},
			wantLen: 1,
		},
		{
			name: "corrupt json",
			setup: func(t *testing.T, h *History) {
				require.NoError(t, os.WriteFile(h.path, []byte("not valid json {{{"), 0644))
			},
			wantErr: true,
		},
		{
			name:     "file not exist",
			notExist: true,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			assert.Len(t, h.Entries, tt.wantLen)
		})
	}
}

// --- save ---

func TestSave(t *testing.T) {
	tests := []struct {
		name    string
		entries []Entry
		check   func(t *testing.T, data []byte)
	}{
		{
			name:    "writes json with entries",
			entries: []Entry{{Source: "/src/b.txt", Destination: "/dst/b.txt", Timestamp: time.Now(), BatchID: "batch_save"}},
			check: func(t *testing.T, data []byte) {
				var loaded []Entry
				require.NoError(t, json.Unmarshal(data, &loaded))
				require.Len(t, loaded, 1)
				assert.Equal(t, "batch_save", loaded[0].BatchID)
			},
		},
		{
			name:    "empty entries writes empty array",
			entries: []Entry{},
			check: func(t *testing.T, data []byte) {
				assert.Contains(t, string(data), "[]")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHistory(t, 10)
			h.Entries = tt.entries
			require.NoError(t, h.save())

			data, err := os.ReadFile(h.path)
			require.NoError(t, err)
			tt.check(t, data)
		})
	}
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
	for range 10 {
		<-done
	}
	assert.Len(t, h.GetBatch("batch_concurrent"), 10)
}

// --- load then Add round-trip ---

func TestHistory_LoadAndAddRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "movelooper.json")
	h := &History{path: path, maxBatches: 10}

	require.NoError(t, h.Add(makeEntry("batch_1")))
	require.NoError(t, h.Add(makeEntry("batch_2")))

	h2 := &History{path: path, maxBatches: 10}
	require.NoError(t, h2.load())
	assert.Len(t, h2.Entries, 2)
}

// --- RemoveCategoryFromBatch ---

func TestRemoveCategoryFromBatch(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(h *History)
		batchID    string
		categories []string
		wantLen    int
		wantBatch  bool
	}{
		{
			name: "removes matching entries, keeps others",
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
			name: "batch becomes empty — no entries remain for that batch",
			setup: func(h *History) {
				_ = h.Add(Entry{Source: "/a", Destination: "/b", BatchID: "b1", Category: "images"})
			},
			batchID:    "b1",
			categories: []string{"images"},
			wantLen:    0,
			wantBatch:  false,
		},
		{
			name: "entry with empty Category is not removed",
			setup: func(h *History) {
				_ = h.Add(Entry{Source: "/a", Destination: "/b", BatchID: "b1", Category: ""})
			},
			batchID:    "b1",
			categories: []string{"images"},
			wantLen:    1,
			wantBatch:  true,
		},
		{
			name: "unknown batchID — no-op, no error",
			setup: func(h *History) {
				_ = h.Add(Entry{Source: "/a", Destination: "/b", BatchID: "b1", Category: "images"})
			},
			batchID:    "b99",
			categories: []string{"images"},
			wantLen:    1,     // b1 entry untouched
			wantBatch:  false, // b99 was never present
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHistory(t, 10)
			tt.setup(h)
			require.NoError(t, h.RemoveCategoryFromBatch(tt.batchID, tt.categories))
			assert.Len(t, h.Entries, tt.wantLen)
			hasBatch := false
			for _, e := range h.Entries {
				if e.BatchID == tt.batchID {
					hasBatch = true
					break
				}
			}
			assert.Equal(t, tt.wantBatch, hasBatch)
		})
	}
}
