# Token Reference

Tokens are placeholders like `{year}` that expand to file or run metadata. They are used in two destination fields:

- `organize-by` — builds sub-directories inside the destination path
- `rename` — produces the full destination filename (include `{ext}` to keep the extension)

A subset of tokens is also available in `archive.name`.

---

## Availability

| Token | `organize-by` | `rename` | `archive.name` |
|---|:---:|:---:|:---:|
| `{name}`, `{ext}`, `{ext-upper}`, `{ext-lower}`, `{ext-reverse}` | ✓ | ✓ | — |
| `{name-slug}`, `{name-snake}`, `{name-upper}`, `{name-lower}` | ✓ | ✓ | — |
| `{name-alpha}`, `{name-ascii}`, `{name-initials}`, `{name-reverse}` | ✓ | ✓ | — |
| `{name-trunc:N}` | ✓ | ✓ | — |
| `{mod-year}`, `{mod-month}`, `{mod-day}`, `{mod-date}`, `{mod-weekday}` | ✓ | ✓ | — |
| `{created-year}`, `{created-month}`, `{created-day}`, `{created-date}` | ✓ | ✓ | — |
| `{year}`, `{month}`, `{day}`, `{date}`, `{weekday}` | ✓ | ✓ | ✓ |
| `{hour}`, `{minute}`, `{second}`, `{timestamp}` | ✓ | ✓ | ✓ |
| `{size-range}` | ✓ | ✓ | — |
| `{category}` | ✓ | ✓ | ✓ |
| `{hostname}`, `{username}`, `{os}` | ✓ | ✓ | ✓ |
| `{mime}`, `{mime-type}`, `{mime-ext}` | ✓ | — | — |
| `{seq}`, `{seq:N}`, `{seq-alpha}`, `{seq-roman}` | — | ✓ | — |
| `{md5}`, `{md5:N}`, `{sha256:N}` | — | ✓ | — |

---

## File identification

| Token | Expands to | Example |
|---|---|---|
| `{name}` | Filename without extension | `report` |
| `{ext}` | Extension, lowercase | `pdf` |
| `{ext-upper}` | Extension, uppercase | `PDF` |
| `{ext-lower}` | Extension, lowercase (alias for `{ext}`) | `pdf` |
| `{ext-reverse}` | Extension reversed | `fdp` |

---

## Name transforms

All transforms operate on the filename without extension.

| Token | Transform | `"My Résumé 2024"` → |
|---|---|---|
| `{name-slug}` | ASCII, lowercase, non-alphanumeric → `-` | `my-resume-2024` |
| `{name-snake}` | ASCII, lowercase, non-alphanumeric → `_` | `my_resume_2024` |
| `{name-upper}` | Uppercase | `MY RÉSUMÉ 2024` |
| `{name-lower}` | Lowercase | `my résumé 2024` |
| `{name-alpha}` | Strips non-alphanumeric characters | `MyRssum2024` |
| `{name-ascii}` | Strips diacritics and non-ASCII | `My Resume 2024` |
| `{name-initials}` | First letter of each word | `mr2` |
| `{name-reverse}` | Reverses the string | `4202 émuséR yM` |
| `{name-trunc:N}` | First N characters | `{name-trunc:5}` → `My Ré` |

---

## Modification date

Based on the file's last modification time.

| Token | Format | Example |
|---|---|---|
| `{mod-year}` | 4-digit year | `2025` |
| `{mod-month}` | 2-digit month | `04` |
| `{mod-day}` | 2-digit day | `16` |
| `{mod-date}` | `YYYY-MM-DD` | `2025-04-16` |
| `{mod-weekday}` | Day name | `Wednesday` |

---

## Creation date

Based on the file's creation time. On systems where creation time is unavailable, falls back to modification time.

| Token | Format | Example |
|---|---|---|
| `{created-year}` | 4-digit year | `2024` |
| `{created-month}` | 2-digit month | `11` |
| `{created-day}` | 2-digit day | `03` |
| `{created-date}` | `YYYY-MM-DD` | `2024-11-03` |

---

## Run date and time

Resolved to the moment movelooper runs, the same for every file in the batch.

| Token | Format | Example |
|---|---|---|
| `{year}` | 4-digit year | `2025` |
| `{month}` | 2-digit month | `04` |
| `{day}` | 2-digit day | `16` |
| `{date}` | `YYYY-MM-DD` | `2025-04-16` |
| `{weekday}` | Day name | `Wednesday` |
| `{hour}` | 24h hour | `15` |
| `{minute}` | Minute | `04` |
| `{second}` | Second | `05` |
| `{timestamp}` | `YYYYMMDD-HHmmss` | `20250416-150405` |

---

## Size range

| Token | Expands to | Range |
|---|---|---|
| `{size-range}` | `tiny` | < 1 MB |
| | `small` | 1 MB – 100 MB |
| | `medium` | 100 MB – 1 GB |
| | `large` | ≥ 1 GB |

Useful in `organize-by` to bin files by size without knowing exact values:

```yaml
organize-by: "{size-range}/{ext}"   # large/mp4, small/pdf, …
```

---

## Category and system

| Token | Expands to |
|---|---|
| `{category}` | Category name as configured |
| `{hostname}` | Machine hostname |
| `{username}` | Current OS user |
| `{os}` | `linux`, `darwin`, or `windows` |

---

## MIME type (`organize-by` only)

MIME tokens detect the file's real type from its content (magic bytes), independent of the extension. They are only available in `organize-by`; combine with `extensions: [all]` to route by content type.

| Token | Expands to | Example |
|---|---|---|
| `{mime}` | Full MIME type | `image/jpeg` |
| `{mime-type}` | Top-level type | `image` |
| `{mime-ext}` | Common extension for the type | `jpg` |

```yaml
source:
  path: ~/Downloads
  extensions: [all]
destination:
  path: ~/Sorted
  organize-by: "{mime-type}/{mime-ext}"   # image/jpg, video/mp4, …
```

---

## Sequence (`rename` only)

Sequence tokens auto-increment based on files already present in the destination directory. The counter seeds from the highest existing number found, so adding files to a non-empty directory never collides.

| Token | Expands to | Example |
|---|---|---|
| `{seq}` | Integer, no padding | `1`, `2`, `42` |
| `{seq:N}` | Zero-padded to N digits | `{seq:3}` → `001`, `002` |
| `{seq-alpha}` | Excel-style letters | `a`, `b`, …, `z`, `aa`, `ab` |
| `{seq-roman}` | Roman numerals (lowercase) | `i`, `ii`, `iii`, `iv` |

```yaml
rename: "{seq:4}_{name}.{ext}"    # 0001_report.pdf, 0002_invoice.pdf
rename: "{seq-roman}_{name}.{ext}" # i_report.pdf, ii_invoice.pdf
```

> In `--dry-run`, sequence tokens are shown as literals (`{seq}`, `{seq:3}`) rather than resolved values, since the destination directory may not exist yet.

---

## Hash (`rename` only)

Hash tokens compute a checksum of the file's content. Useful for deduplication-aware naming.

| Token | Expands to | Example |
|---|---|---|
| `{md5}` | First 8 hex chars of MD5 | `a1b2c3d4` |
| `{md5:N}` | First N hex chars of MD5 | `{md5:16}` → `a1b2c3d4e5f6a7b8` |
| `{sha256:N}` | First N hex chars of SHA-256 | `{sha256:12}` → `3f8a9c1d2e4b` |

MD5 is used for file identification only, not for security.

```yaml
rename: "{name}-{sha256:8}.{ext}"   # report-3f8a9c1d.pdf
```

> In `--dry-run`, hash tokens are shown as literals since reading the file for hashing would be a side effect.

---

## `archive.name` tokens

When `action: archive`, the `archive.name` field supports only tokens that do not depend on a specific file:

`{category}` `{year}` `{month}` `{day}` `{date}` `{weekday}` `{hour}` `{minute}` `{second}` `{timestamp}` `{hostname}` `{username}` `{os}`

```yaml
archive:
  format: zip
  name: "{category}_{date}"   # images_2025-04-16.zip
```

---

## Examples

### Date-based photo library

```yaml
destination:
  path: ~/Pictures
  organize-by: "{mod-year}/{mod-month}"
  rename: "{mod-date}_{name}.{ext}"
# ~/Pictures/2025/04/2025-04-16_sunset.jpg
```

### Unique names with hash

```yaml
destination:
  path: ~/Archive/docs
  rename: "{name-slug}-{sha256:8}.{ext}"
# ~/Archive/docs/annual-report-3f8a9c1d.pdf
```

### Numbered batch import

```yaml
destination:
  path: ~/Scans
  rename: "scan_{seq:4}.{ext}"
# ~/Scans/scan_0001.pdf, scan_0002.pdf, …
```

### Route by real content type

```yaml
source:
  extensions: [all]
destination:
  organize-by: "{mime-type}"
  # image/, video/, application/, …
```
