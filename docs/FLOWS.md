# Flow Diagrams

Three interactive diagrams of the paths that actually touch your files: the one-shot run, watch mode, and undo. Each one is a self-contained page with search, focus, relationship tracing, guided views, and light/dark themes.

Every diagram was generated from a typed JSON source, checked against the code in this repository, and the source is published next to it.

---

## One-shot run

`movelooper` (and `movelooper --dry-run`). Config is loaded once, then every enabled category is scanned, filtered file by file, and acted on. `action: archive` leaves the per-file path, and each processed file becomes a history entry.

Hooks are deliberately left out of the diagram: they are user shell commands, not engine steps, and drawing them broke the line of the flow. A `before` hook runs ahead of the scan and an `after` hook once the category is done, with `ML_FILES_MOVED` in its environment. See [Hooks](/HOOKS.md).

<!-- markdownlint-disable MD033 -->
<iframe src="../diagrams/one-shot-run.html" title="movelooper one-shot run flow" width="100%" height="1000" style="border:1px solid #30363d;border-radius:8px" loading="lazy"></iframe>
<!-- markdownlint-enable MD033 -->

<!-- markdownlint-disable MD033 -->
<p><a href="../diagrams/one-shot-run.html" target="_blank" rel="noopener">Open full page ↗</a> · <a href="../diagrams/one-shot-run.workflow.json" target="_blank" rel="noopener">JSON source</a></p>
<!-- markdownlint-enable MD033 -->

Related: [Categories](/CATEGORIES.md) · [Filters](/FILTERS.md) · [Actions](/ACTIONS.md) · [Conflicts](/CONFLICTS.md) · [Hooks](/HOOKS.md)

---

## Watch mode

`movelooper watch`. A PID lock keeps a single watcher, fsnotify events refresh a per-file timestamp, and a ticker moves only the files that have gone quiet for `watch.delay`. Hooks are skipped here by design, and `action: archive` categories are not watched.

<!-- markdownlint-disable MD033 -->
<iframe src="../diagrams/watch-mode.html" title="movelooper watch mode flow" width="100%" height="1000" style="border:1px solid #30363d;border-radius:8px" loading="lazy"></iframe>
<!-- markdownlint-enable MD033 -->

<!-- markdownlint-disable MD033 -->
<p><a href="../diagrams/watch-mode.html" target="_blank" rel="noopener">Open full page ↗</a> · <a href="../diagrams/watch-mode.workflow.json" target="_blank" rel="noopener">JSON source</a></p>
<!-- markdownlint-enable MD033 -->

Related: [Watch Mode](/WATCH.md) · [Configuration](/CONFIGURATION.md)

---

## Undo

`movelooper undo`. The batch is read from history and replayed in reverse. A moved file goes back to its source path; undoing a copy or a symlink deletes the destination instead, which is why a copy is held back when the original is gone or the copy was edited since.

<!-- markdownlint-disable MD033 -->
<iframe src="../diagrams/undo.html" title="movelooper undo flow" width="100%" height="1000" style="border:1px solid #30363d;border-radius:8px" loading="lazy"></iframe>
<!-- markdownlint-enable MD033 -->

<!-- markdownlint-disable MD033 -->
<p><a href="../diagrams/undo.html" target="_blank" rel="noopener">Open full page ↗</a> · <a href="../diagrams/undo.workflow.json" target="_blank" rel="noopener">JSON source</a></p>
<!-- markdownlint-enable MD033 -->

Related: [Undo](/UNDO.md) · [Guarantees and Limits](/GUARANTEES.md)
