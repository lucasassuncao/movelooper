<!-- markdownlint-disable MD033 -->
<p align="center">
  <img src="docs/movelooper2.png" alt="Movelooper logo" width="300" height="300">
</p>
<!-- markdownlint-enable MD033 -->

🌀 **Movelooper** is a modern CLI tool that automatically organizes and moves your files based on configurable categories. No manual sorting, no chaos. Perfect for keeping your Downloads, Dev, or Media folders clean and structured.

![made with Go](https://img.shields.io/badge/made_with-Go-blue?logo=go) ![type CLI](https://img.shields.io/badge/type-CLI-green) ![license MIT](https://img.shields.io/badge/license-MIT-lightgrey)

## Features

### Organize

- Category-based rules: match files by extension, regex, glob, age, and size
- Organize files by using templates: `{ext}`, `{mod-year}`, `{mod-month}`, `{size-range}`, and [more](#organize-by-tokens)
- Wildcard `extensions: [all]` to catch any file type
- Conflict strategies per category: `rename`, `overwrite`, `skip`, `hash_check`

### Automate

- Watch mode: monitors directories in real-time, moves files as they stabilize
- Undo: revert the last batch or any specific batch from history
- `--dry-run` on move, watch, and undo to preview before committing

### Configure

- Split config across multiple YAML files with `import:`
- Interactive setup wizard (`init -i`) or generate from a template (`init -t full`)
- `config show` to inspect the merged config · `config validate` to catch errors early
- Self-update with `self-update`

## How It Works

`movelooper` reads your configuration file (`movelooper.yaml` or `conf/movelooper.yaml`), scans all extensions listed per category, and moves matching files from the source to the destination. 

The optional `organize-by` field controls how files are placed inside `<destination>/` using a template — for example `{ext}/{mod-year}/{mod-month}` places a `.jpg` modified in April 2025 into `<destination>/jpg/2025/04/`.

## Installation

Download the latest version from the [releases page](https://github.com/lucasassuncao/movelooper/releases), then extract the binary and add it to your system's PATH.

## First-time Setup

Use the interactive wizard (recommended):

```bash
movelooper init -i
```

Or initialize from a template:

```bash
movelooper init -t full
```

Save to a custom path:

```bash
movelooper init -o /path/to/movelooper.yaml
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

Each entry in the `categories` list has the following top-level fields:

| Field         | Type   | Required | Default | Description                                          |
|---------------|--------|----------|---------|------------------------------------------------------|
| `name`        | string | yes      | —       | Label for the category (used in logs and undo history) |
| `enabled`     | bool   | no       | `true`  | When `false`, the category is skipped in all modes  |
| `source`      | object | yes      | —       | Where to scan for files (see below)                  |
| `destination` | object | yes      | —       | Where to move files and how (see below)              |

#### `source` block

| Field        | Type      | Required | Description                                                                 |
|--------------|-----------|----------|-----------------------------------------------------------------------------|
| `path`       | string    | yes      | Directory to scan for files                                                 |
| `extensions` | []string  | yes      | File extensions to match (without the dot). Use `["all"]` to match any extension |
| `filter`     | object    | no       | Optional filters applied to files in this source (see below)               |

#### `source.filter` block

| Field      | Type      | Description                                                                        |
|------------|-----------|------------------------------------------------------------------------------------|
| `regex`    | string    | Regex filter applied to the filename. Mutually exclusive with `glob`               |
| `glob`     | string    | Glob filter applied to the filename. Mutually exclusive with `regex`               |
| `ignore`   | []string  | Glob patterns for filenames to skip (case-insensitive)                             |
| `min-age`  | duration  | Only move files older than this value (e.g. `24h`, `168h`)                        |
| `max-age`  | duration  | Only move files newer than this value (e.g. `720h`, `8760h`)                      |
| `min-size` | string    | Only move files at least this large (e.g. `500KB`, `10MB`, `1GB`)                 |
| `max-size` | string    | Only move files at most this large (e.g. `10MB`, `1GB`)                           |

#### `destination` block

| Field               | Type   | Required | Default  | Description                                                                     |
|---------------------|--------|----------|----------|---------------------------------------------------------------------------------|
| `path`              | string | yes      | —        | Root directory where matched files are moved                                    |
| `conflict-strategy` | string | no       | `rename` | What to do when a file already exists at the destination (see below)            |
| `organize-by`       | string | no       | —        | Template defining subdirectory structure under `path` (see tokens below). Leave empty to move files directly into `path` |

#### Conflict strategies

| Value        | Behavior                                                                              |
|--------------|---------------------------------------------------------------------------------------|
| `rename`     | Appends `(1)`, `(2)`, … to the filename until it is unique (default)                 |
| `overwrite`  | Deletes the existing destination file and replaces it                                 |
| `skip`       | Leaves the source file untouched                                                      |
| `hash_check` | Compares SHA-256 hashes; deletes source if identical, renames if different            |
| `newest`     | Keeps the file with the most recent modification time; skips source if dest is newer  |
| `oldest`     | Keeps the file with the oldest modification time; skips source if dest is older       |
| `larger`     | Keeps the larger file; skips source if dest is larger or equal                        |
| `smaller`    | Keeps the smaller file; skips source if dest is smaller or equal                      |

#### Filename filters (`filter.regex` and `filter.glob`)

Both fields narrow which files within a category are matched, in addition to `extensions`.
**They are mutually exclusive** — defining both in the same category is a configuration error.

- `filter.regex` accepts any valid Go regular expression and is matched against the full filename.
- `filter.glob` accepts a shell-style pattern (`*`, `?`) and supports brace expansion (`report_{2024,2025}_*`).
  Matching is case-insensitive.
- `filter.ignore` uses the same glob syntax as `filter.glob` and is always evaluated independently of both.

#### `organize-by` tokens

| Token | Example | Description |
|---|---|---|
| `{ext}` | `jpg` | File extension, lowercase |
| `{ext-upper}` | `JPG` | File extension, uppercase |
| `{name}` | `photo` | Filename without extension |
| `{mod-year}` | `2025` | File modification year |
| `{mod-month}` | `04` | File modification month (zero-padded) |
| `{mod-day}` | `08` | File modification day (zero-padded) |
| `{mod-date}` | `2025-04-08` | Shorthand for `{mod-year}-{mod-month}-{mod-day}` |
| `{mod-weekday}` | `Tuesday` | File modification weekday |
| `{created-year}` | `2025` | File creation year (falls back to mod time on Linux) |
| `{created-month}` | `04` | File creation month |
| `{created-day}` | `08` | File creation day |
| `{created-date}` | `2025-04-08` | Shorthand for `{created-year}-{created-month}-{created-day}` |
| `{year}` | `2025` | Run date year |
| `{month}` | `04` | Run date month |
| `{day}` | `08` | Run date day |
| `{date}` | `2025-04-08` | Run date shorthand |
| `{weekday}` | `Tuesday` | Run date weekday |
| `{size-range}` | `small` | `tiny` (<1 MB) · `small` (1–100 MB) · `medium` (100 MB–1 GB) · `large` (≥1 GB) |
| `{category}` | `images` | Category name from config |

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
    source:
      path: C:\Users\johndoe\Downloads
      extensions: [jpg, jpeg, png, gif, bmp, webp]
      filter:
        ignore:
          - screenshot_*
          - "*_temp.*"
        min-age: 24h
    destination:
      path: C:\Users\johndoe\images
      conflict-strategy: rename
      organize-by: "{ext}"         # files go to images/jpg/, images/png/, etc.

  - name: videos
    source:
      path: C:\Users\johndoe\Downloads
      extensions: [mp4, avi, mkv, mov, wmv]
      filter:
        min-size: 100MB
    destination:
      path: C:\Users\johndoe\videos
      conflict-strategy: overwrite
      organize-by: "{ext}"

  - name: dated-docs
    source:
      path: C:\Users\johndoe\Downloads
      extensions: [pdf, txt, log]
      filter:
        regex: '^\d{4}-\d{2}-\d{2}_.*'
    destination:
      path: C:\Users\johndoe\dated
      # organize-by not set — all files go directly into dated/

  - name: reports
    source:
      path: C:\Users\johndoe\Downloads
      extensions: [pdf, docx]
      filter:
        glob: "report_*"
    destination:
      path: C:\Users\johndoe\reports

  - name: everything-else
    source:
      path: C:\Users\johndoe\Downloads
      extensions: [all]            # matches any file extension
    destination:
      path: C:\Users\johndoe\sorted
      conflict-strategy: hash_check
      organize-by: "{ext}"         # organizes into sorted/jpg/, sorted/pdf/, etc.
```

## Splitting your config with imports

For large configs, you can split `categories` across multiple YAML files using the top-level `import:` key. Import paths are relative to the file that declares them. Circular imports are detected and reported as an error.

**`movelooper.yaml`** — main file (holds `configuration`, imports category files):

```yaml
configuration:
  output: console
  log-level: info
  watch-delay: 5m

import:
  - categories/media.yaml
  - categories/documents.yaml
  - categories/wallhaven.yaml
```

**`categories/wallhaven.yaml`** — imported file (only `categories`):

```yaml
categories:
  - name: wallhaven
    source:
      path: C:\Users\you\Downloads
      extensions: [jpg, png]
      filter:
        regex: "^wallhaven"
    destination:
      path: C:\Users\you\Walls\Wallhaven
      conflict-strategy: hash_check
```

Imported files can also have their own `import:` for nested splitting. Use `movelooper config show` to inspect the final merged configuration.

## Commands and Flags

### `movelooper` (default — move files once)

```bash
movelooper [flags]
```

| Flag           | Short | Description                        |
|----------------|-------|------------------------------------|
| `--dry-run`    |       | Show what would be moved without moving files |
| `--show-files` |       | List individual files detected     |
| `--config`     | `-c`  | Path to a custom config file       |
| `--version`    |       | Print the current version          |

### `movelooper watch` — real-time monitoring

Monitors all source directories and moves files as they appear, after they stabilize (controlled by `watch-delay`).

```bash
movelooper watch
movelooper watch --dry-run                        # preview matched files without moving
movelooper watch --config /path/to/movelooper.yaml
```

| Flag        | Description                                          |
|-------------|------------------------------------------------------|
| `--dry-run` | Log matched files with their intended destination without moving them |

### `movelooper undo` — revert a batch

```bash
movelooper undo                             # undo the most recent batch
movelooper undo --list                      # list all recorded batches
movelooper undo --dry-run                   # preview what would be restored
movelooper undo batch_1718000000            # undo a specific move batch
movelooper undo batch_1718000000 --dry-run  # preview a specific batch restore
movelooper undo watch_1718000000000000000   # undo a specific watch batch
```

| Flag        | Short | Description                                                    |
|-------------|-------|----------------------------------------------------------------|
| `--list`    | `-l`  | List all recorded batches                                      |
| `--dry-run` |       | Preview which files would be restored without moving any files |

### `movelooper init` — generate config

```bash
movelooper init -i                          # interactive wizard
movelooper init -t full                     # from template
movelooper init -o /path/to/custom.yaml     # custom output path
movelooper init -f                          # force overwrite existing config
```

| Flag           | Short | Description                                              |
|----------------|-------|----------------------------------------------------------|
| `--interactive`| `-i`  | Launch the interactive wizard                            |
| `--template`   | `-t`  | Template to use (default: `basic`)                       |
| `--output`     | `-o`  | Path to write the config file                            |
| `--force`      | `-f`  | Overwrite existing config file                           |

### `movelooper config validate` — validate config

Loads and validates the configuration file without moving any files.

```bash
movelooper config validate
movelooper config validate --config /path/to/movelooper.yaml
```

### `movelooper config show` — inspect active config

Prints the active configuration as resolved in memory after startup, including all defaults filled in. Useful for verifying what movelooper is actually using.

```bash
movelooper config show
movelooper config show --config /path/to/movelooper.yaml
```

### `movelooper self-update` — update the binary

Downloads the latest release from GitHub and replaces the current binary. The old binary is saved as `movelooper.exe.old` and cleaned up on the next run.

```bash
movelooper self-update
movelooper self-update --repo lucasassuncao/movelooper
```

## Tips

- Run with `--dry-run` first to preview actions before organizing real files — works on `movelooper`, `watch`, and `undo`.
- Use `watch --dry-run` to test your rules in real-time without moving anything.
- Use `undo --dry-run` to inspect what a restore would do before committing.
- Use `watch` mode to automatically keep your Downloads folder clean at all times.
- Use `undo --list` to inspect past operations and roll back any batch.
- Use `enabled: false` to temporarily pause a category without deleting it from the config.
- Use `source.extensions: [all]` with `destination.organize-by: "{ext}"` as a catch-all category that organizes any file by its real extension.
- Use `import:` to split a large config into per-category files — combine with `config show` to verify the merged result.
- Run `movelooper config show` to verify which configuration values are actually in effect.
- Run `movelooper config validate` to catch config errors before running the tool.
- Add `filter.ignore` patterns to skip screenshots, drafts, or temp files from being moved.
- Use `filter.glob` for simple name patterns (`report_*.pdf`) and `filter.regex` for complex ones (`^\d{4}-.*`).
- Use `filter.min-age` to avoid moving files that are still being downloaded.
- Add `movelooper watch` to a cron job or Windows Task Scheduler for fully automatic cleanup.
- Run `movelooper self-update` to always stay on the latest release.
