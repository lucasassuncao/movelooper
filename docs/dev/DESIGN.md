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
        └── subcommands: watch, undo, config, init, self-update
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
| `internal/models` | Pure data types shared across packages. No logic, no imports from other internal packages. |
| `internal/helper` | File operations, filters, conflict resolution, hooks, groupby/rename templates. |
| `internal/history` | Reads and writes the JSON operation log. Thread-safe. |
| `internal/scanner` | Scans a directory and maps extensions to built-in category names (`init --scan`). |
| `internal/terminal` | Terminal width detection for log formatting. |
| `internal/updater` | Self-update logic (GitHub releases). |

**Dependency rule:** `models` imports nothing internal. `helper` and `history` import only `models`. `config` imports `helper`, `history`, and `models`. `cmd` imports all of the above. This is a strict acyclic graph — no cycles, no upward imports.

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

**Conflict resolution** (`internal/helper/conflict.go`)

```
ConflictResolver interface { Resolve(...) / SkipMessage() string }
  renameResolver, overwriteResolver, skipResolver,
  hashCheckResolver, newestResolver, oldestResolver,
  largerResolver, smallerResolver

conflictResolvers map[string]ConflictResolver
```

`SkipMessage()` is part of the interface so each resolver owns its own log message. `applyConflictStrategy` in `fileops.go` does not need to know strategy names to produce log output — adding a new resolver requires only a new struct and one map entry.

**File actions** (`internal/helper/fileops.go`)

```
FileAction interface { Execute(src, dst string) error }
  moveAction, copyAction, symlinkAction

fileActions map[string]FileAction
```

Unknown or empty action names fall back to `moveAction`. This is consistent with how conflict resolution handles unknown strategies.

**Rule for adding a new strategy/action/resolver:** implement the interface, add one entry to the map. No other files change.

---

### 3.2 Builder

`AppBuilder` (`internal/config/builder.go`) constructs a fully initialised `*Movelooper` through a chain of named steps:

```go
config.NewAppBuilder(m, configPath).
    ResolveConfig().
    ConfigureLogger().
    LoadConfig().
    LoadCategories().
    InitHistory().
    ValidateDirectories().
    Build()
```

Each method is a no-op when a previous step has already set an error (error accumulation pattern). `Build()` returns the first error encountered.

**Why Builder instead of Functional Options:** the startup steps have strict ordering dependencies (logger requires config, categories require koanf, history requires `HistoryLimit`). Functional Options are unordered by design and do not model dependencies well. Builder makes the sequence explicit and each step independently testable.

**Why not in `cmd`:** the config package owns all the knowledge needed to construct a `Movelooper` (path resolution, YAML parsing, logger setup, category validation). Keeping `AppBuilder` there avoids leaking config logic into the command layer. `preRunHandler` in `cmd/root.go` becomes a thin orchestrator (~12 lines) responsible only for user-facing error formatting and closing the log file on failure.

---

### 3.3 Context Object

`MoveContext` (`internal/helper/fileops.go`) carries the dependencies needed by file-move operations:

```go
type MoveContext struct {
    Logger  *pterm.Logger
    History *history.History
}
```

It is intentionally narrow — callers supply only what file operations need, not the full `Movelooper` object. This prevents helper functions from depending on application-level state and makes them easier to test in isolation.

---

### 3.4 Adapter

`fileInfoDirEntry` (`internal/cmd/watch.go`) adapts `os.FileInfo` to the `os.DirEntry` interface. Watch mode detects files via `fsnotify` events and retrieves metadata with `os.Lstat`, but downstream helpers (`MoveFiles`, filters) all expect `os.DirEntry`. The adapter bridges this gap without modifying either side.

---

## 4. Data flow — move operation

```
runMove(m, ...)
  │
  ├── filterCategories(m.Categories, names, includeDisabled)
  │
  └── for each category:
        processCategoryMove(m, category, ...)
          │
          ├── RunHook(category.Hooks.Before, ...)        ← optional
          ├── helper.ReadDirectory(category.Source.Path)
          │
          └── for each extension:
                filterFilesForExtension(category, files, moved, extension)
                  └── matchesCategory → MatchesFilter (recursive any/all/leaf)
                │
                MoveFiles(MoveContext, category, filteredFiles, extension, batchID)
                  └── for each file:
                        ResolveGroupBy  → destDir
                        ResolveRename   → destName
                        applyConflictStrategy → ConflictResolver.Resolve()
                        dispatchAction  → FileAction.Execute()
                        history.Add(Entry{...})
          │
          └── RunHook(category.Hooks.After, ...)         ← optional
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

1. In `internal/helper/fileops.go`, define:
   ```go
   type hardlinkAction struct{}
   func (a *hardlinkAction) Execute(src, dst string) error { return os.Link(src, dst) }
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

1. In `internal/helper/conflict.go`, define a struct implementing `ConflictResolver`:
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

### Add a new template token (e.g. `{hostname}`)

1. In `internal/helper/groupby.go`, add the token to `knownTokens`.
2. Handle the token in `ResolveGroupBy` and/or `ResolveRename`.

### Add a new AppBuilder step

1. Add a method on `*AppBuilder` in `internal/config/builder.go` following the error-accumulation pattern.
2. Call it in the chain in `preRunHandler` (`internal/cmd/root.go`).

---

## 7. Key decisions and trade-offs

### Koanf is discarded after startup

Koanf is used only to load and validate the YAML file. Once `AppBuilder.Build()` returns, all configuration lives in typed Go structs. This avoids string-keyed config access scattered through the codebase and makes it impossible to read a config key that no longer exists.

### `models` has no logic and no internal imports

All shared types live in `models` with no dependencies on other internal packages. This prevents import cycles and makes the data model readable without understanding any subsystem.

### History is always written per-file, not per-batch

`history.Add` is called immediately after each successful file operation. This means a partial run (e.g. interrupted mid-batch) still produces a correct, undoable history. The batch ID groups entries logically without requiring an atomic commit.

### Watch mode uses a stability window, not inotify CLOSE_WRITE

`fsnotify` does not expose `CLOSE_WRITE` on all platforms. Instead, watch mode records when a file was first detected and only moves it after its `ModTime` has been stable for longer than `watch-delay` (default 5 minutes). This handles large or slow file writes correctly on all supported platforms.

### Conflict resolution and file actions are open/closed

Both `ConflictResolver` and `FileAction` follow the Open/Closed Principle: adding a new strategy requires only a new struct and one map entry. The dispatch functions (`applyConflictStrategy`, `dispatchAction`) never need to be modified. `SkipMessage()` is part of the `ConflictResolver` interface for the same reason — each resolver owns its log message so the dispatcher stays ignorant of strategy names.

### `AppBuilder` error accumulation

Each builder method checks `b.err != nil` at the start and returns immediately if an error was already set. This allows the fluent chain to be written without intermediate error checks while still stopping at the first failure. The trade-off is that only the first error is surfaced; subsequent steps never run, so their errors are never seen. This is acceptable because startup errors are fatal and the first one is always the most actionable.
