# Contributing Guide

**Languages:** [English](CONTRIBUTING.md) · [Русский](CONTRIBUTING-RU.md) · [中文](CONTRIBUTING-ZH.md) · [日本語](CONTRIBUTING-JA.md) · [Español](CONTRIBUTING-ES.md)

Thank you for contributing to LANdapter! We welcome fixes, docs, and new features.

This document explains how to report issues, propose changes, and open pull requests.

---

## Reporting bugs

Create a **GitHub Issue**.

**Before opening:**

- Search for duplicates.
- Use the latest `main` branch.

**Include:**

- Go, PostgreSQL, Node.js versions.
- OS and version.
- Short problem description.
- Steps to reproduce.
- Expected vs actual behavior.
- Logs and screenshots (UI issues).

See `.github/ISSUE_TEMPLATE.md` if present.

---

## Feature requests

Open an Issue with the `enhancement` label:

- Problem being solved.
- Proposed approach.
- Alternatives considered.

Discuss before large implementations.

---

## Development workflow

### 1. Fork and clone

```bash
git clone git@github.com:YOUR_USER/LANdapter.git
cd LANdapter
```

### 2. Branch

- `fix/short-description` – bug fixes
- `feature/feature-name` – features
- `docs/change` – documentation

```bash
git checkout -b feature/awesome-new-feature
```

### 3. Code layout

- `cmd/` – master and agent entrypoints
- `internal/` – private packages
- `storage/` – PostgreSQL storage layer
- `web/` – React frontend
- `migrations/` – SQL migrations
- `docs/` – documentation

See [ARCHITECTURE.md](ARCHITECTURE.md) for design details.

### 4. Commits

Use [Conventional Commits](https://www.conventionalcommits.org/) when possible:

- `feat: add support for Linux ARM64`
- `fix: prevent panic when upload dir is missing`
- `docs: update API reference`
- `test: add unit tests for installer`

### 5. Tests

```bash
make test          # all tests
make test-unit     # unit only
make test-integration  # needs PostgreSQL
make test-cover    # coverage
```

New behavior should include tests.

### 6. Linting

**Go:**

```bash
golangci-lint run ./...
```

**Frontend (`web/`):**

```bash
npm run lint   # if configured
```

Match existing React/Tailwind style.

### 7. Formatting

```bash
go fmt ./...
```

Use Prettier in the IDE for React when available.

### 8. Manual verification

```bash
make build
make run-master   # terminal 1
make run-agent    # terminal 2
```

### 9. Pull request

Push to your fork and open a PR against `main`.

**PR description:**

- Linked Issue (e.g. `Closes #42`).
- Summary of changes.
- How to test.
- UI screenshots when relevant.

---

## Code style

### Go

- `gofmt` required.
- Handle errors explicitly.
- Prefer dependency injection over globals.
- Godoc on exported symbols.

### React

- Functional components and hooks.
- `PascalCase.jsx` components; `camelCase.js` utilities.
- Tailwind over inline styles.
- Small, reusable components.

---

## Documentation

Update `docs/` when changing API, config, or behavior:

- API: [README.API.md](README.API.md) and other language variants (`README.API-*.md`).
- Config: [CONFIG.md](CONFIG.md) and localized `CONFIG-*.md`.
- Architecture: [ARCHITECTURE.md](ARCHITECTURE.md) and localized copies.

When adding a new `docs/*-RU.md` (or other source locale), add matching English, Chinese, Japanese, and Spanish files with the standard language link bar at the top.

---

## Review process

Maintainers will review within a few days. Address feedback; merged PRs land on `main`.

Questions: open an Issue with the `question` label.

Thank you for helping improve LANdapter!
