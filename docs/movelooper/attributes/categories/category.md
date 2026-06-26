# Category

## Examples

For usage examples, see [Category presets](../../examples/category.md).

## Arguments

The following arguments are supported:

| Name | Type | Description | Required | Default |
|------|------|-------------|----------|---------|
| name | string | Human-readable identifier for this category. Used in logs, history, and the --category filter flag. | Yes | - |
| enabled | bool | Whether this category is active. Must be explicitly set to true; omitting this field disables the category. | No | false |
| [source](./source.md) | object | Source directory configuration: which path to watch, which extensions to include, and how deep to scan. | Yes | - |
| [destination](./destination.md) | object | Destination configuration: where to place matched files, how to name them, and what to do on conflicts. | Yes | - |
| [hooks](./hooks.md) | object | Optional shell commands to run before and after each file is moved. | No | - |

