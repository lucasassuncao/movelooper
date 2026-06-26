# Category Examples

## Preset: action-copy

```yaml
categories:
    - name: photos-backup
      enabled: true
      source:
        path: ~/Downloads
        extensions:
            - jpg
            - jpeg
            - heic
      destination:
        path: ~/Downloads/Pictures/backup
        organize-by: '{year}/{month}'
        conflict-strategy: skip
        action: copy
```

## Preset: action-symlink

```yaml
categories:
    - name: media-links
      enabled: true
      source:
        path: ~/Downloads
        extensions:
            - mp4
            - mkv
            - avi
      destination:
        path: ~/Downloads/Media/links
        organize-by: '{year}'
        conflict-strategy: skip
        action: symlink
```

## Preset: conflict-hash-check

```yaml
categories:
    - name: archives
      enabled: true
      source:
        path: ~/Downloads
        extensions:
            - zip
            - tar
            - gz
            - rar
            - 7z
      destination:
        path: ~/Downloads/archives
        organize-by: '{ext}'
        conflict-strategy: hash_check
```

## Preset: conflict-larger

```yaml
categories:
    - name: archives-large
      enabled: true
      source:
        path: ~/Downloads
        extensions:
            - zip
            - 7z
      destination:
        path: ~/Downloads/archives/large
        organize-by: '{ext}'
        conflict-strategy: larger
```

## Preset: conflict-newest

```yaml
categories:
    - name: videos
      enabled: true
      source:
        path: ~/Downloads
        extensions:
            - mp4
            - mkv
            - avi
      destination:
        path: ~/Downloads/videos
        organize-by: '{year}'
        conflict-strategy: newest
```

## Preset: conflict-oldest

```yaml
categories:
    - name: documents
      enabled: true
      source:
        path: ~/Downloads
        extensions:
            - pdf
            - doc
            - docx
      destination:
        path: ~/Downloads/documents
        organize-by: '{year}'
        conflict-strategy: oldest
```

## Preset: conflict-overwrite

```yaml
categories:
    - name: config-sync
      enabled: true
      source:
        path: ~/Downloads
        extensions:
            - yaml
            - json
            - toml
      destination:
        path: ~/Downloads/config
        organize-by: '{ext}'
        conflict-strategy: overwrite
```

## Preset: conflict-rename

```yaml
categories:
    - name: photos
      enabled: true
      source:
        path: ~/Downloads
        extensions:
            - jpg
            - jpeg
            - png
      destination:
        path: ~/Downloads/photos
        organize-by: '{year}/{month}'
        conflict-strategy: rename
```

## Preset: conflict-skip

```yaml
categories:
    - name: music
      enabled: true
      source:
        path: ~/Downloads
        extensions:
            - mp3
            - flac
            - wav
      destination:
        path: ~/Downloads/music
        organize-by: '{ext}'
        conflict-strategy: skip
```

## Preset: conflict-smaller

```yaml
categories:
    - name: photos-small
      enabled: true
      source:
        path: ~/Downloads
        extensions:
            - jpg
            - jpeg
      destination:
        path: ~/Downloads/photos/small
        organize-by: '{year}'
        conflict-strategy: smaller
```

## Preset: filter-age

```yaml
categories:
    - name: old-downloads
      enabled: true
      source:
        path: ~/Downloads
        extensions:
            - pdf
            - zip
            - exe
            - dmg
        filter:
            age:
                min: 720h0m0s
                max: 8760h0m0s
      destination:
        path: ~/Downloads/old
        organize-by: '{year}'
        conflict-strategy: skip
```

## Preset: filter-all

```yaml
categories:
    - name: recent-docs
      enabled: true
      source:
        path: ~/Downloads
        extensions:
            - pdf
            - docx
        filter:
            all:
                - size:
                    min: 100KB
                - age:
                    max: 168h0m0s
                - not:
                    - match:
                        glob: '*_draft.*'
      destination:
        path: ~/Downloads/recent-docs
        organize-by: '{year}/{month}'
        conflict-strategy: rename
```

## Preset: filter-any

```yaml
categories:
    - name: reports
      enabled: true
      source:
        path: ~/Downloads
        extensions:
            - pdf
            - xlsx
            - csv
        filter:
            any:
                - match:
                    regex: ^report_
                - match:
                    glob: summary_*
                - size:
                    min: 1MB
      destination:
        path: ~/Downloads/reports
        organize-by: '{ext}'
        conflict-strategy: rename
```

## Preset: filter-match-glob

```yaml
categories:
    - name: screenshots
      enabled: true
      source:
        path: ~/Downloads
        extensions:
            - png
            - jpg
        filter:
            match:
                glob: screenshot_????-??-??_*
      destination:
        path: ~/Downloads/screenshots
        organize-by: '{year}/{month}'
        conflict-strategy: rename
```

## Preset: filter-match-literal

```yaml
categories:
    - name: annas-archive
      enabled: true
      source:
        path: ~/Downloads
        extensions:
            - pdf
        filter:
            match:
                literal: Anna's Archive.pdf
      destination:
        path: ~/Downloads/books
        conflict-strategy: skip
```

## Preset: filter-match-regex

```yaml
categories:
    - name: dated-reports
      enabled: true
      source:
        path: ~/Downloads
        extensions:
            - pdf
            - csv
            - xlsx
        filter:
            match:
                regex: ^\d{4}-\d{2}-\d{2}_.*
      destination:
        path: ~/Downloads/reports
        organize-by: '{year}/{month}'
        conflict-strategy: rename
```

## Preset: filter-not

```yaml
categories:
    - name: documents
      enabled: true
      source:
        path: ~/Downloads
        extensions:
            - doc
            - docx
            - odt
        filter:
            not:
                - match:
                    glob: '*_temp.*'
                - match:
                    glob: '*_backup.*'
                - match:
                    glob: ~*
      destination:
        path: ~/Downloads/documents
        organize-by: '{ext}'
        conflict-strategy: rename
```

## Preset: filter-size

```yaml
categories:
    - name: large-videos
      enabled: true
      source:
        path: ~/Downloads
        extensions:
            - mp4
            - mkv
        filter:
            size:
                min: 100MB
                max: 10GB
      destination:
        path: ~/Downloads/videos
        organize-by: '{year}/{month}'
        conflict-strategy: hash_check
```

## Preset: hooks-after

```yaml
categories:
    - name: videos-after
      enabled: true
      source:
        path: ~/Downloads
        extensions:
            - mp4
            - mkv
        filter:
            size:
                min: 100MB
      destination:
        path: ~/Downloads/videos
        organize-by: '{year}/{month}'
        conflict-strategy: hash_check
      hooks:
        after:
            shell: bash
            on-failure: warn
            run:
                - echo "$ML_FILES_MOVED files moved to $ML_DEST_PATH"
```

## Preset: hooks-before

```yaml
categories:
    - name: videos-before
      enabled: true
      source:
        path: ~/Downloads
        extensions:
            - mp4
            - mkv
        filter:
            size:
                min: 100MB
      destination:
        path: ~/Downloads/videos
        organize-by: '{year}/{month}'
        conflict-strategy: hash_check
      hooks:
        before:
            shell: bash
            on-failure: abort
            run:
                - mkdir -p ~/Downloads/videos
                - echo "moving $ML_SOURCE_PATH"
```

## Preset: recursive

```yaml
categories:
    - name: documents-recursive
      enabled: true
      source:
        path: ~/Downloads
        extensions:
            - pdf
            - doc
            - docx
            - txt
        recursive: true
        max-depth: 3
        exclude-paths:
            - ~/Downloads/archives
            - ~/Downloads/temp
      destination:
        path: ~/Downloads/documents
        organize-by: '{year}/{month}'
        conflict-strategy: hash_check
```

