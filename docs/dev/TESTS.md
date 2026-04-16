# Test Coverage Reference

Overview of all test cases across the movelooper project.

---

## `internal/config` — Configuration loading

### `config_test.go`

| Test | What it verifies | Expected |
|---|---|---|
| `TestInitConfig_FileNotFound` | Returns `ErrConfigNotFound` when the file does not exist | error (`ErrConfigNotFound`) |
| `TestInitConfig_MalformedYAML` | Returns error for invalid YAML | error |
| `TestInitConfig_ValidMinimalConfig` | Loads a minimal config without errors | no error |
| `TestInitConfig_EmptyFile` | Empty file is valid YAML, no error | no error |
| `TestUnmarshalConfig_ValidCategory` | Deserializes categories correctly | no error, 1 category returned |
| `TestUnmarshalConfig_MissingExtensions` | Error when `extensions` is absent | error containing `"source.extensions are required"` |
| `TestUnmarshalConfig_InvalidRegex` | Error for invalid regex pattern | error containing `"invalid regex"` |
| `TestUnmarshalConfig_RegexAndGlobMutuallyExclusive` | Error when both `regex` and `glob` are set | error containing `"mutually exclusive"` |
| `TestUnmarshalConfig_MinSizeGreaterThanMaxSize` | Error when `min-size > max-size` | error containing `"min-size"` |
| `TestUnmarshalConfig_MinAgeGreaterThanMaxAge` | Error when `min-age > max-age` | error containing `"min-age"` |
| `TestUnmarshalConfig_CaseInsensitiveRegexCompiled` | Regex compiled with `(?i)` when `case-sensitive: false` | no error, `CompiledRegex` matches `"REPORT"` |
| `TestUnmarshalConfig_CaseSensitiveRegexCompiled` | Regex compiled without `(?i)` when `case-sensitive: true` | no error, `CompiledRegex` does not match `"REPORT"` |
| `TestUnmarshalConfig_SizeBytesPopulated` | `MinSizeBytes`/`MaxSizeBytes` are correctly populated | no error, `MinSizeBytes == 1024`, `MaxSizeBytes == 10485760` |
| `TestUnmarshalConfig_InvalidGlob` | Error for invalid glob pattern | error |
| `TestLoadConfig_Defaults` | `WatchDelay` and `HistoryLimit` use default values when absent | `WatchDelay == 5m`, `HistoryLimit == 50` |
| `TestLoadConfig_CustomValues` | Custom `output`, `log-level`, `watch-delay`, `history-limit` are read correctly | no error, fields match YAML values |
| `TestLoadConfig_WatchDelayFallback` | Missing `watch-delay` falls back to default | `WatchDelay == 5m` |
| `TestValidateCategory_Action` | empty / move / copy / symlink / invalid / uppercase | Only `move`, `copy`, `symlink`, and empty are accepted; others error | no error for valid values, error containing `"action"` otherwise |
| `TestValidateCategory_Rename` | empty / valid template / unknown token | Unknown tokens in `rename` are rejected at validation time | no error for valid, error containing `"rename"` for unknown token |

### `logging_test.go`

| Test | What it verifies | Expected |
|---|---|---|
| `TestParseLogLevel_AllLevels` | All log level strings map to the correct `pterm.LogLevel` (including unknown → info) | each input maps to its correct level |
| `TestLogWriterFactory_KnownStrategies` | `console`, `file`, `log`, `both` all return a non-nil strategy | non-nil strategy for each |
| `TestLogWriterFactory_UnknownFallsToConsole` | Unknown output mode falls back to `consoleStrategy` | strategy is `consoleStrategy` |
| `TestLogWriterFactory_EmptyFallsToConsole` | Empty output mode falls back to `consoleStrategy` | strategy is `consoleStrategy` |
| `TestConsoleStrategy_WriterReturnsStdout` | Returns stdout writer with nil closer | no error, closer is nil |
| `TestFileStrategy_WriterCreatesFile` | Creates the log file and returns a non-nil closer | no error, file created, closer non-nil |
| `TestFileStrategy_WriterErrorWhenNoLogFile` | Error when `log-file` is not set | error containing `"log-file is required"` |
| `TestMultiStrategy_WriterCreatesFileAndMultiWriter` | Returns a multi-writer (stdout + file) and a closer | no error, writer and closer non-nil |
| `TestMultiStrategy_WriterErrorWhenNoLogFile` | Error when `log-file` is not set | error |
| `TestConfigureLogger_ConsoleOutput` | Logger configured with console output, nil closer | no error, closer is nil |
| `TestConfigureLogger_FileOutput` | Logger configured with file output, non-nil closer | no error, closer non-nil |
| `TestConfigureLogger_BothOutput` | Logger configured with both outputs, non-nil closer | no error, closer non-nil |
| `TestConfigureLogger_UnknownOutputDefaultsToConsole` | Unknown output defaults to console without error | no error, closer is nil |
| `TestConfigureLogger_ShowCallerEnabled` | `show-caller: true` sets `logger.ShowCaller` | `logger.ShowCaller == true` |

### `imports_test.go`

| Test | What it verifies | Expected |
|---|---|---|
| `TestResolveImports_NoImport` | File without `import:` returns categories unchanged | no error, 1 category |
| `TestResolveImports_EmptyFile` | Empty file is accepted without error | no error, empty output |
| `TestResolveImports_WithSingleImport` | Categories from the imported file are merged | no error, 2 categories, no `import:` key in output |
| `TestResolveImports_ImportWithNoCategoriesInMain` | Main file without `categories:` still receives imported ones | no error, 1 category |
| `TestResolveImports_CircularImport` | Circular import is detected and returns error | error containing `"circular import"` |
| `TestResolveImports_MissingImportFile` | Non-existent imported file returns error | error |
| `TestResolveImports_NestedImports` | Nested imports (A→B→C) are resolved recursively | no error, 3 categories |
| `TestResolveImports_MalformedYAML` | Malformed YAML in imported file returns error | error |
| `TestResolveImports_MultipleImports` | Multiple imports are all merged | no error, 2 categories |
| `TestResolveImports_SiblingCircularChain` | Indirect cycle (B→C→B) is detected | error containing `"circular import"` |

---

## `internal/cmd` — CLI commands

### `move_integration_test.go`

| Test | Subcases | What it verifies | Expected |
|---|---|---|---|
| `TestRunMove` | moves files by extension | Moves files matched by extension | no error, file in dst, non-matching files stay in src |
| | dry-run does not move | Dry-run does not move any file | no error, file stays in src |
| | disabled category skipped | Disabled category is ignored | no error, file stays in src |
| | conflict rename | Name conflict generates a `(1)` suffix | no error, `file.txt` and `file(1).txt` both exist in dst |
| | organize by ext template | Template `{ext}` creates the correct subdirectory | no error, file at `dst/jpg/image.jpg` |
| `TestRunMove_MultipleCategories` | — | Multiple categories move files to distinct destinations | no error, each file in its correct dst |
| `TestRunMove_FileClaimedByFirstCategory` | — | Disputed file goes to the first matching category | no error, file in exactly one dst |
| `TestFilterFilesForExtension` | filters correctly by extension | Filters only files with the correct extension | 2 entries returned |
| | skips already moved files | Already-moved file is excluded from the list | empty result |
| `TestFormatBytes` | 0 B / 512 B / 1.00 KB / … | Converts bytes to human-readable string (B, KB, MB, GB) | each input maps to its expected string |
| `TestMovedSet` | — | `movedSet` marks and queries files correctly | `has` returns false before mark, true after |

### `integration_test.go`

| Test | Subcases | What it verifies | Expected |
|---|---|---|---|
| `TestRunMove_Filters` | regex filter moves only matching files | Regex filter moves only files matching the pattern | no error, matching file in dst, non-matching stays in src |
| | glob filter moves only matching files | Glob filter moves only files matching the pattern | no error, matching file in dst, non-matching stays in src |
| | ignore pattern skips ignored files | Files matching ignore pattern stay in source | no error, non-ignored in dst, ignored stays in src |
| | min size filter skips small files | Files smaller than `min-size` are not moved | no error, large file in dst, small stays in src |
| | min age filter skips recent files | Files newer than `min-age` are not moved | no error, old file in dst, recent stays in src |
| | multiple extensions in one category | A category with multiple extensions moves all of them | no error, jpg and png in dst, pdf stays in src |
| | all extension moves everything | Extension `all` moves any file type | no error, all 3 files in dst |
| | show-files dry-run does not move | `show-files + dry-run` does not move and does not error | no error, file stays in src |
| `TestValidateDirectories_MissingDirsNoError` | — | Missing directories only warn, no panic | no panic |
| `TestResolveConfigPath` | explicit path returns path | Valid explicit path is resolved correctly | no error, resolved path equals input |
| | explicit path not found returns error | Non-existent explicit path returns error | error |
| `TestRunMove_CopyAction` | — | `action: copy` copies the file, leaving source intact | no error, file in dst and in src |
| `TestRunMove_CopyWithRename` | — | `action: copy` + `rename` template produces the renamed file at dst | no error, renamed file in dst, original stays in src |
| `TestRunMove_SymlinkWithConflictRename` | — | `action: symlink` with a conflicting dst creates a renamed symlink | no error, original dst untouched, symlink as `file(1).txt` |

### `watch_test.go`

| Test | Subcases | What it verifies | Expected |
|---|---|---|---|
| `TestFileInfoDirEntry` | regular file | Adapter returns correct type info for a regular file | `IsDir() == false`, `Type().IsRegular() == true` |
| | directory | Adapter correctly identifies a directory | `IsDir() == true`, `Type().IsRegular() == false` |
| `TestMatchesExtensionAndFilters` | matches extension | File with correct extension matches the category | `true` |
| | wrong extension | Wrong extension does not match | `false` |
| | non-existent file | Non-existent file returns false | `false` |
| | regex filter matches | Regex filter passes a matching filename | `true` |
| | regex filter no match | Regex filter rejects a non-matching filename | `false` |
| `TestResolveDryRunDest` | no template returns dst | Without `organize-by`, returns the base destination | path equals base dst |
| | ext template appends subdir | With `{ext}`, returns destination with subdirectory | path equals `dst/jpg` |
| | non-existent file falls back to dst | Non-existent file falls back to base destination | path equals base dst |
| `TestAttemptMoveFile` | dry-run does not move | Dry-run does not move the file | no error, file stays in src |
| | no matching category stays | File with no matching category is not moved | no error, file stays in src |
| | moves matching file | File with matching category is moved | no error, file in dst, absent from src |
| | ignores file from wrong source dir | File in a different directory than watched is ignored | no error, file stays in src |
| `TestPerformInitialScan` | adds matching files | Initial scan adds only files matching the category | tracker has 1 entry for the pdf, txt excluded |
| | skips disabled category | Disabled category is skipped in initial scan | tracker is empty |
| | ignores ignored files | Files matching ignore pattern are excluded from tracker | tracker has 1 entry (non-ignored file only) |
| `TestProcessPendingFiles` | moves stable file | File with old ModTime is moved | no error, file in dst, absent from src |
| | skips fresh file | Recently modified file is not moved | file stays in src |
| | removes deleted file from tracker | Externally deleted file is removed from tracker | tracker no longer contains the ghost path |
| | dry-run does not move | Dry-run does not move a stable file | file stays in src |

### `undo_test.go`

| Test | Subcases | What it verifies | Expected |
|---|---|---|---|
| `TestUndoBatch` | dry-run reports would restore | Dry-run reports what would be restored without moving | no error, file stays at dst |
| | dry-run warns missing destination | Warns when file is no longer at destination | no error |
| | dry-run warns occupied source | Warns when source location is already occupied | no error, both files unchanged |
| | batch not found returns error | Non-existent batch returns error with clear message | error containing `"not found in history"` |
| `TestPrintBatchList` | no batches | Empty history does not return error | no error |
| | with batches | History with batches is listed without error | no error |
| `TestUndoCmd_NilHistory_ReturnsError` | — | `UndoCmd` with nil history returns immediate error | error containing `"history tracking is not initialized"` |
| `TestUndoCopyOrSymlink_RemovesDst` | — | `undoCopyOrSymlink` removes the destination file, leaving source intact | no error, dst removed, src unchanged |
| `TestUndoSymlink_RemovesLink` | — | `undoCopyOrSymlink` removes a symlink without touching the original | no error, link removed, src intact |
| `TestUndoBatch_CopyDryRun` | — | Dry-run of a `copy` batch reports removal without touching the file | no error, dst file still exists |

---

## `internal/helper` — File operations and filters

### `fileops_extra_test.go`

| Test | What it verifies | Expected |
|---|---|---|
| `TestApplyConflictStrategy_NoConflict` | No destination file — returns original path, no skip | `skip == false`, `resolved == dstFile` |
| `TestApplyConflictStrategy_SkipStrategy` | `skip` strategy skips and leaves source untouched | `skip == true`, src file still exists |
| `TestApplyConflictStrategy_RenameStrategy` | `rename` strategy returns a path with `(1)` suffix | `skip == false`, resolved path contains `"(1)"` |
| `TestApplyConflictStrategy_OverwriteStrategy` | `overwrite` strategy returns the original destination path | `skip == false`, `resolved == dstFile` |
| `TestApplyConflictStrategy_HashCheck_Duplicate` | Identical file (same hash): source removed, skip returned | `skip == true`, src file removed |
| `TestApplyConflictStrategy_NewestStrategy_SrcNewer` | Source newer than destination: move proceeds | `skip == false`, `resolved == dstFile` |
| `TestApplyConflictStrategy_OldestStrategy_SrcOlder` | Source older than destination: move proceeds | `skip == false`, `resolved == dstFile` |
| `TestApplyConflictStrategy_LargerStrategy_SrcLarger` | Source larger than destination: move proceeds | `skip == false`, `resolved == dstFile` |
| `TestApplyConflictStrategy_SmallerStrategy_SrcSmaller` | Source smaller than destination: move proceeds | `skip == false`, `resolved == dstFile` |
| `TestApplyConflictStrategy_UnknownFallsToRename` | Unknown strategy falls back to rename | `skip == false`, resolved path contains `"(1)"` |
| `TestIsCrossDeviceError_NonLinkError` | Non-`*os.LinkError` returns false | `false` |
| `TestIsCrossDeviceError_NilError` | Nil error returns false | `false` |
| `TestIsCrossDeviceError_LinkErrorWithPermission` | `*os.LinkError` with permission error returns false | `false` |
| `TestGenerateLogArgs_ReturnsNamePairs` | Matching files produce `"name", "<filename>"` pairs | slice of length 4 (2 pdfs × 2 elements) |
| `TestGenerateLogArgs_NoMatch` | No matching extension returns empty slice | empty slice |
| `TestGenerateLogArgs_AllExtension` | Extension `all` matches every file | slice of length 4 (2 files × 2 elements) |
| `TestMoveFiles_DefaultsToRenameWhenNoStrategy` | Empty `ConflictStrategy` defaults to rename | no error, `file(1).txt` exists in dst |

### `fileops_test.go`

| Test | What it verifies | Expected |
|---|---|---|
| `TestCreateDirectory_CreatesNew` | Creates a directory that does not exist | no error, directory exists |
| `TestCreateDirectory_Idempotent` | Does not fail if the directory already exists | no error |
| `TestReadDirectory_ReturnsEntries` | Returns entries from an existing directory | no error, 2 entries returned |
| `TestReadDirectory_NonExistentReturnsError` | Returns error for a non-existent directory | error |
| `TestCopyFile_CopiesContent` | Copies file content correctly | no error, dst content equals src content |
| `TestCopyFile_PreservesModTime` | Preserves the modification timestamp | no error, dst ModTime equals src ModTime |
| `TestMoveFiles_MovesMatchingExtension` | Moves files by the correct extension | no error, pdf in dst, jpg stays in src |
| `TestMoveFiles_SkipsOnConflictSkipStrategy` | `skip` strategy does not move on conflict | no error, empty moved list, src file still exists |
| `TestMoveFiles_WithOrganizeBy` | `organize-by` template creates the correct subdirectory | no error, file at `dst/jpg/photo.jpg` |
| `TestMoveFiles_ExtAllMovesAll` | Extension `all` moves any file | no error, 2 files moved |
| `TestDispatchAction_Move` | `action: move` moves file to dst and removes src | no error, file in dst, absent from src |
| `TestDispatchAction_Copy` | `action: copy` copies file to dst, source stays | no error, file in both src and dst, contents equal |
| `TestDispatchAction_Symlink` | `action: symlink` creates a symlink at dst pointing to src | no error, dst is a symlink (skipped if privileges unavailable) |
| `TestMoveFiles_RenameTemplate` | `rename` template produces correctly named file at dst | no error, `images_photo.jpg` in dst, original stays in src |

### `filters_extra_test.go`

| Test | What it verifies | Expected |
|---|---|---|
| `TestMeetsAgeSizeFilters_MinAgeOnly` | File older than `min-age` passes | `true` |
| `TestMeetsAgeSizeFilters_MinAgeFails` | File newer than `min-age` fails | `false` |
| `TestMeetsAgeSizeFilters_MaxAgeOnly` | File newer than `max-age` passes | `true` |
| `TestMeetsAgeSizeFilters_MaxAgeFails` | File older than `max-age` fails | `false` |
| `TestMeetsAgeSizeFilters_MinSizeOnly` | File larger than `min-size` passes | `true` |
| `TestMeetsAgeSizeFilters_MinSizeFails` | File smaller than `min-size` fails | `false` |
| `TestMeetsAgeSizeFilters_MaxSizeOnly` | File smaller than `max-size` passes | `true` |
| `TestMeetsAgeSizeFilters_MaxSizeFails` | File larger than `max-size` fails | `false` |
| `TestMeetsAgeSizeFilters_AllConstraints_Pass` | File satisfying all age and size constraints passes | `true` |
| `TestMeetsAgeSizeFilters_AllConstraints_OneFails` | File failing any single constraint fails overall | `false` |
| `TestMatchesNameFilters_RegexMatch` | Compiled regex matches/rejects filenames correctly | `true` for match, `false` for non-match |
| `TestMatchesNameFilters_RegexNoMatch` | Anchored regex rejects non-matching filename | `false` |
| `TestMatchesNameFilters_IncludePatterns` | `include` list requires at least one pattern to match | `true` for matches, `false` otherwise |
| `TestMatchesNameFilters_GlobMatch` | `glob` pattern matches/rejects filenames correctly | `true` for match, `false` for non-match |

### `filters_test.go`

| Test | What it verifies | Expected |
|---|---|---|
| `TestMatchesIgnorePatterns` | Ignore patterns are applied correctly | `true` for ignored files, `false` otherwise |
| `TestExpandGlobPattern` | Expands `*.{jpg,png}` into separate patterns | correct list of expanded patterns |
| `TestMatchesGlob` | Glob matches names with configurable case sensitivity | `true`/`false` per case |
| `TestValidateGlob` | Validates glob pattern syntax | no error for valid, error for invalid |
| `TestHasExtension` | Checks file extension (case insensitive) | `true`/`false` per case |
| `TestMatchesAnyExtension` | Checks against a list of extensions | `true` if any matches, `false` otherwise |
| `TestMatchesNameFilters` | Applies regex, glob, and include together | `true`/`false` per scenario |
| `TestParseSize` | Converts strings like `"1 MB"` to bytes | correct int64 value or error for invalid input |
| `TestMeetsMinAge` | Checks if file is older than the minimum | `true`/`false` per case |
| `TestMeetsMaxAge` | Checks if file is newer than the maximum | `true`/`false` per case |
| `TestMeetsMinSize` | Checks if file is larger than the minimum | `true`/`false` per case |
| `TestMeetsMaxSize` | Checks if file is smaller than the maximum | `true`/`false` per case |
| `TestMeetsAgeSizeFilters_NoConstraints` | Without constraints, any file passes | `true` |

### `conflict_test.go`

| Test | What it verifies | Expected |
|---|---|---|
| `TestGetUniqueDestinationPath_NoConflict` | No conflict: returns the original path | no error, path unchanged |
| `TestGetUniqueDestinationPath_Conflict` | Conflict: adds `(1)` suffix | no error, path contains `"(1)"` |
| `TestGetUniqueDestinationPath_MultipleConflicts` | Multiple conflicts increment the suffix | no error, path contains `"(2)"` |
| `TestResolveConflict_UnknownFallsToRename` | Unknown strategy falls back to rename | no error, renamed path returned |
| `TestRenameResolver` | `rename` strategy adds suffix to the file | no error, resolved path has suffix |
| `TestOverwriteResolver_RemovesDst` | `overwrite` strategy deletes the destination | no error, dst file removed |
| `TestSkipResolver` | `skip` strategy does not move the file | `shouldMove == false` |
| `TestHashCheckResolver_DuplicateRemovesSrc` | Identical file (same hash): source is removed | `shouldMove == false`, src removed |
| `TestHashCheckResolver_DifferentRenames` | Different file: falls back to rename | no error, renamed path returned |
| `TestNewestResolver_SrcNewer` | Newer source replaces destination | `shouldMove == true` |
| `TestNewestResolver_DstNewer` | Newer destination: source is discarded | `shouldMove == false` |
| `TestOldestResolver_SrcOlder` | Older source replaces destination | `shouldMove == true` |
| `TestOldestResolver_DstOlder` | Older destination: source is discarded | `shouldMove == false` |
| `TestLargerResolver_SrcLarger` | Larger source replaces destination | `shouldMove == true` |
| `TestLargerResolver_DstLarger` | Larger destination: source is discarded | `shouldMove == false` |
| `TestSmallerResolver_SrcSmaller` | Smaller source replaces destination | `shouldMove == true` |
| `TestSmallerResolver_DstSmaller` | Smaller destination: source is discarded | `shouldMove == false` |

### `groupby_test.go`

| Test | What it verifies | Expected |
|---|---|---|
| `TestFileSizeRange` | Classifies files as tiny/small/medium/large | correct label per size |
| `TestResolveGroupBy_EmptyTemplate` | Empty template returns empty string | `""` |
| `TestResolveGroupBy_ExtToken` | `{ext}` returns the file extension | `"jpg"` |
| `TestResolveGroupBy_ExtUpperToken` | `{ext-upper}` returns extension in uppercase | `"JPG"` |
| `TestResolveGroupBy_CategoryToken` | `{category}` returns the category name | category name string |
| `TestResolveGroupBy_RunDateTokens` | `{year}`, `{month}`, `{day}` return current date parts | correct date part strings |
| `TestResolveGroupBy_ModDateTokens` | `{mod-year}`, `{mod-month}`, `{mod-day}` return file modification date parts | correct date part strings |
| `TestResolveGroupBy_SizeRange` | `{size-range}` returns the file size range | one of `tiny/small/medium/large` |
| `TestResolveGroupBy_CombinedTemplate` | Combined template with multiple tokens | correctly interpolated string |
| `TestValidateTemplate` | empty / valid / unknown / mixed tokens | Rejects unknown `{token}` values, accepts all known ones | no error for valid, error for unknown |
| `TestResolveRename` | empty / ext / ext-upper / mod-date / category / run-date | Resolves rename template to correct filename using file metadata | expected filename string per case |

---

## `internal/history` — Operation history

### `history_extra_test.go`

| Test | What it verifies | Expected |
|---|---|---|
| `TestLoad_ValidJSON` | Loads entries correctly from a valid JSON file | no error, 1 entry with correct `BatchID` |
| `TestLoad_CorruptJSON` | Returns error when JSON file is malformed | error |
| `TestLoad_FileNotExist` | Returns `os.IsNotExist` error when file is absent | `os.IsNotExist(err) == true` |
| `TestSave_WritesJSON` | Persists entries to disk as valid JSON | no error, file contains correct entry |
| `TestSave_EmptyEntriesWritesEmptyArray` | Empty entries are serialized as `[]` | no error, file contains `[]` |
| `TestAdd_UnwritablePath` | Returns error when the history path is not writable | error |
| `TestAdd_ConcurrentSafe` | 10 concurrent `Add` calls produce exactly 10 entries without data races | no error, 10 entries in batch |
| `TestNewBatchID_UniquePerSecond` | Both generated IDs have the `batch_` prefix | both IDs contain `"batch_"` |
| `TestGetAllBatches_Empty` | Returns empty slice when history has no entries | empty slice |
| `TestRemoveBatch_NonExistentBatchNoError` | Removing a non-existent batch does not error or affect others | no error, existing batch unchanged |
| `TestPrune_SingleBatchUnderLimit` | Single batch below limit is retained | 1 batch returned |
| `TestHistory_LoadAndAddRoundTrip` | Entries added to one instance are readable by a fresh instance on the same file | no error, 2 entries in new instance |

### `history_test.go`

| Test | What it verifies | Expected |
|---|---|---|
| `TestAdd_PersistsToDisk` | Added entry is written to the JSON file | no error, JSON file contains 1 entry |
| `TestGetBatch_ReturnsCorrectEntries` | Returns only the entries for the requested batch | 2 entries for batch A, 1 for batch B |
| `TestGetBatch_UnknownIDReturnsEmpty` | Non-existent batch returns empty list | empty slice |
| `TestGetLastBatchID_ReturnsLast` | Returns the ID of the most recent batch | no error, ID equals `"batch_2"` |
| `TestGetLastBatchID_EmptyHistoryErrors` | Returns error when history is empty | error |
| `TestGetAllBatches_OrderedOldestFirst` | Batches are listed from oldest to newest | 2 batches in insertion order |
| `TestRemoveBatch_RemovesEntries` | Removes batch entries from memory | no error, batch A empty, batch B intact |
| `TestRemoveBatch_PersistsRemoval` | Removal is persisted to the JSON file | no error, removed batch absent from JSON |
| `TestPrune_KeepsMaxBatches` | When limit is exceeded, the oldest batch is removed | 2 batches remain, oldest gone |
| `TestNewBatchID_HasPrefix` | Batch IDs start with `batch_` | ID contains `"batch_"` |
| `TestNewWatchBatchID_HasPrefix` | Watch batch IDs start with `watch_` | ID contains `"watch_"` |
| `TestNewWatchBatchID_UniquePerCall` | 50 consecutively generated IDs are all unique | no collisions in 50 iterations |
