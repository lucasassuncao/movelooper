# movelooper Architecture

Diagrams describing the internal design and runtime behaviour of movelooper.

---

## Package dependencies

High-level view of which packages depend on which.

```mermaid
graph LR
    cmd --> config
    cmd --> scanner
    cmd --> filters
    cmd --> fileops
    cmd --> hooks
    cmd --> history
    cmd --> tokens
    cmd --> models

    config --> models
    config --> logger

    fileops --> models
    fileops --> filters
    fileops --> history
    fileops --> logger
    fileops --> tokens

    scanner --> models
    filters --> models
    hooks --> logger
    tokens --> models
```

---

## One-shot run (`movelooper`)

What happens when you run `movelooper` (or `movelooper --dry-run`).

```mermaid
flowchart TD
    Start([movelooper]) --> LoadConfig[Load config\nconfig.NewApp]
    LoadConfig --> Filter[Apply --category filter\nFilterCategories]
    Filter --> ForEach[For each enabled category]

    ForEach --> BeforeHook{Before hook\ndefined?}
    BeforeHook -- yes --> RunBefore[Run before hook\nhooks.RunHook]
    BeforeHook -- no --> Scan
    RunBefore --> Scan

    Scan[Scan source dir\nscanner.WalkSource] --> GroupExt[Group files by extension]
    GroupExt --> ForExt[For each extension]

    ForExt --> Match[Filter: extension + CategoryFilter\nfilters.MatchesFilter]
    Match --> DryRun{Dry run?}

    DryRun -- yes --> LogPlanned[Log planned moves\ntokens.ResolveGroupBy\ntokens.ResolveRename]
    DryRun -- no --> MoveFiles[fileops.MoveFiles\nper source directory]

    MoveFiles --> NextExt{More extensions?}
    LogPlanned --> NextExt
    NextExt -- yes --> ForExt
    NextExt -- no --> AfterHook{After hook\ndefined?}

    AfterHook -- yes --> RunAfter[Run after hook\nML_FILES_MOVED env vars]
    AfterHook -- no --> NextCat
    RunAfter --> NextCat

    NextCat{More categories?}
    NextCat -- yes --> ForEach
    NextCat -- no --> Summary[Print summary\nmoved / size / failures]
    Summary --> End([done])
```

---

> **`action: archive`** bypasses the per-file `fileActions` path. Instead of moving each file, `processCategoryMove` collects all of the category's matched files and hands them to `internal/archive`, which streams them into one `.zip`/`.tar.gz` written atomically (temp file + rename). Sources are deleted only when `keep-source: false` and the archive was written successfully. Archive batches are recorded in history as non-undoable and are skipped in watch mode.

## Watch mode (`movelooper watch`)

Watch mode uses two goroutines: an event loop that listens to filesystem notifications and a ticker loop that moves files once their writes have stabilised.

```mermaid
flowchart TD
    Start([movelooper watch]) --> Lock[Acquire PID lock file\nprevent duplicate watchers]
    Lock --> Register[Register source dirs\nfsnotify.Watcher]
    Register --> InitScan[Initial scan\nadd existing files to tracker]
    InitScan --> Goroutines[Spawn goroutines]

    Goroutines --> EventLoop
    Goroutines --> TickerLoop
    Goroutines --> WaitSignal[Wait for SIGINT / SIGTERM]

    subgraph EventLoop [Event loop goroutine]
        E1[fsnotify Create / Write event] --> E2[Add or update file in tracker heap\nwith current timestamp]
        E2 --> E1
    end

    subgraph TickerLoop [Ticker loop goroutine]
        T1[Ticker fires\nevery poll interval] --> T2[Pop files whose last event is\nolder than watch.delay from the min-heap]
        T2 --> T3{Any due files?}
        T3 -- no --> T1
        T3 -- yes --> T4[attemptMoveFile\nmatch category → fileops.MoveFiles]
        T4 --> T1
    end

    WaitSignal --> Shutdown[Release lock file\nclose watcher]
    Shutdown --> End([done])
```

> Hooks are intentionally skipped in watch mode. A warning is printed at startup for any category that defines before/after hooks.

---

## Category model

Structure of a single category entry in `movelooper.yaml`.

```mermaid
classDiagram
    class Category {
        +string Name
        +bool Enabled
        +CategorySource Source
        +CategoryDestination Destination
        +CategoryHooks Hooks
    }

    class CategorySource {
        +string Path
        +string[] Extensions
        +CategoryFilter Filter
        +bool Recursive
        +int MaxDepth
        +string[] ExcludePaths
    }

    class CategoryDestination {
        +string Path
        +string OrganizeBy
        +ConflictStrategy ConflictStrategy
        +Action Action
        +string Rename
    }

    class CategoryFilter {
        +MatchFilter Match
        +AgeFilter Age
        +SizeFilter Size
        +CategoryFilter[] Any
        +CategoryFilter[] All
        +CategoryFilter[] Not
    }

    class MatchFilter {
        +string Literal
        +string Regex
        +string Glob
        +bool CaseSensitive
    }

    class AgeFilter {
        +Duration Min
        +Duration Max
    }

    class SizeFilter {
        +string Min
        +string Max
    }

    class CategoryHooks {
        +CategoryHook Before
        +CategoryHook After
    }

    class CategoryHook {
        +string Shell
        +string OnFailure
        +string[] Run
    }

    Category --> CategorySource : source
    Category --> CategoryDestination : destination
    Category --> CategoryHooks : hooks
    CategorySource --> CategoryFilter : filter
    CategoryFilter --> MatchFilter : match
    CategoryFilter --> AgeFilter : age
    CategoryFilter --> SizeFilter : size
    CategoryFilter --> CategoryFilter : any / all / not
    CategoryHooks --> CategoryHook : before / after
```

---

## Filter evaluation

`filters.MatchesFilter` evaluates a `CategoryFilter` tree recursively.
`not` is always checked first so it can veto regardless of `any`/`all`.

```mermaid
flowchart TD
    Start([MatchesFilter]) --> CheckNot{not list\nnon-empty?}

    CheckNot -- yes --> AnyNotMatch{Any sub-filter\nin not matches?}
    AnyNotMatch -- yes --> Reject([false])
    AnyNotMatch -- no --> CheckAny

    CheckNot -- no --> CheckAny{any list\nnon-empty?}

    CheckAny -- yes --> AnyMatch{Any sub-filter\nin any matches?}
    AnyMatch -- yes --> Accept([true])
    AnyMatch -- no --> Reject2([false])

    CheckAny -- no --> CheckAll{all list\nnon-empty?}

    CheckAll -- yes --> AllMatch{All sub-filters\nin all match?}
    AllMatch -- yes --> Accept2([true])
    AllMatch -- no --> Reject3([false])

    CheckAll -- no --> LeafMatch[Check leaf filters\nmatch + age + size]
    LeafMatch --> LeafResult([true / false])
```

Each `CategoryFilter` leaf checks up to three independent constraints, all of which must pass (implicit AND):

| Field | Matches when |
|---|---|
| `match.glob` / `match.regex` / `match.literal` | filename satisfies the name pattern |
| `age.min` / `age.max` | modification time falls within the age window |
| `size.min` / `size.max` | file size falls within the size window |

---

## Per-file processing pipeline

What `fileops.MoveFiles` does for each individual file.

```mermaid
flowchart TD
    Start([file matched by scanner]) --> Tokens[Resolve destination\ntokens.ResolveGroupBy\ntokens.ResolveRename]
    Tokens --> MkDir[Create destination dir\nos.MkdirAll]
    MkDir --> Exists{Destination\nalready exists?}

    Exists -- no --> Action
    Exists -- yes --> Conflict[Apply conflict strategy]

    Conflict --> Skip{Strategy says\nskip?}
    Skip -- yes --> Done([file skipped])
    Skip -- no --> Action

    Action[Perform action] --> ActType{Action type}
    ActType -- move --> Move[os.Rename\nor copy+delete cross-device]
    ActType -- copy --> Copy[io.Copy\npreserve timestamps]
    ActType -- symlink --> Symlink[os.Symlink\nabsolute source path]

    Move --> History
    Copy --> History
    Symlink --> History

    History[Record in history\nhistory.Entry\nsource · dest · batchID · action · category]
    History --> Done2([file processed])
```

---

## Conflict strategies

How each strategy resolves a naming collision when the destination file already exists.

```mermaid
flowchart TD
    Collision([destination exists]) --> Strategy{conflict-strategy}

    Strategy -- skip --> SkipFile([skip — leave source untouched])

    Strategy -- rename --> Rename[Append counter suffix\nfile.txt → file 1 .txt]
    Rename --> Proceed([proceed with new name])

    Strategy -- overwrite --> Overwrite[Replace destination\nswap-aside on Windows for rollback]
    Overwrite --> Proceed2([proceed — original name])

    Strategy -- hash_check --> Hash[Compare SHA-256 of source\nand destination]
    Hash --> HashMatch{Identical\ncontent?}
    HashMatch -- yes --> RemoveSrc[Remove source\nduplicate]
    RemoveSrc --> SkipFile2([skip])
    HashMatch -- no --> Rename2[Append counter suffix\nas per rename strategy]
    Rename2 --> Proceed3([proceed with new name])

    Strategy -- newest --> NewestCmp{Source ModTime\n> dest ModTime?}
    NewestCmp -- no --> SkipFile3([skip — destination is newer])
    NewestCmp -- yes --> SwapNewest[Swap destination aside]
    SwapNewest --> Proceed4([proceed — original name])

    Strategy -- oldest --> OldestCmp{Source ModTime\n< dest ModTime?}
    OldestCmp -- no --> SkipFile4([skip — destination is older])
    OldestCmp -- yes --> SwapOldest[Swap destination aside]
    SwapOldest --> Proceed5([proceed — original name])

    Strategy -- larger --> LargerCmp{Source size\n> dest size?}
    LargerCmp -- no --> SkipFile5([skip — destination is larger])
    LargerCmp -- yes --> SwapLarger[Swap destination aside]
    SwapLarger --> Proceed6([proceed — original name])

    Strategy -- smaller --> SmallerCmp{Source size\n< dest size?}
    SmallerCmp -- no --> SkipFile6([skip — destination is smaller])
    SmallerCmp -- yes --> SwapSmaller[Swap destination aside]
    SwapSmaller --> Proceed7([proceed — original name])
```

> **Swap-aside**: for strategies that replace the destination (`overwrite`, `newest`, `oldest`, `larger`, `smaller`), the existing file is renamed to a temporary backup before the action runs. If the action fails, the backup is restored atomically.
