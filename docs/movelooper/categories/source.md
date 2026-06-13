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
| match | object | Filename matching rules. Sub-fields literal, regex, and glob are mutually exclusive. | No | - |
| age | object | Age window: min and max as Go durations. | No | - |
| size | object | Size window: min and max as human-readable strings. | No | - |
| any | array[object] | OR logic: file must match at least one sub-filter. | No | - |
| all | array[object] | AND logic: file must match all sub-filters simultaneously. | No | - |
| not | array[object] | Exclusion: file is rejected if any entry matches. | No | - |

#### filter.match

| Name | Type | Description | Required | Default |
|------|------|-------------|----------|---------|
| literal | string | Exact filename match. Mutually exclusive with regex and glob. | No | - |
| regex | string | RE2 regular expression matched against the filename (without path). Mutually exclusive with literal and glob. | No | - |
| glob | string | Glob pattern matched against the filename (without path). Mutually exclusive with literal and regex. | No | - |
| case-sensitive | bool | Whether literal/regex/glob matching is case-sensitive. | No | false |

#### filter.age

| Name | Type | Description | Required | Default |
|------|------|-------------|----------|---------|
| min | duration | Only match files older than this duration. Accepts Go duration strings (e.g. 24h, 168h). | No | - |
| max | duration | Only match files newer than this duration. | No | - |

#### filter.size

| Name | Type | Description | Required | Default |
|------|------|-------------|----------|---------|
| min | string | Only match files at least this large. Accepts human-readable sizes — KB/MB/GB/TB are decimal (powers of 1000), KiB/MiB/GiB/TiB are binary (powers of 1024). | No | - |
| max | string | Only match files no larger than this size. Same units as min. | No | - |

#### any

The following arguments are supported:

| Name | Type | Description | Required | Default |
|------|------|-------------|----------|---------|
| match | object | Filename matching rules. Sub-fields literal, regex, and glob are mutually exclusive. | No | - |
| age | object | Age window: min and max as Go durations. | No | - |
| size | object | Size window: min and max as human-readable strings. | No | - |
| any | array | OR logic: file must match at least one sub-filter. | No | - |
| all | array | AND logic: file must match all sub-filters simultaneously. | No | - |
| not | array | Exclusion: file is rejected if any entry matches. | No | - |

#### all

The following arguments are supported:

| Name | Type | Description | Required | Default |
|------|------|-------------|----------|---------|
| match | object | Filename matching rules. Sub-fields literal, regex, and glob are mutually exclusive. | No | - |
| age | object | Age window: min and max as Go durations. | No | - |
| size | object | Size window: min and max as human-readable strings. | No | - |
| any | array | OR logic: file must match at least one sub-filter. | No | - |
| all | array | AND logic: file must match all sub-filters simultaneously. | No | - |
| not | array | Exclusion: file is rejected if any entry matches. | No | - |

#### not

The following arguments are supported:

| Name | Type | Description | Required | Default |
|------|------|-------------|----------|---------|
| match | object | Filename matching rules. Sub-fields literal, regex, and glob are mutually exclusive. | No | - |
| age | object | Age window: min and max as Go durations. | No | - |
| size | object | Size window: min and max as human-readable strings. | No | - |
| any | array | OR logic: file must match at least one sub-filter. | No | - |
| all | array | AND logic: file must match all sub-filters simultaneously. | No | - |
| not | array | Exclusion: file is rejected if any entry matches. | No | - |
