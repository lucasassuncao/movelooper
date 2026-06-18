package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lucasassuncao/movelooper/internal/history"
)

const (
	pickerHeaderLines = 3
	pickerFooterLines = 2
	maxColFilename    = 60
	maxColRestoreTo   = 55
)

var (
	pickerSelectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	pickerDimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	pickerFooterStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	pickerTitleStyle    = lipgloss.NewStyle().Bold(true)
)

type batchPicker struct {
	batches        []history.BatchSummary
	cursor         int
	viewport       int
	height         int
	selected       string
	quitting       bool
	previewing     bool
	previewEntries []history.Entry
	previewTable   table.Model
	hist           *history.History
}

func newBatchPicker(batches []history.BatchSummary, hist *history.History) batchPicker {
	reversed := make([]history.BatchSummary, len(batches))
	for i, b := range batches {
		reversed[len(batches)-1-i] = b
	}
	return batchPicker{batches: reversed, height: 24, hist: hist}
}

func (m batchPicker) listVisible() int {
	n := m.height - pickerHeaderLines - pickerFooterLines
	if n < 1 {
		n = 1
	}
	return n
}

func (m batchPicker) tableHeight() int {
	// title(1) + blank(1) + table-header(1) + border(1) + blank-after(1) + footer(1) = 6 fixed lines
	n := m.height - 6
	if n < 3 {
		n = 3
	}
	return n
}

func (m batchPicker) Init() tea.Cmd { return nil }

func (m batchPicker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		if m.previewing {
			m.previewTable.SetHeight(m.tableHeight())
		}
		return m, nil
	case tea.KeyMsg:
		if m.previewing {
			return m.updatePreview(msg)
		}
		return m.updateList(msg.String())
	}
	return m, nil
}

func (m batchPicker) updateList(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			if m.cursor < m.viewport {
				m.viewport = m.cursor
			}
		}
	case "down", "j":
		if m.cursor < len(m.batches)-1 {
			m.cursor++
			if m.cursor >= m.viewport+m.listVisible() {
				m.viewport = m.cursor - m.listVisible() + 1
			}
		}
	case "p":
		m.previewEntries = m.hist.GetBatch(m.batches[m.cursor].BatchID)
		m.previewTable = buildPreviewTable(m.previewEntries, m.tableHeight())
		m.previewing = true
	case "enter":
		m.selected = m.batches[m.cursor].BatchID
		m.quitting = true
		return m, tea.Quit
	case "q", "esc", "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func (m batchPicker) updatePreview(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.selected = m.batches[m.cursor].BatchID
		m.quitting = true
		return m, tea.Quit
	case "p", "esc":
		m.previewing = false
		return m, nil
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	}
	var cmd tea.Cmd
	m.previewTable, cmd = m.previewTable.Update(msg)
	return m, cmd
}

func (m batchPicker) View() string {
	if m.quitting {
		return ""
	}
	if m.previewing {
		return m.viewPreview()
	}
	return m.viewList()
}

func (m batchPicker) viewList() string {
	var sb strings.Builder
	sb.WriteString("  Select a batch to undo\n\n")

	end := m.viewport + m.listVisible()
	if end > len(m.batches) {
		end = len(m.batches)
	}

	for i := m.viewport; i < end; i++ {
		b := m.batches[i]
		label := fmt.Sprintf("  ○  %-24s  %3d files  %s",
			b.BatchID, b.Count, b.Timestamp.Format("2006-01-02 15:04:05"),
		)
		if i == 0 {
			label += "  (most recent)"
		}
		if i == m.cursor {
			label = strings.Replace(label, "○", "●", 1)
			sb.WriteString(pickerSelectedStyle.Render(label) + "\n")
		} else {
			sb.WriteString(pickerDimStyle.Render(label) + "\n")
		}
	}

	sb.WriteString("\n")
	sb.WriteString(pickerFooterStyle.Render(
		"  [↑↓ / jk] navigate   [p] preview files   [Enter] select   [q / Esc] cancel",
	))
	return sb.String()
}

func (m batchPicker) viewPreview() string {
	batch := m.batches[m.cursor]
	var sb strings.Builder

	sb.WriteString(pickerTitleStyle.Render(
		fmt.Sprintf("Preview: %s   (%d files to restore)", batch.BatchID, len(m.previewEntries)),
	))
	sb.WriteString("\n\n")
	sb.WriteString(m.previewTable.View())
	sb.WriteString("\n")
	sb.WriteString(pickerFooterStyle.Render(
		"  [↑↓ / jk] scroll   [Enter] select & restore   [p / Esc] back   [q] cancel",
	))
	return sb.String()
}

func runeLen(s string) int { return len([]rune(s)) }

func buildPreviewTable(entries []history.Entry, height int) table.Model {
	if height > len(entries) {
		height = len(entries)
	}

	filenameW := runeLen("FILENAME")
	restoreW := runeLen("RESTORE TO")
	for _, e := range entries {
		if w := runeLen(filepath.Base(e.Destination)); w > filenameW {
			filenameW = w
		}
		if w := runeLen(filepath.Dir(e.Source)); w > restoreW {
			restoreW = w
		}
	}
	if filenameW > maxColFilename {
		filenameW = maxColFilename
	}
	if restoreW > maxColRestoreTo {
		restoreW = maxColRestoreTo
	}

	columns := []table.Column{
		{Title: "FILENAME", Width: filenameW},
		{Title: "RESTORE TO", Width: restoreW},
	}

	rows := make([]table.Row, len(entries))
	for i, e := range entries {
		rows[i] = table.Row{
			truncate(filepath.Base(e.Destination), filenameW),
			truncatePath(filepath.Dir(e.Source), restoreW),
		}
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithHeight(height),
		table.WithFocused(true),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("8")).
		BorderBottom(true).
		Bold(true).
		Foreground(lipgloss.Color("8")).
		PaddingLeft(0)
	s.Cell = s.Cell.PaddingLeft(0)
	t.SetStyles(s)

	return t
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}

func truncatePath(p string, max int) string {
	r := []rune(p)
	if len(r) <= max {
		return p
	}
	return "…" + string(r[len(r)-(max-1):])
}

func pickBatch(batches []history.BatchSummary, hist *history.History) (string, error) {
	p := tea.NewProgram(newBatchPicker(batches, hist))
	m, err := p.Run()
	if err != nil {
		return "", err
	}
	return m.(batchPicker).selected, nil
}
