# source

## Arguments

The following arguments are supported:

| Name | Type | Format | Description | Required | Default |
|------|------|--------|-------------|----------|---------|
| path | string | directory | Directory to watch for incoming files. | Yes | - |
| extensions | []string | - | File extensions to match (without the leading dot). Use the special value "all" to match every file. | Yes | - |
| filter | object | - | Optional filtering rules applied to each matched file. All populated sub-fields must match (AND logic) unless any/all are used. | No | - |
| recursive | bool | - | Whether to scan sub-directories of the source path. Combine with max-depth to limit depth. | No | false |
| max-depth | int | - | Maximum sub-directory depth when recursive is true. 0 means unlimited. | No | 0 |
| exclude-paths | []string | - | Absolute paths to skip during recursive walk. The destination path is always auto-excluded. | No | - |

### filter

The following arguments are supported:

| Name | Type | Description | Required | Default |
|------|------|-------------|----------|---------|
| match | object | Name-based filter: glob, regex, or literal match (pick one). | No | - |
| age | object | Modification-time constraints. | No | - |
| size | object | File-size constraints. | No | - |
| any | []object | OR logic: file must match at least one sub-filter. | No | - |
| all | []object | AND logic: file must match all sub-filters simultaneously. | No | - |
| not | []object | NOT logic: exclude files matching any of these sub-filters. | No | - |

#### match

The following arguments are supported:

| Name | Type | Format | Description | Required | Default |
|------|------|--------|-------------|----------|---------|
| literal | string | - | Exact filename match (whole name must equal this string). Mutually exclusive with regex and glob. | No | - |
| regex | string | regex | RE2 regular expression matched against the filename. Mutually exclusive with glob and literal. | No | - |
| glob | string | glob | Glob pattern matched against the filename. Mutually exclusive with regex and literal. | No | - |
| case-sensitive | bool | - | Whether matching is case-sensitive. Applies to regex, glob, and literal. | No | false |

#### age

The following arguments are supported:

| Name | Type | Format | Description | Required | Default |
|------|------|--------|-------------|----------|---------|
| min | duration | duration | Only match files older than this duration. | No | - |
| max | duration | duration | Only match files newer than this duration. | No | - |

#### size

The following arguments are supported:

| Name | Type | Description | Required | Default |
|------|------|-------------|----------|---------|
| min | string | Only match files at least this large. KB/MB/GB/TB are decimal; KiB/MiB/GiB/TiB are binary. | No | - |
| max | string | Only match files no larger than this size. | No | - |

