package ui

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/lucasassuncao/movelooper/internal/core"
	"github.com/lucasassuncao/movelooper/internal/models"
)

// StartUI initializes and runs the Fyne UI
func StartUI(m *models.Movelooper) {
	a := app.NewWithID("com.lucasassuncao.movelooper")
	w := a.NewWindow("Movelooper")
	w.Resize(fyne.NewSize(800, 600))

	// --- Logs Tab Setup ---
	logEntry := widget.NewMultiLineEntry()
	logEntry.TextStyle = fyne.TextStyle{Monospace: true}
	logEntry.Wrapping = fyne.TextWrapWord
	logEntry.Disable() // Read-only

	// Redirect Logger
	logWriter := NewLogWriter(logEntry)
	multiWriter := MultiLogWriter(logWriter)
	m.Logger = m.Logger.WithWriter(multiWriter)

	// --- Dashboard Tab Setup ---
	statusLabel := widget.NewLabel("Status: Idle")
	statusLabel.TextStyle = fyne.TextStyle{Bold: true}

	var watchCancel context.CancelFunc
	var watchRunning bool

	runOnceBtn := widget.NewButtonWithIcon("Run Once", theme.MediaPlayIcon(), func() {
		statusLabel.SetText("Status: Running...")
		go func() {
			err := core.RunOnce(m, false, false)
			if err != nil {
				m.Logger.Error("Run error", m.Logger.Args("error", err.Error()))
			} else {
				m.Logger.Info("Run complete")
			}
			statusLabel.SetText("Status: Idle")
		}()
	})

	watchBtn := widget.NewButtonWithIcon("Start Watcher", theme.VisibilityIcon(), nil)
	watchBtn.OnTapped = func() {
		if watchRunning {
			// Stop Watcher
			if watchCancel != nil {
				watchCancel()
			}
			watchRunning = false
			watchBtn.SetText("Start Watcher")
			watchBtn.SetIcon(theme.VisibilityIcon())
			statusLabel.SetText("Status: Idle")
			m.Logger.Info("Watcher stopped")
		} else {
			// Start Watcher
			ctx, cancel := context.WithCancel(context.Background())
			watchCancel = cancel
			watchRunning = true
			watchBtn.SetText("Stop Watcher")
			watchBtn.SetIcon(theme.MediaStopIcon())
			statusLabel.SetText("Status: Watching...")

			go func() {
				err := core.StartWatcher(ctx, m)
				if err != nil {
					m.Logger.Error("Watcher error", m.Logger.Args("error", err.Error()))
					// Reset UI state if watcher crashes
					watchRunning = false
					watchBtn.SetText("Start Watcher")
					watchBtn.SetIcon(theme.VisibilityIcon())
					statusLabel.SetText("Status: Error")
				}
			}()
		}
	}

	dashboardContent := container.NewVBox(
		widget.NewLabelWithStyle("Movelooper Dashboard", fyne.TextAlignCenter, fyne.TextStyle{Bold: true, Italic: true}),
		layout.NewSpacer(),
		statusLabel,
		layout.NewSpacer(),
		runOnceBtn,
		watchBtn,
		layout.NewSpacer(),
	)

	// --- Config Tab Setup ---
	// For now, just list categories
	configList := widget.NewList(
		func() int {
			return len(m.Categories)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			cat := m.Categories[i]
			o.(*widget.Label).SetText(fmt.Sprintf("%s: %s -> %s", cat.Name, cat.Source, cat.Destination))
		},
	)

	configContent := container.NewBorder(
		widget.NewLabel("Current Configuration (Read-Only)"),
		nil, nil, nil,
		configList,
	)

	// --- Tabs ---
	tabs := container.NewAppTabs(
		container.NewTabItemWithIcon("Dashboard", theme.HomeIcon(), dashboardContent),
		container.NewTabItemWithIcon("Logs", theme.DocumentIcon(), container.NewScroll(logEntry)),
		container.NewTabItemWithIcon("Config", theme.SettingsIcon(), configContent),
	)

	w.SetContent(tabs)

	// Initial Log
	m.Logger.Info("Movelooper UI Started")

	w.ShowAndRun()
}
