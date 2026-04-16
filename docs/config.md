# Config File Reference (`movelooper.yaml`)

## `configuration` block

| Field            | Type     | Required | Default   | Description                                                                        |
|------------------|----------|----------|-----------|------------------------------------------------------------------------------------|
| `output`         | string   | no       | `console` | Where to write logs: `console`, `file`, or `both`                                 |
| `log-file`       | string   | no       | —         | Path to the log file (required when `output` is `file` or `both`)                 |
| `log-level`      | string   | no       | `info`    | Log verbosity: `trace`, `debug`, `info`, `warn`, `error`, `fatal`                 |
| `show-caller`    | bool     | no       | `false`   | Include the source location in log lines                                           |
| `watch-delay`    | duration | no       | `5m`      | How long a file must be stable before `watch` moves it (e.g. `30s`, `5m`)        |
| `history-limit`  | int      | no       | `50`      | Maximum number of batches retained in undo history                                 |

## `categories` block

Each entry in the `categories` list has the following top-level fields:

| Field         | Type   | Required | Default | Description                                            |
|---------------|--------|----------|---------|--------------------------------------------------------|
| `name`        | string | yes      | —       | Label for the category (used in logs and undo history) |
| `enabled`     | bool   | no       | `true`  | When `false`, the category is skipped in all modes     |
| `source`      | object | yes      | —       | Where to scan for files (see below)                    |
| `destination` | object | yes      | —       | Where to move files and how (see below)                |

### `source` block

| Field        | Type     | Required | Description                                                                        |
|--------------|----------|----------|------------------------------------------------------------------------------------|
| `path`       | string   | yes      | Directory to scan for files                                                        |
| `extensions` | []string | yes      | File extensions to match (without the dot). Use `["all"]` to match any extension  |
| `filter`     | object   | no       | Optional filters applied to files in this source (see below)                      |

### `source.filter` block

| Field      | Type      | Description                                                             |
|------------|-----------|-------------------------------------------------------------------------|
| `regex`    | string    | Regex filter applied to the filename. Mutually exclusive with `glob`    |
| `glob`     | string    | Glob filter applied to the filename. Mutually exclusive with `regex`    |
| `ignore`   | []string  | Glob patterns for filenames to skip (case-insensitive)                  |
| `min-age`  | duration  | Only move files older than this value (e.g. `24h`, `168h`)             |
| `max-age`  | duration  | Only move files newer than this value (e.g. `720h`, `8760h`)           |
| `min-size` | string    | Only move files at least this large (e.g. `500KB`, `10MB`, `1GB`)      |
| `max-size` | string    | Only move files at most this large (e.g. `10MB`, `1GB`)                |
| `any`      | []filter  | OR between groups — file passes if at least one group matches           |
| `all`      | []filter  | AND between groups — file passes only if every group matches            |

### Filter logic with `any` and `all`

By default, all fields in `filter` are combined with AND — a file must satisfy every condition. Use `any` or `all` for explicit boolean logic between groups.

**Rules:**
- `any` and `all` are mutually exclusive at the same level
- `any`/`all` and direct fields (`glob`, `regex`, `min-size`, etc.) cannot be mixed at the same level
- `any`/`all` must contain at least one entry
- Within each group, existing rules apply: `regex` and `glob` are mutually exclusive

#### Simple filter — AND (unchanged)

```yaml
filter:
  glob: "report_*"
  min-size: 500KB
  min-age: 1h
# Matches: glob AND min-size AND min-age
```

#### `any` — OR between groups

```yaml
filter:
  any:
    - glob: "report_*"
      min-size: 500KB
    - glob: "invoice_*"
      min-age: 1h
# Matches: (report_* AND >500KB) OR (invoice_* AND >1h old)
```

#### `all` — explicit AND between groups

```yaml
filter:
  all:
    - glob: "report_*"
    - min-size: 500KB
# Equivalent to simple filter — both conditions must pass
```

#### `any` inside `all` — most useful nested case

```yaml
filter:
  all:
    - min-size: 1MB
    - any:
        - glob: "report_*"
        - glob: "invoice_*"
# Matches: (report_* OR invoice_*) AND >1MB
```

#### `all` inside `any`

```yaml
filter:
  any:
    - all:
        - glob: "report_*"
        - min-size: 500KB
    - all:
        - regex: '^\d{4}-.*'
        - min-age: 24h
# Matches: (report_* AND >500KB) OR (date-prefixed name AND >24h old)
```

### `destination` block

| Field               | Type   | Required | Default  | Description                                                                                                      |
|---------------------|--------|----------|----------|------------------------------------------------------------------------------------------------------------------|
| `path`              | string | yes      | —        | Root directory where matched files are placed                                                                    |
| `action`            | string | no       | `move`   | Operation to perform: `move`, `copy`, or `symlink`                                                               |
| `rename`            | string | no       | —        | Template to rename the file at the destination (same tokens as `organize-by`). Leave empty to keep original name |
| `conflict-strategy` | string | no       | `rename` | What to do when a file already exists at the destination (see below)                                             |
| `organize-by`       | string | no       | —        | Template defining subdirectory structure under `path` (see tokens below). Leave empty to place files directly    |

### Conflict strategies

| Value        | Behavior                                                                              |
|--------------|---------------------------------------------------------------------------------------|
| `rename`     | Appends `(1)`, `(2)`, … to the filename until it is unique (default)                 |
| `overwrite`  | Deletes the existing destination file and replaces it                                 |
| `skip`       | Leaves the source file untouched                                                      |
| `hash_check` | Compares SHA-256 hashes; deletes source if identical, renames if different            |
| `newest`     | Keeps the file with the most recent modification time; skips source if dest is newer  |
| `oldest`     | Keeps the file with the oldest modification time; skips source if dest is older       |
| `larger`     | Keeps the larger file; skips source if dest is larger or equal                        |
| `smaller`    | Keeps the smaller file; skips source if dest is smaller or equal                      |

### Actions (`destination.action`)

| Value     | Behavior                                                                                                                                              |
|-----------|-------------------------------------------------------------------------------------------------------------------------------------------------------|
| `move`    | Moves the file from source to destination and removes it from the source (default)                                                                    |
| `copy`    | Copies the file to the destination, leaving the original in place                                                                                     |
| `symlink` | Creates a symbolic link at the destination pointing to the source file. On Windows, requires elevated privileges; failures are logged as warnings per file |

### Rename template (`destination.rename`)

Renames the file at the destination using a template. Supports the same tokens as `organize-by` (see table below). Leave empty to keep the original filename.

```yaml
destination:
  path: ~/Backup/photos
  action: copy
  rename: "{mod-date}_{name}.{ext}"   # e.g., photo.jpg → 2025-04-16_photo.jpg
```

### Filename filters (`filter.regex` and `filter.glob`)

Both fields narrow which files within a category are matched, in addition to `extensions`.
**They are mutually exclusive** — defining both in the same category is a configuration error.

- `filter.regex` accepts any valid Go regular expression and is matched against the full filename.
- `filter.glob` accepts a shell-style pattern (`*`, `?`) and supports brace expansion (`report_{2024,2025}_*`). Matching is case-insensitive.
- `filter.ignore` uses the same glob syntax as `filter.glob` and is always evaluated independently of both.

### `organize-by` tokens

These tokens are also valid in `destination.rename`.

| Token            | Example      | Description                                                                    |
|------------------|--------------|--------------------------------------------------------------------------------|
| `{ext}`          | `jpg`        | File extension, lowercase                                                      |
| `{ext-upper}`    | `JPG`        | File extension, uppercase                                                      |
| `{name}`         | `photo`      | Filename without extension                                                     |
| `{mod-year}`     | `2025`       | File modification year                                                         |
| `{mod-month}`    | `04`         | File modification month (zero-padded)                                          |
| `{mod-day}`      | `08`         | File modification day (zero-padded)                                            |
| `{mod-date}`     | `2025-04-08` | Shorthand for `{mod-year}-{mod-month}-{mod-day}`                               |
| `{mod-weekday}`  | `Tuesday`    | File modification weekday                                                      |
| `{created-year}` | `2025`       | File creation year (falls back to mod time on Linux)                           |
| `{created-month}`| `04`         | File creation month                                                            |
| `{created-day}`  | `08`         | File creation day                                                              |
| `{created-date}` | `2025-04-08` | Shorthand for `{created-year}-{created-month}-{created-day}`                   |
| `{year}`         | `2025`       | Run date year                                                                  |
| `{month}`        | `04`         | Run date month                                                                 |
| `{day}`          | `08`         | Run date day                                                                   |
| `{date}`         | `2025-04-08` | Run date shorthand                                                             |
| `{weekday}`      | `Tuesday`    | Run date weekday                                                               |
| `{size-range}`   | `small`      | `tiny` (<1 MB) · `small` (1–100 MB) · `medium` (100 MB–1 GB) · `large` (≥1 GB) |
| `{category}`     | `images`     | Category name from config                                                      |

## Full example

```yaml
configuration:
  output: both
  log-file: ~/.movelooper/logs/movelooper.log
  log-level: info
  show-caller: false
  watch-delay: 5m

categories:
  - name: images
    source:
      path: ~/Downloads
      extensions: [jpg, jpeg, png, gif, bmp, webp]
      filter:
        ignore:
          - screenshot_*
          - "*_temp.*"
        min-age: 24h
    destination:
      path: ~/images
      conflict-strategy: rename
      organize-by: "{ext}"         # files go to images/jpg/, images/png/, etc.

  - name: videos
    source:
      path: ~/Downloads
      extensions: [mp4, avi, mkv, mov, wmv]
      filter:
        min-size: 100MB
    destination:
      path: ~/videos
      conflict-strategy: overwrite
      organize-by: "{ext}"

  - name: dated-docs
    source:
      path: ~/Downloads
      extensions: [pdf, txt, log]
      filter:
        regex: '^\d{4}-\d{2}-\d{2}_.*'
    destination:
      path: ~/dated
      # organize-by not set — all files go directly into dated/

  - name: reports
    source:
      path: ~/Downloads
      extensions: [pdf, docx]
      filter:
        glob: "report_*"
    destination:
      path: ~/reports

  - name: everything-else
    source:
      path: ~/Downloads
      extensions: [all]            # matches any file extension
    destination:
      path: ~/sorted
      conflict-strategy: hash_check
      organize-by: "{ext}"         # organizes into sorted/jpg/, sorted/pdf/, etc.

  - name: photos-backup
    source:
      path: ~/Downloads
      extensions: [jpg, jpeg, raw]
    destination:
      path: ~/Backup/photos
      action: copy                          # keep original in Downloads
      rename: "{mod-date}_{name}.{ext}"    # e.g., photo.jpg → 2025-04-16_photo.jpg
      conflict-strategy: skip

  - name: media-server-links
    source:
      path: ~/Downloads
      extensions: [mp4, mkv, mov]
      filter:
        min-size: 500MB
    destination:
      path: ~/MediaServer/movies
      action: symlink                       # link without moving the file
      organize-by: "{mod-year}"
      conflict-strategy: rename
```

## Splitting your config with imports

For large configs, you can split `categories` across multiple YAML files using the top-level `import:` key. Import paths are relative to the file that declares them. Circular imports are detected and reported as an error.

**`movelooper.yaml`** — main file (holds `configuration`, imports category files):

```yaml
configuration:
  output: console
  log-level: info
  watch-delay: 5m

import:
  - categories/media.yaml
  - categories/documents.yaml
  - categories/wallhaven.yaml
```

**`categories/wallhaven.yaml`** — imported file (only `categories`):

```yaml
categories:
  - name: wallhaven
    source:
      path: ~/Downloads
      extensions: [jpg, png]
      filter:
        regex: "^wallhaven"
    destination:
      path: ~/Walls/Wallhaven
      conflict-strategy: hash_check
```

Imported files can also have their own `import:` for nested splitting. Use `movelooper config show` to inspect the final merged configuration.

---

[← Back to README](../README.md)
