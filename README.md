<!-- markdownlint-disable MD033 -->
<p align="center">
  <img src="docs/pages/movelooper2.png" alt="Movelooper logo" width="300" height="300">
</p>
<!-- markdownlint-enable MD033 -->

🌀 **Movelooper** is a modern CLI tool that automatically organizes and moves your files based on configurable categories.

Are your files a mess? **Movelooper** fixes that.\
Tired of moving files by hand? **Movelooper** does it for you.\
Scared of losing something? Every move is recorded and undoable.\
Not sure it will work? Run `--dry-run` and see exactly what happens before touching anything.

For example, your Downloads folder has 847 files... You haven't sorted them in 6 months. You know you won't do it manually.\
You want to organize them by file type, date, and size, but you also want to rename them in a consistent way.\
You want to avoid duplicates and conflicts and do it quickly and safely.

That's why you use `movelooper`.

Write one YAML config file, run `movelooper`, and it will automatically move and organize your files into the right folders.\
Movelooper can also watch your folders in real-time and move files as they arrive, so you never have to worry about clutter again.

## Features

### Organize

- Move files from source to destination based on categories defined in a YAML config file
- Select actions per category: move, copy, symlink, or archive (.zip or .tar.gz), see [Actions](https://lucasassuncao.github.io/movelooper/docs/#/ACTIONS) for all available actions
- Filter files by extension, regex, glob, age, size, and real content type (magic bytes), see [Filters](https://lucasassuncao.github.io/movelooper/docs/#/FILTERS) for all available filters
- Configure conflict strategies per category: rename, overwrite, skip, hash_check, and more, see [Conflict Strategies](https://lucasassuncao.github.io/movelooper/docs/#/CONFLICTS) for all available strategies
- Organize files into subdirectories using template tokens: `{ext}`, `{mod-year}`, `{mod-month}`, `{size-range}`, see [Tokens](https://lucasassuncao.github.io/movelooper/docs/#/TOKENS) for all available tokens
- Rename files at the destination using a rich token engine, see [Tokens](https://lucasassuncao.github.io/movelooper/docs/#/TOKENS) for all available tokens
- Use a catch-all category with `extensions: [all]` to organize any file type by its real extension
- Keep a history of all moves in `~/.movelooper/history/movelooper.json` for auditing and undoing

### Automate

- Use `--dry-run` to preview what would happen without moving any files
- Use Watch mode to automatically move files as they arrive in the source folder, see [Watch Mode](https://lucasassuncao.github.io/movelooper/docs/#/WATCH) for reference
- Use Undo command to roll back any batch of moves, or preview what would be undone with `undo --dry-run`
- Use Hooks to trigger scripts or webhooks after each category, for example to notify, log, or validate the move, see [Hooks](https://lucasassuncao.github.io/movelooper/docs/#/HOOKS) for reference

### Configure

- Split config across multiple YAML files and import them using `import:` statements
- Use the `edit` command to open a rich interactive TUI editor for your config file, with validation on save
- Self-update with `self-update`

## How It Works

`movelooper` reads your configuration file (defaults to `movelooper.yaml` or `conf/movelooper.yaml`),\
it scans all extensions listed per category, and processes matching files from the source to the destination\
following the rules defined in the config. It keeps a history of all moves in `~/.movelooper/history/movelooper.json` so you can undo any batch any time.

## Getting Started

Follow the [Getting Started](https://lucasassuncao.github.io/movelooper/docs/#/GETTING-STARTED) guide to install and set up `movelooper`.

## Documentation

See the [Documentation](https://lucasassuncao.github.io/movelooper/docs/) for detailed information on how to use `movelooper`, including configuration options, commands, and examples.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on reporting issues and submitting pull requests.
