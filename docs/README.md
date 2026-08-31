<!-- markdownlint-disable MD033 -->
<p align="center">
  <img src="../movelooper2.png" alt="Movelooper logo" width="300" height="300">
</p>
<!-- markdownlint-enable MD033 -->

🌀 **Movelooper** is a modern CLI tool that automatically organizes your files, using a declarative rules engine where configurable categories describe what goes where and Movelooper does it.

## Given this `movelooper.yaml` file

```yaml
categories:
  - name: images
    enabled: true
    source:
      path: ./Downloads
      extensions: [jpg, png]
    destination:
      path: ./Pictures
      organize-by: "{mod-year}/{mod-month}"   # subfolders by modification date
      conflict-strategy: hash_check           # identical file already there? drop the duplicate

  - name: documents
    enabled: true
    source:
      path: ./Downloads
      extensions: [pdf]
      filter:
        not:
          - match:
              glob: "draft_*"                 # leave drafts alone
    destination:
      path: ./Documents
      rename: "{mod-date}_{name}.{ext}"       # report.pdf -> 2026-08-31_report.pdf
      conflict-strategy: rename
```

Note: That file is `examples/demo/movelooper.yaml`, the exact config running in the recording below.

## Demo

<!-- markdownlint-disable MD033 -->
<p align="left">
  <img src="../examples/demo/demo.gif" alt="movelooper sorting a Downloads folder: the files before, the run with --show-files, and the resulting tree" width="100%">
</p>
<!-- markdownlint-enable MD033 -->

## This is what it does

```text
BEFORE                          AFTER

Downloads/                      Downloads/
├── vacation.jpg                ├── draft_notes.pdf
├── screenshot.png              └── setup.exe
├── invoice_2026-03.pdf
├── report.pdf                  Pictures/2026/08/
├── draft_notes.pdf             ├── vacation.jpg
└── setup.exe                   └── screenshot.png

                                Documents/
                                ├── 2026-08-31_invoice_2026-03.pdf
                                └── 2026-08-31_report.pdf
```

`draft_notes.pdf` stayed because a filter excluded it. `setup.exe` stayed because no category claimed it. A file no rule matches is a file movelooper never touches.

Run `movelooper --dry-run` first and you get every destination resolved and printed, with nothing moved. Run `movelooper undo` afterwards and the whole batch goes back.

## Install

Download the binary for your platform from the [releases page](https://github.com/lucasassuncao/movelooper/releases) and put it on your `PATH`. There is nothing else to install: no runtime, no dependencies. Linux, macOS and Windows are built and tested on every change.

```bash
movelooper --version
movelooper self-update      # later, to upgrade in place
```

## Quick start

```bash
movelooper edit             # build the config in an interactive TUI
movelooper validate         # check it before it ever touches a file
movelooper --dry-run        # see exactly what would happen
movelooper                  # do it
movelooper undo             # change your mind
```

With no config anywhere, `movelooper edit` starts a new one at `~/.movelooper/conf/movelooper.yaml`. The editor lists every available field with its allowed values, so the config is discoverable without reading the reference. See [Getting Started](/GETTING-STARTED.md) for the guided version.

## What it can do

**Rules**

- Filter by extension, literal, regex, glob, age, size, and real content type (magic bytes), composed with `any`, `all`, and `not`. See [Filters](/FILTERS.md).
- Choose the action per category: `move`, `copy`, `symlink`, or `archive` into a `.zip`/`.tar.gz`. See [Actions](/ACTIONS.md).
- Decide what happens when the destination is occupied: `rename`, `overwrite`, `skip`, `hash_check`, `newest`, `oldest`, `larger`, `smaller`. See [Conflict Strategies](/CONFLICTS.md).
- Build destination subfolders and filenames from tokens: dates, extension, size range, category, counters, content hashes. See [Tokens](/TOKENS.md).
- Catch everything a rule missed with `extensions: [all]`, organized by its real extension.

**Running**

- Organize on demand, over any folder you point at: Downloads, a scanner's output, a photo dump, whatever another process writes into.
- Or run `watch` in the background and let files be organized as they arrive, once they stop changing. See [Watch Mode](/WATCH.md).
- Trigger scripts or webhooks around each category with [Hooks](/HOOKS.md).

**Config**

- One YAML file, or many: split it up and pull the pieces in with `import:`.
- `movelooper edit` for the guided TUI, `movelooper validate` for CI and cron.
- `validate` also warns about configs that are valid but lose files in ways people rarely intend, before the run instead of afterwards from the missing files.

## Safety

**Movelooper never destroys the last copy of a file.** Bytes are only removed where they demonstrably exist somewhere else: `hash_check` drops a source whose twin is already at the destination, and `archive` only deletes originals when you set `keep-source: false` and they are inside the archive. There is no delete rule, and nothing is destroyed to make room.

Every completed move is recorded in `~/.movelooper/history/movelooper.json`, and `undo` replays any batch backwards. If the history cannot be written, movelooper says so and exits non-zero instead of reporting a success you cannot reverse. An interrupted run still records what it moved.

The full contract, including what is deliberately *not* promised, is in [Guarantees and Limits](/GUARANTEES.md).

## Where to go next

- [Getting Started](/GETTING-STARTED.md): install and build your first config
- [Configuration](/CONFIGURATION.md): the `configuration:` block, logging, watch, history, defaults, imports
- [Categories](/CATEGORIES.md): the `categories:` block, source, destination, hooks
- [Commands and Flags](/COMMANDS.md): every command and flag
- [Cookbook](/COOKBOOK.md): ready-made recipes
- [FAQ](/FAQ.md): common questions
- [Troubleshooting](/TROUBLESHOOTING.md): when something does not behave
