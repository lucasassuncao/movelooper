# Interactive Config Editor

`movelooper edit` opens a TUI editor for your configuration file. It is the fastest way to create or modify a config without leaving the terminal, with validation on every save.

```bash
movelooper edit
```

---

## Layout

**Block list**\
The first screen. The left pane lists the top-level blocks of your config, grouped as `ADDED` (in the file), `AVAILABLE` (declared by the schema but not in the file yet), `UNKNOWN` (in the file but not in the schema) and `PASSTHROUGH` (kept as written, such as `import`). The right pane shows the whole file as highlighted YAML and follows the selection. Below it, the **Hint/Example** pane describes the selected block.

**Block editor**\
**Enter** on a block opens it. The left pane is the block's field tree; the right pane previews the block's YAML, and becomes an editable YAML buffer when focused. The **Hint/Example** pane describes the selected field. A list or map field opens its own editor one level down, and the header shows the path as a breadcrumb (`categories › images › source › filter`).

The **Hint/Example** pane only informs, so **Tab** never stops on it. **h** shows or hides it, and **Ctrl+H** focuses it to scroll a long hint.

---

## Keybindings

The legend at the bottom always lists the keys of the screen in front, and **?** opens the full list.

**Block list**

| Key | Action |
|---|---|
| **↑ / ↓** | Move between blocks |
| **Enter / →** | Open the block, or add it when it is not in the file yet |
| **Tab** | Switch between the block list and the file preview |
| **/** | Filter the list |
| **p** | Pick a whole-document preset |
| **h** | Show or hide the Hint/Example pane |
| **Ctrl+H** | Focus the Hint/Example pane to scroll it |
| **Ctrl+D** | Delete the selected block |
| **Ctrl+S** | Save the file |
| **Ctrl+U / Ctrl+Y** | Undo / redo the last document change |
| **Ctrl+R** | Reload the file from disk |
| **Ctrl+L** | Validate without saving |
| **q** | Quit (asks first when there are unsaved changes) |

In the file preview, **↑ / ↓** scroll, and **Tab** or **Esc** go back to the list.

**Block editor**

| Key | Action |
|---|---|
| **↑ / ↓** | Move between fields |
| **→ / ←** | Expand / collapse a field |
| **Enter** | Add the field, or open a nested list or map |
| **Tab** | Switch between the field tree and the YAML buffer |
| **p** | Pick a preset for this block, when it has any |
| **h** | Show or hide the Hint/Example pane |
| **Ctrl+H** | Focus the Hint/Example pane to scroll it |
| **Ctrl+D** | Remove the field (or delete the entry, in a list) |
| **Ctrl+U / Ctrl+Y** | Undo / redo the last edit |
| **Ctrl+S** | Apply the block to the document and return to the list |
| **Ctrl+L** | Validate without saving |
| **Esc** | Go up one level; from the top, back to the list (asks first when the block has unapplied changes) |

**Ctrl+C** quits from any screen.

---

## Saving and validation

**Ctrl+S** in the block editor only applies the block to the document; the file is written by **Ctrl+S** in the block list. That save validates the entire config first: if there are errors, a dialog lists them and nothing is written. Use `--no-validate-on-save` to save anyway; you are then asked to confirm, with the errors listed as warnings.

If the file changed on disk since it was opened, saving always asks before overwriting it, even with `--no-save-confirm`.

---

## Creating a new config file

With no config in any of the search locations, `movelooper edit` opens an empty one at `~/.movelooper/conf/movelooper.yaml` and creates the directory on save, so a fresh install needs nothing but this command.

Use `--output` to write somewhere else instead:

```bash
movelooper edit --output ~/projects/app/movelooper.yaml
```

Useful for bootstrapping a new config or creating a category file for use with `import:`.

---

## Themes

```bash
movelooper edit --theme grape
movelooper edit --list-themes   # see all available themes
```

The default theme is `plain`.

---

## Flags

| Flag | Description |
|---|---|
| `--theme` | Theme name (default: `plain`) |
| `--list-themes` | Browse every available theme and its category in a scrollable table (`q` quits) |
| `--output`, `-o` | Write to this file instead of the loaded config |
| `--no-save-confirm` | Skip the save confirmation dialog |
| `--no-delete-confirm` | Skip the block-delete confirmation dialog |
| `--no-validate-on-save` | Allow saving with validation errors |
| `--dump` | Record every editor action to a JSONL trace file for bug reports (the path is printed on exit) |
| `--dump-path` | Write the session trace to this file instead of a temp file (implies `--dump`) |
| `--config` | Load this config file (default: standard lookup) |

See [Commands](/COMMANDS.md) for the full flag reference for all commands.
