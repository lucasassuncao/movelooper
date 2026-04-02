# 🌀 movelooper

<!-- markdownlint-disable MD033 -->
<p align="center">
  <img src="./movelooper.png" alt="Movelooper logo" width="360">
</p>
<!-- markdownlint-enable MD033 -->

**movelooper** is a modern CLI tool that automatically organizes and moves your files based on configurable categories. No manual sorting, no chaos. Perfect for keeping your Downloads, Dev, or Media folders clean and structured.

![made with Go](https://img.shields.io/badge/made_with-Go-blue?logo=go) ![type CLI](https://img.shields.io/badge/type-CLI-green) ![license MIT](https://img.shields.io/badge/license-MIT-lightgrey)

## Features

- Smart file organization based on configurable categories
- Multiple predefined templates (`basic`, `music`, `video`, `images`, `books`, `archives`, `installers`, `regex`, `full`)
- Interactive setup wizard
- Dry-run (`--preview` / `--dry-run`) to simulate moves safely
- Watch mode (`watch`) — monitors directories and moves files in real-time
- Undo support (`undo`) — revert the last or any specific batch of moves
- Regex and glob pattern filtering per category
- Ignore patterns to skip specific files
- Conflict strategies: `rename`, `overwrite`, `skip`, `hash_check`
- Show filenames with `--show-files`
- Logging support (`console`, `file`, or `both`)
- Custom config path with `--config` / `-c`
- Runs automatically — `movelooper` defaults to the move operation

## How It Works

`movelooper` reads your configuration file (`movelooper.yaml` or `conf/movelooper.yaml`), scans all extensions listed per category, and moves matching files from the source to the destination, organizing them into subfolders by extension.

## Installation

Download the latest version from the [releases page](https://github.com/lucasassuncao/movelooper/releases), then extract the binary and add it to your system's PATH.

## First-time Setup

Use the interactive wizard (recommended):

```bash
movelooper init -i
```

Or initialize from a template:

```bash
movelooper init -t media
```

Force overwrite an existing config:

```bash
movelooper init -f
```

### Available Templates

| Template     | Description                                          |
|--------------|------------------------------------------------------|
| `basic`      | One category: images                                 |
| `music`      | One category: audio files                            |
| `video`      | One category: video files                            |
| `images`     | One category: image files (includes SVG)             |
| `books`      | One category: documents and e-books                  |
| `archives`   | One category: compressed files                       |
| `installers` | One category: executable installers                  |
| `regex`      | One category with a date-prefix regex filter         |
| `full`       | All categories combined with conflict strategies     |

## Config File Reference (`movelooper.yaml`)

```yaml
configuration:
  output: both                        # console | file | both
  log-file: ~/.movelooper/logs/movelooper.log
  log-level: info                     # trace | debug | info | warn | error | fatal
  show-caller: false
  watch-delay: 5m                     # delay before moving a stable file in watch mode

categories:
  - name: images
    source: C:\Users\johndoe\Downloads
    destination: C:\Users\johndoe\images
    extensions: [jpg, jpeg, png, gif, bmp, webp]
    conflict_strategy: rename         # rename | overwrite | skip | hash_check
    ignore:
      - screenshot_*
      - "*_temp.*"

  - name: videos
    source: C:\Users\johndoe\Downloads
    destination: C:\Users\johndoe\Downloads\videos
    extensions: [mp4, avi, mkv, mov, wmv]
    conflict_strategy: overwrite

  - name: dated-docs
    source: C:\Users\johndoe\Downloads
    destination: C:\Users\johndoe\dated
    extensions: [pdf, txt, log]
    regex: '^\d{4}-\d{2}-\d{2}_.*'   # optional: filter by filename regex

  - name: reports
    source: C:\Users\johndoe\Downloads
    destination: C:\Users\johndoe\reports
    extensions: [pdf, docx]
    glob: "report_*"                  # optional: filter by glob pattern
```

## Commands and Flags

### `movelooper` (default — move files once)

```bash
movelooper [flags]
```

| Flag           | Short | Description                        |
|----------------|-------|------------------------------------|
| `--preview`    | `-p`  | Dry-run: show what would be moved  |
| `--dry-run`    |       | Alias for `--preview`              |
| `--show-files` |       | List individual files detected     |
| `--config`     | `-c`  | Path to a custom config file       |
| `--version`    |       | Print the current version          |

### `movelooper watch` — real-time monitoring

Monitors all source directories and moves files as they appear, after they stabilize (controlled by `watch-delay`).

```bash
movelooper watch
movelooper watch --config /path/to/movelooper.yaml
```

### `movelooper undo` — revert a batch

```bash
movelooper undo                           # undo the most recent batch
movelooper undo --list                    # list all recorded batches
movelooper undo batch_1718000000          # undo a specific move batch
movelooper undo watch_1718000000000000000 # undo a specific watch batch
```

### `movelooper init` — generate config

```bash
movelooper init -i       # interactive wizard
movelooper init -t full  # from template
movelooper init -f       # force overwrite existing config
```

## Tips

- Run with `-p` first to preview actions before organizing real files.
- Use `watch` mode to automatically keep your Downloads folder clean at all times.
- Use `undo --list` to inspect past operations and roll back any batch.
- Add `ignore` patterns to skip screenshots, drafts, or temp files from being moved.
- Use `glob` for simple name patterns (`report_*.pdf`) and `regex` for complex ones (`^\d{4}-.*`).
- Add `movelooper watch` to a cron job or Windows Task Scheduler for fully automatic cleanup.
