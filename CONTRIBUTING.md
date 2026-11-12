# Contributing Guide

This guide explains how to contribute to GoRead2.

## Getting Started

### Development Setup

1. **Fork and clone**:

   Fork the repository by clicking "Fork" at [github.com/jeffreyp/goread2](https://github.com/jeffreyp/goread2), then clone the fork:

   ```bash
   git clone https://github.com/your-username/goread2.git
   cd goread2
   ```

2. **Install dependencies and set up the environment**: see [docs/setup.md](docs/setup.md) for prerequisites, environment variables, and the local OAuth setup.

3. **Run tests**:
   ```bash
   make test
   ```

4. **Start the development server**:
   ```bash
   make dev
   ```
   or `make dev-login` to sign in locally without Google OAuth (see [docs/testing.md](docs/testing.md#local-dev-login-make-dev-login)).

## Development Workflow

### Making Changes

1. **Create a feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make changes**, following the patterns already in the surrounding code rather than inventing new ones.

3. **Write tests** for new functionality:
   ```bash
   # Unit tests (co-located with the code they test)
   touch internal/<package>/your_feature_test.go

   # Integration tests
   touch test/integration/your_feature_integration_test.go
   ```
   See [docs/testing.md](docs/testing.md) for the test helpers available in `test/helpers/` and real examples to follow.

4. **Run tests frequently**:
   ```bash
   # Quick test run (unit tests)
   go test ./internal/...

   # Full test suite using Makefile (recommended)
   make test

   # Or use the test script directly
   ./test.sh
   ```

5. **Commit with a clear message**, ending with a bd issue ID if one exists:
   ```bash
   git add .
   git commit -m "Add RSS feed auto-discovery from <link rel='alternate'> tags (<bd-id>)"
   ```

### Pull Request Process

1. **Push the branch**:
   ```bash
   git push origin feature/your-feature-name
   ```

2. **Create a pull request** on GitHub with a clear title and description, a reference to any related issue, and a screenshot or demo for UI changes.

3. **Address review feedback**: push updates to the same branch and respond to comments.

4. **Merge requirements**: CI checks pass, review approval, no merge conflicts, and the relevant doc updated if the change touches behavior, configuration, or an API (see [Documentation](#documentation) below).

## Documentation

### Writing Style

All documentation in `docs/`, `README.md`, and this file must be written in a professional, plain, and precise tone. Write as if for a colleague reading a reference manual, not a friend. Do not add commentary that the content already demonstrates (for example, avoid a line like "the caching strategy is dead simple" above a section that already shows it is simple).

Follow these rules:

- Avoid em dashes unless there is no other clear way to write the sentence. Use a period, comma, or a "which"/"that" clause instead.
- Put the main point of a sentence first, not after a leading clause. Write "This is good for both the goose and the gander," not "This is not only good for the goose, it is good for the gander."
- Name the specific thing first, then describe it in an appositive, rather than burying the name in a parenthetical. Write "GoRead2 uses Inter, a clean, sans-serif font," not "GoRead2 uses a clean, sans-serif font (Inter)."
- Do not address the reader as "you" or "your." Refer to features and actions generically instead: write "shows subscribed feeds," not "shows your subscribed feeds"; "on navigating away," not "when you navigate away."
- Before editing a document, read it in full. Look for existing text to streamline or clarify, rather than only appending new material.
- Documentation describes the current state of the product, not its history. Do not use a doc as a TODO list or a record of past decisions. Record decisions in a commit message or a bd issue, not in `docs/`.
- Do not cite bd issue IDs or the dates of past events. Dates that are part of a current fact, such as a renewal date, or that appear in an example command, are fine.
- Do not use emoji in prose.
- Do not name real people or give personal email addresses. Refer to roles, such as "the maintainer," and use `example.com` addresses in examples. A configured value that names an account, such as a required GitHub reviewer, may be given as it is configured.
- Refer to code by file and function or type, such as "`SignInGoogleUser` in `internal/auth/auth.go`," not by line number, which goes stale.
- Use real values or clearly marked placeholders, such as `<session-id>` or `your-client-id`, in examples. Do not invent identifiers, output, or figures that look real.

### Where docs live

Root-level files use uppercase names (`README.md`, `CLAUDE.md`, `CONTRIBUTING.md`, `LICENSE`), the standard open-source convention that GitHub and tooling give special prominence. Files inside `docs/` use lowercase with hyphens (`setup.md`, `apple-releases.md`).

[docs/README.md](docs/README.md) indexes the guides by audience; add a new guide there and to the map below. Each guide opens with a summary and a table of contents and ends with Related Documentation. Give a guide a Troubleshooting section only when it has troubleshooting content of its own or [docs/troubleshooting.md](docs/troubleshooting.md) has a section on its topic.

### Doc map

Before working in a feature area, read the corresponding doc for context. If a change alters behavior, configuration, or an API, update the doc in the same commit.

| When touching… | Read/update… |
|----------------|--------------|
| `internal/auth/`, OAuth flow, sessions | `docs/authentication.md` |
| `internal/handlers/admin*`, `cmd/admin/` | `docs/admin.md` |
| `internal/services/subscription*`, Stripe webhooks, `cmd/setup-stripe/` | `docs/stripe.md` |
| Environment variables (`internal/config/`, `internal/secrets/`, `cmd/validate-config/`) | `docs/configuration.md` |
| `internal/cache/`, HTTP cache headers | `docs/performance.md` |
| `internal/database/`, schema migrations, `index.yaml`, how the components fit together | `docs/architecture.md` |
| API endpoints (`internal/handlers/`) | `docs/api.md` |
| Deployment config (`app.yaml`, `cron.yaml`, secrets), `deploy-staging.yml`, `deploy-prod.yml`, `rollback.yml`, `scripts/smoke-check.sh`, `scripts/post-promote-health-watch.sh` | `docs/deployment.md` |
| Monitoring (`monitoring/`) | `docs/monitoring.md` |
| Performance, query optimization | `docs/performance.md` |
| Security controls, input validation | `docs/security.md` |
| Personal data stored, account deletion, data export, retention (`internal/services/user_data_service.go`, `internal/services/retention_service.go`, `internal/handlers/account_handler.go`) | `docs/privacy.md` and `web/templates/privacy.html` |
| Cookies, browser storage, or third-party requests from the web app | `docs/privacy.md` (cookie inventory) and `web/templates/privacy.html` |
| Test infrastructure, `test.sh`, `test.yml`, `docs.yml`, `scripts/with-datastore-emulator.sh`, `scripts/check-benchmark-regression.sh`, `cmd/generate-test-articles/` | `docs/testing.md` |
| Error messages and log lines that operators act on | `docs/troubleshooting.md` |
| User-facing features (UI, keyboard shortcuts) | `docs/features.md` |
| `ios/` (SwiftUI app, Xcode project, fastlane) | `docs/apple-apps.md` |
| `ios/fastlane/`, `ios-release.yml`, `macos-release.yml` | `docs/apple-releases.md` |

## Frontend Architecture

GoRead2 splits its frontend into a core bundle plus several smaller bundles loaded on demand with native ES6 dynamic imports, rather than one large script: `modals.js` (modal interactions), `account.js` (the account page), and `animations.js`/`animations-integration.js`. Check current sizes with `ls -la web/static/js/*.min.js` after `make build-frontend`; this doc doesn't keep its own copy of them, since they change with every edit to the bundles.

`index.html` passes each lazily-loaded bundle's versioned URL to `app.js` in a `data-*-src` attribute (for example `data-modals-src`), so the module is cached like every other asset (see [performance.md](docs/performance.md#versioned-css-and-js)) and `app.js` falls back to the unversioned source file when the attribute is absent, as in tests.

To add a new lazy-loaded module:

1. Create a new module file (e.g., `feature.js`) exporting a class or functions.
2. Add a lazy-load helper to `app.js` that `import()`s it and caches the result, following `loadModalManager` as a model.
3. Pass the module's versioned URL from `index.html` in a `data-` attribute on the `app.min.js` script tag, as `data-modals-src` does.
4. Add the new file to the frontend build in `package.json` and the `Makefile`.
5. Update this section if the loading pattern itself changes.

## Testing Requirements

New functionality needs a unit test co-located with the code (`internal/<package>/*_test.go`) and, for a new or changed API endpoint, an integration test in `test/integration/`. A UI change needs a Jest test in `web/tests/`. [docs/testing.md](docs/testing.md) covers the test helpers in `test/helpers/` (in-memory SQLite, a test HTTP server, mock feed servers) and has real examples to copy from; `internal/handlers/feed_handler_test.go` and `test/integration/api_test.go` are reasonable starting points for a handler and an integration test respectively.

## Security Guidelines

- Filter every database query by the authenticated user's ID; there is no other isolation between accounts.
- Validate and sanitize all input, especially feed URLs (SSRF) and anything rendered as HTML.
- Every state-changing request needs a valid session and CSRF token.
- New protected endpoints must go through `RequireAuth` (or `RequireAdmin` for admin-only ones).

See [docs/security.md](docs/security.md) for how each control is implemented, and `test/security/` for the regression suite that enforces them in CI.

## Issue Tracking with Beads

GoRead2 uses **bd (beads)** for all issue tracking. Issues live in a local Dolt database, synced via `refs/dolt/data` on the git remote; `.beads/issues.jsonl` is a passive export of that data, committed to git for visibility.

### Common Commands

```bash
# Find work with no blockers
bd ready

# Create a new issue
bd create --title="Description" --type=bug|feature|task --priority=2

# Show issue details
bd show <id>

# Claim and start working
bd update <id> --claim

# Close when done
bd close <id>

# Search
bd search "keyword"
```

### Issue Types

| Type | Use |
|------|-----|
| `bug` | Something broken |
| `feature` | New functionality |
| `task` | Tests, docs, refactoring |
| `epic` | Large feature with subtasks |
| `chore` | Maintenance, tooling |

### Priority Levels

| Priority | Meaning |
|----------|---------|
| `0` / P0 | Critical (security, data loss, broken build) |
| `1` / P1 | High |
| `2` / P2 | Medium (default) |
| `3` / P3 | Low |
| `4` / P4 | Backlog |

## Issue Reporting

For external users who may not have Beads installed, open a GitHub issue instead.

### Bug Reports

Include these details:

- **Environment**: Local or App Engine
- **Steps to reproduce**: Exact sequence of actions
- **Expected behavior**: What should happen
- **Actual behavior**: What actually happens
- **Logs**: Relevant error messages and stack traces (redact secrets)

### Feature Requests

Describe the use case, a proposed solution, and any alternatives considered.

## Release Process

There is no versioned release for the web app: every commit to `main` that passes CI deploys automatically to a zero-traffic staging version, and a maintainer promotes it to production after a manual approval and a smoke check. See [docs/deployment.md](docs/deployment.md) for the full pipeline. The native Apple apps do use versioned releases through TestFlight and a notarized DMG; see [docs/apple-releases.md](docs/apple-releases.md).

## Getting Help

- [Setup Guide](docs/setup.md) - Local installation
- [Testing Guide](docs/testing.md) - Running and writing tests
- [API Reference](docs/api.md) - Every endpoint
- [Troubleshooting Guide](docs/troubleshooting.md) - Common issues

Questions about contributing, or feedback on a pull request, belong as comments on the relevant GitHub issue or pull request.
