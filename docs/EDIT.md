# Interactive Config Editor

`movelooper edit` opens a two-panel TUI editor for your configuration file. It is the fastest way to create or modify a config without leaving the terminal, with validation on every save.

```bash
movelooper edit
```

---

## Layout

**Left panel — block list**\
Lists the top-level blocks in your config: `configuration`, and each of your categories. Navigate with **↑ / ↓**, open a block with **Enter**.

**Right panel — field editor**\
Shows the fields of the selected block. Each field has a type-appropriate control: text inputs for strings, toggles for booleans, dropdowns for enums. Nested objects expand inline.

---

## Keybindings

| Key | Action |
|---|---|
| **↑ / ↓** | Move between items |
| **Enter** | Open a block / confirm a value |
| **Esc** | Go back / cancel |
| **Tab** | Next field |
| **Ctrl+S** | Save |
| **Ctrl+U** | Undo last edit |
| **Ctrl+Y** | Redo |

---

## Saving and validation

**Ctrl+S** validates the entire config before writing. If there are errors, the editor shows them inline and refuses to save. Use `--no-validate-on-save` to override (a warning is shown, the file is still written).

---

## Creating a new config file

Use `--output` to write to a different file than the one loaded:

```bash
movelooper edit --output ~/projects/app/movelooper.yaml
```

Useful for bootstrapping a new config or creating a category file for use with `import:`.

---

## Themes

```bash
movelooper edit --theme dracula
movelooper edit --list-themes   # see all available themes
```

The default theme is `dark`. The same `--theme` flag applies to `show-docs`.

---

## Flags

| Flag | Description |
|---|---|
| `--theme` | Theme name (default: `dark`) |
| `--list-themes` | List available themes and exit |
| `--output`, `-o` | Write to this file instead of the loaded config |
| `--no-save-confirm` | Skip the save confirmation dialog |
| `--no-delete-confirm` | Skip the block-delete confirmation dialog |
| `--no-validate-on-save` | Allow saving with validation errors |
| `--config` | Load this config file (default: standard lookup) |

See [Commands](/COMMANDS.md) for the full flag reference for all commands.
