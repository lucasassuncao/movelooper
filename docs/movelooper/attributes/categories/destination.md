# destination

## Arguments

The following arguments are supported:

| Name | Type | Format | Description | Required | Default |
|------|------|--------|-------------|----------|---------|
| path | string | directory | Directory where matched files are placed. | Yes | - |
| organize-by | string | organize-by pattern | Token pattern used to build sub-directories inside the destination path. Leave empty to place all files directly. | No | - |
| conflict-strategy | string | - | What to do when a file with the same name already exists at the destination. | No | rename |
| action | string | - | File operation to perform. 'move' removes the source; 'copy' keeps it; 'symlink' creates a symbolic link. | No | move |
| rename | string | rename pattern | Token pattern for the destination filename (without extension). Leave empty to keep the original name. | No | - |

