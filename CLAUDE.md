# GoRead2 Claude Code Guide

GoRead2 is a multi-user RSS reader written in Go with a JavaScript/CSS frontend, deployed to Google App Engine. It uses Google Datastore for persistence, Google OAuth for authentication, and Stripe for subscriptions.

## Build, Test, Lint

```bash
make build          # compile Go binary
make build-frontend # minify JS + CSS (requires npm)
make all            # build-frontend + build + test-quick (default)
make test           # run test suite via ./test.sh
make test-datastore # same suite against a temporary Datastore emulator (stopped on exit)
make lint           # golangci-lint
make docs           # check markdown files for broken links/anchors, writing-style violations, and stale doc-to-code references (requires npm)
make dev            # start local dev server (validates config first)
make dev-login      # dev server with a local sign-in instead of Google OAuth (see docs/testing.md)
```

Run `make test` and `make lint` after any Go changes before committing.

Without an emulator, the Datastore-backed tests skip. To run them, use `make test-datastore`, or `./scripts/with-datastore-emulator.sh <command>` for a narrower command or for `make dev-login` against Datastore. Never start the emulator directly with `gcloud beta emulators datastore start`, in the background or in a terminal pane: it keeps running after the session ends. The wrapper shuts it down when the command exits, fails, or is interrupted. Run `make docs` after any changes to root-level or `docs/` markdown files.

## Documentation Conventions

[CONTRIBUTING.md](CONTRIBUTING.md#documentation) has the writing style rules, where docs live, and the feature area → doc mapping. Before working in a feature area, read the mapped doc for context; if your change alters behaviour, configuration, or APIs, update that doc in the same commit. Those rules govern documentation written for people; this file is instructions for coding agents and addresses its reader directly, so it does not follow them itself.

## Commit Conventions

Every commit message must end with two trailers: one naming the Claude model that wrote the change, and one for the maintainer. For example, from Claude Opus 5.5:

```
Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Co-Authored-By: Jeffrey Pratt <jeffrey@jeffreypratt.org>
```

## Issue Tracking: bd (beads)

This project uses **bd (beads)** for ALL issue tracking. Do NOT use markdown TODOs, TodoWrite, TaskCreate, or other tracking methods. Run `bd prime` after compaction or at the start of a new session to reload full workflow context.

### Workflow

1. **Find ready work**: `bd ready`
2. **Claim atomically**: `bd update <id> --claim`
3. **Implement, test, document**
4. **Discovered new work?** `bd create --title="Found bug" --description="Details" -p 1 --deps discovered-from:<parent-id>`
5. **Close**: `bd close <id>` (or `bd close <id1> <id2> ...` for multiple)
6. **Commit beads state** alongside code: `.beads/issues.jsonl` should always be committed with the related code changes.

### Key Commands

```bash
bd ready                                              # unblocked issues
bd show <id>                                          # detailed issue view
bd create --title="..." --description="..." -t task -p 2  # new issue
bd update <id> --claim                                # claim + mark in_progress
bd close <id> --reason "Done"                         # complete
bd dep add <issue> <depends-on>                       # add dependency
bd dolt push / bd dolt pull                           # sync with remote
bd remember "insight"                                 # persistent knowledge across sessions (not MEMORY.md)
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

- Use bd for ALL task tracking, never markdown TODO lists or external trackers
- Use `--json` flag when parsing output programmatically
- Link discovered work with `discovered-from` dependencies
- Check `bd ready` before asking "what should I work on?"

Architecture in one line: issues live in a local Dolt DB; sync uses `refs/dolt/data` on the git remote; `.beads/issues.jsonl` is a passive export. See https://github.com/gastownhall/beads/blob/main/docs/SYNC_CONCEPTS.md for details and anti-patterns.

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

# 5. Clean up any stashes or stale remote branches, then hand off context for the next session
```

Never stop before pushing, since that leaves work stranded locally. Never say "ready to push when you are"; push it. If push fails, resolve and retry.

## Ephemeral Planning Documents

Do NOT create planning or design documents (PLAN.md, DESIGN.md, ARCHITECTURE.md, etc.) in the repo root. If you need to create one, place it in `history/` which is gitignored.

## Multi-Agent Orchestration (CAO)

When `CAO_TERMINAL_ID` is set, you are in a multi-agent session:

- **Supervisor**: Coordinates work via beads issues, never writes code directly
- **Developer**: Implements tasks from `bd show <id>`; iterates on reviewer feedback
- **Reviewer**: Reviews diffs; approves or requests changes

Beads issues carry all task context (title, description, acceptance criteria, design notes). Pass the issue ID between agents; they fetch details via `bd show <id>`.
