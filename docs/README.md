# movelooper

![made with Go](https://img.shields.io/badge/made_with-Go-blue?logo=go) ![type CLI](https://img.shields.io/badge/type-CLI-green) ![license MIT](https://img.shields.io/badge/license-MIT-lightgrey)

**Movelooper** is a modern CLI tool that automatically organizes and moves your files based on configurable categories. No manual sorting, no chaos. Perfect for keeping your Downloads, Dev, or Media folders clean and structured.

## Features

### Organize

- Category-based rules: match files by extension, regex, glob, age, size, and real content type (`filter.mime: "image/*"`, magic bytes)
- Organize files into subdirectories using templates: `{ext}`, `{mod-year}`, `{mod-month}`, `{size-range}`, and [more](TOKENS.md)
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
- Hooks: run shell commands before and after each category — notify, log, call webhooks, or trigger scripts

### Configure

- Split config across multiple YAML files with `import:`
- `edit` — interactive TUI editor for the config file · `validate` to check all rules · `show-docs` to browse the field reference in the terminal
- Self-update with `self-update`

## Installation

Download the latest version from the [releases page](https://github.com/lucasassuncao/movelooper/releases), then extract the binary and add it to your system's PATH.

## First-time setup

```bash
movelooper edit
```

Or write the config manually — see [Configuration](CONFIGURATION.md) and [Categories](CATEGORIES.md) for all fields.

## Quick example

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
      action: copy
      rename: "{mod-date}_{name}.{ext}"
      conflict-strategy: skip
```

## Tips

**Safety:** run with `--dry-run` first to preview actions. Use `undo --list` to inspect past operations and roll back any batch.

**Filters:** add `filter.not` to skip screenshots, drafts, or temp files. Use `filter.age.min` to avoid moving files still being downloaded.

**Actions:** use `action: copy` to back up without removing originals. Use `action: symlink` to link into a media server folder.

**Automation:** add `movelooper watch` to a cron job or Windows Task Scheduler. Run `movelooper self-update` to stay on the latest release.

## Where to go next

- [Getting Started](/GETTING-STARTED.md) — install, first category, dry-run, undo
- [Configuration](/CONFIGURATION.md) — logging, watch delay, history, defaults
- [Categories](/CATEGORIES.md) — source, destination, hooks
- [Actions](/ACTIONS.md) — move, copy, symlink, archive
- [Conflicts](/CONFLICTS.md) — rename, overwrite, skip, hash_check, newest, oldest, larger, smaller
- [Tokens](/TOKENS.md) — full token reference for `organize-by` and `rename`
- [Filters](/FILTERS.md) — match, age, size, mime, and boolean composition
- [Hooks](/HOOKS.md) — before/after hooks, `ML_*` env vars, examples
- [Watch Mode](/WATCH.md) — stability detection, delay tuning, running automatically
- [Undo](/UNDO.md) — batch IDs, history file, limitations per action type
- [Cookbook](/COOKBOOK.md) — ready-to-use configs for common scenarios
- [Commands](/COMMANDS.md) — all CLI commands and flags
- [Editor (TUI)](/EDIT.md) — interactive config editor, keybindings, themes
- [FAQ](/FAQ.md) — common questions
