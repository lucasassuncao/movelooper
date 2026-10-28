# Development Guide

Practical reference for contributing to movelooper. Covers setup, daily workflow, testing, code conventions, and the release process.

---

## Table of Contents

1. [Prerequisites](#1-prerequisites)
2. [Getting started](#2-getting-started)
3. [Daily workflow](#3-daily-workflow)
4. [Testing](#4-testing)
5. [Code conventions](#5-code-conventions)
6. [Adding new features](#6-adding-new-features)
7. [CI pipeline](#7-ci-pipeline)
8. [Release process](#8-release-process)
9. [Serving the docs site locally](#9-serving-the-docs-site-locally)

---

## 1. Prerequisites

| Tool | Version | Purpose |
|---|---|---|
| Go | 1.27.0+ | Build and test |
| Make | any | Task runner |
| Git | any | Version control |
| PowerShell (`pwsh`) | 7.4+ | Runs `scripts/pwsh` on Windows; `make check-scripts` needs it on every OS |
| VHS, ttyd, ffmpeg | any | Only for `make demo` |

All other tools (golangci-lint, goreleaser, gosec, gotestsum, gomarkdoc, syft, grype, git-cliff, shellcheck, PSScriptAnalyzer, caddy) are pinned in `scripts/make/tools.mk` and installed into `./.gobin` on first use, so no global installation is required and every target runs the pinned version rather than whatever is on `PATH`. Bumping a version in `scripts/make/tools.mk` does not invalidate an already installed binary, run `make clean-tools` first.

---

## 2. Getting started

```bash
git clone https://github.com/lucasassuncao/movelooper
cd movelooper
go mod download
```

Build for the current platform:

```bash
make build
# binary written to dist/
```

Run without building:

```bash
make run
```

---

## 3. Daily workflow

### Available `make` targets

```
make help            # list all targets with descriptions
make build           # build binary for current platform (via goreleaser)
make build-all       # build for all platforms (linux/windows/darwin × amd64/arm64)
make run             # go run main.go
make fmt             # gofmt ./...
make lint            # golangci-lint run
make test            # run tests (testdox format)
make test-watch      # rerun tests on file changes
make test-coverage   # tests + HTML and XML coverage reports
make security        # gosec static analysis
make sbom            # CycloneDX SBOM of the dependency tree (syft)
make vuln            # scan the SBOM for known vulnerabilities (grype)
make docs            # regenerate package READMEs (gomarkdoc) + config reference (generate-docs)
make stage-docs      # assemble the docs site into site/, as GitHub Pages publishes it
make serve-docs      # stage-docs, then serve site/ on http://localhost:8080 (caddy)
make demo            # record examples/demo/demo.gif from demo.tape (vhs)
make deps            # go mod download && go mod tidy
make notes           # print the git-cliff notes the next `make tag` would write
make tag             # create and push an annotated tag with git-cliff notes (VERSION=v1.2.3)
make release-notes   # git-cliff notes of the tag at HEAD into release-notes.md (cliff.toml)
make release-check   # goreleaser check + a local snapshot release into dist/
make check-scripts   # every script has a twin, shellcheck and PSScriptAnalyzer pass
make tools           # install every pinned tool into ./.gobin
make clean-tools     # remove ./.gobin so the next target reinstalls the pinned tools
make clean           # remove build artifacts, installed tools and cache
make all             # deps + fmt + docs + lint + security + test-coverage + govulncheck + sbom + vuln
```

### Before committing

Run at minimum:

```bash
make fmt && make test
```

CI enforces formatting, dependency tidiness, tests, lint, security, and generated docs, all of these must pass on `main`.

### Commit messages

Commits follow [Conventional Commits](https://www.conventionalcommits.org): `type(scope): subject`. The type is not decoration: git-cliff groups the tag and release notes by it (`cliff.toml`), so a commit lands under Features, Bug Fixes, Refactor, Performance, Documentation, Build, CI, Tests, Chores or, without a type, Other.

- **Type:** `feat`, `fix`, `refactor`, `perf`, `docs`, `build`, `ci`, `test` or `chore`, picked by what the change does for whoever uses the code.
- **Scope:** optional, the area touched (`config`, `fileops`, `release`...). Reuse a scope already in `git log` rather than inventing a synonym.
- **Breaking change:** `!` before the colon (`feat(config)!: ...`), and the body says what users must change.
- **Subject:** imperative mood, lowercase after the colon, no trailing period, around 50 characters.
- **Body:** optional prose, wrapped at about 72 columns, saying why the change exists: the bug's mechanism, the constraint behind the approach, what was ruled out. The diff already shows what changed. The whole message reaches the release page, so the body is written for its readers too.

```
feat(release): write tag and release notes with git-cliff

goreleaser's changelog kept only each commit's subject, so the body,
where the reason for a change lives, never reached the release page.
git-cliff now renders every commit with its whole message.
```

---

## 4. Testing

### Running tests

```bash
make test               # all packages, testdox format
make test-watch         # rerun on file changes (useful during TDD)
make test-coverage      # generates coverage.html and coverage.xml
go test ./internal/fileops/... -run TestMoveFiles -v   # single package, filtered
```

### Test organisation

- Tests live in the **same package** as the code they test (e.g. `package fileops`, not `package fileops_test`). This allows testing unexported functions directly, which is intentional, internal invariants are worth testing.
- Each source file has its own test file (`scan.go` → `scan_test.go`), so tests live next to the code they exercise. There are no catch-all test files; when a test file grows too long, the source file usually wants splitting too.
- The `internal/cmd/integration_test.go` file contains end-to-end tests that exercise the full command flow against real temporary directories.

### Writing tests

**Prefer table-driven tests** with `tests := []struct{...}` for any function with multiple scenarios. Use subtests (`t.Run`) so failures are reported by name.

```go
func TestMyFunc(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid input", "foo", "FOO", false},
        {"empty input", "", "", true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := MyFunc(tt.input)
            if tt.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

**Use `t.TempDir()`** for all filesystem tests. It is automatically cleaned up after the test and avoids leftover files when tests fail.

**Use `require` for fatal assertions, `assert` for non-fatal ones.** A failing `require` stops the test immediately; `assert` continues and collects all failures.

```go
require.NoError(t, err)       // test cannot continue if this fails
assert.Equal(t, want, got)    // collect all assertion failures, then report
```

**Silent logger in tests.** Use `logger.Discard()` when a function requires a `*slog.Logger`. It drops all output so test logs stay clean.

**No mocks.** Tests hit real files and real functions. External state (filesystem, OS executable path) is controlled via `t.TempDir()` and carefully crafted inputs. This is intentional, mocks can mask real integration bugs.

---

## 5. Code conventions

### General

- **No comments that explain what the code does.** Well-named identifiers do that. Comments should explain *why*: a non-obvious constraint, a platform workaround, a subtle invariant.
- **No unused error handling.** Only handle errors that can actually occur. Do not add fallbacks for scenarios that cannot happen.
- **YAGNI.** Do not add abstractions or configuration options for hypothetical future requirements.
- **Three similar lines before abstracting.** Duplication is cheaper than the wrong abstraction.

### Packages and imports

Follow the dependency rule in [`DESIGN.md`](DESIGN.md) §2: leaf packages import nothing internal, `cmd` imports everything, and nothing imports upward. It lives only there, so it cannot drift from a copy.

Import groups in each file follow standard Go convention (enforced by `gofmt`/`gci`):

1. Standard library
2. Third-party
3. Internal (`github.com/lucasassuncao/movelooper/...`)

### Error handling

- Wrap errors with `%w` so callers can use `errors.Is` / `errors.As`.
- Return errors; do not panic in library code.
- Prefer `fmt.Errorf("context: %w", err)` over bare `err` returns when the call site is ambiguous.
- Sentinel errors (`var ErrFoo = fmt.Errorf(...)`) are defined only when callers need to test for a specific condition with `errors.Is`.

### Extending the Strategy registries

Each registry (file actions, conflict resolvers, log writers) follows the same shape. When adding a new entry:

1. Define a private struct implementing the interface.
2. Add one entry to the map. Do not modify the dispatch function.
3. For `ConflictResolver`, implement both `Resolve()` and `SkipMessage()`.

See [`DESIGN.md`](DESIGN.md) §6 for step-by-step extension guides.

### Security-sensitive patterns

- File paths that come from user config or directory walks are passed to `filepath.Clean` or opened with `#nosec G304` annotations where gosec would otherwise flag them. Do not remove these annotations blindly, they exist because the path source is documented.
- History and log files are created with mode `0600` (owner read/write only).
- Directories are created with mode `0750`.

### Linter configuration

golangci-lint is configured in `.golangci.yaml`. Active linters include:

- `staticcheck`, `gocritic`, correctness and idiom checks
- `gocognit`, `gocyclo`, `nestif`, complexity limits (cognitive complexity ≤ 35)
- `dupl`, duplicate code detection
- `misspell`, spelling in comments and strings
- `lll`, line length ≤ 300 characters
- `unparam`, unused function parameters

Run `make lint` before opening a PR. The CI pipeline will fail if lint errors are present.

---

## 6. Adding new features

### Checklist

- [ ] Read [`DESIGN.md`](DESIGN.md) to understand where the change belongs
- [ ] Follow the extension guides in [`DESIGN.md`](DESIGN.md) §6 for registries and the builder
- [ ] Write tests before or alongside the implementation (TDD preferred)
- [ ] Run `make all` (deps + fmt + docs + lint + security + test-coverage + govulncheck + sbom + vuln) before pushing
- [ ] Update [`DESIGN.md`](DESIGN.md) if a new pattern or architectural decision is introduced

### Where to put new code

| What | Where |
|---|---|
| New file action (`hardlink`, etc.) | `internal/fileops/fileops.go` (struct + map entry) + `internal/config/config.go` (`validActions`) |
| New conflict strategy | `internal/fileops/conflict.go` (struct + map entry) |
| New log output mode | `internal/config/logging.go` (struct + map entry) |
| New template token | `internal/tokens/resolve.go` (`buildStaticPairs`) + `internal/tokens/validate.go` (`knownTokens`) |
| New startup step | `internal/config/builder.go` (method) + `internal/cmd/root.go` (chain call) |
| New CLI command | `internal/cmd/<command>.go` + registered in `RootCmd` |
| New config field | `internal/models/movelooper.go` (struct) + `internal/config/appconfig.go` (`LoadConfig`) |

---

## 7. CI pipeline

The CI workflow (`.github/workflows/ci.yml`) runs on every push and pull request to `main`. Its jobs run in parallel:

| Job | Runs on | Steps |
|---|---|---|
| Checks | Linux | `go mod tidy`, `make fmt` and `make docs`, each followed by `git diff`, so an uncommitted result fails the job |
| Test | Linux, Windows, macOS | `make lint`, `make security` (gosec, medium+) and `make test-coverage`; coverage uploaded as artifact |
| Supply chain | Linux | `make govulncheck` (vulnerabilities reachable from this code), then `make vuln` (grype, medium+); SBOM uploaded as artifact |
| Deploy GitHub Pages | Linux | Publishes the docs site once Checks passes; never on pull requests |

Lint and security run on every OS because Go only compiles the files of the target platform, so `*_windows.go` and `*_unix.go` are only analyzed on their own OS.

**Checks, Test and Supply chain must pass before merging.** There are no exceptions. A red Supply chain job with nothing changed in the repo usually means a CVE was published against a dependency.

If the docs check fails, run `make docs` locally, review the diff, and commit the result.

---

## 8. Release process

Releases are managed with goreleaser and GitHub Actions. Binaries are built for:

- `linux/amd64`, `linux/arm64`
- `windows/amd64`, `windows/arm64`
- `darwin/amd64`, `darwin/arm64`

### Creating a release

1. Ensure `main` is clean and all CI checks pass.
2. Optionally preview: `make notes` prints the tag message body, and `make release-check` runs the release pipeline without publishing.
3. Create and push an annotated tag:
   ```bash
   make tag VERSION=v1.2.3
   ```
4. The release workflow picks up the tag, builds all platform binaries, and publishes the GitHub release automatically.

### Version naming

Follow [Semantic Versioning](https://semver.org):

- `PATCH` (`v1.2.3` → `v1.2.4`): bug fixes, no API or config changes
- `MINOR` (`v1.2.3` → `v1.3.0`): new features, backwards-compatible config changes
- `MAJOR` (`v1.2.3` → `v2.0.0`): breaking changes to the config format or CLI interface

Pre-release versions (`-alpha`, `-beta`, `-rc.1`) are marked as pre-release on GitHub automatically by goreleaser when the tag contains a pre-release suffix.

---

## 9. Serving the docs site locally

The landing page is plain HTML and works directly from the filesystem. The docs section (docsify) does not, it uses `fetch()` to load `.md` files at runtime, which browsers block on `file://` origins. A local HTTP server is required:

```bash
make serve-docs
```

It first runs `make stage-docs`, which assembles the site into `site/`: the contents of `docs/pages/` as the root, the markdown files of `docs/` next to them, the `docs/movelooper/` and `docs/dev/` folders, and `examples/demo/demo.gif`. Then it installs the pinned [Caddy](https://caddyserver.com) into `./.gobin` on first use and serves `site/` until you press Ctrl+C.

| URL | Page |
|---|---|
| `http://localhost:8080/` | Landing page |
| `http://localhost:8080/docs/` | Docs (docsify) |

The Deploy GitHub Pages job runs the same `make stage-docs` and publishes `site/`, so the local preview has the layout of the published site. Edit the files under `docs/` and `examples/`, never under `site/`: it is rebuilt from scratch on every run.
