# Undo

Every run of `movelooper` records a batch in the history file. `movelooper undo` lets you restore files from any recorded batch — interactively or by batch ID.

---

## What is a batch?

A batch is the set of all moves made in a single run of `movelooper`. Each batch has a unique ID:

- One-shot runs: `batch_a1b2c3d4e5f6a7b8`
- Watch-mode runs: `watch_0f1e2d3c4b5a6978`

The batch ID is printed at the end of each run and in `undo --list`.

---

## Interactive picker

```bash
movelooper undo
```

Opens a picker listing all recorded batches. Use **↑ / ↓** to select, **Enter** to confirm, **Esc** to cancel.

## List recorded batches

```bash
movelooper undo --list
```

## Undo a specific batch

```bash
movelooper undo batch_a1b2c3d4e5f6a7b8
```

## Preview before restoring

```bash
movelooper undo --dry-run
movelooper undo batch_a1b2c3d4e5f6a7b8 --dry-run
```

Recommended before any large undo. Prints what would be restored without moving files.

## Partial undo (by category)

```bash
movelooper undo --category images
movelooper undo batch_a1b2c3d4e5f6a7b8 --category images,docs
```

Only entries from the specified categories are reverted. If the batch becomes empty after the partial undo, it is removed from history.

---

## Behavior by action type

| Action | What undo does |
|---|---|
| `move` | Moves the file back to its original source path |
| `copy` | Removes the copied file at the destination. The original is never touched |
| `symlink` | Removes the symbolic link at the destination. The source file is never touched |
| `archive` | **Cannot be undone.** Archive batches do not appear in undo history |

If the source file no longer exists at undo time, movelooper logs a warning and skips it. The rest of the batch is still restored.

---

## History file

Stored at `~/.movelooper/history/movelooper.json` by default. Configurable under `configuration.history`:

```yaml
configuration:
  history:
    limit: 100                                     # keep the last 100 batches (default)
    file: ~/.movelooper/history/movelooper.json    # custom path
    enabled: true                                  # set false to disable tracking entirely
```

When `limit` is reached, the oldest batches are evicted automatically.

---

## Flags

| Flag | Short | Description |
|---|---|---|
| `--list` | `-l` | List all recorded batches |
| `--dry-run` | | Preview which files would be restored |
| `--category` | | Comma-separated category names to undo (default: all) |

See [Commands](/COMMANDS.md) for the full flag reference including `--format json`.
