# Commands and Flags

## `movelooper` — move files once

Scans all enabled categories and moves matching files from source to destination. If a category defines `hooks`, the `before` hook runs before files are processed and the `after` hook runs when processing is complete.

```bash
movelooper [flags]
```

| Flag                  | Short | Description                                                          |
|-----------------------|-------|----------------------------------------------------------------------|
| `--dry-run`           |       | Show what would be moved without moving files                        |
| `--show-files`        |       | List individual files detected                                       |
| `--config`            | `-c`  | Path to a custom config file                                         |
| `--version`           |       | Print the current version                                            |
| `--category`          |       | Comma-separated list of category names to process (default: all)     |
| `--include-disabled`  |       | Include categories with `enabled: false`                             |

```bash
movelooper --category images                 # run only the "images" category
movelooper --category images,docs            # run "images" and "docs"
movelooper --include-disabled                # run all categories including disabled
movelooper --category archive --include-disabled  # run a disabled category explicitly
```

## `movelooper watch` — real-time monitoring

Monitors all source directories and moves files as they appear, after they stabilize (controlled by `watch-delay`). Hooks are executed per category on each triggered move, same as in the default move command.

```bash
movelooper watch
movelooper watch --dry-run                         # preview matched files without moving
movelooper watch --config /path/to/movelooper.yaml
movelooper watch --category images                 # watch only the "images" category
```

| Flag                  | Description                                                               |
|-----------------------|---------------------------------------------------------------------------|
| `--dry-run`           | Log matched files with their intended destination without moving them     |
| `--category`          | Comma-separated list of category names to monitor (default: all)          |
| `--include-disabled`  | Include categories with `enabled: false`                                  |

## `movelooper undo` — revert a batch

```bash
movelooper undo                                      # open interactive batch picker
movelooper undo --list                               # list all recorded batches
movelooper undo --dry-run                            # preview what would be restored
movelooper undo batch_1718000000                     # undo a specific move batch
movelooper undo batch_1718000000 --dry-run           # preview a specific batch restore
movelooper undo watch_1718000000000000000            # undo a specific watch batch
movelooper undo --category images                    # undo only "images" entries from the last batch
movelooper undo batch_1718000000 --category images,docs  # partial undo on a specific batch
```

| Flag          | Short | Description                                                        |
|---------------|-------|--------------------------------------------------------------------|
| `--list`      | `-l`  | List all recorded batches                                          |
| `--dry-run`   |       | Preview which files would be restored without moving any files     |
| `--category`  |       | Comma-separated list of category names to undo (default: all)      |

> **Note:** Undoing a `copy` batch removes the copied file at the destination. Undoing a `symlink` batch removes the symbolic link. The source file is never touched in either case.
>
> When using `--category`, only entries from the specified categories are reverted. If the batch becomes empty after the partial undo, it is removed from history entirely. Entries recorded before category tracking was added (older history) are skipped with a warning.

## `movelooper edit` — interactive config editor

Opens the configuration file in an interactive two-panel TUI editor. The left panel lists top-level configuration keys; pressing Enter opens the block editor where sub-fields can be toggled and edited. The editor validates the file on save.

```bash
movelooper edit
movelooper edit --theme dracula
movelooper edit --list-themes
movelooper edit --output /path/to/new.yaml
movelooper edit --config /path/to/movelooper.yaml
```

| Flag                    | Description                                                              |
|-------------------------|--------------------------------------------------------------------------|
| `--theme`               | Theme name (default: `dark`) — run `--list-themes` to see options        |
| `--list-themes`         | List available theme names and exit                                      |
| `--output`              | Save to this file instead of the loaded config (load path is unchanged)  |
| `--no-save-confirm`     | Skip the save confirmation dialog                                        |
| `--no-delete-confirm`   | Skip the block-delete confirmation dialog                                |
| `--no-validate-on-save` | Allow saving even when validators report errors (a warning is shown)     |

**Keybindings:** `Ctrl+S` save · `Ctrl+U` undo · `Ctrl+Y` redo · `Esc` quit

## `movelooper validate` — validate config file

Loads and validates the configuration file, reporting all rule violations. Exits with a non-zero status when errors are found.

```bash
movelooper validate
movelooper validate --format table
movelooper validate --format json --summary
movelooper validate --strict
movelooper validate --config /path/to/movelooper.yaml
```

| Flag        | Short | Description                                                                         |
|-------------|-------|-------------------------------------------------------------------------------------|
| `--format`  | `-f`  | Output format: `pretty` (default), `plain`, `table`, `json`                        |
| `--summary` |       | Show only total error counts, not individual violations                             |
| `--strict`  |       | Also verify that `source.path` and `destination.path` directories exist on disk    |

## `movelooper config` — show resolved config path

Prints the absolute path of the configuration file that would be loaded, after applying default search locations and the `--config` override.

```bash
movelooper config
movelooper config --config /path/to/movelooper.yaml
```

## `movelooper show-docs` — browse field reference in terminal

Renders the full field reference for `configuration` and `category` blocks directly in the terminal.

```bash
movelooper show-docs
movelooper show-docs --section source
movelooper show-docs --theme dracula
movelooper show-docs --list-themes
```

| Flag            | Description                                                                                |
|-----------------|--------------------------------------------------------------------------------------------|
| `--section`     | Show only docs matching this topic (case-insensitive, partial match)                      |
| `--theme`       | Theme name (default: `dark`) — run `--list-themes` to see options                         |
| `--list-themes` | List available theme names and exit                                                        |

## `movelooper self-update` — update the binary

Downloads the latest release from GitHub and replaces the current binary. The old binary is saved as `movelooper.exe.old` and cleaned up on the next run.

```bash
movelooper self-update
movelooper self-update --repo lucasassuncao/movelooper
```

---

[← Back to README](../README.md)
