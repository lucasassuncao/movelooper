# Configuration Reference

## `configuration` block

Global settings grouped into four sub-sections: `logging`, `watch`, `history`, and `defaults`.

### `logging`

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `output` | string | no | `console` | Where to write logs: `console`, `file`, or `both` |
| `level` | string | no | `info` | Log verbosity: `trace`, `debug`, `info`, `warn`, `error`, `fatal` |
| `file` | string | no | — | Path to the log file (required when `output` is `file` or `both`) |
| `show-caller` | bool | no | `false` | Include the source location in log lines |
| `format` | string | no | `pretty` | `pretty` for the human console renderer; `json` for structured slog lines |
| `color` | string | no | `auto` | ANSI color for `pretty`: `auto` (console only), `always`, `never`. Ignored for `json` |
| `max-width` | int | no | `70` | Column width for wrapping `pretty` lines (20–500). Ignored for `json` |

The global `--format` flag overrides `format` for a single run: `movelooper --format json` forces structured JSON regardless of the configured value.

### `watch`

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `delay` | duration | no | `5m` | How long a file must be stable before `watch` moves it (e.g. `30s`, `5m`) |
| `poll-interval` | duration | no | `5s` | How often watch re-checks pending files for stability (keep shorter than `delay`) |

See [Watch Mode](/WATCH.md) for how stability detection works, delay tuning, and running automatically.

### `history`

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `enabled` | bool | no | `true` | Whether move events are recorded for undo |
| `limit` | int | no | `100` | Maximum number of batches retained in undo history |
| `file` | string | no | `~/.movelooper/history/movelooper.json` | Path to the history JSON file (supports `~`) |

### `defaults` (optional)

Fallback destination settings applied to any category that omits them. Per-category values always win.

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `conflict-strategy` | string | no | — | Fallback for `destination.conflict-strategy` (same 8 allowed values) |
| `action` | string | no | — | Fallback for `destination.action`: `move`, `copy`, `symlink` |
| `organize-by` | string | no | — | Fallback for `destination.organize-by` template |

---

## `import` key

Split `categories` across multiple YAML files using the top-level `import:` key. Import paths are relative to the file that declares them. Circular imports are detected and reported as an error.

**`movelooper.yaml`** — main file:

```yaml
configuration:
  logging:
    output: console
    level: info
  watch:
    delay: 5m
  history:
    limit: 100

import:
  - categories/media.yaml
  - categories/documents.yaml
  - categories/wallhaven.yaml
```

**`categories/wallhaven.yaml`** — imported file (only `categories:`):

```yaml
categories:
  - name: wallhaven
    source:
      path: ~/Downloads
      extensions: [jpg, png]
      filter:
        match:
          regex: "^wallhaven"
    destination:
      path: ~/Walls/Wallhaven
      conflict-strategy: hash_check
```

Imported files can also have their own `import:` for nested splitting.
