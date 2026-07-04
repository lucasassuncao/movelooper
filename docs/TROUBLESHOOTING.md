# Troubleshooting

Common problems and how to fix them.

---

## Files are not being moved

Work through this checklist in order:

**1. Run with `--dry-run --show-files` to see what movelooper detects:**

```bash
movelooper --dry-run --show-files
```

If the file appears in the output, it would be moved on a real run. If it does not appear, continue below.

**2. Check that the category is enabled.**

```yaml
categories:
  - name: images
    enabled: true    # must be true
```

Run `movelooper --include-disabled` to temporarily process disabled categories.

**3. Check the file extension.**

Extensions in the config are matched without the leading dot. `jpg` matches `photo.jpg`, not `.jpg`. `extensions: [all]` matches every file.

**4. Check `filter.age.min`.**

If your config has `filter.age.min: 10m`, files modified less than 10 minutes ago are skipped. This is intentional — it avoids moving files that are still downloading. Wait and retry, or lower the value for testing.

**5. Check that `source.path` exists and is correct.**

```bash
movelooper validate --strict    # verifies that source and destination paths exist on disk
```

**6. Check if the file was already claimed by an earlier category.**

A file is processed by the **first** matching category in the list. If an earlier category already claimed it (or skipped it), later categories won't see it. Re-order categories or use `--category <name>` to run a specific one:

```bash
movelooper --dry-run --category images --show-files
```

---

## Watch mode is not moving files

**1. Is the process still running?**

```bash
# Check if movelooper is running
ps aux | grep movelooper          # Linux/macOS
Get-Process movelooper            # Windows PowerShell
```

If it stopped, restart it. Consider setting up a [systemd service or launchd agent](COOKBOOK.md#8-running-automatically-set-and-forget) so it restarts automatically.

**2. Is `watch.delay` too high?**

`watch.delay` (default `5m`) is how long a file must be stable (no new writes) before watch moves it. If a file was just dropped and you waited less than `delay`, it hasn't been picked up yet. Check your config:

```yaml
configuration:
  watch:
    delay: 5m    # lower this for testing, e.g. 30s
```

**3. Does the category use `action: archive`?**

Archive categories are skipped in watch mode — movelooper prints a warning at startup for each one. Use the one-shot `movelooper` command for archive categories.

**4. Do the categories have hooks?**

Hooks (`before`/`after`) run only on the one-shot `movelooper` command. In watch mode, hooks are ignored silently.

---

## Config file not found

movelooper looks for its config in this order:

1. Path passed via `--config /path/to/file.yaml`
2. `./movelooper.yaml` (current directory)
3. `./conf/movelooper.yaml` (current directory)

To see exactly which file would be loaded:

```bash
movelooper config
```

If you always run movelooper from a different directory than your config, use `--config` with an absolute path:

```bash
movelooper --config /home/youruser/movelooper.yaml
movelooper watch --config /home/youruser/movelooper.yaml
```

This is especially important for scheduled tasks (cron, systemd, Task Scheduler), where the working directory is not your home folder.

---

## Undo did not restore my files

**Undoing a `copy` batch removes the copy, it does not restore anything.**

When `action: copy` is used, the source file was never moved — it is still in the original location. Undoing a copy batch removes the copy that was created at the destination.

**Undoing a `symlink` batch removes the symlink, not the target.**

The file the symlink pointed to is not touched.

**Archive batches cannot be undone.**

`action: archive` packs files into a zip or tar.gz and removes the originals. This operation is not reversible through `movelooper undo`.

**The batch is not in the history list.**

History is stored in `~/.movelooper/history/movelooper.json` by default. If this file was deleted, or if `history.enabled: false` is set in your config, no batches were recorded.

```bash
movelooper undo --list    # see all recorded batches
```

---

## Validate reports errors I don't understand

Run validate with `--format table` for a cleaner view:

```bash
movelooper validate --format table
```

Common validation errors:

| Error | Cause | Fix |
|---|---|---|
| `source.path is required` | No path set on the source block | Add `path:` under `source:` |
| `destination.path is required` | No path set on the destination block | Add `path:` under `destination:` |
| `unknown conflict strategy` | Typo in `conflict-strategy` value | Valid values: `rename`, `overwrite`, `skip`, `hash_check` |
| `unknown action` | Typo in `action` value | Valid values: `move`, `copy`, `symlink`, `archive` |
| `extensions must not be empty` | Empty `extensions:` list | Add at least one extension, or use `[all]` |
| `invalid filter.age duration` | Duration format wrong | Use Go duration format: `10m`, `2h`, `1h30m` |

---

## The wrong shell is being used for hooks

movelooper defaults to `$SHELL` on Linux/macOS and `cmd` on Windows. Set `shell:` explicitly to avoid surprises:

```yaml
hooks:
  before:
    shell: bash      # Linux/macOS
    # shell: pwsh   # Windows PowerShell Core
    # shell: cmd    # Windows Command Prompt
    run:
      - echo "hello"
```

On Windows with PowerShell, use `$env:ML_CATEGORY` syntax instead of `$ML_CATEGORY`:

```powershell
Write-Host "Moving files for $($env:ML_CATEGORY)"
```

---

## Still stuck?

- Run `movelooper show-docs` to browse the full field reference in the terminal
- Check [Configuration](CONFIGURATION.md) and [Categories](CATEGORIES.md) for all fields and their defaults
- Open an issue at [github.com/lucasassuncao/movelooper](https://github.com/lucasassuncao/movelooper/issues)
