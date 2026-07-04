# Hooks

Hooks let you run shell commands before and/or after a category processes files. Use them to send notifications, call webhooks, log to external systems, or trigger dependent scripts.

> Hooks run only on the one-shot `movelooper` command. `movelooper watch` logs a warning at startup for any category that defines hooks and skips them.

---

## Structure

```yaml
hooks:
  before:
    shell: bash
    on-failure: abort
    run:
      - echo "Starting $ML_CATEGORY..."
  after:
    shell: bash
    on-failure: warn
    run:
      - echo "$ML_FILES_MOVED files moved (batch $ML_BATCH_ID)"
```

Both `before` and `after` are optional and independent.

| Field | Required | Values | Description |
|---|---|---|---|
| `shell` | no | any executable | Shell to run commands. Defaults to `$SHELL` on Unix/macOS, `cmd` on Windows. Use `pwsh` for PowerShell Core. |
| `on-failure` | yes | `abort`, `warn` | What to do when a command exits non-zero. `abort` stops the sequence; `warn` logs and continues. |
| `run` | yes | list of strings | Commands executed in order, each as a separate shell invocation. |

### Lifecycle

- `before` runs before any files are processed. If it fails with `on-failure: abort`, the entire category is skipped — no files are moved.
- `after` runs after all files have been processed. If it fails, the files are already moved; only the hook result is affected.
- Each command in `run` is a separate shell invocation. A non-zero exit from any command triggers `on-failure`.

---

## Environment variables

### Available in both `before` and `after`

| Variable | Description |
|---|---|
| `ML_CATEGORY` | Category name as configured |
| `ML_SOURCE_PATH` | Source directory path |
| `ML_DEST_PATH` | Destination root path |
| `ML_DRY_RUN` | `true` when running with `--dry-run`, `false` otherwise |
| `ML_ACTION` | `move`, `copy`, or `symlink` |

### Available only in `after`

| Variable | Description |
|---|---|
| `ML_FILES_MOVED` | Number of files successfully processed |
| `ML_FILES_SKIPPED` | Number of files skipped (e.g. `conflict-strategy: skip`) |
| `ML_FILES_FAILED` | Number of files that failed to process |
| `ML_BATCH_ID` | Batch ID — pass to `movelooper undo <id>` to revert this specific batch |
| `ML_ARCHIVE_PATH` | Path to the created archive (only when `action: archive`) |

---

## Platform notes

**Windows — PowerShell Core (`pwsh`):** invoked with `-NonInteractive -NoProfile` automatically. Use `$env:ML_*` syntax.

**Windows — `cmd`:** use `%ML_CATEGORY%` syntax. Multi-line scripts are not supported; use `pwsh` instead.

**Linux / macOS — `bash` / `zsh`:** use `$ML_*` syntax. Multi-line scripts with YAML block scalars (`|`) work naturally.

---

## Examples

### Notify on completion (Linux/macOS)

```yaml
hooks:
  after:
    shell: bash
    on-failure: warn
    run:
      - |
        if [ "$ML_FILES_MOVED" -gt 0 ]; then
          notify-send "movelooper" "$ML_FILES_MOVED file(s) moved (batch $ML_BATCH_ID)"
        fi
```

### Notify on completion (Windows)

```yaml
hooks:
  after:
    shell: pwsh
    on-failure: warn
    run:
      - |
        if ($env:ML_FILES_MOVED -gt 0) {
          Add-Type -AssemblyName System.Windows.Forms
          [System.Windows.Forms.MessageBox]::Show(
            "$($env:ML_FILES_MOVED) file(s) moved (batch $($env:ML_BATCH_ID))",
            "movelooper"
          )
        }
```

### Call a webhook after every run

```yaml
hooks:
  after:
    shell: bash
    on-failure: warn
    run:
      - |
        curl -s -X POST https://example.com/webhook \
          -H "Content-Type: application/json" \
          -d "{\"category\":\"$ML_CATEGORY\",\"moved\":$ML_FILES_MOVED,\"batch\":\"$ML_BATCH_ID\"}"
```

### Abort if a required directory is missing

```yaml
hooks:
  before:
    shell: bash
    on-failure: abort
    run:
      - test -d "/Volumes/Backup" || (echo "Backup volume not mounted" && exit 1)
```

### Log to a file

```yaml
hooks:
  after:
    shell: bash
    on-failure: warn
    run:
      - |
        echo "$(date -Iseconds) [$ML_CATEGORY] moved=$ML_FILES_MOVED skipped=$ML_FILES_SKIPPED failed=$ML_FILES_FAILED batch=$ML_BATCH_ID" \
          >> ~/movelooper-audit.log
```

### Auto-undo on post-move validation failure

```yaml
hooks:
  after:
    shell: bash
    on-failure: warn
    run:
      - |
        if ! /usr/local/bin/validate-files.sh "$ML_DEST_PATH"; then
          echo "Validation failed — undoing batch $ML_BATCH_ID"
          movelooper undo "$ML_BATCH_ID"
        fi
```

### Dry-run awareness

The `ML_DRY_RUN` variable lets you skip side effects during previews:

```yaml
hooks:
  after:
    shell: bash
    on-failure: warn
    run:
      - |
        if [ "$ML_DRY_RUN" = "false" ] && [ "$ML_FILES_MOVED" -gt 0 ]; then
          curl -s https://example.com/webhook -d "moved=$ML_FILES_MOVED"
        fi
```
