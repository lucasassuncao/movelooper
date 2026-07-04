# Actions

The `action` field in `destination` controls what movelooper does with each matched file.

| Value | Behavior |
|---|---|
| `move` | Moves the file, removing it from the source (default) |
| `copy` | Copies the file, leaving the original in place |
| `symlink` | Creates a symbolic link at the destination pointing to the source file |
| `archive` | Packs all matched files into one `.zip` or `.tar.gz` |

---

## `move`

Default action. The file is moved from source to destination and removed from the source.

```yaml
destination:
  path: ~/sorted
  action: move  # optional — this is the default
```

## `copy`

Copies the file to the destination. The original stays in the source directory. Useful for backups or when you want the same file in two places.

```yaml
destination:
  path: ~/Backup/photos
  action: copy
  conflict-strategy: skip
```

## `symlink`

Creates a symbolic link at the destination pointing to the source file. The original is not moved or copied.

> On Windows, creating symlinks requires elevated privileges (run as administrator or enable Developer Mode).

```yaml
destination:
  path: ~/MediaServer/movies
  action: symlink
  organize-by: "{mod-year}"
  conflict-strategy: rename
```

## `archive`

Packs all files matched by the category into a single `.zip` or `.tar.gz` archive at the destination. Requires an `archive:` block.

**Constraints:**
- Not available in watch mode
- Cannot be undone
- Only `rename`, `overwrite`, and `skip` conflict strategies apply. See [Conflict Strategies](/CONFLICTS.md)

### `archive` block

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `format` | string | yes | — | `zip` or `tar.gz` |
| `name` | string | no | category name | Archive filename base (extension added automatically). Supports: `{category}`, `{date}`, `{timestamp}`, `{hostname}`, `{username}`, `{os}` |
| `compression` | string | no | `best` | `none`, `fast`, or `best` |
| `keep-source` | bool | no | `true` | Keep originals; `false` deletes sources after a successful write |
| `flatten` | bool | no | `false` | Put all files at the archive root; `false` preserves sub-paths |

The after-hook receives `ML_ARCHIVE_PATH` with the path to the created archive.

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
