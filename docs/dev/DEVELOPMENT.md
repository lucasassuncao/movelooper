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

---

## 1. Prerequisites

| Tool | Version | Purpose |
|---|---|---|
| Go | 1.23+ | Build and test |
| Make | any | Task runner |
| Git | any | Version control |

All other tools (golangci-lint, goreleaser, gosec, gotestsum, gomarkdoc) are invoked via `go run` and require no global installation. They are pinned to specific versions in the `Makefile`.

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
make docs            # regenerate package README files (gomarkdoc)
make deps            # go mod download && go mod tidy
make clean           # remove build artifacts and test cache
make all             # fmt + docs + lint + security + test-coverage
```

### Before committing

Run at minimum:

```bash
make fmt && make test
```

CI enforces formatting, dependency tidiness, tests, lint, security, and generated docs — all of these must pass on `main`.

### Commit messages

- One short subject line (imperative mood, 50 chars or fewer)
- Up to 3–5 terse bullet points in the body when context is needed
- No trailing period on the subject line

```
refactor: replace dispatchAction switch with FileAction Strategy registry

- add FileAction interface and moveAction/copyAction/symlinkAction structs
- register in fileActions map; unknown names fall back to move
- remove fmt.Errorf("unknown action") path
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

- Tests live in the **same package** as the code they test (`package helper`, not `package helper_test`). This allows testing unexported functions directly, which is intentional — internal invariants are worth testing.
- Each package has one or more `*_test.go` files. Extra test files (e.g. `fileops_extra_test.go`, `filters_extra_test.go`) group related tests that would make the main test file too long.
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

**Silent logger in tests.** Use the `newTestLogger()` helper (defined in `helper/fileops_test.go`) when a function requires a `*pterm.Logger`. It disables all output so test logs stay clean.

**No mocks.** Tests hit real files and real functions. External state (filesystem, OS executable path) is controlled via `t.TempDir()` and carefully crafted inputs. This is intentional — mocks can mask real integration bugs.

### Test coverage reference

See [`TESTS.md`](TESTS.md) for the full list of test cases per file.

---

## 5. Code conventions

### General

- **No comments that explain what the code does.** Well-named identifiers do that. Comments should explain *why*: a non-obvious constraint, a platform workaround, a subtle invariant.
- **No unused error handling.** Only handle errors that can actually occur. Do not add fallbacks for scenarios that cannot happen.
- **YAGNI.** Do not add abstractions or configuration options for hypothetical future requirements.
- **Three similar lines before abstracting.** Duplication is cheaper than the wrong abstraction.

### Packages and imports

Follow the existing dependency rule (see [`DESIGN.md`](DESIGN.md) §2):

```
models → (nothing internal)
helper, history → models
config → helper, history, models
cmd → all of the above
```

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

- File paths that come from user config or directory walks are passed to `filepath.Clean` or opened with `#nosec G304` annotations where gosec would otherwise flag them. Do not remove these annotations blindly — they exist because the path source is documented.
- History and log files are created with mode `0600` (owner read/write only).
- Directories are created with mode `0750`.

### Linter configuration

golangci-lint is configured in `.golangci.yaml`. Active linters include:

- `staticcheck`, `gocritic` — correctness and idiom checks
- `gocognit`, `gocyclo`, `nestif` — complexity limits (cognitive complexity ≤ 35)
- `dupl` — duplicate code detection
- `misspell` — spelling in comments and strings
- `lll` — line length ≤ 300 characters
- `unparam` — unused function parameters

Run `make lint` before opening a PR. The CI pipeline will fail if lint errors are present.

---

## 6. Adding new features

### Checklist

- [ ] Read [`DESIGN.md`](DESIGN.md) to understand where the change belongs
- [ ] Follow the extension guides in [`DESIGN.md`](DESIGN.md) §6 for registries and the builder
- [ ] Write tests before or alongside the implementation (TDD preferred)
- [ ] Run `make all` (fmt + docs + lint + security + test-coverage) before pushing
- [ ] Update [`TESTS.md`](TESTS.md) with any new test cases
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

The CI workflow (`.github/workflows/ci.yml`) runs on every push and pull request to `main`. Steps run in order:

| Step | Command | What it checks |
|---|---|---|
| Format | `make fmt` + `git diff` | All code is `gofmt`-formatted |
| Dependencies | `go mod tidy` + `git diff` | `go.mod` and `go.sum` are up to date |
| Test | `make test-coverage` | All tests pass; coverage report uploaded as artifact |
| Lint | `make lint` | golangci-lint passes |
| Security | `make security` | gosec finds no medium+ severity issues |
| Docs | `make docs` + `git diff` | Package README files are regenerated and committed |

**All steps must pass before merging.** There are no exceptions.

If the docs check fails, run `make docs` locally, review the diff, and commit the result.

---

## 8. Release process

Releases are managed with goreleaser and GitHub Actions. Binaries are built for:

- `linux/amd64`, `linux/arm64`
- `windows/amd64`, `windows/arm64`
- `darwin/amd64`, `darwin/arm64`

### Creating a release

1. Ensure `main` is clean and all CI checks pass.
2. Create and push an annotated tag:
   ```bash
   make tag VERSION=v1.2.3
   ```
3. The release workflow picks up the tag, builds all platform binaries, and publishes the GitHub release automatically.

### Version naming

Follow [Semantic Versioning](https://semver.org):

- `PATCH` (`v1.2.3` → `v1.2.4`): bug fixes, no API or config changes
- `MINOR` (`v1.2.3` → `v1.3.0`): new features, backwards-compatible config changes
- `MAJOR` (`v1.2.3` → `v2.0.0`): breaking changes to the config format or CLI interface

Pre-release versions (`-alpha`, `-beta`, `-rc.1`) are marked as pre-release on GitHub automatically by goreleaser when the tag contains a pre-release suffix.
