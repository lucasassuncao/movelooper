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
| `copy`, when the original is gone or the copy was edited | Keeps the copy and warns. See below |
| `symlink` | Removes the symbolic link at the destination. The source file is never touched |
| `archive` | **Cannot be undone.** Archive batches do not appear in undo history |

If the source file no longer exists at undo time, movelooper logs a warning and skips it. The rest of the batch is still restored.

## When undo refuses to delete a copy

Undoing a `copy` deletes the file at the destination, which reverses the run only while the original is still where it came from. Two cases turn that deletion into a real loss, and movelooper holds back on both:

- **The original is gone.** Someone deleted or moved it after the run, so the copy at the destination is the last one left.
- **The copy was modified after it was made.** Removing it would discard edits that exist nowhere else.

In both cases the file is kept, a warning names it, and the entry stays in history so you can act on it later.

```bash
movelooper undo batch_a1b2c3d4e5f6a7b8 --force
```

`--force` removes those copies anyway. It changes nothing else: the guards on `move` undo, which never overwrite a source path that something else now occupies, always apply.

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
