package ui

import (
	"io"
	"os"
	"sync"

	"fyne.io/fyne/v2/widget"
)

// LogWriter implements io.Writer to capture logs and display them in a Fyne widget
type LogWriter struct {
	entry *widget.Entry
	mu    sync.Mutex
}

// NewLogWriter creates a new LogWriter
func NewLogWriter(entry *widget.Entry) *LogWriter {
	return &LogWriter{entry: entry}
}

// Write implements io.Writer
func (w *LogWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	text := string(p)
	currentText := w.entry.Text

	// Simple truncation to prevent memory issues
	if len(currentText) > 100000 {
		currentText = currentText[50000:]
	}

	w.entry.SetText(currentText + text)
	w.entry.Refresh() // Ensure the widget updates

	return len(p), nil
}

// MultiLogWriter creates a writer that writes to both stdout and the UI
func MultiLogWriter(uiWriter io.Writer) io.Writer {
	return io.MultiWriter(os.Stdout, uiWriter)
}
