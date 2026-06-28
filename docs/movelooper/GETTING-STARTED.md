# Getting Started with movelooper

This guide takes you from zero to a working config in a few minutes. By the end you will have at least one category moving files automatically, know how to preview before committing, and know how to undo if something goes wrong.

---

## 1. Install

Download the binary for your platform from the [releases page](https://github.com/lucasassuncao/movelooper/releases), extract it, and place it somewhere on your `PATH`.

Verify the installation:

```bash
movelooper --version
```

---

## 2. Create a config file

movelooper looks for its config at `movelooper.yaml` in the current directory, or at `conf/movelooper.yaml`. You can also point to any file with `--config`.

The fastest way to create one is to open the interactive editor:

```bash
movelooper edit
```

The editor validates on save, so you can iterate quickly without leaving the terminal. Alternatively, write the file by hand — the rest of this guide shows exactly what to put in it.

---

## 3. Config structure

A `movelooper.yaml` has two top-level blocks:

```yaml
configuration:   # global settings: logging, watch delay, history
  logging:
    output: console
    level: info
  watch:
    delay: 5m

categories:      # list of rules — one category per set of files to organize
  - name: images
    enabled: true
    source: ...
    destination: ...
```

Every category is independent. A file is processed by the first category it matches in the list.

---

## 4. Your first category

A category needs at minimum: a `name`, `enabled: true`, a `source.path`, `source.extensions`, and a `destination.path`.

```yaml
categories:
  - name: images
    enabled: true
    source:
      path: ~/Downloads
      extensions: [jpg, jpeg, png, webp, gif]
    destination:
      path: ~/Pictures
      conflict-strategy: rename     # append (1), (2)… if the file already exists
```

This moves every `.jpg`, `.jpeg`, `.png`, `.webp`, and `.gif` from `~/Downloads` to `~/Pictures`. If a file with the same name already exists at the destination, the incoming file is renamed with a counter suffix (`photo(1).jpg`, `photo(2).jpg`, …).

---

## 5. Validate the config

Before running, check that your config is valid:

```bash
movelooper validate
```

This catches unknown fields, missing required values, and filter mistakes. Fix any errors reported before continuing.

---

## 6. Dry run first

Always preview before moving real files:

```bash
movelooper --dry-run
```

This prints where each file *would* go without touching anything. Add `--show-files` to also list each file's name:

```bash
movelooper --dry-run --show-files
```

---

## 7. Run it

When the dry-run output looks right, run without the flag:

```bash
movelooper
```

movelooper logs a summary: how many files were moved per category and the total size. Every move is recorded in the history file so you can undo it.

To run only specific categories, use `--category`:

```bash
movelooper --category images
movelooper --category images,docs
```

---

## 8. Add more categories

### Organize into subdirectories

Use `organize-by` to build sub-folders inside the destination automatically:

```yaml
- name: photos
  enabled: true
  source:
    path: ~/Downloads
    extensions: [jpg, jpeg, raw, heic]
  destination:
    path: ~/Pictures
    organize-by: "{year}/{month}"      # ~/Pictures/2025/04/
    conflict-strategy: hash_check      # skip if identical, rename if different
```

Common tokens for `organize-by`:

| Token | Expands to |
|---|---|
| `{year}` | 4-digit modification year |
| `{month}` | 2-digit modification month |
| `{day}` | 2-digit modification day |
| `{ext}` | file extension |

### Rename files at the destination

Use `rename` to apply a template to the filename:

```yaml
destination:
  path: ~/Videos/sorted
  rename: "{year}-{month}-{day}_{name}"   # clip.mp4 → 2025-04-16_clip.mp4
```

### Filter by name, age, or size

```yaml
source:
  path: ~/Downloads
  extensions: [pdf]
  filter:
    age:
      min: 10m          # skip files modified less than 10 minutes ago (still downloading)
    match:
      glob: "invoice_*" # only files whose name starts with "invoice_"
```

### Combine filters with any / all / not

```yaml
filter:
  not:
    - match:
        glob: "*_draft*"     # never move drafts
  any:
    - match:
        glob: "invoice_*"
    - match:
        glob: "receipt_*"
```

See [Filter evaluation](ARCHITECTURE.md#filter-evaluation) for the full logic.

### Copy or symlink instead of move

```yaml
destination:
  path: ~/Backup/docs
  action: copy       # keep the original in ~/Downloads
```

```yaml
destination:
  path: ~/MediaServer/movies
  action: symlink    # create a link without duplicating the file
```

---

## 9. Watch mode

Watch mode monitors source directories in real-time and moves files as they stabilize (i.e. after no new writes for `watch.delay`). It is useful for keeping your Downloads folder clean automatically.

```bash
movelooper watch
```

To start watch mode automatically at login, add `movelooper watch` to your shell profile, a cron job, or a systemd user service.

> Hooks (`before`/`after`) do not run in watch mode — they run only on the one-shot `movelooper` command.

### Stability delay

`watch.delay` (default `5m`) controls how long a file must be unchanged before watch moves it. Increase it if you work with large files that take a while to download:

```yaml
configuration:
  watch:
    delay: 10m
```

---

## 10. Undo

Every move is recorded in the history file. To revert the last operation:

```bash
movelooper undo
```

This opens an interactive batch picker. Select a batch with the arrow keys and press Enter to restore. Use `--dry-run` to preview first:

```bash
movelooper undo --dry-run
```

To undo only a specific category within a batch:

```bash
movelooper undo --category images
```

---

## 11. Split a large config with imports

When your config grows, split it into separate files:

```yaml
# movelooper.yaml
configuration:
  logging:
    output: console
    level: info

import:
  - conf/images.yaml
  - conf/docs.yaml
  - conf/videos.yaml
```

Each imported file is a standalone YAML file with a `categories:` block. movelooper merges them at load time.

---

## Where to go next

- [Architecture](ARCHITECTURE.md) — package diagram, one-shot and watch-mode flows, category model, filter tree, conflict strategies
- [Config File Reference](../CONFIG.md) — every field with types, defaults, and examples
- [Commands and Flags](../COMMANDS.md) — all CLI commands and flags
- [Attribute reference](attributes/configuration/configuration.md) — browsable field docs (also available in the terminal via `movelooper show-docs`)
