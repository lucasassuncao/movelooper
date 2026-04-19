# Changelog

All notable changes to movelooper are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Versioning follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Changed
- `dispatchAction` refactored from switch to `FileAction` Strategy interface + registry map
- `ConflictResolver` interface extended with `SkipMessage()` — each resolver owns its skip log message, eliminating OCP violation in `applyConflictStrategy`
- `preRunHandler` refactored into `AppBuilder` chainable steps (`ResolveConfig`, `ConfigureLogger`, `LoadConfig`, `LoadCategories`, `InitHistory`, `ValidateDirectories`)
- `resolveConfigPath` moved from `cmd` to `config` package as `ResolveConfigPath`

### Docs
- Added `docs/dev/DESIGN.md` — architecture and design patterns guide
- Added `docs/dev/DEVELOPMENT.md` — contributor workflow, conventions, CI, and release process
- Added `CHANGELOG.md`
- Added `CONTRIBUTING.md`

---

## [2.11.0] — 2026-04-17

### Added
- Before/after hooks for categories — run shell commands before and after a category processes files
- Hook `on-failure` policy: `abort` stops the category, `warn` logs and continues
- `ML_CATEGORY`, `ML_SOURCE_PATH`, `ML_DEST_PATH`, `ML_DRY_RUN`, `ML_ACTION`, `ML_FILES_MOVED`, `ML_FILES_FAILED`, `ML_BATCH_ID` environment variables injected into hook processes

---

## [2.10.0] — 2026-04-17

### Added
- `movelooper init --scan` — scans a directory and generates a config file from detected file types using a built-in extension dictionary

---

## [2.9.0] — 2026-04-17

### Added
- `{seq}` and `{seq:N}` tokens in `destination.rename` — auto-incrementing sequence number with optional zero-padded width (e.g. `{seq:4}` → `0001`)
- `{seq}` is valid only in `rename`, not in `organize-by`

---

## [2.8.0] — 2026-04-16

### Added
- `--category` flag on `move`, `watch`, and `undo` — comma-separated list of category names to process
- `--include-disabled` flag — include categories with `enabled: false`
- `undo --category` removes only matched category entries from a batch, leaving others intact
- `Category` field added to history entries

---

## [2.7.0] — 2026-04-16

### Added
- `any` and `all` boolean operators in `source.filter` — compose filters with OR (`any`) and AND (`all`) logic
- Operators can be nested arbitrarily; a node may not mix `any`/`all` with direct fields at the same level

---

## [2.6.0] — 2026-04-16

### Added
- `destination.action` field — `move` (default), `copy`, or `symlink`
- `destination.rename` template field — rename files at destination using tokens (`{name}`, `{ext}`, `{mod-date}`, `{category}`, `{seq}`, etc.)
- `undo` for `copy` and `symlink` removes the destination instead of restoring the source

---

## [2.5.3] — 2026-04-11

### Fixed
- CI: updated deprecated Node.js 20 GitHub Actions to Node.js 24
- goreleaser: corrected `archives.format` configuration

---

## [2.5.2] — 2026-04-11

### Changed
- Expanded test coverage across `helper`, `history`, and `cmd` packages
- Added `gosec` security scan to CI pipeline
- Restructured `docs/` directory

---

## [2.5.1] — 2026-04-11

### Fixed
- Watch batch IDs now use `crypto/rand` to prevent collisions under concurrent events
- History disk write moved outside the mutex to prevent lock contention
- `getUniqueDestinationPath` now errors after 1 000 attempts instead of looping indefinitely
- `os.Chtimes` failure surfaced as `ErrTimestampPreserve` warning instead of being silently discarded
- `LogCloser` is now deferred-closed on `preRunHandler` failure to prevent file handle leak
- Watch critical sections use `defer mu.Unlock()` via closures for correctness

---

## [2.5.0] — 2026-04-09

### Changed
- YAML config templates are now embedded at build time via `go:embed`; on-the-fly template generation removed

---

## [2.4.0] — 2026-04-09

### Changed
- Logger configuration refactored to use the Factory + Strategy pattern (`writerBuilder` interface, `logWriterStrategies` map)

---

## [2.3.0] — 2026-04-09

### Changed
- Configuration loading migrated from Viper to [koanf](https://github.com/knadh/koanf) — stricter parsing, no global state, better composability

---

## [2.2.0] — 2026-04-09

### Added
- `case-sensitive` field in `source.filter` — controls whether regex, glob, and include pattern matching is case-sensitive (default: false)
- `include` patterns in `source.filter` — whitelist glob patterns; a file must match at least one to pass

---

## [2.1.2] — 2026-04-09

### Fixed
- Resolved panic, error-handling gaps, and dead-code issues identified by static analysis

---

## [2.1.1] — 2026-04-09

### Fixed
- `movelooper config show` output updated to reflect `source.filter` nesting; column alignment corrected

---

## [2.1.0] — 2026-04-09

### Added
- `newest`, `oldest`, `larger`, and `smaller` conflict strategies in `destination.conflict-strategy`

---

## [2.0.0] — 2026-04-08

### Breaking changes
- Category schema redesigned — `source` and `destination` are now nested objects
- `organize-by` template system introduced for subdirectory organisation (`{ext}`, `{mod-year}`, `{size-range}`, `{category}`, etc.)
- Previous flat category format is not compatible

### Added
- `destination.organize-by` template field
- Full token set: `{ext}`, `{ext-upper}`, `{mod-year}`, `{mod-month}`, `{mod-day}`, `{mod-date}`, `{year}`, `{month}`, `{day}`, `{date}`, `{size-range}`, `{category}`

---

## [1.20.1] — 2026-04-08

### Fixed
- Surfaced the real underlying error when a config `import:` file fails to load

---

## [1.20.0] — 2026-04-08

### Added
- `import:` directive in config files — merge categories from one or more YAML files into the main config
- Circular import detection

---

## [1.19.1] — 2026-04-08

### Added
- GitHub Actions CI pipeline — format check, dependency check, tests, lint, security scan, docs generation

---

## [1.18.0] — 2026-04-07

### Added
- Startup validation warns when source or destination directories do not exist
- Watch mode logs every move event
- End-of-run summary: total files moved, total bytes, categories skipped

---

## [1.17.0] — 2026-04-07

### Added
- `--dry-run` flag for `undo` — preview what would be restored without changing anything

---

## [1.16.0] — 2026-04-07

### Added
- `enabled` field on categories — set `enabled: false` to skip a category without removing it

---

## [1.15.0] — 2026-04-07

### Added
- `use_extension_subfolder` (now `organize-by: {ext}`) — automatically place files in a subdirectory named after their extension

---

## [1.14.0] — 2026-04-07

### Changed
- `AppConfig` merged into `Configuration` — single config struct, no nested duplication

---

## [1.13.2] — 2026-04-07

### Fixed
- Self-update no longer prompts or errors when already on the latest version

---

## [1.13.1] — 2026-04-07

### Fixed
- Replaced Windows-specific path fallbacks with cross-platform `os.TempDir()`

---

## [1.13.0] — 2026-04-07

### Changed
- Batch ID generation centralised (`NewBatchID`, `NewWatchBatchID`)
- Ticker interval named constant (`tickerInterval`)
- Hash helpers moved to dedicated location

---

## [1.12.1] — 2026-04-07

### Fixed
- `CreateDirectory` simplified and errors surfaced correctly
- Watch mode surfaces errors instead of swallowing them
- Deduplicated `ErrUserAborted` handling

---

## [1.12.0] — 2026-04-07

### Changed
- `helper` package split into focused files (`fileops`, `filters`, `conflict`, `groupby`, `hooks`)
- Global variables removed
- Logging standardised across all packages
- Dead code removed

---

## [1.11.0] — 2026-04-07

### Changed
- Template functions (`organize-by`, `group-by`) refactored from repetitive switch statements to a data-driven approach

---

## [1.10.1] — 2026-04-07

### Fixed
- Cross-device move detection corrected — falls back to copy+delete only for `EXDEV`/`ERROR_NOT_SAME_DEVICE`, not for other `*os.LinkError` values
- File mode and timestamps preserved on copy

---

## [1.10.0] — 2026-04-07

### Added
- `--output` flag on `movelooper init` — choose `console`, `file`, or `both` when generating the config

---

## [1.9.0] — 2026-04-02

### Added
- `max-age` and `max-size` filter fields (complement to `min-age`/`min-size`)
- `movelooper self-update` command — downloads the latest release from GitHub

---

## [1.8.0] — 2026-04-02

### Added
- `--config` / `-c` flag unified across all commands
- `movelooper config validate` — validates the config file without running any operations
- `min-age` and `min-size` filter fields
- `history-limit` config field — controls how many batches are retained

---

## [1.7.0] — 2026-03-18

### Added
- `--version` flag

---

## [1.6.3] — 2026-03-17

### Fixed
- Error propagation corrected in several paths where errors were silently discarded
- Watch shutdown made graceful on all platforms
- Initial scan fixed to correctly populate the file tracker on startup
- Cross-device move handling improved

---

## [1.6.2] — 2026-03-17

### Fixed
- Log file handle leak on shutdown
- Mutex contention in watch mode reduced
- Regex pre-compilation validated at config load time

---

## [1.6.1] — 2026-03-17

### Fixed
- Graceful watch shutdown on SIGINT/SIGTERM
- History pruning applied correctly when limit is exceeded
- Regex compiled once at startup instead of per-file

---

## [1.6.0] — 2025-12-03

### Added
- `movelooper undo` command — restores files moved in a previous batch

### Changed
- Category logic simplified; duplicate matching logic consolidated into the `helper` package

---

## [1.5.1] — 2025-12-01

### Fixed
- Log messages when using regex filter with `use_extension_subfolder`

---

## [1.5.0] — 2025-11-30

### Added
- `regex` filter field in category `source` — move only files whose names match a regular expression

---

## [1.4.0] — 2025-11-30

### Changed
- Interactive `init` forms migrated from `survey` to [charmbracelet/huh](https://github.com/charmbracelet/huh)

---

## [1.3.0] — 2025-11-20

### Added
- `movelooper watch` command — monitors source directories and moves files in real time as they appear and stabilise

---

## [1.2.0] — 2025-11-20

### Added
- Conflict strategies: `rename` (default), `overwrite`, `skip`, `hash_check`
- `hash_check` deduplicates identical files by SHA-256 hash

---

## [1.1.0] — 2025-11-13

### Changed
- `move` subcommand removed — file moving is now the default behaviour of the root command (`movelooper`)

---

## [1.0.0] — 2025-10-29

### Added
- Initial stable release
- Root command moves files from configured source directories to destination directories
- Category-based configuration via `movelooper.yaml`
- `movelooper init` — interactive config file generator
- `movelooper config show` — displays the loaded configuration
- Logging to console, file, or both via `pterm`
- Cobra CLI framework, Viper config loading

[Unreleased]: https://github.com/lucasassuncao/movelooper/compare/v2.11.0...HEAD
[2.11.0]: https://github.com/lucasassuncao/movelooper/compare/v2.10.0...v2.11.0
[2.10.0]: https://github.com/lucasassuncao/movelooper/compare/v2.9.0...v2.10.0
[2.9.0]: https://github.com/lucasassuncao/movelooper/compare/v2.8.0...v2.9.0
[2.8.0]: https://github.com/lucasassuncao/movelooper/compare/v2.7.0...v2.8.0
[2.7.0]: https://github.com/lucasassuncao/movelooper/compare/v2.6.0...v2.7.0
[2.6.0]: https://github.com/lucasassuncao/movelooper/compare/v2.5.3...v2.6.0
[2.5.3]: https://github.com/lucasassuncao/movelooper/compare/v2.5.2...v2.5.3
[2.5.2]: https://github.com/lucasassuncao/movelooper/compare/v2.5.1...v2.5.2
[2.5.1]: https://github.com/lucasassuncao/movelooper/compare/v2.5.0...v2.5.1
[2.5.0]: https://github.com/lucasassuncao/movelooper/compare/v2.4.0...v2.5.0
[2.4.0]: https://github.com/lucasassuncao/movelooper/compare/v2.3.0...v2.4.0
[2.3.0]: https://github.com/lucasassuncao/movelooper/compare/v2.2.0...v2.3.0
[2.2.0]: https://github.com/lucasassuncao/movelooper/compare/v2.1.2...v2.2.0
[2.1.2]: https://github.com/lucasassuncao/movelooper/compare/v2.1.1...v2.1.2
[2.1.1]: https://github.com/lucasassuncao/movelooper/compare/v2.1.0...v2.1.1
[2.1.0]: https://github.com/lucasassuncao/movelooper/compare/v2.0.0...v2.1.0
[2.0.0]: https://github.com/lucasassuncao/movelooper/compare/v1.20.1...v2.0.0
[1.20.1]: https://github.com/lucasassuncao/movelooper/compare/v1.20.0...v1.20.1
[1.20.0]: https://github.com/lucasassuncao/movelooper/compare/v1.19.1...v1.20.0
[1.19.1]: https://github.com/lucasassuncao/movelooper/compare/v1.19.0...v1.19.1
[1.19.0]: https://github.com/lucasassuncao/movelooper/compare/v1.18.0...v1.19.0
[1.18.0]: https://github.com/lucasassuncao/movelooper/compare/v1.17.0...v1.18.0
[1.17.0]: https://github.com/lucasassuncao/movelooper/compare/v1.16.0...v1.17.0
[1.16.0]: https://github.com/lucasassuncao/movelooper/compare/v1.15.0...v1.16.0
[1.15.0]: https://github.com/lucasassuncao/movelooper/compare/v1.14.0...v1.15.0
[1.14.0]: https://github.com/lucasassuncao/movelooper/compare/v1.13.2...v1.14.0
[1.13.2]: https://github.com/lucasassuncao/movelooper/compare/v1.13.1...v1.13.2
[1.13.1]: https://github.com/lucasassuncao/movelooper/compare/v1.13.0...v1.13.1
[1.13.0]: https://github.com/lucasassuncao/movelooper/compare/v1.12.1...v1.13.0
[1.12.1]: https://github.com/lucasassuncao/movelooper/compare/v1.12.0...v1.12.1
[1.12.0]: https://github.com/lucasassuncao/movelooper/compare/v1.11.0...v1.12.0
[1.11.0]: https://github.com/lucasassuncao/movelooper/compare/v1.10.1...v1.11.0
[1.10.1]: https://github.com/lucasassuncao/movelooper/compare/v1.10.0...v1.10.1
[1.10.0]: https://github.com/lucasassuncao/movelooper/compare/v1.9.0...v1.10.0
[1.9.0]: https://github.com/lucasassuncao/movelooper/compare/v1.8.0...v1.9.0
[1.8.0]: https://github.com/lucasassuncao/movelooper/compare/v1.7.0...v1.8.0
[1.7.0]: https://github.com/lucasassuncao/movelooper/compare/v1.6.3...v1.7.0
[1.6.3]: https://github.com/lucasassuncao/movelooper/compare/v1.6.2...v1.6.3
[1.6.2]: https://github.com/lucasassuncao/movelooper/compare/v1.6.1...v1.6.2
[1.6.1]: https://github.com/lucasassuncao/movelooper/compare/v1.6.0...v1.6.1
[1.6.0]: https://github.com/lucasassuncao/movelooper/compare/v1.5.1...v1.6.0
[1.5.1]: https://github.com/lucasassuncao/movelooper/compare/v1.5.0...v1.5.1
[1.5.0]: https://github.com/lucasassuncao/movelooper/compare/v1.4.0...v1.5.0
[1.4.0]: https://github.com/lucasassuncao/movelooper/compare/v1.3.0...v1.4.0
[1.3.0]: https://github.com/lucasassuncao/movelooper/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/lucasassuncao/movelooper/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/lucasassuncao/movelooper/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/lucasassuncao/movelooper/releases/tag/v1.0.0
