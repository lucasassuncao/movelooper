# Guarantees and Limits

This page is the contract. It says what movelooper promises about your files, what it deliberately does not promise, and what it will never try to be.

## What movelooper is

**A declarative router for local files.** You declare what should move, from where, to where. movelooper carries out exactly that.

The configuration file is the source of truth. movelooper does not second-guess it, does not prompt for confirmation on a rule you wrote, and does not refuse a rule because it looks risky. If you declare `conflict-strategy: overwrite`, files get overwritten. That is the feature, not an accident.

The corollary matters just as much: because a run never argues with your config, **the time to find out what a rule does is before the run**, with `validate` and `--dry-run`. Both exist for that.

## What movelooper guarantees

**Every completed move is recorded, and you are told when it is not.** Moves are recorded after they happen, in a history file, and `undo` replays them backwards. If the history cannot be written, movelooper says so and exits with a non-zero status. It never reports a clean success for moves it cannot undo.

**An interrupted run still records what it moved.** Press Ctrl+C during a run and movelooper stops starting new work, writes the history for the files it already moved, and prints the batch ID to undo them with.

**A preview touches nothing.** `--dry-run` only reads. It resolves every destination, reports which of them already exist and what your conflict strategy will do to them, and writes nothing.

**A file is never processed twice in one run.** Categories are matched in order and the first match claims the file.

**Undo never turns into a deletion.** Undoing a `copy` removes the file at the destination, which only reverses the run while the original is still at the source. When the original is gone, or the copy was modified after it was made, the file is kept and a warning names it. `undo --force` removes it anyway, on your say-so.

**No partial file is left behind.** A copy that fails partway is removed. A destination displaced by `overwrite` or a comparison strategy is set aside first and restored if the operation fails.

**The same behaviour on Linux, macOS, and Windows.** All three are tested on every change. Where a platform genuinely cannot do something (`action: symlink` needs administrator rights or Developer Mode on Windows), movelooper fails with a clear message instead of half-doing it.

## What movelooper does not guarantee

**History survives a power cut or a `kill -9`.** History is written once, at the end of a run, to keep disk writes down. A run killed without warning leaves its files moved and no record of it. Ctrl+C and SIGTERM are handled; abrupt termination is not.

**`action: archive` cannot be undone.** `undo` refuses archive batches. With `keep-source: false` the originals are deleted and the archive becomes the only copy. `validate` warns about this.

**`hash_check` deletes duplicate sources.** When the destination already holds a byte-identical file, the source is removed. No content is lost (the bytes still exist at the destination), but the source file is gone.

**Two simultaneous runs are not coordinated.** The main command takes no lock, so a scheduled run and a manual run can overlap. Nothing is corrupted (a file is claimed by exactly one of them) and a file that another process already moved is not counted as a failure, but the work is not divided intelligently either.

## What movelooper will never be

Not a sync tool. Not a backup tool. Not a deduplicator. Not a media library manager.

Those are different jobs with different failure modes. You are free to build rules that serve those ends (that is your call, and the config is yours), but movelooper will not grow commands that scan your machine, index content, or reconcile two directories. It routes the files you declare, and stops there.

## Watch mode limits

Watch mode is an addition to the one-shot command, not a replacement for it. It handles one directory per category, one file at a time. It deliberately does not support:

| Not supported in watch | Use instead |
|---|---|
| `recursive: true`: only the top-level source directory is monitored | the one-shot command, scheduled |
| `action: archive`: archiving is a batch operation | the one-shot command |
| `hooks`: before/after run per category, not per file | the one-shot command |

movelooper warns about each of these at startup for any category that configures them. These are boundaries, not missing features.

Only one watcher runs at a time, enforced by an operating-system lock that is released automatically when the process ends, including on a crash. There is no stale lock file to clean up by hand.
