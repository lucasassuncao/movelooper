# Categories Reference

Each entry in the `categories` list defines a rule: which files to match and where to put them.

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `name` | string | yes | — | Label used in logs and undo history. Must be unique. |
| `enabled` | bool | no | `false` | Must be explicitly `true`; omitting the field disables the category |
| `source` | object | yes | — | Where to scan for files |
| `destination` | object | yes | — | Where to place files and how |
| `hooks` | object | no | — | Shell commands to run before/after processing |

---

## `source`

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `path` | string | yes | — | Directory to scan |
| `extensions` | []string | yes | — | Extensions to match (without dot). Use `["all"]` to match any file |
| `filter` | object | no | — | Additional filters (see [Filters](/FILTERS.md)) |
| `recursive` | bool | no | `false` | Scan subdirectories recursively |
| `max-depth` | int | no | `0` | Max recursion depth; `0` = unlimited (only used with `recursive: true`) |
| `exclude-paths` | []string | no | `[]` | Absolute paths to skip during recursive walk. The destination is always auto-excluded |

---

## `destination`

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `path` | string | yes | — | Root directory where matched files are placed |
| `action` | string | no | `move` | File operation: `move`, `copy`, `symlink`, or `archive` |
| `organize-by` | string | no | — | Token template for sub-directories under `path` (see [Tokens](/TOKENS.md)) |
| `rename` | string | no | — | Token template for the destination filename (see [Tokens](/TOKENS.md)). Empty = keep original |
| `conflict-strategy` | string | no | `rename` | What to do when a file already exists at the destination |
| `archive` | object | no* | — | Required when `action: archive` |

### Conflict strategies

| Value | Behavior |
|---|---|
| `rename` | Appends `(1)`, `(2)`, … until the name is unique (default) |
| `overwrite` | Replaces the existing file |
| `skip` | Leaves the source file untouched |
| `hash_check` | Compares SHA-256 hashes; skips if identical, renames if different |
| `newest` | Keeps the file with the most recent modification time |
| `oldest` | Keeps the file with the oldest modification time |
| `larger` | Keeps the larger file |
| `smaller` | Keeps the smaller file |

### Actions

| Value | Behavior |
|---|---|
| `move` | Moves the file, removing it from the source (default) |
| `copy` | Copies the file, leaving the original in place |
| `symlink` | Creates a symbolic link at the destination. On Windows requires elevated privileges |
| `archive` | Packs all matched files into one `.zip` or `.tar.gz`. Not available in `watch` mode; cannot be undone |

### `archive` block

Required when `action: archive`.

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `format` | string | yes | — | `zip` or `tar.gz` |
| `name` | string | no | category name | Archive filename base (extension added automatically). Supports: `{category}`, `{date}`, `{timestamp}`, `{hostname}`, `{username}`, `{os}` |
| `compression` | string | no | `best` | `none`, `fast`, or `best` |
| `keep-source` | bool | no | `true` | Keep originals; `false` deletes sources after a successful write |
| `flatten` | bool | no | `false` | Put all files at the archive root; `false` preserves sub-paths |

Only `rename`, `overwrite`, and `skip` conflict strategies apply to archive. The after-hook receives `ML_ARCHIVE_PATH`.

```yaml
destination:
  path: ~/Downloads/archives
  action: archive
  conflict-strategy: rename
  archive:
    format: zip
    name: "{category}_{date}"
    compression: best
    keep-source: true
```

---

## `hooks`

Optional shell commands to run before and/or after a category is processed. `before` can abort the move if it fails; `after` receives file counts and the batch ID.

> Hooks run only on the one-shot `movelooper` command, not in `watch` mode.

See [Hooks](/HOOKS.md) for the full reference: fields, all `ML_*` environment variables, platform notes, and examples.

---

## Full example

```yaml
configuration:
  logging:
    output: both
    level: info
    file: ~/.movelooper/logs/movelooper.log
  watch:
    delay: 5m
  history:
    limit: 100

categories:
  - name: images
    enabled: true
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
      organize-by: "{ext}"

  - name: videos
    enabled: true
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

  - name: photos-backup
    enabled: true
    source:
      path: ~/Downloads
      extensions: [jpg, jpeg, raw]
    destination:
      path: ~/Backup/photos
      action: copy
      rename: "{mod-date}_{name}.{ext}"
      conflict-strategy: skip

  - name: media-server-links
    enabled: true
    source:
      path: ~/Downloads
      extensions: [mp4, mkv, mov]
      filter:
        size:
          min: 500MB
    destination:
      path: ~/MediaServer/movies
      action: symlink
      organize-by: "{mod-year}"
      conflict-strategy: rename

  - name: everything-else
    enabled: true
    source:
      path: ~/Downloads
      extensions: [all]
    destination:
      path: ~/sorted
      conflict-strategy: hash_check
      organize-by: "{ext}"
```
