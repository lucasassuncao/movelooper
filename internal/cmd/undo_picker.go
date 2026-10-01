package cmd

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/lucasassuncao/bezel/draw"
	"github.com/lucasassuncao/bezel/layout"
	"github.com/lucasassuncao/bezel/list"
	"github.com/lucasassuncao/bezel/overlay"
	"github.com/lucasassuncao/bezel/shell"
	"github.com/lucasassuncao/bezel/table"
	"github.com/lucasassuncao/bezel/theme"
	"github.com/lucasassuncao/movelooper/internal/history"
)

const maxColRestoreTo = 55

// The picker's own keys arrive as these messages, sent by its actions.
type (
	pickerPreviewMsg struct{}
	pickerBackMsg    struct{}
	pickerSelectMsg  struct{}
)

var (
	pickerListActions = []shell.Action{
		shell.Help(), shell.Move(), shell.Quit(),
		shell.Custom("p", "preview files", shell.Send(pickerPreviewMsg{})),
		shell.Custom("enter", "select", shell.Send(pickerSelectMsg{})),
	}
	pickerPreviewActions = []shell.Action{
		shell.Help(), shell.Scroll(), shell.Quit(),
		shell.Custom("enter", "select & restore", shell.Send(pickerSelectMsg{})),
		shell.Custom("p/esc", "back", shell.Send(pickerBackMsg{}), shell.WithKey("p", "esc")),
	}
	previewColumns = []table.Column{{Title: "FILENAME", Max: 60, Flex: true}, {Title: "RESTORE TO"}}
)

// batchPicker lists the history's batches, newest first, and returns the one
// chosen. p swaps the list for a table of the files that batch would restore.
type batchPicker struct {
	sh       shell.Shell
	th       theme.Resolved
	batches  []history.BatchSummary
	list     list.Model
	hist     *history.History
	selected string

	previewing bool
	preview    [][]string // filename, restore-to
	offset     int
}

func newBatchPicker(batches []history.BatchSummary, hist *history.History) batchPicker {
	reversed := slices.Clone(batches)
	slices.Reverse(reversed)
	rows := make([]list.Row, len(reversed))
	for i, b := range reversed {
		label := fmt.Sprintf("%-24s  %3d files  %s", b.BatchID, b.Count, b.Timestamp.Format("2006-01-02 15:04:05"))
		if i == 0 {
			label += "  (most recent)"
		}
		rows[i] = list.Row{Label: label, Value: b.BatchID}
	}
	keys := list.DefaultKeys()
	keys.Filter = key.NewBinding() // no filter: the keys stay what they were

	th := theme.Resolve(theme.ThemeDefault, true)
	m := batchPicker{th: th, batches: reversed, hist: hist, list: list.New(rows, 0).WithKeys(keys)}
	m.sh = shell.New(shell.Config{Layout: layout.Fill("batches"), Theme: th, Title: "Select a batch to undo"})
	m.sh = m.sh.SetActions(pickerListActions...)
	return m
}

// Init asks the terminal for its background so the theme can match it.
func (m batchPicker) Init() tea.Cmd { return tea.RequestBackgroundColor }

func (m batchPicker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		m.th = theme.Resolve(theme.ThemeDefault, msg.IsDark())
		m.sh = m.sh.SetTheme(m.th)
		return m, nil
	case tea.WindowSizeMsg:
		m.sh, _, _ = m.sh.Update(msg)
		return m.resize(), nil
	case overlay.CloseMsg, overlay.PushMsg:
		var cmd tea.Cmd
		m.sh, _, cmd = m.sh.Update(msg)
		return m, cmd
	case pickerPreviewMsg:
		return m.openPreview(), nil
	case pickerBackMsg:
		m.previewing = false
		m.sh = m.sh.SetLayout(layout.Fill("batches")).SetActions(pickerListActions...)
		return m.resize(), nil
	case pickerSelectMsg:
		m.selected = m.chosen()
		return m, tea.Quit
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		// The shell runs the actions and, while the help is open, takes every key.
		if sh, handled, cmd := m.sh.Update(msg); handled {
			m.sh = sh
			return m, cmd
		}
		if m.previewing {
			return m.scrollPreview(msg), nil
		}
		m.list, _ = m.list.Update(msg)
		return m, nil
	}
	return m, nil
}

// resize fits the list to its pane after the terminal or the layout changed.
func (m batchPicker) resize() batchPicker {
	m.list = m.list.SetHeight(draw.InnerRect(m.sh.Rect("batches")).H)
	return m
}

func (m batchPicker) chosen() string {
	if r := m.list.Selected(); r != nil {
		return r.Value.(string)
	}
	return ""
}

func (m batchPicker) openPreview() batchPicker {
	entries := m.hist.GetBatch(m.chosen())
	m.preview = make([][]string, len(entries))
	for i, e := range entries {
		m.preview[i] = []string{filepath.Base(e.Destination), truncatePath(filepath.Dir(e.Source), maxColRestoreTo)}
	}
	m.previewing, m.offset = true, 0
	m.sh = m.sh.SetLayout(layout.Fill("preview")).SetActions(pickerPreviewActions...)
	return m
}

func (m batchPicker) scrollPreview(msg tea.KeyPressMsg) batchPicker {
	visible := m.previewRows()
	switch msg.String() {
	case "up":
		m.offset--
	case "down":
		m.offset++
	case "pgup":
		m.offset -= visible
	case "pgdown":
		m.offset += visible
	}
	m.offset = min(max(m.offset, 0), max(len(m.preview)-visible, 0))
	return m
}

// previewRows is how many file rows fit under the table's header.
func (m batchPicker) previewRows() int {
	return max(draw.InnerRect(m.sh.Rect("preview")).H-1, 1)
}

func (m batchPicker) View() tea.View {
	panes := map[string]shell.Pane{
		"batches": {Title: "Batches", Body: func(layout.Rect) string { return m.list.View(m.th) }},
	}
	if m.previewing {
		panes = map[string]shell.Pane{"preview": {
			Title: fmt.Sprintf("Preview: %s (%d files to restore)", m.chosen(), len(m.preview)),
			Body:  m.previewBody,
		}}
	}
	v := tea.NewView(m.sh.View(panes))
	v.AltScreen = true
	return v
}

func (m batchPicker) previewBody(r layout.Rect) string {
	widths := table.Fit(previewColumns, m.preview, r.W)
	lines := []string{m.th.TableHeader.Render(table.Titles(previewColumns, widths))}
	end := min(m.offset+m.previewRows(), len(m.preview))
	for _, row := range m.preview[m.offset:end] {
		lines = append(lines, table.Row(widths, row))
	}
	return strings.Join(lines, "\n")
}

// truncatePath keeps the tail of a path, the part that tells folders apart.
func truncatePath(p string, maxW int) string {
	r := []rune(p)
	if len(r) <= maxW {
		return p
	}
	return "…" + string(r[len(r)-(maxW-1):])
}

func pickBatch(batches []history.BatchSummary, hist *history.History) (string, error) {
	final, err := tea.NewProgram(newBatchPicker(batches, hist)).Run()
	if err != nil {
		return "", err
	}
	return final.(batchPicker).selected, nil
}
