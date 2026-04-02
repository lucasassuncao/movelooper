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

### `configuration` block

| Field         | Type     | Required | Default  | Description                                              |
|---------------|----------|----------|----------|----------------------------------------------------------|
| `output`      | string   | no       | `console`| Where to write logs: `console`, `file`, or `both`       |
| `log-file`    | string   | no       | —        | Path to the log file (required when `output` is `file` or `both`) |
| `log-level`   | string   | no       | `info`   | Log verbosity: `trace`, `debug`, `info`, `warn`, `error`, `fatal` |
| `show-caller` | bool     | no       | `false`  | Include the source location in log lines                 |
| `watch-delay`    | duration | no       | `5m`  | How long a file must be stable before `watch` moves it (e.g. `30s`, `5m`) |
| `history-limit`  | int      | no       | `50`  | Maximum number of batches retained in undo history                         |

### `categories` block

Each entry in the `categories` list accepts the following fields:

| Field               | Type       | Required | Default  | Description                                              |
|---------------------|------------|----------|----------|----------------------------------------------------------|
| `name`              | string     | yes      | —        | Label for the category (used in logs)                    |
| `source`            | string     | yes      | —        | Directory to scan for files                              |
| `destination`       | string     | yes      | —        | Root directory where files are moved (organized into `<destination>/<extension>/`) |
| `extensions`        | []string   | yes      | —        | List of file extensions to match (without the dot)       |
| `conflict_strategy` | string     | no       | `rename` | What to do when the destination file already exists (see below) |
| `regex`             | string     | no       | —        | Optional regex filter applied to the filename. Mutually exclusive with `glob` |
| `glob`              | string     | no       | —        | Optional glob filter applied to the filename. Mutually exclusive with `regex` |
| `ignore`            | []string   | no       | —        | Glob patterns for filenames to skip (case-insensitive)   |
| `min-age`           | duration   | no       | —        | Only move files whose modification time is older than this value (e.g. `24h`, `168h`) |
| `min-size`          | string     | no       | —        | Only move files at least this large (e.g. `500KB`, `10MB`, `1GB`)  |

#### Conflict strategies

| Value        | Behavior                                                                 |
|--------------|--------------------------------------------------------------------------|
| `rename`     | Appends `(1)`, `(2)`, … to the filename until it is unique (default)    |
| `overwrite`  | Deletes the existing destination file and replaces it                    |
| `skip`       | Leaves the source file untouched                                         |
| `hash_check` | Compares SHA-256 hashes; deletes source if identical, renames if different |

#### Filename filters (`regex` and `glob`)

Both fields narrow which files within a category are matched, in addition to `extensions`.
**They are mutually exclusive** — defining both in the same category is a configuration error.

- `regex` accepts any valid Go regular expression and is matched against the full filename.
- `glob` accepts a shell-style pattern (`*`, `?`) and supports brace expansion (`report_{2024,2025}_*`).
  Matching is case-insensitive.
- `ignore` uses the same glob syntax as `glob` and is always evaluated independently of both.

### Full example

```yaml
configuration:
  output: both
  log-file: ~/.movelooper/logs/movelooper.log
  log-level: info
  show-caller: false
  watch-delay: 5m

categories:
  - name: images
    source: C:\Users\johndoe\Downloads
    destination: C:\Users\johndoe\images
    extensions: [jpg, jpeg, png, gif, bmp, webp]
    conflict_strategy: rename
    ignore:
      - screenshot_*
      - "*_temp.*"

  - name: videos
    source: C:\Users\johndoe\Downloads
    destination: C:\Users\johndoe\videos
    extensions: [mp4, avi, mkv, mov, wmv]
    conflict_strategy: overwrite

  - name: dated-docs
    source: C:\Users\johndoe\Downloads
    destination: C:\Users\johndoe\dated
    extensions: [pdf, txt, log]
    regex: '^\d{4}-\d{2}-\d{2}_.*'

  - name: reports
    source: C:\Users\johndoe\Downloads
    destination: C:\Users\johndoe\reports
    extensions: [pdf, docx]
    glob: "report_*"
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
