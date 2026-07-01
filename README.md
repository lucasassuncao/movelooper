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
- Rich rename tokens: name transforms (`{name-slug}`, `{name-snake}`, `{name-upper}`, `{name-trunc:N}`, …), system info (`{hostname}`, `{username}`, `{os}`), time (`{hour}`, `{minute}`, `{timestamp}`), hashes (`{md5}`, `{sha256:N}`), and sequences (`{seq}`, `{seq-alpha}`, `{seq-roman}`)
- Wildcard `extensions: [all]` to catch any file type
- Conflict strategies per category: `rename`, `overwrite`, `skip`, `hash_check`, and more
- `action: copy` or `action: symlink` to back up or link files without moving them
- `action: archive` to pack a whole category into a single `.zip`/`.tar.gz` at the destination
- `rename` template to rename files at the destination using the same token engine

### Automate

- Watch mode: monitors directories in real-time, moves files as they stabilize
- Undo: interactive batch picker to select and revert a batch · pass a batch ID to skip the picker
- `--dry-run` on move and undo to preview before committing
- Hooks: run shell commands before and after each category — notify, log, call webhooks, or trigger scripts (one-shot `movelooper` run only, not `watch`)

### Configure

- Split config across multiple YAML files with `import:`
- `edit` — interactive TUI editor for the config file · `validate` to check all rules · `show-docs` to browse the field reference in the terminal
- Self-update with `self-update`

## How It Works

`movelooper` reads your configuration file (`movelooper.yaml` or `conf/movelooper.yaml`), scans all extensions listed per category, and processes matching files from the source to the destination.

The optional `organize-by` field controls how files are placed inside `<destination>/` using a template — for example `{ext}/{mod-year}/{mod-month}` places a `.jpg` modified in April 2025 into `<destination>/jpg/2025/04/`.

## Installation

Download the latest version from the [releases page](https://github.com/lucasassuncao/movelooper/releases), then extract the binary and add it to your system's PATH.

## First-time Setup

Create a `movelooper.yaml` config file and open it in the interactive editor:

```bash
movelooper edit
```

Or write the config manually — see [Config File Reference](docs/config.md) for all fields and a [full example](docs/config.md#full-example).

## Quick Example

```yaml
configuration:
  logging:
    output: console
    level: info
  watch:
    delay: 5m

categories:
  - name: images
    source:
      path: ~/Downloads
      extensions: [jpg, jpeg, png, webp]
      filter:
        not:
          - match:
              glob: "screenshot_*"
        age:
          min: 10m
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

- [Architecture Overview](docs/ARCHITECTURE.md) — how `movelooper` works, the config structure, and the token engine
- [Config File Reference](docs/CONFIG.md) — all fields, conflict strategies, actions, rename templates, tokens, full example, and config imports
- [Commands and Flags](docs/COMMANDS.md) — all CLI commands with flags and usage examples

## Tips

### Safety & Dry-run

- Run with `--dry-run` first to preview actions before organizing real files - works on `movelooper` and `undo`.
- Use `undo --dry-run` to inspect what a restore would do before committing.
- Use `undo --list` to inspect past operations and roll back any batch.

### Configuration

- Use `enabled: false` to temporarily pause a category without deleting it from the config.
- Use `import:` to split a large config into per-category files.
- Run `movelooper edit` to open the config in the interactive TUI editor — it validates on save.

### Filters

- Add `filter.not` patterns to skip screenshots, drafts, or temp files from being moved.
- Use `filter.age.min` to avoid moving files that are still being downloaded.
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
