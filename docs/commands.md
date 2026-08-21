# Commands and Flags

## `movelooper` — move files once

Scans all enabled categories and moves matching files from source to destination. If a category defines `hooks`, the `before` hook runs before files are processed and the `after` hook runs when processing is complete.

```bash
movelooper [flags]
```

| Flag                  | Short | Description                                                          |
|-----------------------|-------|----------------------------------------------------------------------|
| `--dry-run`           |       | Show what would be moved without moving files, and report which destinations already exist |
| `--show-files`        |       | List each detected file and, after moving, its destination (one block per category) |
| `--config`            | `-c`  | Path to a custom config file                                         |
| `--format`            |       | Log output format: `pretty` (default) or `json`. Overrides `configuration.logging.format` |
| `--version`           |       | Print the current version                                            |
| `--category`          |       | Comma-separated list of category names to process (default: all)     |
| `--include-disabled`  |       | Include categories with `enabled: false`                             |

`--config` and `--format` are global flags: they apply to every command (`movelooper`, `watch`, `undo`, …). `--format json` emits structured slog JSON lines instead of the pretty console renderer, useful for piping to a log aggregator.

A run stops early in two cases: when you interrupt it with Ctrl+C, and when the destination becomes unwritable (a full disk, or a directory the process cannot write to) for several files in a row. In both cases the history of what was already moved is written before exiting, and the batch ID is printed so you can `undo` it. Every other failure skips the file and the run continues, reporting the total at the end. See [Guarantees and Limits](/GUARANTEES.md).

`--dry-run` resolves every destination without touching anything, and reports any destination that already exists together with what your `conflict-strategy` will do to it. Two tokens are the exception: `{seq}` and `{sha256:N}` are left as literal placeholders in the preview, because resolving them would mean scanning the destination and reading each file — a preview never does either.

```bash
movelooper --category images                 # run only the "images" category
movelooper --category images,docs            # run "images" and "docs"
movelooper --include-disabled                # run all categories including disabled
movelooper --category archive --include-disabled  # run a disabled category explicitly
movelooper --dry-run --format json           # preview as JSON lines
movelooper watch --format json               # structured logs in watch mode
```

With `action: archive`, a category is packed into a single `.zip`/`.tar.gz` at the destination instead of moving files individually. `--dry-run` lists what would be archived. Archive is not processed in `watch` mode (a warning is printed at startup) and archive batches cannot be undone.

## `movelooper watch` — real-time monitoring

Monitors all source directories and moves files as they appear, after they stabilize (controlled by `watch.delay`). Hooks are executed per category on each triggered move, same as in the default move command.

```bash
movelooper watch
movelooper watch --config /path/to/movelooper.yaml
movelooper watch --category images                 # watch only the "images" category
```

| Flag                  | Description                                                               |
|-----------------------|---------------------------------------------------------------------------|
| `--show-files`        | Log each file and its destination as it is moved                          |
| `--category`          | Comma-separated list of category names to monitor (default: all)          |
| `--include-disabled`  | Include categories with `enabled: false`                                  |

## `movelooper undo` — revert a batch

```bash
movelooper undo                                      # open interactive batch picker
movelooper undo --list                               # list all recorded batches
movelooper undo --dry-run                            # preview what would be restored
movelooper undo batch_a1b2c3d4e5f6a7b8               # undo a specific move batch
movelooper undo batch_a1b2c3d4e5f6a7b8 --dry-run     # preview a specific batch restore
movelooper undo watch_0f1e2d3c4b5a6978               # undo a specific watch batch
movelooper undo --category images                    # undo only "images" entries from the last batch
movelooper undo batch_a1b2c3d4e5f6a7b8 --category images,docs  # partial undo on a specific batch
movelooper undo --dry-run --format json              # preview the restore as JSON lines
```

| Flag          | Short | Description                                                        |
|---------------|-------|--------------------------------------------------------------------|
| `--list`      | `-l`  | List all recorded batches                                          |
| `--dry-run`   |       | Preview which files would be restored without moving any files     |
| `--category`  |       | Comma-separated list of category names to undo (default: all)      |

The global `--format json` also applies here: undo's restore/dry-run logs (`file(s) restored`, `[dry-run] would restore file(s)`) are emitted as structured JSON lines.

> **Note:** Undoing a `copy` batch removes the copied file at the destination. Undoing a `symlink` batch removes the symbolic link. The source file is never touched in either case.
>
> When using `--category`, only entries from the specified categories are reverted. If the batch becomes empty after the partial undo, it is removed from history entirely. Entries recorded before category tracking was added (older history) are skipped with a warning.

## `movelooper edit` — interactive config editor

Opens the configuration file in an interactive two-panel TUI editor. The left panel lists top-level configuration keys; pressing Enter opens the block editor where sub-fields can be toggled and edited. The editor validates the file on save.

```bash
movelooper edit
movelooper edit --theme grape
movelooper edit --list-themes
movelooper edit --output /path/to/new.yaml
movelooper edit --config /path/to/movelooper.yaml
```

| Flag                    | Description                                                              |
|-------------------------|--------------------------------------------------------------------------|
| `--theme`               | Theme name (default: `plain`) — run `--list-themes` to see options       |
| `--list-themes`         | List available theme names and exit                                      |
| `--output`, `-o`        | Save to this file instead of the loaded config (load path is unchanged)  |
| `--no-save-confirm`     | Skip the save confirmation dialog                                        |
| `--no-delete-confirm`   | Skip the block-delete confirmation dialog                                |
| `--no-validate-on-save` | Allow saving even when validators report errors (a warning is shown)     |

**Keybindings:** `Ctrl+S` save · `Ctrl+U` undo · `Ctrl+Y` redo · `Esc` quit

## `movelooper validate` — validate config file

Loads and validates the configuration file, reporting all rule violations. Exits with a non-zero status when errors are found.

Alongside errors, `validate` reports **warnings**: configurations that are perfectly valid and will run exactly as written, but that lose files in ways people rarely intend. Warnings never fail the command and never stop a run — the config stays the source of truth. They are reported here so you find out before the run instead of afterwards, from the missing files.

| Warning | Why it matters |
|---|---|
| A `rename` template with no per-file token, plus `conflict-strategy: overwrite` | Every file resolves to the same name, so each one overwrites the previous and only the last survives |
| `source.path` and `destination.path` are the same directory | Files are processed onto themselves |
| `action: archive` with `keep-source: false` | The originals are deleted and archive batches cannot be undone, so the archive becomes the only copy |
| `conflict-strategy: overwrite` with no `organize-by` and no `rename` | Everything lands in one directory under its original name, replacing existing files with matching names |
| Two categories reading the same source with overlapping extensions | The first match wins, so the later category may never see those files |

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

> On `validate`, `--format` controls the **validation report** rendering (`pretty`/`plain`/`table`/`json`), not the log format; its local `-f` shadows the global logging `--format` here.
| `--strict`  |       | Also verify that `source.path` and `destination.path` directories exist on disk    |

## `movelooper config` — show resolved config path

Prints the absolute path of the configuration file that would be loaded, after applying default search locations and the `--config` override.

```bash
movelooper config
movelooper config --config /path/to/movelooper.yaml
```

## `movelooper self-update` — update the binary

Downloads a release from GitHub and replaces the current binary. The old binary is saved with a `.old` suffix (e.g. `movelooper.exe.old` on Windows) and cleaned up on the next run.

```bash
movelooper self-update                          # install the latest stable release
movelooper self-update --list                   # list available releases
movelooper self-update --list --prerelease      # include rc/beta/alpha in the list
movelooper self-update --version v1.2.0         # install a specific release tag
movelooper self-update --repo lucasassuncao/movelooper
```

| Flag           | Description                                                                |
|----------------|----------------------------------------------------------------------------|
| `--repo`       | GitHub repository in `owner/repo` format                                   |
| `--version`    | Install this specific release tag (e.g. `v1.2.0`) instead of the latest     |
| `--list`       | List available releases and exit                                           |
| `--prerelease` | Include prereleases (rc/beta/alpha) in `--list`, or as the latest target   |
| `--limit`      | Maximum number of releases to show with `--list` (default `20`, max `100`)  |

---
