package history

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const defaultMaxBatches = 100

// NewBatchID returns a collision-resistant batch ID for a one-shot move operation.
func NewBatchID() string { return newBatchID("batch") }

// NewWatchBatchID returns a collision-resistant batch ID for a watch-mode move operation.
func NewWatchBatchID() string { return newBatchID("watch") }

func newBatchID(prefix string) string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(b)
}

// Entry represents a single file operation
type Entry struct {
	Source      string    `json:"source"`
	Destination string    `json:"destination"`
	Timestamp   time.Time `json:"timestamp"`
	BatchID     string    `json:"batch_id"`
	Action      string    `json:"action"`
	Category    string    `json:"category"`
}

// History manages the log of file operations
type History struct {
	mu         sync.Mutex
	entries    []Entry
	batchDeque []string       // batch IDs in insertion order, no duplicates
	batchCount map[string]int // number of entries per batch ID
	path       string
	maxBatches int
}

// NewHistory creates a new History manager. path is the file where history is
// persisted; limit controls the maximum number of batches retained (values
// less than 1 fall back to defaultMaxBatches).
func NewHistory(path string, limit int) (*History, error) {
	if limit < 1 {
		limit = defaultMaxBatches
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, err
	}

	h := &History{
		path:       path,
		maxBatches: limit,
		batchCount: make(map[string]int),
	}

	if err := h.load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return h, nil
}

// rebuildIndex reconstructs batchDeque and batchCount from h.entries.
// Must be called with h.mu held, or before the History is shared.
func (h *History) rebuildIndex() {
	h.batchDeque = nil
	h.batchCount = make(map[string]int, len(h.entries))
	for _, e := range h.entries {
		if h.batchCount[e.BatchID] == 0 {
			h.batchDeque = append(h.batchDeque, e.BatchID)
		}
		h.batchCount[e.BatchID]++
	}
}

// Add records a new entry: it updates the in-memory state, prunes old batches
// past the limit, and rewrites the whole history file as an indented JSON array
// via an atomic temp-file rename.
func (h *History) Add(entry Entry) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.entries = append(h.entries, entry)
	if h.batchCount == nil {
		h.batchCount = make(map[string]int)
	}
	if h.batchCount[entry.BatchID] == 0 {
		h.batchDeque = append(h.batchDeque, entry.BatchID)
	}
	h.batchCount[entry.BatchID]++

	h.prune()
	return h.save()
}

// prune removes the oldest batches, keeping at most maxBatches.
// Uses batchDeque for an O(1) limit check; only scans entries when pruning.
func (h *History) prune() {
	if len(h.batchDeque) <= h.maxBatches {
		return
	}

	excess := len(h.batchDeque) - h.maxBatches
	toRemove := make(map[string]bool, excess)
	for _, id := range h.batchDeque[:excess] {
		toRemove[id] = true
		delete(h.batchCount, id)
	}
	h.batchDeque = h.batchDeque[excess:]

	newEntries := make([]Entry, 0, len(h.entries))
	for _, e := range h.entries {
		if !toRemove[e.BatchID] {
			newEntries = append(newEntries, e)
		}
	}
	h.entries = newEntries
}

// BatchSummary holds a brief description of a batch for listing purposes
type BatchSummary struct {
	BatchID   string
	Count     int
	Timestamp time.Time
}

// GetAllBatches returns one summary per batch, ordered oldest → newest
func (h *History) GetAllBatches() []BatchSummary {
	h.mu.Lock()
	defer h.mu.Unlock()

	firstTimestamp := make(map[string]time.Time, len(h.batchDeque))
	for _, e := range h.entries {
		if _, ok := firstTimestamp[e.BatchID]; !ok {
			firstTimestamp[e.BatchID] = e.Timestamp
		}
	}

	summaries := make([]BatchSummary, 0, len(h.batchDeque))
	for _, id := range h.batchDeque {
		summaries = append(summaries, BatchSummary{
			BatchID:   id,
			Count:     h.batchCount[id],
			Timestamp: firstTimestamp[id],
		})
	}
	return summaries
}

// GetLastBatchID returns the ID of the most recent batch
func (h *History) GetLastBatchID() (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.entries) == 0 {
		return "", fmt.Errorf("history is empty")
	}

	return h.entries[len(h.entries)-1].BatchID, nil
}

// GetBatch returns all entries for a given batch ID
func (h *History) GetBatch(batchID string) []Entry {
	h.mu.Lock()
	defer h.mu.Unlock()

	batch := make([]Entry, 0)
	for _, entry := range h.entries {
		if entry.BatchID == batchID {
			batch = append(batch, entry)
		}
	}
	return batch
}

// RemoveBatch removes all entries for a given batch ID
func (h *History) RemoveBatch(batchID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	newEntries := make([]Entry, 0, len(h.entries))
	for _, entry := range h.entries {
		if entry.BatchID != batchID {
			newEntries = append(newEntries, entry)
		}
	}

	original := h.entries
	h.entries = newEntries
	if err := h.save(); err != nil {
		h.entries = original
		h.rebuildIndex()
		return err
	}
	h.rebuildIndex()
	return nil
}

// RemoveCategoryFromBatch removes entries belonging to any of the given category
// names from the specified batch. If the batch becomes empty after removal, its
// reference is also gone. Entries with an empty Category field are never matched.
// Returns the number of entries removed.
func (h *History) RemoveCategoryFromBatch(batchID string, categories []string) (int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	catSet := make(map[string]bool, len(categories))
	for _, c := range categories {
		catSet[c] = true
	}

	newEntries := make([]Entry, 0, len(h.entries))
	for _, e := range h.entries {
		if e.BatchID == batchID && e.Category != "" && catSet[e.Category] {
			continue
		}
		newEntries = append(newEntries, e)
	}

	removed := len(h.entries) - len(newEntries)
	original := h.entries
	h.entries = newEntries
	if err := h.save(); err != nil {
		h.entries = original
		h.rebuildIndex()
		return 0, err
	}
	h.rebuildIndex()
	return removed, nil
}

// RemoveEntries removes specific entries from history, matched by BatchID and Source path.
// Only successfully restored entries should be passed so failed restores remain in history.
func (h *History) RemoveEntries(entries []Entry) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	toRemove := make(map[string]bool, len(entries))
	for _, e := range entries {
		toRemove[e.BatchID+"\x00"+e.Source] = true
	}

	newEntries := make([]Entry, 0, len(h.entries))
	for _, e := range h.entries {
		if !toRemove[e.BatchID+"\x00"+e.Source] {
			newEntries = append(newEntries, e)
		}
	}

	original := h.entries
	h.entries = newEntries
	if err := h.save(); err != nil {
		h.entries = original
		h.rebuildIndex()
		return err
	}
	h.rebuildIndex()
	return nil
}

// save writes h.entries to disk atomically using a temp file + rename, as an
// indented JSON array (2-space). Callers must hold h.mu.
func (h *History) save() error {
	entries := h.entries
	if entries == nil {
		entries = []Entry{}
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	tmp := h.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil { //#nosec G304 -- path is set by the application at startup from config, not from user input
		return err
	}
	return os.Rename(tmp, h.path)
}

// load reads the history file into h.entries. It supports two formats:
//   - JSON array (current): the whole file is an indented array, detected by a
//     leading '['.
//   - NDJSON (legacy): one JSON object per line, written by an earlier version;
//     malformed lines are skipped to tolerate a partial write from a crash.
//
// Callers must hold h.mu or call before the History is shared.
func (h *History) load() error {
	data, err := os.ReadFile(h.path)
	if err != nil {
		return err
	}

	// current: whole-file JSON array
	if content := bytes.TrimSpace(data); len(content) > 0 && content[0] == '[' {
		var entries []Entry
		if err := json.Unmarshal(data, &entries); err != nil {
			return err
		}
		h.entries = entries
		h.rebuildIndex()
		return nil
	}

	// legacy NDJSON: skip malformed lines (e.g. partial last line after a crash)
	sc := bufio.NewScanner(bytes.NewReader(data))
	var entries []Entry
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var e Entry
		if err := json.Unmarshal(line, &e); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	if err := sc.Err(); err != nil {
		return err
	}
	h.entries = entries
	h.rebuildIndex()
	return nil
}
