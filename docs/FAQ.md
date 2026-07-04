# FAQ

## Does movelooper work on Windows?

Yes. All features work on Windows with one exception: `action: symlink` requires elevated privileges (run as administrator) or [Developer Mode](https://learn.microsoft.com/en-us/windows/apps/get-started/enable-your-device-for-development) enabled.

## What happens if the destination directory doesn't exist?

movelooper creates it automatically, including any intermediate directories.

## Can a category have multiple source paths?

Not within a single category. Each category has one `source.path`. To cover multiple sources, define multiple categories pointing to the same destination.

## Can I process subdirectories?

Yes. Set `recursive: true` in the source block. Optionally limit depth with `max-depth`.

```yaml
source:
  path: ~/Documents
  extensions: [pdf]
  recursive: true
  max-depth: 3
```

## What's the difference between `hash_check` and `skip`?

`skip` checks only whether a file with the same name already exists. `hash_check` also compares SHA-256 file content: if the content is identical it skips; if it's different it renames the incoming file. Use `hash_check` to avoid duplicates without risking silent data loss.

See [Conflict Strategies](/CONFLICTS.md) for all options.

## Can I undo individual files instead of whole batches?

The minimum undo unit is a category within a batch. You can narrow with `--category`, but not to a single file. All files from the selected categories in the batch are restored.

## What happens if the source file no longer exists at undo time?

movelooper logs a warning and skips it. The remaining files in the batch are still restored.

## I accidentally moved important files. What now?

Run `movelooper undo` immediately and select the batch from the interactive picker. Use `--dry-run` first to confirm what will come back.

See [Undo](/UNDO.md) for the full reference.

## Does watch mode process existing files or only new ones?

Only files that arrive (or are written to) after watch starts. Files already in the source directory at startup are not processed.

## How do I schedule movelooper to run automatically?

See [Watch Mode](/WATCH.md#running-automatically) for systemd, launchd, and Windows Task Scheduler setup.

## Can two categories match the same file?

A file is processed by the first matching category in the list and skipped by all subsequent ones. Category order matters.

## What does `enabled: false` do?

A category with `enabled: false` is completely ignored at runtime. To include it explicitly, pass `--include-disabled`.

## Can I validate my config without running?

Yes:

```bash
movelooper validate
movelooper validate --strict   # also checks that source/destination paths exist on disk
```

## Where is the history file?

At `~/.movelooper/history/movelooper.json` by default. You can change the path or retention limit under `configuration.history`. See [Undo](/UNDO.md).
