# source

## Arguments

The following arguments are supported:

| Name | Type | Description | Required | Default |
|------|------|-------------|----------|---------|
| path | string | Directory to watch for incoming files. | Yes | - |
| extensions | array | File extensions to match (without the leading dot). Use the special value "all" to match every file. | Yes | - |
| filter | object | Optional filtering rules applied to each matched file. All populated sub-fields must match (AND logic) unless any/all are used. | No | - |
| recursive | bool | Whether to scan sub-directories of the source path. Combine with max-depth to limit depth. | No | false |
| max-depth | int | Maximum sub-directory depth when recursive is true. 0 means unlimited. | No | 0 |
| exclude-paths | array | Sub-paths (relative to the source path) to skip during scanning. | No | - |

### filter

The following arguments are supported:

| Name | Type | Description | Required | Default |
|------|------|-------------|----------|---------|
| regex | string | RE2 regular expression matched against the filename (without path). Mutually exclusive with glob. | No | - |
| glob | string | Glob pattern matched against the filename (without path). Mutually exclusive with regex. | No | - |
| include | array | Filenames must match at least one of these glob patterns. | No | - |
| ignore | array | Filenames matching these patterns are excluded. Takes precedence over include. | No | - |
| case-sensitive | bool | Whether extension and glob/include/ignore matching is case-sensitive. | No | false |
| min-age | duration | Only match files older than this duration. Accepts Go duration strings (e.g. 24h, 168h). | No | - |
| max-age | duration | Only match files newer than this duration. | No | - |
| min-size | string | Only match files at least this large. Accepts human-readable sizes — KB/MB/GB/TB are decimal (powers of 1000), KiB/MiB/GiB/TiB are binary (powers of 1024). | No | - |
| max-size | string | Only match files no larger than this size. Same units as min-size. | No | - |
| any | array[object] | OR logic: file must match at least one sub-filter. | No | - |
| all | array[object] | AND logic: file must match all sub-filters simultaneously. | No | - |

#### any

The following arguments are supported:

| Name | Type | Description | Required | Default |
|------|------|-------------|----------|---------|
| regex | string | RE2 regular expression matched against the filename (without path). Mutually exclusive with glob. | No | - |
| glob | string | Glob pattern matched against the filename (without path). Mutually exclusive with regex. | No | - |
| include | array | Filenames must match at least one of these glob patterns. | No | - |
| ignore | array | Filenames matching these patterns are excluded. Takes precedence over include. | No | - |
| case-sensitive | bool | Whether extension and glob/include/ignore matching is case-sensitive. | No | false |
| min-age | duration | Only match files older than this duration. Accepts Go duration strings (e.g. 24h, 168h). | No | - |
| max-age | duration | Only match files newer than this duration. | No | - |
| min-size | string | Only match files at least this large. Accepts human-readable sizes — KB/MB/GB/TB are decimal (powers of 1000), KiB/MiB/GiB/TiB are binary (powers of 1024). | No | - |
| max-size | string | Only match files no larger than this size. Same units as min-size. | No | - |
| any | array | OR logic: file must match at least one sub-filter. | No | - |
| all | array | AND logic: file must match all sub-filters simultaneously. | No | - |

#### all

The following arguments are supported:

| Name | Type | Description | Required | Default |
|------|------|-------------|----------|---------|
| regex | string | RE2 regular expression matched against the filename (without path). Mutually exclusive with glob. | No | - |
| glob | string | Glob pattern matched against the filename (without path). Mutually exclusive with regex. | No | - |
| include | array | Filenames must match at least one of these glob patterns. | No | - |
| ignore | array | Filenames matching these patterns are excluded. Takes precedence over include. | No | - |
| case-sensitive | bool | Whether extension and glob/include/ignore matching is case-sensitive. | No | false |
| min-age | duration | Only match files older than this duration. Accepts Go duration strings (e.g. 24h, 168h). | No | - |
| max-age | duration | Only match files newer than this duration. | No | - |
| min-size | string | Only match files at least this large. Accepts human-readable sizes — KB/MB/GB/TB are decimal (powers of 1000), KiB/MiB/GiB/TiB are binary (powers of 1024). | No | - |
| max-size | string | Only match files no larger than this size. Same units as min-size. | No | - |
| any | array | OR logic: file must match at least one sub-filter. | No | - |
| all | array | AND logic: file must match all sub-filters simultaneously. | No | - |

