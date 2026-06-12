# hooks

## Arguments

The following arguments are supported:

| Name | Type | Description | Required | Default |
|------|------|-------------|----------|---------|
| before | object | Hook executed before the file operation. If it fails, the move is aborted (unless on-failure is 'warn'). | No | - |
| after | object | Hook executed after the file operation completes successfully. | No | - |

### before

The following arguments are supported:

| Name | Type | Description | Required | Default |
|------|------|-------------|----------|---------|
| shell | string | Shell interpreter for hook commands. | No | /bin/sh |
| on-failure | string | What to do if a hook command exits non-zero: abort the file's operation, or warn and continue. | Yes | abort |
| run | array | Shell commands executed in order. | Yes | - |

### after

The following arguments are supported:

| Name | Type | Description | Required | Default |
|------|------|-------------|----------|---------|
| shell | string | Shell interpreter for hook commands. | No | /bin/sh |
| on-failure | string | What to do if a hook command exits non-zero: abort the file's operation, or warn and continue. | Yes | abort |
| run | array | Shell commands executed in order. | Yes | - |

