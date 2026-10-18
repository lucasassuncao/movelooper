# Contributing to movelooper

Thank you for your interest in contributing. This document covers how to report issues, propose changes, and submit pull requests.

---

## Reporting issues

Open a GitHub issue and include:

- **What you expected** to happen
- **What actually happened** (paste the full error output or log)
- **Steps to reproduce**: the minimal config and command that triggers the issue
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

- [ ] `make all` passes locally (fmt, docs, lint, security, tests, vulnerability scans)
- [ ] New behaviour is covered by tests
- [ ] `docs/dev/DESIGN.md` updated if a new architectural pattern or decision was introduced
- [ ] Commit messages follow the project style (see below)

### Commit message style

[Conventional Commits](https://www.conventionalcommits.org): `type(scope): subject`, in imperative mood with no trailing period, plus an optional prose body that says why the change exists. The type decides where the commit appears in the release notes, so it matters. The full rule is in the [Development Guide](docs/dev/DEVELOPMENT.md#commit-messages).

```
fix(undo): stop undo of a copy from deleting the last copy

Undoing a copy removes the file at the destination, which reverses the
run only while the original is still at the source. Nothing checked
that: deleting the original and then undoing left no copy at all, and
editing the copy before an undo took the edits with it. Undo now keeps
the file in both cases and warns; --force removes it anyway.
```

---

## Code of conduct

Be direct and constructive. Critique code, not people. Assume good intent.
