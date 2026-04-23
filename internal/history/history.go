package history

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const defaultMaxBatches = 50

// NewBatchID returns a collision-resistant batch ID for a one-shot move operation.
func NewBatchID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("batch_%d", time.Now().UnixNano())
	}
	return "batch_" + hex.EncodeToString(b)
}

// NewWatchBatchID returns a collision-resistant batch ID for a watch-mode move operation.
func NewWatchBatchID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("watch_%d", time.Now().UnixNano())
	}
	return "watch_" + hex.EncodeToString(b)
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
	Entries    []Entry `json:"entries"`
	path       string
	maxBatches int
}

// NewHistory creates a new History manager. limit controls the maximum number
// of batches retained; values less than 1 fall back to defaultMaxBatches.
func NewHistory(limit int) (*History, error) {
	if limit < 1 {
		limit = defaultMaxBatches
	}

	ex, err := os.Executable()
	if err != nil {
		return nil, err
	}

	historyDir := filepath.Join(filepath.Dir(ex), "history")
	if err := os.MkdirAll(historyDir, 0750); err != nil {
		return nil, err
	}

	path := filepath.Join(historyDir, "movelooper.json")

	h := &History{
		path:       path,
		maxBatches: limit,
	}

	if err := h.load(); err != nil {
		// If file doesn't exist, start with empty history
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return h, nil
}

// Add appends a new entry to the history.
func (h *History) Add(entry Entry) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	snapshot := make([]Entry, len(h.Entries))
	copy(snapshot, h.Entries)
	h.Entries = append(h.Entries, entry)
	h.prune()
	if err := h.save(); err != nil {
		h.Entries = snapshot
		return err
	}
	return nil
}

// prune removes the oldest batches, keeping at most maxBatches
func (h *History) prune() {
	seen := make(map[string]bool)
	var batchOrder []string
	for _, e := range h.Entries {
		if !seen[e.BatchID] {
			seen[e.BatchID] = true
			batchOrder = append(batchOrder, e.BatchID)
		}
	}

	if len(batchOrder) <= h.maxBatches {
		return
	}

	toRemove := make(map[string]bool)
	for _, id := range batchOrder[:len(batchOrder)-h.maxBatches] {
		toRemove[id] = true
	}

	var newEntries []Entry
	for _, e := range h.Entries {
		if !toRemove[e.BatchID] {
			newEntries = append(newEntries, e)
		}
	}
	h.Entries = newEntries
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

	seen := make(map[string]*BatchSummary)
	var order []string
	for _, e := range h.Entries {
		if _, ok := seen[e.BatchID]; !ok {
			seen[e.BatchID] = &BatchSummary{BatchID: e.BatchID, Timestamp: e.Timestamp}
			order = append(order, e.BatchID)
		}
		seen[e.BatchID].Count++
	}

	summaries := make([]BatchSummary, 0, len(order))
	for _, id := range order {
		summaries = append(summaries, *seen[id])
	}
	return summaries
}

// GetLastBatchID returns the ID of the most recent batch
func (h *History) GetLastBatchID() (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.Entries) == 0 {
		return "", fmt.Errorf("history is empty")
	}

	return h.Entries[len(h.Entries)-1].BatchID, nil
}

// GetBatch returns all entries for a given batch ID
func (h *History) GetBatch(batchID string) []Entry {
	h.mu.Lock()
	defer h.mu.Unlock()

	var batch []Entry
	for _, entry := range h.Entries {
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

	newEntries := make([]Entry, 0, len(h.Entries))
	for _, entry := range h.Entries {
		if entry.BatchID != batchID {
			newEntries = append(newEntries, entry)
		}
	}

	original := h.Entries
	h.Entries = newEntries
	if err := h.save(); err != nil {
		h.Entries = original
		return err
	}
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

	newEntries := make([]Entry, 0, len(h.Entries))
	for _, e := range h.Entries {
		if e.BatchID == batchID && e.Category != "" && catSet[e.Category] {
			continue
		}
		newEntries = append(newEntries, e)
	}

	removed := len(h.Entries) - len(newEntries)
	original := h.Entries
	h.Entries = newEntries
	if err := h.save(); err != nil {
		h.Entries = original
		return 0, err
	}
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

	newEntries := make([]Entry, 0, len(h.Entries))
	for _, e := range h.Entries {
		if !toRemove[e.BatchID+"\x00"+e.Source] {
			newEntries = append(newEntries, e)
		}
	}

	original := h.Entries
	h.Entries = newEntries
	if err := h.save(); err != nil {
		h.Entries = original
		return err
	}
	return nil
}

func (h *History) load() error {
	data, err := os.ReadFile(h.path)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &h.Entries)
}

func (h *History) save() error {
	data, err := json.MarshalIndent(h.Entries, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(h.path, data, 0600)
}
