# Filters

Filters let you narrow which files a category processes. A file must pass every filter defined on a source — the top-level fields are implicitly ANDed. For more complex logic, use `any`, `all`, and `not`.

---

## Filter types

### `match` — filename pattern

Constrains by filename. Pick exactly one of `glob`, `regex`, or `literal`.

```yaml
filter:
  match:
    glob: "invoice_*"          # glob pattern
    # regex: "^invoice_\\d{4}" # RE2 regex — mutually exclusive with glob
    # literal: "summary.pdf"   # exact filename — mutually exclusive with glob/regex
    case-sensitive: false       # default; set true for case-sensitive matching
```

| Field | Description |
|---|---|
| `glob` | Shell-style wildcard: `*` matches any characters, `?` matches one |
| `regex` | RE2 regular expression matched against the filename |
| `literal` | Exact filename match (whole name must equal this string) |
| `case-sensitive` | Applies to all three match types; default `false` |

### `age` — modification time

Constrains by how old the file is relative to the current time. Accepts Go duration strings: `10m`, `24h`, `168h` (7 days), `720h` (30 days).

```yaml
filter:
  age:
    min: 10m     # skip files modified less than 10 minutes ago (still downloading)
    max: 720h    # skip files older than 30 days
```

| Field | Meaning |
|---|---|
| `min` | File must be **older** than this duration |
| `max` | File must be **newer** than this duration |

### `size` — file size

Constrains by file size. Decimal units: `KB`, `MB`, `GB`, `TB`. Binary units: `KiB`, `MiB`, `GiB`, `TiB`.

```yaml
filter:
  size:
    min: 1MB     # skip files smaller than 1 MB
    max: 500MB   # skip files larger than 500 MB
```

| Field | Meaning |
|---|---|
| `min` | File must be **at least** this large |
| `max` | File must be **no larger** than this |

Decimal units (`KB`, `MB`, `GB`, `TB`) are powers of 1000; binary units (`KiB`, `MiB`, `GiB`, `TiB`) are powers of 1024. Suffixes are case-insensitive, decimals are allowed (`1.5MB`), and a bare number means bytes.

| Suffix | Meaning | Example |
|---|---|---|
| `KB` / `MB` / `GB` / `TB` | Powers of 1000 | `10MB` = 10 000 000 bytes |
| `KiB` / `MiB` / `GiB` / `TiB` | Powers of 1024 | `10MiB` = 10 485 760 bytes |

The two families are not interchangeable — `10MiB` is ~4.9% larger than `10MB`. Rule of thumb: `MB` matches what disk vendors and network tools report; `MiB` matches what Windows Explorer shows as "MB".

> **Note:** Earlier versions of movelooper treated `KB`/`MB`/`GB`/`TB` as binary. If your config predates this change, switch to `KiB`/`MiB`/`GiB`/`TiB` to keep the exact same thresholds.

### `mime` — real content type

Matches against the file's detected MIME type (read from magic bytes, not the extension). The value is a glob pattern matched against the full MIME string.

```yaml
filter:
  mime: "image/*"           # any image type
  # mime: "application/pdf"
  # mime: "video/*"
```

Combine with `extensions: [all]` when you want to route files by real content type regardless of extension:

```yaml
source:
  path: ~/Downloads
  extensions: [all]
  filter:
    mime: "image/*"
```

---

## Boolean composition

Use `any`, `all`, and `not` to combine multiple filters. Each takes a list of filters that follow the same structure (including nested `any`/`all`/`not`).

### `any` — OR

The file must match **at least one** of the listed filters.

```yaml
filter:
  any:
    - match:
        glob: "invoice_*"
    - match:
        glob: "receipt_*"
```

### `all` — AND

The file must match **all** listed filters simultaneously. Equivalent to listing them at the top level, but explicit.

```yaml
filter:
  all:
    - size:
        min: 100KB
    - age:
        max: 168h     # modified within the last 7 days
```

### `not` — exclude

The file is excluded if it matches **any** of the listed filters.

```yaml
filter:
  not:
    - match:
        glob: "*_draft*"
    - match:
        glob: "*_temp*"
```

---

## Combining types at the top level

All populated top-level fields are ANDed. The file must pass every one:

```yaml
filter:
  age:
    min: 5m          # not still downloading
  size:
    min: 10KB        # not an empty placeholder
  match:
    glob: "report_*" # name starts with "report_"
```

---

## Nesting examples

### Only large recent files whose name matches a pattern

```yaml
filter:
  match:
    glob: "export_*"
  size:
    min: 5MB
  age:
    max: 48h
```

### Invoices or receipts, but not drafts

```yaml
filter:
  any:
    - match:
        glob: "invoice_*"
    - match:
        glob: "receipt_*"
  not:
    - match:
        glob: "*_draft*"
```

### PDFs that are recent OR large (but never tiny temp files)

```yaml
filter:
  any:
    - age:
        max: 24h
    - size:
        min: 10MB
  not:
    - size:
        max: 1KB
```

### Route images by real type, not just extension

```yaml
source:
  path: ~/Downloads
  extensions: [all]
  filter:
    mime: "image/*"
    size:
      min: 50KB       # skip tiny thumbnails
```

---

## Filter evaluation order

1. `extensions` is checked first (before any filter block).
2. Within the filter block, `match`, `age`, `size`, and `mime` are evaluated first, then `any`, `all`, and `not` are composed on top.
3. A file proceeds only when every condition is satisfied.
