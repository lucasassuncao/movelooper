package cmd

import (
	"path/filepath"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lucasassuncao/bezel/shell"
	"github.com/lucasassuncao/movelooper/internal/history"
)

func newTestPicker(t *testing.T) batchPicker {
	t.Helper()
	hist, err := history.NewHistory(filepath.Join(t.TempDir(), "history.json"), 0)
	require.NoError(t, err)
	now := time.Now()
	require.NoError(t, hist.AddBatch([]history.Entry{
		{Source: "/in/a.txt", Destination: "/out/a.txt", BatchID: "batch_old", Action: "move", Timestamp: now.Add(-time.Hour)},
	}))
	require.NoError(t, hist.AddBatch([]history.Entry{
		{Source: "/in/b.txt", Destination: "/out/b.txt", BatchID: "batch_new", Action: "move", Timestamp: now},
		{Source: "/in/c.txt", Destination: "/out/c.txt", BatchID: "batch_new", Action: "move", Timestamp: now},
	}))
	p := newBatchPicker(hist.GetAllBatches(), hist)
	m, _ := p.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	return m.(batchPicker)
}

// pressKeys feeds each key and, like the runtime, the messages its commands
// return; it stops on quit and returns the quit command.
func pressKeys(p batchPicker, keys ...tea.KeyPressMsg) (batchPicker, tea.Cmd) {
	var cmd tea.Cmd
	for _, k := range keys {
		var msg tea.Msg = k
		for msg != nil {
			var m tea.Model
			m, cmd = p.Update(msg)
			p = m.(batchPicker)
			if cmd == nil {
				break
			}
			if msg = cmd(); isQuit(msg) {
				return p, cmd
			}
		}
	}
	return p, cmd
}

func isQuit(msg tea.Msg) bool {
	_, ok := msg.(tea.QuitMsg)
	return ok
}

var (
	keyEnter = tea.KeyPressMsg{Code: tea.KeyEnter}
	keyEsc   = tea.KeyPressMsg{Code: tea.KeyEscape}
	keyDown  = tea.KeyPressMsg{Code: tea.KeyDown}
	keyJ     = tea.KeyPressMsg{Code: 'j', Text: "j"}
	keyP     = tea.KeyPressMsg{Code: 'p', Text: "p"}
	keyQ     = tea.KeyPressMsg{Code: 'q', Text: "q"}
)

func TestBatchPicker_EnterSelectsTheMostRecentFirst(t *testing.T) {
	p, cmd := pressKeys(newTestPicker(t), keyEnter)
	assert.Equal(t, "batch_new", p.selected)
	require.NotNil(t, cmd)
	assert.IsType(t, tea.QuitMsg{}, cmd())
}

func TestBatchPicker_DownMovesDown(t *testing.T) {
	p, _ := pressKeys(newTestPicker(t), keyDown, keyEnter)
	assert.Equal(t, "batch_old", p.selected)
}

func TestBatchPicker_JDoesNotMove(t *testing.T) {
	p, _ := pressKeys(newTestPicker(t), keyJ, keyEnter)
	assert.Equal(t, "batch_new", p.selected)
}

func TestBatchPicker_QCancels(t *testing.T) {
	p, cmd := pressKeys(newTestPicker(t), keyQ)
	assert.Empty(t, p.selected)
	assert.IsType(t, tea.QuitMsg{}, cmd())
}

func TestBatchPicker_EscInTheListDoesNotQuit(t *testing.T) {
	p, cmd := pressKeys(newTestPicker(t), keyEsc)
	assert.Nil(t, cmd)
	assert.Empty(t, p.selected)
}

func TestBatchPicker_PreviewListsTheBatchFiles(t *testing.T) {
	p, _ := pressKeys(newTestPicker(t), keyP)
	require.True(t, p.previewing)
	view := ansi.Strip(p.View().Content)
	assert.Contains(t, view, "Preview: batch_new (2 files to restore)")
	assert.Contains(t, view, "b.txt")
	assert.Contains(t, view, "RESTORE TO")

	p, _ = pressKeys(p, keyEsc)
	assert.False(t, p.previewing)
	assert.Contains(t, ansi.Strip(p.View().Content), "(most recent)")
}

func TestBatchPicker_EnterInPreviewSelects(t *testing.T) {
	p, _ := pressKeys(newTestPicker(t), keyP, keyEnter)
	assert.Equal(t, "batch_new", p.selected)
}

func TestBatchPicker_HelpIsShownAndEscClosesItWithoutQuitting(t *testing.T) {
	p := newTestPicker(t)
	assert.Contains(t, ansi.Strip(p.View().Content), "[?] help")

	p, _ = pressKeys(p, tea.KeyPressMsg{Code: '?', Text: "?"})
	require.True(t, p.sh.HasOverlay(), "? opens the help")

	p, _ = pressKeys(p, keyEsc)
	assert.False(t, p.sh.HasOverlay(), "esc closes the help")
	assert.Empty(t, p.selected)
}

func TestBatchPicker_ActionsShareNoKey(t *testing.T) {
	require.NoError(t, shell.Check(pickerListActions...))
	require.NoError(t, shell.Check(pickerPreviewActions...))
}

func TestBatchPicker_LegendPutsThePrebuiltKeysFirst(t *testing.T) {
	view := ansi.Strip(newTestPicker(t).View().Content)
	assert.Contains(t, view, "[?] help  [↑/↓] move  [q] quit  [p] preview files  [enter] select")
}
