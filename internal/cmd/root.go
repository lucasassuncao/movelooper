// Package cmd contains the command line interface commands for the Movelooper application
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/lucasassuncao/movelooper/internal/config"
	"github.com/lucasassuncao/movelooper/internal/helper"
	"github.com/lucasassuncao/movelooper/internal/history"
	"github.com/lucasassuncao/movelooper/internal/models"

	"github.com/spf13/cobra"
)

// RootCmd represents the base command when called without any subcommands
func RootCmd(m *models.Movelooper) *cobra.Command {
	var (
		dryRun    bool
		showFiles bool
	)

	cmd := &cobra.Command{
		Use:   "movelooper",
		Short: "movelooper is a CLI tool for organizing and moving files",
		Long: `movelooper organizes and moves files from source directories to destination directories,
based on configurable categories.

By default, it runs the move operation automatically.
Use -p / --preview / --dry-run for a dry-run preview, and --show-files to display filenames.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			configPath, _ := cmd.Flags().GetString("config")
			return preRunHandler(m, configPath)
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			if m.LogCloser != nil {
				return m.LogCloser.Close()
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			categories, err := config.UnmarshalConfig(m)
			if err != nil {
				return err
			}
			m.Categories = categories

			batchID := fmt.Sprintf("batch_%d", time.Now().Unix())

			// Track moved files to avoid processing them multiple times
			movedFiles := make(map[string]bool)

			for _, category := range m.Categories {
				files, err := helper.ReadDirectory(category.Source)
				if err != nil {
					m.Logger.Error("failed to read directory",
						m.Logger.Args("path", category.Source),
						m.Logger.Args("error", err.Error()),
					)
					continue
				}

				// Extensions are mandatory; regex/glob act as additional name filters
				for _, extension := range category.Extensions {
					// Build candidate list: not already moved, not ignored, matches extension
					var filteredFiles []os.DirEntry
					for _, file := range files {
						filePath := filepath.Join(category.Source, file.Name())
						if movedFiles[filePath] {
							continue
						}
						if helper.MatchesIgnorePatterns(file.Name(), category.Ignore) {
							continue
						}
						if !helper.HasExtension(file, extension) || !file.Type().IsRegular() {
							continue
						}
						// Apply optional regex/glob name filter
						if category.Regex != "" && !helper.MatchesRegex(file.Name(), category.CompiledRegex) {
							continue
						}
						if category.Glob != "" && !helper.MatchesGlob(file.Name(), category.Glob) {
							continue
						}
						filteredFiles = append(filteredFiles, file)
					}

					count := len(filteredFiles)
					switch count {
					case 0:
						m.Logger.Info(fmt.Sprintf("No .%s files found", extension))
					default:
						message := fmt.Sprintf("%d .%s files to move", count, extension)
						if showFiles {
							logArgs := helper.GenerateLogArgs(filteredFiles, extension)
							if len(logArgs) > 0 {
								m.Logger.Warn(message, m.Logger.Args(logArgs...))
							} else {
								m.Logger.Warn(message)
							}
						} else {
							m.Logger.Warn(message)
						}
					}

					if !dryRun && count > 0 {
						dirPath := filepath.Join(category.Destination, extension)
						if err := helper.CreateDirectory(dirPath); err != nil {
							m.Logger.Error("failed to create directory", m.Logger.Args("error", err.Error()))
						}
						helper.MoveFiles(m, category, filteredFiles, extension, batchID)
						for _, file := range filteredFiles {
							filePath := filepath.Join(category.Source, file.Name())
							movedFiles[filePath] = true
						}
					}
				}
			}

			if dryRun {
				m.Logger.Info("Dry-run complete (no files were moved).")
			}

			return nil
		},
	}

	cmd.PersistentFlags().StringP("config", "c", "", "Path to configuration file (e.g., /path/to/movelooper.yaml)")
	cmd.PersistentFlags().BoolVarP(&dryRun, "preview", "p", false, "Run in dry-run (preview) mode without moving files")
	cmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Alias for --preview")
	cmd.PersistentFlags().BoolVar(&showFiles, "show-files", false, "Show list of individual files detected")

	// Add subcommands
	cmd.AddCommand(InitCmd())
	cmd.AddCommand(WatchCmd(m))
	cmd.AddCommand(UndoCmd(m))

	return cmd
}

// preRunHandler handles the necessary configuration before command execution
func preRunHandler(m *models.Movelooper, configPath string) error {
	var options []config.ViperOptions

	if configPath != "" {
		// Se um caminho específico foi fornecido, use-o
		dir := filepath.Dir(configPath)
		filename := filepath.Base(configPath)
		ext := filepath.Ext(filename)
		nameWithoutExt := filename[:len(filename)-len(ext)]

		options = []config.ViperOptions{
			config.WithConfigName(nameWithoutExt),
			config.WithConfigType(ext[1:]),
			config.WithConfigPath(dir),
		}
	} else {
		ex, err := os.Executable()
		if err != nil {
			return fmt.Errorf("error getting executable: %v", err)
		}

		options = []config.ViperOptions{
			config.WithConfigName("movelooper"),
			config.WithConfigType("yaml"),
			config.WithConfigPath(filepath.Dir(ex)),
			config.WithConfigPath(filepath.Join(filepath.Dir(ex), "conf")),
		}
	}

	err := config.InitConfig(m.Viper, options...)
	if err != nil {
		if configPath != "" {
			return fmt.Errorf("configuration file not found at '%s'", configPath)
		}
		return fmt.Errorf("configuration file not found\n\nPlease run 'movelooper init' to create a configuration file")
	}

	logger, closer, err := config.ConfigureLogger(m.Viper)
	if err != nil {
		return fmt.Errorf("failed to configure logger: %v", err)
	}

	m.Logger = logger
	m.LogCloser = closer

	hist, err := history.NewHistory()
	if err != nil {
		// Log warning but don't fail app if history fails
		m.Logger.Warn("Failed to initialize history tracking", m.Logger.Args("error", err.Error()))
	} else {
		m.History = hist
	}

	return nil
}
