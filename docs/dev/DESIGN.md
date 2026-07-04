# Design & Architecture Guide

This document explains the architectural decisions behind movelooper — why the code is structured the way it is, which patterns are used and where, and what the rules are for extending each subsystem. It is intended for contributors who want to understand the project before making changes.

---

## Table of Contents

1. [High-level architecture](#1-high-level-architecture)
2. [Package responsibilities](#2-package-responsibilities)
3. [Design patterns](#3-design-patterns)
4. [Data flow — move operation](#4-data-flow--move-operation)
5. [Configuration lifecycle](#5-configuration-lifecycle)
6. [Extending the system](#6-extending-the-system)
7. [Key decisions and trade-offs](#7-key-decisions-and-trade-offs)

---

## 1. High-level architecture

```
main.go
  └── cmd.RootCmd(m *Movelooper, version)
        ├── PersistentPreRunE → config.AppBuilder → populates m
        ├── RunE             → runMove(m, ...)
        └── subcommands: watch, undo, edit, validate, config, self-update, show-docs
```

The application is built around a single shared state object, `models.Movelooper`, which is created empty in `main.go` and populated by `AppBuilder` before every command runs. Commands read from it; they never write back. This keeps command logic stateless and easy to test.

Koanf is used only during startup. Once `AppBuilder.Build()` returns, the koanf instance is discarded and the rest of the application works exclusively with typed structs (`m.Config`, `m.Categories`, `m.Logger`, `m.History`).

---

## 2. Package responsibilities

| Package | Responsibility |
|---|---|
| `main` | Entry point. Creates `*Movelooper`, delegates everything to `cmd`. |
| `internal/cmd` | CLI commands (cobra). Orchestrates operations; contains no file I/O logic. |
| `internal/config` | Reads, validates, and builds configuration. Houses `AppBuilder`. |
| `internal/models` | Core data types. Imports `logger`, `history`, and `tokens` to type the `Movelooper` struct fields and validate template patterns. |
| `internal/fileops` | File operations: move, copy, symlink, conflict resolution, `MoveContext`. |
| `internal/filters` | Extension matching, match (literal/regex/glob)/age/size/not filters, `MatchesFilter`. |
| `internal/hooks` | Shell hook execution (`RunHook`). |
| `internal/tokens` | Template token resolution (`ResolveGroupBy`, `ResolveRename`) and validation. |
| `internal/history` | Reads and writes the JSON operation log. Thread-safe. |
| `internal/scanner` | Walks a category's source directory and returns the files eligible for moving (`WalkSource`). |
| `internal/archive` | Packs sets of files into zip or tar.gz archives. Config-agnostic: takes explicit (source, entry-name) pairs. |
| `internal/content` | Detects a file's real MIME type from magic bytes, independent of extension. Wraps `gabriel-vasile/mimetype`. |
| `internal/logger` | `Logger` interface (thin wrapper over `*pterm.Logger`). Lets non-`cmd` packages accept a logger without importing pterm directly. |
| `internal/terminal` | Terminal width detection for log formatting. |
| `internal/updater` | Self-update logic (GitHub releases). |

**Dependency rule:** `logger`, `content`, and `history` are leaf packages — they import nothing internal. `tokens` imports `content`. `models` imports `history`, `logger`, and `tokens` (to type `Movelooper` fields and validate template patterns). `fileops`, `filters`, `hooks`, and `scanner` import `models` and other leaves as needed. `config` imports `filters`, `tokens`, `history`, and `models`. `cmd` imports all of the above. The graph is strictly acyclic — no upward imports.

---

## 3. Design patterns

### 3.1 Strategy + Registry

Used in three places. Each follows the same shape: an interface, a map of named implementations, and a factory/dispatcher that looks up the right implementation by name. Unknown names fall back to a safe default.

**Logging output** (`internal/config/logging.go`)

```
writerBuilder interface
  consoleStrategy  →  os.Stdout
  fileStrategy     →  log file
  multiStrategy    →  io.MultiWriter(stdout, file)

logWriterStrategies map[string]writerBuilder
logWriterFactory(output string) writerBuilder
```

**Conflict resolution** (`internal/fileops/conflict.go`)

```
ConflictResolver interface { Resolve(...) / SkipMessage() string }
  renameResolver, overwriteResolver, skipResolver, hashCheckResolver,
  comparatorResolver  (newest / oldest / larger / smaller — one type,
                       parameterised by a comparison predicate + skip message)

conflictResolvers map[string]ConflictResolver
```

`SkipMessage()` is part of the interface so each resolver owns its own log message. `applyConflictStrategy` in `fileops.go` does not need to know strategy names to produce log output — adding a new resolver requires only one implementation and one map entry (or, for a value comparison, a new `comparatorResolver` entry).

**File actions** (`internal/fileops/fileops.go`)

```
FileAction interface { Execute(ctx context.Context, src, dst string) error }
  moveAction, copyAction, symlinkAction

fileActions map[string]FileAction
```

Unknown or empty action names fall back to `moveAction`. This is consistent with how conflict resolution handles unknown strategies.

**Rule for adding a new strategy/action/resolver:** implement the interface, add one entry to the map. No other files change.

---

### 3.2 Builder

`NewApp` (`internal/config/builder.go`) constructs a fully initialised `*Movelooper` by running a fixed sequence of functional options:

```go
config.NewApp(m, configPath,
    config.WithLogger(),
    config.WithFormatOverride(formatOverride),
    config.WithConfig(),
    config.WithCategories(),
    config.WithHistory(),
    config.WithValidateDirs(),
)
```

Each `With*` option enables one initialization step. Steps always run in declaration order and share an error-stop: if any step fails, subsequent steps are skipped and the first error is returned.

**Why not in `cmd`:** the config package owns all the knowledge needed to construct a `Movelooper` (path resolution, YAML parsing, logger setup, category validation). Keeping this logic there avoids leaking it into the command layer. `preRunHandler` in `cmd/root.go` stays a thin orchestrator responsible only for user-facing error formatting and closing the log file on failure.

---

### 3.3 Context Object

`MoveContext` (`internal/fileops/fileops.go`) carries the dependencies needed by file-move operations:

```go
type MoveContext struct {
    Logger  logger.Logger
    History *history.History
}
```

It is intentionally narrow — callers supply only what file operations need, not the full `Movelooper` object. This prevents file operation functions from depending on application-level state and makes them easier to test in isolation.

`MoveFiles` also accepts a `context.Context` as its first argument. The context is checked at the start of each file iteration, allowing the caller to cancel a long-running batch (e.g. on SIGINT in watch mode).

---

### 3.4 Adapter

`fileInfoDirEntry` (`internal/cmd/watch_helper.go`) adapts `os.FileInfo` to the `os.DirEntry` interface. Watch mode detects files via `fsnotify` events and retrieves metadata with `os.Lstat`, but downstream helpers (`MoveFiles`, filters) all expect `os.DirEntry`. The adapter bridges this gap without modifying either side.

---

## 4. Data flow — move operation

```
runMove(ctx, m, ...)
  │
  ├── filterCategories(m.Categories, names, includeDisabled)
  │
  └── for each category:
        processCategoryMove(ctx, m, category, ...)
          │
          ├── hooks.RunHook(ctx, category.Hooks.Before, ...)   ← optional
          ├── scanner.WalkSource(ctx, category.Source, autoExclude)  → all entries (honours recursive/max-depth/excludes)
          │     └── group entries by extension (one pass)
          │
          └── for each configured extension:
                for each candidate: matchesCategory → filters.MatchesFilter (recursive any/all/leaf)
                  └── a per-category `seen` set claims each file for the first matching extension
                │
                fileops.MoveFiles(ctx, MoveContext, MoveRequest{Category, Files, Extension, BatchID, SourceDir})
                  └── for each file:
                        [ctx.Done() check]
                        tokens.ResolveGroupBy  → destDir
                        tokens.ResolveRename   → destName
                        applyConflictStrategy  → ConflictResolver.Resolve()
                        dispatchAction         → FileAction.Execute()
                        history.Add(Entry{...})
          │
          └── hooks.RunHook(ctx, category.Hooks.After, ...)    ← optional
```

The `movedSet` in `runMove` tracks absolute paths already processed in the current batch. A file claimed by the first matching category cannot be claimed again by a later one.

---

## 5. Configuration lifecycle

```
YAML file
  └── config.ResolveConfigPath   → absolute path
  └── config.ResolveImports      → merged YAML bytes (import: chaining)
  └── koanf.Load                 → in-memory key-value store
        │
        ├── config.ConfigureLogger  → *pterm.Logger (Strategy: writerBuilder)
        ├── config.LoadConfig       → models.Configuration
        └── config.UnmarshalConfig  → []*models.Category
              └── validateCategory  → compileRegex, validateFilter, validateHooks
```

After `AppBuilder.Build()` returns, koanf is discarded. No other part of the application reads YAML or accesses koanf.

**Import chaining** (`internal/config/imports.go`): a config file may declare `import: [other.yaml]`. `ResolveImports` merges all imported files' `categories:` blocks into the main file before loading into koanf. Circular imports are detected via a visited-path set.

---

## 6. Extending the system

### Add a new file action (e.g. `hardlink`)

1. In `internal/fileops/fileops.go`, define:
   ```go
   type hardlinkAction struct{}
   func (a *hardlinkAction) Execute(_ context.Context, src, dst string) error { return os.Link(src, dst) }
   ```
2. Register it:
   ```go
   var fileActions = map[string]FileAction{
       ...
       "hardlink": &hardlinkAction{},
   }
   ```
3. Add `"hardlink"` to `validActions` in `internal/config/config.go`.
4. No other files change.

### Add a new conflict strategy (e.g. `newest_or_rename`)

1. In `internal/fileops/conflict.go`, define a struct implementing `ConflictResolver`:
   ```go
   type newestOrRenameResolver struct{}
   func (r *newestOrRenameResolver) Resolve(...) (string, bool, error) { ... }
   func (r *newestOrRenameResolver) SkipMessage() string { return "" }
   ```
2. Register it in `conflictResolvers`.
3. No other files change — `applyConflictStrategy` calls `resolver.SkipMessage()` generically.

### Add a new log output mode (e.g. `syslog`)

1. In `internal/config/logging.go`, define a struct implementing `writerBuilder`.
2. Register it in `logWriterStrategies`.
3. No other files change.

### Add a new template token (e.g. `{mytoken}`)

1. In `internal/tokens/resolve.go`, add the token to `buildStaticPairs`.
2. Add the token name to `knownTokens` in `internal/tokens/validate.go`.
3. No other files change — `ResolveGroupBy` and `ResolveRename` use `buildStaticPairs` generically.

### Add a new AppBuilder step

1. Add a `With*` option function and its corresponding initialization step in `internal/config/builder.go`.
2. Pass the new `With*` to `config.NewApp` in `preRunHandler` (`internal/cmd/root.go`).

---

## 7. Key decisions and trade-offs

### Koanf is discarded after startup

Koanf is used only to load and validate the YAML file. Once `AppBuilder.Build()` returns, all configuration lives in typed Go structs. This avoids string-keyed config access scattered through the codebase and makes it impossible to read a config key that no longer exists.

### `models` depends on `logger`, `history`, and `tokens`

`models` imports three internal packages: `logger` and `history` to type the fields of `Movelooper`, and `tokens` to implement the format-validator functions in `formats.go`. This is intentional — the alternative would be to use raw `interface{}` or duplicate the validator logic. The dependency is one-directional: `logger`, `history`, and `tokens` do not import `models`, so there are no cycles.

### History is always written per-file, not per-batch

`history.Add` is called immediately after each successful file operation. This means a partial run (e.g. interrupted mid-batch) still produces a correct, undoable history. The batch ID groups entries logically without requiring an atomic commit.

### Watch mode uses a stability window, not inotify CLOSE_WRITE

`fsnotify` does not expose `CLOSE_WRITE` on all platforms. Instead, watch mode records the time of the most recent create/write event for each file and only moves it once no further event has arrived for longer than `watch.delay` (default 5 minutes). Pending files live in an indexed min-heap keyed by that timestamp, so each tick pops only the files whose quiet period has elapsed instead of scanning every tracked file. This handles large or slow file writes correctly on all supported platforms.

### Conflict resolution and file actions are open/closed

Both `ConflictResolver` and `FileAction` follow the Open/Closed Principle: adding a new strategy requires only a new struct and one map entry. The dispatch functions (`applyConflictStrategy`, `dispatchAction`) never need to be modified. `SkipMessage()` is part of the `ConflictResolver` interface for the same reason — each resolver owns its log message so the dispatcher stays ignorant of strategy names.

### `AppBuilder` error accumulation

Each builder method checks `b.err != nil` at the start and returns immediately if an error was already set. This allows the fluent chain to be written without intermediate error checks while still stopping at the first failure. The trade-off is that only the first error is surfaced; subsequent steps never run, so their errors are never seen. This is acceptable because startup errors are fatal and the first one is always the most actionable.
