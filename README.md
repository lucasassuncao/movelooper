<!-- markdownlint-disable MD033 -->
<p align="center">
  <img src="docs/pages/movelooper2.png" alt="Movelooper logo" width="300" height="300">
</p>
<!-- markdownlint-enable MD033 -->

🌀 **Movelooper** is a modern CLI tool that automatically organizes and moves your files based on configurable categories. No manual sorting, no chaos. Perfect for keeping your Downloads, Dev, or Media folders clean and structured.

![made with Go](https://img.shields.io/badge/made_with-Go-blue?logo=go) ![type CLI](https://img.shields.io/badge/type-CLI-green) ![license MIT](https://img.shields.io/badge/license-MIT-lightgrey)

## Features

### Organize

- Category-based rules: match files by extension, regex, glob, age, and size
- Organize files into subdirectories using templates: `{ext}`, `{mod-year}`, `{mod-month}`, `{size-range}`, and [more](docs/config.md#organize-by-tokens)
- Wildcard `extensions: [all]` to catch any file type
- Conflict strategies per category: `rename`, `overwrite`, `skip`, `hash_check`, and more
- `action: copy` or `action: symlink` to back up or link files without moving them
- `rename` template to rename files at the destination using the same token engine

### Automate

- Watch mode: monitors directories in real-time, moves files as they stabilize
- Undo: revert the last batch or any specific batch from history
- `--dry-run` on move, watch, and undo to preview before committing
- Hooks: run shell commands before and after each category — notify, log, call webhooks, or trigger scripts

### Configure

- Split config across multiple YAML files with `import:`
- Interactive setup wizard (`init -i`) or generate from a template (`init -t full`)
- `config show` to inspect the merged config · `config validate` to catch errors early
- Self-update with `self-update`

## How It Works

`movelooper` reads your configuration file (`movelooper.yaml` or `conf/movelooper.yaml`), scans all extensions listed per category, and processes matching files from the source to the destination.

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

## Quick Example

```yaml
configuration:
  output: console
  log-level: info
  watch-delay: 5m

categories:
  - name: images
    source:
      path: ~/Downloads
      extensions: [jpg, jpeg, png, webp]
      filter:
        ignore: ["screenshot_*"]
        min-age: 10m
    destination:
      path: ~/Images
      conflict-strategy: rename
      organize-by: "{ext}"

  - name: photos-backup
    source:
      path: ~/Downloads
      extensions: [jpg, jpeg, raw]
    destination:
      path: ~/Backup/photos
      action: copy                          # keep original in Downloads
      rename: "{mod-date}_{name}.{ext}"    # photo.jpg → 2025-04-16_photo.jpg
      conflict-strategy: skip
    hooks:
      before:
        shell: bash
        on-failure: warn
        run:
          - echo "Starting $ML_CATEGORY..."
      after:
        shell: bash
        on-failure: warn
        run:
          - |
            if [ "$ML_FILES_MOVED" -gt 0 ]; then
              echo "$ML_FILES_MOVED files moved. Batch: $ML_BATCH_ID"
            fi
```

## Documentation

- [Config File Reference](docs/config.md) — all fields, conflict strategies, actions, rename templates, tokens, full example, and config imports
- [Commands and Flags](docs/commands.md) — all CLI commands with flags and usage examples

## Tips

### Safety & Dry-run

- Run with `--dry-run` first to preview actions before organizing real files - works on `movelooper`, `watch`, and `undo`.
- Use `watch --dry-run` to test your rules in real-time without moving anything.
- Use `undo --dry-run` to inspect what a restore would do before committing.
- Use `undo --list` to inspect past operations and roll back any batch.

### Configuration

- Use `enabled: false` to temporarily pause a category without deleting it from the config.
- Use `import:` to split a large config into per-category files - combine with `config show` to verify the merged result.
- Run `movelooper config validate` to catch config errors before running the tool.

### Filters

- Add `filter.ignore` patterns to skip screenshots, drafts, or temp files from being moved.
- Use `filter.min-age` to avoid moving files that are still being downloaded.
- Use `source.extensions: [all]` with `destination.organize-by: "{ext}"` as a catch-all that organizes any file by its real extension.

### Actions & Rename

- Use `action: copy` to back up files without removing them from the source.
- Use `action: symlink` to link files into a media server folder without duplicating them.
- Use `rename: "{mod-date}_{name}.{ext}"` to timestamp files as they arrive at the destination.

### Automation

- Use `watch` mode to automatically keep your Downloads folder clean at all times.
- Add `movelooper watch` to a cron job or Windows Task Scheduler for fully automatic cleanup.
- Run `movelooper self-update` to always stay on the latest release.
- Use `hooks.after` with `$ML_BATCH_ID` to trigger an undo script if post-move validation fails.
- On Windows, set `shell: pwsh` and use `$env:ML_*` syntax; on Linux/macOS use `shell: bash` and `$ML_*`.
