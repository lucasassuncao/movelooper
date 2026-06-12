# Configuration

## Arguments

The following arguments are supported:

| Name | Type | Description | Required | Default |
|------|------|-------------|----------|---------|
| output | string | Where log output is written. Use 'both' to write to the console and a file simultaneously. | Yes | console |
| log-file | string | Path to the log file. Only used when output is 'file' or 'both'. Supports ~ for the home directory. | No | ~/movelooper.log |
| log-level | string | Minimum severity level to emit. Lower levels produce more output; 'fatal' produces the least. | Yes | info |
| show-caller | bool | Append the source file and line number to each log entry. Useful when debugging hooks or scanners. | No | false |
| watch-delay | duration | Interval between directory scans in watch mode. Accepts Go duration strings (e.g. 30s, 5m, 1h). | No | 5m |
| history-limit | int | Maximum number of move events kept in the undo history. Older entries are evicted when the limit is reached. | No | 100 |

