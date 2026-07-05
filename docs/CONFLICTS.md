# Conflict Strategies

The `conflict-strategy` field in `destination` controls what happens when a file already exists at the target path.

| Value | Behavior |
|---|---|
| `rename` | Appends `(1)`, `(2)`, … until the name is unique (default) |
| `overwrite` | Replaces the existing file |
| `skip` | Leaves the source file untouched |
| `hash_check` | Compares SHA-256 hashes; skips if identical, renames if different content |
| `newest` | Keeps the file with the most recent modification time |
| `oldest` | Keeps the file with the oldest modification time |
| `larger` | Keeps the larger file |
| `smaller` | Keeps the smaller file |

---

## `rename`

Appends a counter to the filename until a unique name is found: `report.pdf` → `report (1).pdf` → `report (2).pdf`. This is the default strategy.

```yaml
destination:
  path: ~/Images
  conflict-strategy: rename
```

## `overwrite`

Replaces any existing file at the destination without warning. Use when you always want the latest version.

```yaml
destination:
  path: ~/Videos
  conflict-strategy: overwrite
```

## `skip`

Skips the file entirely if one already exists at the destination. The source file is left in place.

```yaml
destination:
  path: ~/Backup
  conflict-strategy: skip
```

## `hash_check`

Computes the SHA-256 hash of both files:
- **Identical content** → source is skipped (no duplicate stored). With `action: move` the duplicate source file is deleted (moving it would have consumed it anyway); with `action: copy` or `symlink` the source is left untouched.
- **Different content** → incoming file is renamed (like `rename`)

Useful for de-duplication: you avoid storing identical files twice without accidentally discarding genuinely different ones.

```yaml
destination:
  path: ~/sorted
  conflict-strategy: hash_check
```

## `newest`

Compares modification times and keeps whichever file is newer. If the source is newer, it replaces the destination; otherwise the source is skipped.

```yaml
destination:
  path: ~/Photos
  conflict-strategy: newest
```

## `oldest`

Keeps whichever file is older. If the source is older than the existing destination file, it replaces it; otherwise the source is skipped.

```yaml
destination:
  path: ~/Archive
  conflict-strategy: oldest
```

## `larger`

Keeps the larger of the two files by byte size. Useful when you always want to preserve the higher-quality version.

```yaml
destination:
  path: ~/Videos
  conflict-strategy: larger
```

## `smaller`

Keeps the smaller of the two files by byte size.

```yaml
destination:
  path: ~/Compressed
  conflict-strategy: smaller
```

---

> For `action: archive`, only `rename`, `overwrite`, and `skip` apply. See [Actions](/ACTIONS.md) for details.
