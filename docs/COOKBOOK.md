# Cookbook

Ready-to-use configurations for common scenarios. Each recipe is a complete, working config — copy, adjust the paths, then run `movelooper --dry-run` to preview before committing.

---

## 1. Downloads folder sorter

Keeps your Downloads clean by routing files into typed folders. Files modified less than 5 minutes ago are skipped (still downloading).

```yaml
configuration:
  logging:
    output: console
    level: info
  watch:
    delay: 5m

categories:
  - name: images
    enabled: true
    source:
      path: ~/Downloads
      extensions: [jpg, jpeg, png, gif, webp, heic, svg]
      filter:
        age:
          min: 5m
    destination:
      path: ~/Downloads/sorted/images
      conflict-strategy: rename

  - name: documents
    enabled: true
    source:
      path: ~/Downloads
      extensions: [pdf, docx, doc, xlsx, xls, pptx, ppt, txt, odt, csv]
      filter:
        age:
          min: 5m
    destination:
      path: ~/Downloads/sorted/documents
      conflict-strategy: rename

  - name: videos
    enabled: true
    source:
      path: ~/Downloads
      extensions: [mp4, mkv, avi, mov, wmv, webm, m4v]
      filter:
        age:
          min: 5m
    destination:
      path: ~/Downloads/sorted/videos
      conflict-strategy: rename

  - name: archives
    enabled: true
    source:
      path: ~/Downloads
      extensions: [zip, tar, gz, rar, 7z, tar.gz, tar.bz2]
      filter:
        age:
          min: 5m
    destination:
      path: ~/Downloads/sorted/archives
      conflict-strategy: rename

  - name: installers
    enabled: true
    source:
      path: ~/Downloads
      extensions: [exe, msi, dmg, pkg, deb, rpm, appimage]
      filter:
        age:
          min: 5m
    destination:
      path: ~/Downloads/sorted/installers
      conflict-strategy: skip    # keep only one copy per installer name
```

> **See also:** [Filters](/FILTERS.md) — `age`, `size`, `match`, and boolean composition.

---

## 2. Photo library organized by date

Moves photos from a camera import folder into a year/month structure. Uses `hash_check` so duplicate photos (same content, different name) are silently skipped.

```yaml
configuration:
  logging:
    output: console
    level: info

categories:
  - name: photos
    enabled: true
    source:
      path: ~/Downloads
      extensions: [jpg, jpeg, raw, cr2, cr3, nef, arw, dng, heic, png]
      filter:
        age:
          min: 1m
    destination:
      path: ~/Pictures/library
      organize-by: "{year}/{month}"      # ~/Pictures/library/2025/04/
      conflict-strategy: hash_check      # skip identical, rename if different content
```

To also rename files with the modification date as prefix:

```yaml
    destination:
      path: ~/Pictures/library
      organize-by: "{year}/{month}"
      rename: "{year}-{month}-{day}_{name}.{ext}"   # 2025-04-16_photo.jpg
      conflict-strategy: hash_check
```

> **See also:** [Tokens](/TOKENS.md) — `{mod-year}`, `{mod-date}`, `{name}`, `{ext}`, and all other template tokens.

---

## 3. Backup without removing originals

Copies files to a backup location and leaves the originals untouched. Uses `skip` so a file that was already backed up is never overwritten.

```yaml
configuration:
  logging:
    output: console
    level: info

categories:
  - name: docs-backup
    enabled: true
    source:
      path: ~/Documents/work
      extensions: [pdf, docx, xlsx, pptx, txt]
    destination:
      path: /Volumes/Backup/work-docs
      action: copy                        # copy, do not move
      organize-by: "{year}/{month}"
      conflict-strategy: skip             # never overwrite an existing backup
```

> **Note:** Undoing a `copy` batch removes the copy at the destination. The original is never touched.

---

## 4. Catch-all: sort everything by extension

Organizes any file in a folder into a subfolder named after its extension. Useful as a final fallback category or as the only category for a "misc" folder.

```yaml
configuration:
  logging:
    output: console
    level: info

categories:
  - name: sort-by-extension
    enabled: true
    source:
      path: ~/Desktop
      extensions: [all]                  # matches every file type
      filter:
        age:
          min: 10m
    destination:
      path: ~/Desktop/sorted
      organize-by: "{ext}"               # ~/Desktop/sorted/pdf/, /mp4/, /docx/ …
      conflict-strategy: rename
```

For even smarter sorting — organize by the **real** file type (not just the extension), add `filter.mime`:

```yaml
    source:
      path: ~/Desktop
      extensions: [all]
      filter:
        age:
          min: 10m
        mime:
          - "image/*"       # only real images, regardless of extension
```

> **See also:** [Tokens](/TOKENS.md) — `{ext}` and all organize-by tokens. [Filters](/FILTERS.md) — `mime`, `age`, and filter composition.

---

## 5. Wallpaper sorter (filter by filename pattern)

Moves only wallpapers downloaded from Wallhaven (whose filenames start with `wallhaven-`) without touching other images in the same folder.

```yaml
configuration:
  logging:
    output: console
    level: info

categories:
  - name: wallhaven
    enabled: true
    source:
      path: ~/Downloads
      extensions: [jpg, jpeg, png]
      filter:
        match:
          regex: "^wallhaven-"           # only files whose name starts with "wallhaven-"
        age:
          min: 1m
    destination:
      path: ~/Pictures/Wallpapers/Wallhaven
      conflict-strategy: hash_check
```

> **See also:** [Filters](/FILTERS.md) — `match.regex`, `match.glob`, and boolean composition.

---

## 6. Split config with imports

When your config grows large, split categories into separate files. The main file holds global settings; each imported file holds one or more categories.

```yaml
# movelooper.yaml — main file
configuration:
  logging:
    output: console
    level: info
  watch:
    delay: 5m

import:
  - conf/images.yaml
  - conf/documents.yaml
  - conf/videos.yaml
```

```yaml
# conf/images.yaml
categories:
  - name: images
    enabled: true
    source:
      path: ~/Downloads
      extensions: [jpg, jpeg, png, webp, heic]
      filter:
        age:
          min: 5m
    destination:
      path: ~/Pictures
      organize-by: "{year}/{month}"
      conflict-strategy: hash_check
```

Run `movelooper edit` to open the merged config in the interactive TUI editor.

> **See also:** [Configuration](/CONFIGURATION.md) — `import:` key and all global settings.

---

## 7. Hook: notify after moving files

Sends a desktop notification after each category run. Only fires when at least one file was moved.

**Linux / macOS (bash)**

```yaml
categories:
  - name: images
    enabled: true
    source:
      path: ~/Downloads
      extensions: [jpg, jpeg, png, webp]
    destination:
      path: ~/Pictures
      conflict-strategy: rename
    hooks:
      after:
        shell: bash
        on-failure: warn                 # a failed notification should not be a hard error
        run:
          - |
            if [ "$ML_FILES_MOVED" -gt 0 ]; then
              notify-send "movelooper" "$ML_FILES_MOVED file(s) moved to Pictures (batch $ML_BATCH_ID)"
            fi
```

**Windows (PowerShell)**

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

> Hooks run only on the one-shot `movelooper` command, not in `watch` mode.

> **See also:** [Hooks](/HOOKS.md) — fields, `on-failure`, all `ML_*` environment variables, and more examples.

---

## 8. Running automatically (set and forget)

See [Watch Mode](/WATCH.md#running-automatically) for systemd, launchd, and Windows Task Scheduler setup.
