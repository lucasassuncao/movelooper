package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/lucasassuncao/bezel/theme"
	"github.com/lucasassuncao/bezel/themebrowser"
	"github.com/lucasassuncao/movelooper/internal/config"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/yedit/editor"
	"github.com/spf13/cobra"
)

// EditCmd returns the "edit" command, which opens an interactive TUI editor
// for the movelooper configuration file.
func EditCmd() *cobra.Command {
	var output string
	var themeName string
	var listThemes bool
	var noSaveConfirm bool
	var noDeleteConfirm bool
	var noValidateOnSave bool
	var dump bool
	var dumpPath string

	cmd := &cobra.Command{
		Use:               "edit",
		Short:             "Edit the movelooper configuration file in an interactive TUI",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
		Long: `Open the movelooper configuration file in an interactive two-panel TUI editor.

The left panel lists top-level configuration keys; pressing Enter opens the
block editor where sub-fields can be toggled and edited. Tab changes pane;
Ctrl+S writes the file; Ctrl+U undoes the last change; Ctrl+Y redoes it;
Esc goes back; q quits. Press ? in the editor for every key.

Use --output to write to a different file than the one loaded (e.g. to
produce a new config from an existing one).`,
		Example: `  # Edit the default configuration file
  movelooper edit

  # Edit with the Grape theme
  movelooper edit --theme grape

  # Browse the available themes in a scrollable table
  movelooper edit --list-themes

  # Load from --config but save to a new file
  movelooper edit --output /path/to/new.yaml

  # Record a session trace to attach to a bug report
  movelooper edit --dump

  # Record the trace to a specific file instead of the OS temp dir
  movelooper edit --dump-path ./trace.jsonl`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if listThemes {
				return themebrowser.BrowseInTerminal()
			}

			selectedTheme, err := theme.Lookup(themeName)
			if err != nil {
				return fmt.Errorf("%w: run 'movelooper edit --list-themes' to see available themes", err)
			}

			configFlag, _ := cmd.Root().PersistentFlags().GetString("config")
			loadPath, savePath, err := resolveEditPaths(configFlag, output)
			if err != nil {
				return err
			}
			output = savePath

			saveTarget := loadPath
			if output != "" {
				saveTarget = output
			}
			if _, statErr := os.Stat(saveTarget); errors.Is(statErr, fs.ErrNotExist) {
				if err := ensureConfigDir(saveTarget); err != nil {
					return err
				}
			}

			movelooperHints, err := buildMovelooperHints()
			if err != nil {
				return fmt.Errorf("building hint source: %w", err)
			}

			res, err := editor.Run(editor.Config{
				Path:                 loadPath,
				SavePath:             output,
				Schema:               &models.Config{},
				Title:                "movelooper",
				BlockPresets:         MovelooperBlockPresets,
				DocPresets:           MovelooperDocPresets,
				EnableHints:          true,
				Metadata:             movelooperHints,
				Theme:                selectedTheme,
				PassthroughKeys:      []string{"import"},
				NoSaveConfirm:        noSaveConfirm,
				NoDeleteConfirm:      noDeleteConfirm,
				NoValidateOnSave:     noValidateOnSave,
				SchemaRecursionDepth: config.MaxFilterNestingDepth - 1,
				Validators:           MovelooperValidators,
				AnimationDuration:    600 * time.Millisecond,
				Trace: editor.Trace{
					Dump:     dump || dumpPath != "",
					DumpPath: dumpPath,
				},
			})
			if err != nil {
				return err
			}
			if res.Saved {
				savedTo := loadPath
				if output != "" {
					savedTo = output
				}
				fmt.Println("configuration saved to", savedTo)
			}
			if res.DumpPath != "" {
				fmt.Println("session trace written to", res.DumpPath)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "Save to this file instead of the loaded config (load path is unchanged)")
	cmd.Flags().StringVar(&themeName, "theme", "plain", "Theme name (run --list-themes to see options)")
	cmd.Flags().BoolVar(&listThemes, "list-themes", false, "Browse the available themes in a scrollable table (q quits)")
	cmd.Flags().BoolVar(&noSaveConfirm, "no-save-confirm", false, "Skip the 'Save changes?' confirmation dialog")
	cmd.Flags().BoolVar(&noDeleteConfirm, "no-delete-confirm", false, "Skip the 'Remove block?' confirmation dialog")
	cmd.Flags().BoolVar(&noValidateOnSave, "no-validate-on-save", false, "Allow saving even when validators report errors (a warning is shown)")
	cmd.Flags().BoolVar(&dump, "dump", false, "Record every editor action to a JSONL trace file for bug reports (path is printed on exit)")
	cmd.Flags().StringVar(&dumpPath, "dump-path", "", "Write the session trace to this file instead of a temp file (implies --dump)")

	return cmd
}

// resolveEditPaths picks the config to open and the optional save target; when
// no config exists, it creates the default path used by future runs.
func resolveEditPaths(configFlag, output string) (loadPath, savePath string, err error) {
	if configFlag != "" {
		return configFlag, output, nil
	}
	if output != "" {
		return output, "", nil
	}
	if resolved, resolveErr := config.ResolveConfigPath(""); resolveErr == nil {
		return resolved, "", nil
	}
	// If no config exists, use the default home path so the first edit creates
	// the file that later runs will look for; the executable dir is a fallback.
	if home, homeErr := os.UserHomeDir(); homeErr == nil {
		return filepath.Join(home, ".movelooper", "conf", "movelooper.yaml"), "", nil
	}
	ex, err := os.Executable()
	if err != nil {
		return "", "", fmt.Errorf("could not determine executable path: %w", err)
	}
	return filepath.Join(filepath.Dir(ex), "conf", "movelooper.yaml"), "", nil
}

// ensureConfigDir creates the parent directory before the editor saves a config.
func ensureConfigDir(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("creating config directory %s: %w", dir, err)
	}
	return nil
}
