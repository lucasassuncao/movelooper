# Contributing to movelooper

Thank you for your interest in contributing. This document covers how to report issues, propose changes, and submit pull requests.

---

## Reporting issues

Open a GitHub issue and include:

- **What you expected** to happen
- **What actually happened** (paste the full error output or log)
- **Steps to reproduce** — the minimal config and command that triggers the issue
- **Environment:** OS, architecture, and movelooper version (`movelooper --version`)

For security vulnerabilities, do **not** open a public issue. Email the maintainer directly.

---

## Proposing changes

For non-trivial changes, open an issue or start a discussion before writing code. This avoids investing time in an implementation that conflicts with the project direction.

Small fixes (typos, one-line corrections) can go straight to a pull request.

---

## Submitting a pull request

1. Fork the repository and create a branch from `main`.
2. Follow the [Development Guide](docs/dev/DEVELOPMENT.md) for setup, workflow, and conventions.
3. Run the full check suite before pushing:
   ```bash
   make all
   ```
4. Open a pull request against `main`.

### PR checklist

- [ ] `make all` passes locally (fmt, docs, lint, security, tests)
- [ ] New behaviour is covered by tests
- [ ] `docs/dev/TESTS.md` updated for any new or removed test cases
- [ ] `docs/dev/DESIGN.md` updated if a new architectural pattern or decision was introduced
- [ ] Commit messages follow the project style (see below)

### Commit message style

One short subject line in imperative mood, 50 characters or fewer. Add up to 3–5 terse bullet points in the body only when context is genuinely needed.

```
feat: add {hostname} template token to destination paths

- add "hostname" entry to knownTokens in groupby.go
- handle token in ResolveGroupBy and ResolveRename
- cover both functions in table-driven tests
```

No trailing period on the subject line.

---

## Code of conduct

Be direct and constructive. Critique code, not people. Assume good intent.
