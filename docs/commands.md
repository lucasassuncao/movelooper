# Commands and Flags

## `movelooper` — move files once

Scans all enabled categories and moves matching files from source to destination.

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

Monitors all source directories and moves files as they appear, after they stabilize (controlled by `watch-delay`).

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
movelooper undo                                      # undo the most recent batch
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

## `movelooper init` — generate config

```bash
movelooper init -i                           # interactive wizard
movelooper init -t full                      # from template
movelooper init -o /path/to/custom.yaml      # custom output path
movelooper init -f                           # force overwrite existing config
```

| Flag            | Short | Description                              |
|-----------------|-------|------------------------------------------|
| `--interactive` | `-i`  | Launch the interactive wizard            |
| `--template`    | `-t`  | Template to use (default: `basic`)       |
| `--output`      | `-o`  | Path to write the config file            |
| `--force`       | `-f`  | Overwrite existing config file           |

### Available templates

| Template     | Description                                                      |
|--------------|------------------------------------------------------------------|
| `basic`      | One category: images                                             |
| `music`      | One category: audio files                                        |
| `video`      | One category: video files                                        |
| `images`     | One category: image files (includes SVG)                         |
| `books`      | One category: documents and e-books                              |
| `archives`   | One category: compressed files                                   |
| `installers` | One category: executable installers                              |
| `regex`      | One category with a date-prefix regex filter                     |
| `full`       | All categories combined, including copy and symlink examples     |

## `movelooper config validate` — validate config

Loads and validates the configuration file without moving any files.

```bash
movelooper config validate
movelooper config validate --config /path/to/movelooper.yaml
```

## `movelooper config show` — inspect active config

Prints the active configuration as resolved in memory after startup, including all defaults filled in. Useful for verifying what movelooper is actually using.

```bash
movelooper config show
movelooper config show --config /path/to/movelooper.yaml
```

## `movelooper self-update` — update the binary

Downloads the latest release from GitHub and replaces the current binary. The old binary is saved as `movelooper.exe.old` and cleaned up on the next run.

```bash
movelooper self-update
movelooper self-update --repo lucasassuncao/movelooper
```

---

[← Back to README](../README.md)
