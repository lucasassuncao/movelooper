# Configuration

## Examples

For usage examples, see [Configuration presets](../../examples/configuration.md).

## Arguments

The following arguments are supported:

| Name | Type | Description | Required | Default |
|------|------|-------------|----------|---------|
| logging | object | Log output settings: destination, severity level, format, file path, and caller info. | Yes | - |
| watch | object | Watch-mode settings. | No | - |
| history | object | Undo-history settings: whether tracking is on, how many batches to keep, and where to store them. | No | - |
| defaults | object | Fallback destination settings applied to any category that omits them. Per-category values always win. | No | - |

### logging

The following arguments are supported:

| Name | Type | Format | Description | Required | Default |
|------|------|--------|-------------|----------|---------|
| output | string | - | Where log output is written. Use 'both' to write to the console and a file simultaneously. 'log' is an alias for 'file'. | Yes | console |
| level | string | - | Minimum severity level to emit. Lower levels produce more output; 'fatal' produces the least. | Yes | info |
| file | string | directory | Path to the log file. Only used when output is 'file' or 'both'. Supports ~ for the home directory. | No | ~/movelooper.log |
| show-caller | bool | - | Append the source file and line number to each log entry. Useful when debugging hooks or scanners. | No | false |
| format | string | - | Log rendering format. 'pretty' is the human-readable console renderer; 'json' emits structured slog JSON lines for log aggregation. | No | pretty |
| color | string | - | ANSI color for the pretty format. 'auto' colors the console but not files; 'always'/'never' force it on or off. Ignored when format is json. | No | auto |
| max-width | int | - | Maximum width, in columns, for wrapping pretty log lines. Ignored when format is json. | No | 70 |

### watch

The following arguments are supported:

| Name | Type | Format | Description | Required | Default |
|------|------|--------|-------------|----------|---------|
| delay | duration | duration | How long a file must go without a new create/write event before it is considered stable and moved. Accepts Go duration strings (e.g. 30s, 5m, 1h). | No | 5m |
| poll-interval | duration | duration | How often watch mode re-checks pending files for stability. Keep it shorter than delay so stable files are picked up promptly. | No | 5s |

### history

The following arguments are supported:

| Name | Type | Format | Description | Required | Default |
|------|------|--------|-------------|----------|---------|
| limit | int | - | Maximum number of move batches kept in the undo history. Older batches are evicted when the limit is reached. | No | 100 |
| file | string | directory | Path to the history file used for undo. Defaults to ~/.movelooper/history/movelooper.json when not set. | No | ~/.movelooper/history/movelooper.json |
| enabled | bool | - | Whether move events are recorded for undo. Set to false to skip history tracking entirely. | No | true |

### defaults

The following arguments are supported:

| Name | Type | Description | Required | Default |
|------|------|-------------|----------|---------|
| conflict-strategy | string | Fallback conflict-strategy for categories that omit destination.conflict-strategy. | No | - |
| action | string | Fallback action for categories that omit destination.action. | No | - |
| organize-by | string | Fallback organize-by template for categories that omit destination.organize-by. | No | - |

