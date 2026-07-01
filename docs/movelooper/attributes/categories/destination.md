# destination

## Arguments

The following arguments are supported:

| Name | Type | Format | Description | Required | Default |
|------|------|--------|-------------|----------|---------|
| path | string | directory | Directory where matched files are placed. | Yes | - |
| organize-by | string | organize-by pattern | Token pattern used to build sub-directories inside the destination path. Leave empty to place all files directly. | No | - |
| conflict-strategy | string | - | What to do when a file with the same name already exists at the destination. | No | rename |
| action | string | - | File operation to perform. 'move' removes the source; 'copy' keeps it; 'symlink' links it; 'archive' packs the whole category into one compressed file (requires the archive block). | No | move |
| rename | string | rename pattern | Token pattern for the destination filename. It becomes the whole filename, so include {ext} to keep the extension (omit it and the file is written without one). Leave empty to keep the original name. | No | - |
| archive | object | - | Archiving options. Required when action is 'archive'. Packs all matched files of the category into one zip/tar.gz at the destination path. | No | - |

### archive

The following arguments are supported:

| Name | Type | Description | Required | Default |
|------|------|-------------|----------|---------|
| format | string | Archive container/compression: 'zip' (universal) or 'tar.gz'. Required — no default. | Yes | - |
| name | string | Archive filename base (extension is added automatically). Supports category/date/system tokens only: {category}, {date}, {timestamp}, {hostname}, {username}, {os}. Empty uses the category name. | No | - |
| compression | string | Compression effort: 'none', 'fast', or 'best'. | No | best |
| keep-source | bool | Keep the original files after archiving. Defaults to true; set false to delete sources after a successful write. | No | true |
| flatten | bool | Put every file at the archive root. Defaults to false, which preserves each file's sub-path relative to the source directory (relevant with recursive scans). | No | false |

