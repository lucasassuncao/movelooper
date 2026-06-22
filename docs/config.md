# Config File Reference (`movelooper.yaml`)

## `configuration` block

Grouped into four sub-sections: `logging`, `watch`, `history`, and the optional `defaults`.

### `logging`

| Field         | Type   | Required | Default   | Description                                                                          |
|---------------|--------|----------|-----------|--------------------------------------------------------------------------------------|
| `output`      | string | no       | `console` | Where to write logs: `console`, `file`, or `both`                                    |
| `level`       | string | no       | `info`    | Log verbosity: `trace`, `debug`, `info`, `warn`, `error`, `fatal`                    |
| `file`        | string | no       | —         | Path to the log file (required when `output` is `file` or `both`)                    |
| `show-caller` | bool   | no       | `false`   | Include the source location in log lines                                             |
| `format`      | string | no       | `pretty`  | `pretty` for the human console renderer; `json` for structured slog lines            |
| `color`       | string | no       | `auto`    | ANSI color for `pretty`: `auto` (console only), `always`, `never`. Ignored for `json` |
| `max-width`   | int    | no       | `70`      | Column width for wrapping `pretty` lines (20–500). Ignored for `json`                |

### `watch`

| Field           | Type     | Required | Default | Description                                                                       |
|-----------------|----------|----------|---------|-----------------------------------------------------------------------------------|
| `delay`         | duration | no       | `5m`    | How long a file must be stable before `watch` moves it (e.g. `30s`, `5m`)         |
| `poll-interval` | duration | no       | `5s`    | How often watch re-checks pending files for stability (keep shorter than `delay`) |

### `history`

| Field     | Type   | Required | Default                                 | Description                                                  |
|-----------|--------|----------|-----------------------------------------|--------------------------------------------------------------|
| `enabled` | bool   | no       | `true`                                  | Whether move events are recorded for undo                    |
| `limit`   | int    | no       | `100`                                   | Maximum number of batches retained in undo history           |
| `file`    | string | no       | `~/.movelooper/history/movelooper.json` | Path to the history JSON file used for undo (supports `~`)   |

### `defaults` (optional)

Fallback destination settings applied to any category that omits them. Per-category values always win.

| Field               | Type   | Required | Default | Description                                                          |
|---------------------|--------|----------|---------|---------------------------------------------------------------------|
| `conflict-strategy` | string | no       | —       | Fallback for `destination.conflict-strategy` (same 8 allowed values) |
| `action`            | string | no       | —       | Fallback for `destination.action`: `move`, `copy`, `symlink`        |
| `organize-by`       | string | no       | —       | Fallback for `destination.organize-by` template                     |

## `categories` block

Each entry in the `categories` list has the following top-level fields:

| Field         | Type   | Required | Default | Description                                            |
|---------------|--------|----------|---------|--------------------------------------------------------|
| `name`        | string | yes      | —       | Label for the category (used in logs and undo history). Must be unique across the list. |
| `enabled`     | bool   | no       | `true`  | When `false`, the category is skipped in all modes     |
| `source`      | object | yes      | —       | Where to scan for files (see below)                    |
| `destination` | object | yes      | —       | Where to move files and how (see below)                |

### `source` block

| Field        | Type     | Required | Description                                                                        |
|--------------|----------|----------|------------------------------------------------------------------------------------|
| `path`           | string   | yes      | Directory to scan for files                                                                                       |
| `extensions`     | []string | yes      | File extensions to match (without the dot). Use `["all"]` to match any extension                                 |
| `filter`         | object   | no       | Optional filters applied to files in this source (see below)                                                     |
| `recursive`      | bool     | no       | When `true`, scans subdirectories recursively (default: `false`)                                                  |
| `max-depth`      | int      | no       | Maximum recursion depth; `0` means unlimited (only used when `recursive: true`, default: `0`)                    |
| `exclude-paths`  | []string | no       | Absolute paths to skip during recursive walk. The destination path is always auto-excluded (default: `[]`)       |

### `source.filter` block

| Field   | Type     | Description                                                                                          |
|---------|----------|------------------------------------------------------------------------------------------------------|
| `match` | object   | Filename matching rules (see below). Sub-fields `literal`, `regex`, and `glob` are mutually exclusive |
| `age`   | object   | Age window: `min` and `max` as Go durations (e.g. `24h`, `168h`)                                    |
| `size`  | object   | Size window: `min` and `max` as human-readable strings (e.g. `500KB`, `10MB`)                       |
| `any`   | []filter | OR between groups — file passes if at least one group matches                                        |
| `all`   | []filter | AND between groups — file passes only if every group matches                                         |
| `not`   | []filter | Exclusion — file is rejected if any entry matches                                                    |

#### `filter.match` block

| Field            | Type   | Description                                                                          |
|------------------|--------|--------------------------------------------------------------------------------------|
| `literal`        | string | Exact filename match. Mutually exclusive with `regex` and `glob`                     |
| `regex`          | string | RE2 regular expression matched against the filename                                  |
| `glob`           | string | Shell-style glob pattern (`*`, `?`). Supports brace expansion (`report_{2024,2025}_*`) |
| `case-sensitive` | bool   | When `true`, matching is case-sensitive (default: `false`)                           |

#### `filter.age` block

| Field | Type     | Description                                                   |
|-------|----------|---------------------------------------------------------------|
| `min` | duration | Only move files older than this value (e.g. `24h`, `168h`)   |
| `max` | duration | Only move files newer than this value (e.g. `720h`, `8760h`) |

#### `filter.size` block

| Field | Type   | Description                                                                                                     |
|-------|--------|-----------------------------------------------------------------------------------------------------------------|
| `min` | string | Only move files at least this large (e.g. `500KB`, `10MB`, `1GB`). `KB`/`MB`/`GB`/`TB` are decimal (powers of 1000); `KiB`/`MiB`/`GiB`/`TiB` are binary (powers of 1024) |
| `max` | string | Only move files at most this large (e.g. `10MB`, `1GB`). Same units as `min`                                   |

### Size units

`size.min` and `size.max` accept two unit families, each with its standard
meaning. Suffixes are case-insensitive, decimals are allowed (`1.5MB`), and a
bare number (`"500"`) means bytes.

| Suffixes | Meaning | Example |
|----------|---------|---------|
| `KB`, `MB`, `GB`, `TB` | Decimal — powers of 1000 | `10MB` = 10 000 000 bytes |
| `KiB`, `MiB`, `GiB`, `TiB` | Binary — powers of 1024 | `10MiB` = 10 485 760 bytes |

The two can be mixed freely (everything is compared in bytes), but they are
**not** interchangeable — `10MiB` is ~4.9% larger than `10MB`. For a file of
10 200 000 bytes:

```yaml
filter:
  size:
    min: 10MB    # file IS moved   (10 200 000 ≥ 10 000 000)
# vs
filter:
  size:
    min: 10MiB   # file is NOT moved (10 200 000 < 10 485 760)
```

Rule of thumb: `MB` matches what disk vendors and network tools report;
`MiB` matches what Windows Explorer displays as "MB". Log output
(`size` in the run summary) always uses the decimal family.

> **Migration note:** earlier versions of movelooper treated `KB`/`MB`/`GB`/`TB`
> as binary (powers of 1024). If your config predates this change, switch those
> values to `KiB`/`MiB`/`GiB`/`TiB` to keep the exact same thresholds.

### Filter logic with `any` and `all`

By default, all fields in `filter` are combined with AND — a file must satisfy every condition. Use `any` or `all` for explicit boolean logic between groups.

**Rules:**
- `any` and `all` are mutually exclusive at the same level
- `any`/`all` and direct fields (`match`, `age`, `size`, `not`) cannot be mixed at the same level
- `any`/`all` must contain at least one entry
- Within each `match` block, `literal`, `regex`, and `glob` are mutually exclusive

#### Simple filter — AND (unchanged)

```yaml
filter:
  match:
    glob: "report_*"
  size:
    min: 500KB
  age:
    min: 1h
# Matches: match AND size.min AND age.min
```

#### `any` — OR between groups

```yaml
filter:
  any:
    - match:
        glob: "report_*"
      size:
        min: 500KB
    - match:
        glob: "invoice_*"
      age:
        min: 1h
# Matches: (report_* AND >500KB) OR (invoice_* AND >1h old)
```

#### `all` — explicit AND between groups

```yaml
filter:
  all:
    - match:
        glob: "report_*"
    - size:
        min: 500KB
# Equivalent to simple filter — both conditions must pass
```

#### `any` inside `all` — most useful nested case

```yaml
filter:
  all:
    - size:
        min: 1MB
    - any:
        - match:
            glob: "report_*"
        - match:
            glob: "invoice_*"
# Matches: (report_* OR invoice_*) AND >1MB
```

#### `all` inside `any`

```yaml
filter:
  any:
    - all:
        - match:
            glob: "report_*"
        - size:
            min: 500KB
    - all:
        - match:
            regex: '^\d{4}-.*'
        - age:
            min: 24h
# Matches: (report_* AND >500KB) OR (date-prefixed name AND >24h old)
```

#### `not` — exclude matching files

```yaml
filter:
  not:
    - match:
        glob: "screenshot_*"
    - match:
        glob: "*_temp.*"
# Moves all files except those matching screenshot_* or *_temp.*
```

#### `not` combined with `any`

```yaml
filter:
  not:
    - match:
        glob: "wallhaven*"
  age:
    min: 10m
# Moves files older than 10 minutes, excluding wallhaven* filenames
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

Use `{seq:N}` to number files sequentially at the destination:

```yaml
destination:
  path: ~/Images
  organize-by: "{ext}"
  rename: "{seq:4}_{name}.{ext}"      # e.g., photo.jpg → 0001_photo.jpg, 0002_photo.jpg
```

The counter is derived by scanning the destination directory for existing leading numbers — no external state file. Each subdirectory (when `organize-by` is set) maintains its own independent sequence.

### Filename matching (`filter.match`)

The `match` sub-block controls which filenames are accepted. `literal`, `regex`, and `glob` are mutually exclusive — specify exactly one.

- `match.literal` performs an exact filename match.
- `match.regex` accepts any valid Go (RE2) regular expression and is matched against the full filename.
- `match.glob` accepts a shell-style pattern (`*`, `?`) and supports brace expansion (`report_{2024,2025}_*`). Matching is case-insensitive by default.
- Set `match.case-sensitive: true` to enable case-sensitive matching for any of the three modes.

Use `not:` (a list of sub-filters) to exclude files that would otherwise match:

```yaml
filter:
  not:
    - match:
        glob: "screenshot_*"
```

### `organize-by` tokens

These tokens are also valid in `destination.rename` unless noted otherwise.

| Token              | Example            | Description                                                                      |
|--------------------|--------------------|----------------------------------------------------------------------------------|
| `{ext}`            | `jpg`              | File extension, lowercase                                                        |
| `{ext-upper}`      | `JPG`              | File extension, uppercase                                                        |
| `{ext-lower}`      | `jpg`              | File extension, lowercase (explicit alias of `{ext}`)                            |
| `{ext-reverse}`    | `gpj`              | File extension reversed                                                          |
| `{name}`           | `photo`            | Filename without extension                                                       |
| `{name-slug}`      | `my-file-name`     | Filename slugified: lowercase, spaces/specials → `-`                             |
| `{name-snake}`     | `my_file_name`     | Filename snake-cased: lowercase, spaces/specials → `_`                           |
| `{name-upper}`     | `PHOTO`            | Filename uppercased                                                              |
| `{name-lower}`     | `photo`            | Filename lowercased                                                              |
| `{name-trunc:N}`   | `phot` (N=4)       | Filename truncated to N runes; returned as-is if shorter. N: 1–255              |
| `{name-alpha}`     | `photo2025`        | Filename with only alphanumeric characters kept                                  |
| `{name-ascii}`     | `Acao_resume`      | Filename with accents/unicode normalized to ASCII equivalents                    |
| `{name-initials}`  | `mvp`              | First letter of each word (split on space, `-`, `_`)                             |
| `{name-reverse}`   | `otohp`            | Filename reversed                                                                |
| `{mod-year}`       | `2025`             | File modification year                                                           |
| `{mod-month}`      | `04`               | File modification month (zero-padded)                                            |
| `{mod-day}`        | `08`               | File modification day (zero-padded)                                              |
| `{mod-date}`       | `2025-04-08`       | Shorthand for `{mod-year}-{mod-month}-{mod-day}`                                 |
| `{mod-weekday}`    | `Tuesday`          | File modification weekday                                                        |
| `{created-year}`   | `2025`             | File creation year (falls back to mod time on Linux)                             |
| `{created-month}`  | `04`               | File creation month                                                              |
| `{created-day}`    | `08`               | File creation day                                                                |
| `{created-date}`   | `2025-04-08`       | Shorthand for `{created-year}-{created-month}-{created-day}`                     |
| `{year}`           | `2025`             | Run date year                                                                    |
| `{month}`          | `04`               | Run date month                                                                   |
| `{day}`            | `08`               | Run date day                                                                     |
| `{date}`           | `2025-04-08`       | Run date shorthand                                                               |
| `{weekday}`        | `Tuesday`          | Run date weekday                                                                 |
| `{hour}`           | `14`               | Run time hour (24h, zero-padded)                                                 |
| `{minute}`         | `35`               | Run time minute (zero-padded)                                                    |
| `{second}`         | `00`               | Run time second (zero-padded)                                                    |
| `{timestamp}`      | `20250416-143500`  | Run date+time compact: `YYYYMMDD-HHmmss`                                         |
| `{size-range}`     | `small`            | `tiny` (<1 MB) · `small` (1–100 MB) · `medium` (100 MB–1 GB) · `large` (≥1 GB) |
| `{category}`       | `images`           | Category name from config                                                        |
| `{hostname}`       | `lucas-pc`         | Machine hostname (`unknown` on failure)                                          |
| `{username}`       | `lucas`            | OS username (`unknown` on failure)                                               |
| `{os}`             | `windows`          | Operating system (`windows`, `linux`, `darwin`, …)                               |
| `{seq}`            | `1`                | Auto-incrementing sequence number (no padding). **`rename` only**                |
| `{seq:N}`          | `0001`             | Sequence number zero-padded to N digits (1 ≤ N ≤ 20). **`rename` only**         |
| `{seq-alpha}`      | `a`, `b`, `aa`     | Alphabetic sequence, Excel-style overflow (a→z→aa→ab…). **`rename` only**       |
| `{seq-roman}`      | `i`, `ii`, `iii`   | Roman numeral sequence. **`rename` only**                                        |
| `{md5}`            | `5d41402a`         | First 8 hex chars of file MD5. **`rename` only**                                 |
| `{md5:N}`          | `5d41402abc4b`     | First N hex chars of MD5 (1 ≤ N ≤ 32). **`rename` only**                        |
| `{sha256:N}`       | `2cf24dba5f`       | First N hex chars of SHA-256 (1 ≤ N ≤ 64). **`rename` only**                    |

### `hooks` block

Optional. Defines shell commands to run before and/or after the category is processed.

```yaml
hooks:
  before:
    shell: bash          # optional — defaults to $SHELL on Unix/Mac, cmd on Windows
    on-failure: abort    # abort | warn
    run:
      - echo "Starting $ML_CATEGORY..."
      - notify-send "Movelooper" "Processing $ML_CATEGORY"
  after:
    shell: pwsh
    on-failure: warn
    run:
      - Write-Host "$ML_FILES_MOVED files moved"
      - Invoke-RestMethod "https://example.com/webhook" -Method Post
```

Both `before` and `after` are optional and independent. If `before` fails with `on-failure: abort`, the category is skipped entirely (no files are moved). If `after` fails, files are already moved and the error is logged.

| Field | Required | Values | Description |
|---|---|---|---|
| `shell` | No | any executable | Shell used to run commands. Defaults to `$SHELL` (Unix/Mac) or `cmd` (Windows). Use `pwsh` or `powershell` for PowerShell on Windows. |
| `on-failure` | Yes | `abort`, `warn` | What to do when a command returns a non-zero exit code. `abort` stops the sequence; `warn` logs and continues. |
| `run` | Yes | list of strings | Commands executed in order, each as a separate shell invocation. |

#### Environment variables injected into hook commands

**Available in both `before` and `after`:**

| Variable | Description |
|---|---|
| `ML_CATEGORY` | Category name |
| `ML_SOURCE_PATH` | Source directory path |
| `ML_DEST_PATH` | Destination root path |
| `ML_DRY_RUN` | `true` or `false` |
| `ML_ACTION` | `move`, `copy`, or `symlink` |

**Available only in `after`:**

| Variable | Description |
|---|---|
| `ML_FILES_MOVED` | Files successfully processed |
| `ML_FILES_FAILED` | Files that failed to process |
| `ML_BATCH_ID` | Batch ID usable with `movelooper undo` |

All variables are prefixed with `ML_` to avoid collision with system environment variables.

#### Hook examples

**Windows — PowerShell Core (`pwsh`)**

```yaml
hooks:
  before:
    shell: pwsh
    on-failure: warn
    run:
      - 'Write-Output "[$env:ML_CATEGORY] Iniciando processamento..."'
  after:
    shell: pwsh
    on-failure: warn
    run:
      - |
        if ($env:ML_FILES_MOVED -gt 0) {
          Write-Output "[$env:ML_CATEGORY] $env:ML_FILES_MOVED arquivos movidos para $env:ML_DEST_PATH"
          Write-Output "Batch ID para undo: $env:ML_BATCH_ID"
        }
```

> **Note:** On Windows, `pwsh` and `powershell` are invoked with `-NonInteractive -NoProfile` automatically to avoid spawning new windows and prevent profile side effects. Use `$env:ML_*` syntax for environment variables.

**Linux / macOS — Bash**

```yaml
hooks:
  before:
    shell: bash
    on-failure: warn
    run:
      - 'echo "[$ML_CATEGORY] Iniciando processamento..."'
  after:
    shell: bash
    on-failure: warn
    run:
      - |
        if [ "$ML_FILES_MOVED" -gt 0 ]; then
          echo "[$ML_CATEGORY] $ML_FILES_MOVED arquivos movidos para $ML_DEST_PATH"
          echo "Batch ID para undo: $ML_BATCH_ID"
        fi
```

> **Note:** On Linux/macOS, use `$ML_*` syntax for environment variables. Multi-line scripts can be written with YAML block scalars (`|`).

## Full example

```yaml
configuration:
  logging:
    output: both
    level: info
    file: ~/.movelooper/logs/movelooper.log
    show-caller: false
  watch:
    delay: 5m
  history:
    limit: 100
    file: ~/.movelooper/history/movelooper.json

categories:
  - name: images
    source:
      path: ~/Downloads
      extensions: [jpg, jpeg, png, gif, bmp, webp]
      filter:
        not:
          - match:
              glob: "screenshot_*"
          - match:
              glob: "*_temp.*"
        age:
          min: 24h
    destination:
      path: ~/images
      conflict-strategy: rename
      organize-by: "{ext}"         # files go to images/jpg/, images/png/, etc.

  - name: videos
    source:
      path: ~/Downloads
      extensions: [mp4, avi, mkv, mov, wmv]
      filter:
        size:
          min: 100MB
    destination:
      path: ~/videos
      conflict-strategy: overwrite
      organize-by: "{ext}"

  - name: dated-docs
    source:
      path: ~/Downloads
      extensions: [pdf, txt, log]
      filter:
        match:
          regex: '^\d{4}-\d{2}-\d{2}_.*'
    destination:
      path: ~/dated
      # organize-by not set — all files go directly into dated/

  - name: reports
    source:
      path: ~/Downloads
      extensions: [pdf, docx]
      filter:
        match:
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
        size:
          min: 500MB
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
  logging:
    output: console
    level: info
  watch:
    delay: 5m
  history:
    limit: 100
    file: ~/.movelooper/history/movelooper.json

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
        match:
          regex: "^wallhaven"
    destination:
      path: ~/Walls/Wallhaven
      conflict-strategy: hash_check
```

Imported files can also have their own `import:` for nested splitting.

---

[← Back to README](../README.md)
