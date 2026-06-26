# Configuration Examples

## Preset: both

```yaml
configuration:
    logging:
        output: both
        level: info
        file: ~/.movelooper/logs/movelooper.log
        show-caller: false
        format: pretty
        color: auto
    watch:
        delay: 5m0s
    history:
        limit: 100
        file: ~/.movelooper/history/movelooper.json
```

## Preset: console-debug

```yaml
configuration:
    logging:
        output: console
        level: debug
        file: ~/.movelooper/logs/movelooper.log
        show-caller: true
        format: pretty
        color: auto
    watch:
        delay: 5m0s
    history:
        limit: 100
        file: ~/.movelooper/history/movelooper.json
```

## Preset: console-error

```yaml
configuration:
    logging:
        output: console
        level: error
        file: ~/.movelooper/logs/movelooper.log
        show-caller: false
        format: pretty
        color: auto
    watch:
        delay: 5m0s
    history:
        limit: 100
        file: ~/.movelooper/history/movelooper.json
```

## Preset: console-fatal

```yaml
configuration:
    logging:
        output: console
        level: fatal
        file: ~/.movelooper/logs/movelooper.log
        show-caller: false
        format: pretty
        color: auto
    watch:
        delay: 5m0s
    history:
        limit: 100
        file: ~/.movelooper/history/movelooper.json
```

## Preset: console-info

```yaml
configuration:
    logging:
        output: console
        level: info
        file: ~/.movelooper/logs/movelooper.log
        show-caller: false
        format: pretty
        color: auto
    watch:
        delay: 5m0s
    history:
        limit: 100
        file: ~/.movelooper/history/movelooper.json
```

## Preset: console-trace

```yaml
configuration:
    logging:
        output: console
        level: trace
        file: ~/.movelooper/logs/movelooper.log
        show-caller: true
        format: pretty
        color: auto
    watch:
        delay: 5m0s
    history:
        limit: 100
        file: ~/.movelooper/history/movelooper.json
```

## Preset: console-warn

```yaml
configuration:
    logging:
        output: console
        level: warn
        file: ~/.movelooper/logs/movelooper.log
        show-caller: false
        format: pretty
        color: auto
    watch:
        delay: 5m0s
    history:
        limit: 100
        file: ~/.movelooper/history/movelooper.json
```

## Preset: file

```yaml
configuration:
    logging:
        output: file
        level: warn
        file: ~/.movelooper/logs/movelooper.log
        show-caller: false
        format: pretty
        color: auto
    watch:
        delay: 5m0s
    history:
        limit: 100
        file: ~/.movelooper/history/movelooper.json
```

## Preset: json

```yaml
configuration:
    logging:
        output: file
        level: info
        file: ~/.movelooper/logs/movelooper.log
        show-caller: false
        format: json
        color: auto
    watch:
        delay: 5m0s
    history:
        limit: 100
        file: ~/.movelooper/history/movelooper.json
```

