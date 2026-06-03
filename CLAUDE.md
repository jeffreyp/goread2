# GoRead2 — Claude Code Guide

GoRead2 is a multi-user RSS reader written in Go with a JavaScript/CSS frontend, deployed to Google App Engine. It uses Google Datastore for persistence, Google OAuth for authentication, and Stripe for subscriptions.

## Build, Test, Lint

```bash
make build          # compile Go binary
make build-frontend # minify JS + CSS (requires npm)
make all            # build-frontend + build + test (default)
make test           # run test suite via ./test.sh
make lint           # golangci-lint
make dev            # start local dev server (validates config first)
```

Run `make test` and `make lint` after any Go changes before committing.

## Documentation Conventions

Root-level files use uppercase names (`README.md`, `CLAUDE.md`, `LICENSE`) — this is the standard open-source convention that GitHub and tooling give special prominence. Files inside `docs/` use lowercase with hyphens (`setup.md`, `feature-flags.md`).

## Commit Conventions

Every commit message must include both trailers:

```
Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-Authored-By: Jeffrey Pratt <jeffrey@jeffreypratt.org>
```

## Issue Tracking — bd (beads)

This project uses **bd (beads)** for ALL issue tracking. Do NOT use markdown TODOs, TaskCreate, or other tracking methods.

### Workflow

1. **Find ready work**: `bd ready`
2. **Claim atomically**: `bd update <id> --claim`
3. **Implement, test, document**
4. **Discovered new work?** `bd create --title="Found bug" --description="Details" -p 1 --deps discovered-from:<parent-id>`
5. **Close**: `bd close <id>`
6. **Commit beads state** alongside code: `.beads/issues.jsonl` should always be committed with the related code changes.

### Key Commands

```bash
bd ready                                              # unblocked issues
bd create --title="..." --description="..." -t task -p 2  # new issue
bd update <id> --claim                                # claim + mark in_progress
bd close <id> --reason "Done"                         # complete
bd close <id1> <id2> ...                              # close multiple
bd dep add <issue> <depends-on>                       # add dependency
bd dolt push / bd dolt pull                           # sync with remote
```

### Issue Types & Priorities

| Type | Use |
|------|-----|
| `bug` | Something broken |
| `feature` | New functionality |
| `task` | Tests, docs, refactoring |
| `epic` | Large feature with subtasks |
| `chore` | Maintenance, tooling |

| Priority | Meaning |
|----------|---------|
| `0` | Critical (security, data loss, broken build) |
| `1` | High |
| `2` | Medium (default) |
| `3` | Low |
| `4` | Backlog |

### Rules

- Use bd for ALL task tracking — never markdown TODO lists or external trackers
- Use `--json` flag when parsing output programmatically
- Link discovered work with `discovered-from` dependencies
- Check `bd ready` before asking "what should I work on?"

## Session Completion Protocol

Work is **not done** until `git push` succeeds.

```bash
# 1. File issues for any remaining work
# 2. Run quality gates if code changed
make test && make lint

# 3. Close/update issue status
bd close <id1> <id2> ...

# 4. Commit and push
git add <files> .beads/issues.jsonl
git commit -m "..."   # include both Co-Authored-By trailers
git pull --rebase
bd dolt push
git push
git status            # must show "up to date with origin"
```

Never stop before pushing — that leaves work stranded locally. If push fails, resolve and retry.

## Ephemeral Planning Documents

Do NOT create planning or design documents (PLAN.md, DESIGN.md, ARCHITECTURE.md, etc.) in the repo root. If you need to create one, place it in `history/` which is gitignored.

## Multi-Agent Orchestration (CAO)

When `CAO_TERMINAL_ID` is set, you are in a multi-agent session:

- **Supervisor**: Coordinates work via beads issues — never writes code directly
- **Developer**: Implements tasks from `bd show <id>`; iterates on reviewer feedback
- **Reviewer**: Reviews diffs; approves or requests changes

Beads issues carry all task context (title, description, acceptance criteria, design notes). Pass the issue ID between agents — they fetch details via `bd show <id>`.
