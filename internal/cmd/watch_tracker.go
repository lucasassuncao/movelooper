package cmd

import (
	"container/heap"
	"sync"
	"time"
)

// trackedFile is one pending file in the watcher's queue.
type trackedFile struct {
	path     string
	detected time.Time // time of the most recent create/write event
	index    int       // position in the heap, maintained by heap.Interface
}

// trackedHeap is a min-heap of pending files ordered by detection time, so the
// root is always the file that has been quiet the longest.
type trackedHeap []*trackedFile

func (h trackedHeap) Len() int           { return len(h) }
func (h trackedHeap) Less(i, j int) bool { return h[i].detected.Before(h[j].detected) }
func (h trackedHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *trackedHeap) Push(x any) {
	tf := x.(*trackedFile)
	tf.index = len(*h)
	*h = append(*h, tf)
}

func (h *trackedHeap) Pop() any {
	old := *h
	n := len(old)
	tf := old[n-1]
	old[n-1] = nil
	tf.index = -1
	*h = old[:n-1]
	return tf
}

// fileTracker is an indexed min-heap of files the watcher has detected but not
// yet moved, ordered by the time of their most recent create/write event. Every
// new event pushes a file's timestamp forward (re-heapified in O(log n)); the
// ticker pops only the files whose stability delay has elapsed, instead of
// scanning every tracked file each tick. Safe for concurrent use.
type fileTracker struct {
	mu    sync.Mutex
	heap  trackedHeap
	index map[string]*trackedFile
}

func newFileTracker() *fileTracker {
	return &fileTracker{index: make(map[string]*trackedFile)}
}

// touch records a create/write event for path at time at, adding it to the queue
// or pushing an existing entry's timestamp forward. It reports whether the file
// was already being tracked.
func (t *fileTracker) touch(path string, at time.Time) (alreadyTracked bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if tf, ok := t.index[path]; ok {
		tf.detected = at
		heap.Fix(&t.heap, tf.index)
		return true
	}
	tf := &trackedFile{path: path, detected: at}
	heap.Push(&t.heap, tf)
	t.index[path] = tf
	return false
}

// due removes and returns the paths whose most recent event is older than
// threshold relative to now. Because the heap is ordered by detection time, it
// stops at the first file that is not yet stable, so a file still receiving
// events is never returned early.
func (t *fileTracker) due(now time.Time, threshold time.Duration) []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	var ready []string
	for t.heap.Len() > 0 {
		top := t.heap[0]
		if now.Sub(top.detected) <= threshold {
			break
		}
		heap.Pop(&t.heap)
		delete(t.index, top.path)
		ready = append(ready, top.path)
	}
	return ready
}
