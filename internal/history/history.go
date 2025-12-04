package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Entry represents a single file move operation
type Entry struct {
	Source      string    `json:"source"`
	Destination string    `json:"destination"`
	Timestamp   time.Time `json:"timestamp"`
	BatchID     string    `json:"batch_id"`
}

// History manages the log of file operations
type History struct {
	mu      sync.Mutex
	Entries []Entry `json:"entries"`
	path    string
}

// NewHistory creates a new History manager
func NewHistory() (*History, error) {
	ex, err := os.Executable()
	if err != nil {
		return nil, err
	}

	historyDir := filepath.Join(filepath.Dir(ex), "history")
	if err := os.MkdirAll(historyDir, 0755); err != nil {
		return nil, err
	}

	path := filepath.Join(historyDir, "movelooper.json")

	h := &History{
		path: path,
	}

	if err := h.load(); err != nil {
		// If file doesn't exist, start with empty history
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return h, nil
}

// Add appends a new entry to the history
func (h *History) Add(entry Entry) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.Entries = append(h.Entries, entry)
	return h.save()
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

	var newEntries []Entry
	for _, entry := range h.Entries {
		if entry.BatchID != batchID {
			newEntries = append(newEntries, entry)
		}
	}

	h.Entries = newEntries
	return h.save()
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

	return os.WriteFile(h.path, data, 0644)
}
