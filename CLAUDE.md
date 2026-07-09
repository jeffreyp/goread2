# GoRead2 Claude Code Guide

GoRead2 is a multi-user RSS reader written in Go with a JavaScript/CSS frontend, deployed to Google App Engine. It uses Google Datastore for persistence, Google OAuth for authentication, and Stripe for subscriptions.

## Build, Test, Lint

```bash
make build          # compile Go binary
make build-frontend # minify JS + CSS (requires npm)
make all            # build-frontend + build + test (default)
make test           # run test suite via ./test.sh
make lint           # golangci-lint
make docs           # check markdown files for broken relative links and anchors (requires npm)
make dev            # start local dev server (validates config first)
```

Run `make test` and `make lint` after any Go changes before committing. Run `make docs` after any changes to root-level or `docs/` markdown files.

## Documentation Conventions

Root-level files use uppercase names (`README.md`, `CLAUDE.md`, `CONTRIBUTING.md`, `LICENSE`); this is the standard open-source convention that GitHub and tooling give special prominence. Files inside `docs/` use lowercase with hyphens (`setup.md`, `feature-flags.md`).

### Writing Style

All documentation, in `docs/` and at the root, must be written in a professional, plain, and precise tone. Write as if for a colleague reading a reference manual, not a friend. Do not add commentary that the content already demonstrates (for example, avoid a line like "the caching strategy is dead simple" above a section that already shows it is simple).

Follow these rules:

- Avoid em dashes unless there is no other clear way to write the sentence. Use a period, comma, or a "which"/"that" clause instead.
- Put the main point of a sentence first, not after a leading clause. Write "This is good for both the goose and the gander," not "This is not only good for the goose, it is good for the gander."
- Before editing a document, read it in full. Look for existing text to streamline or clarify, rather than only appending new material.
- Documentation describes the current state of the product, not its history. Do not use a doc as a TODO list or a record of past decisions. Record decisions in `history/`, a commit message, or a bd issue, not in `docs/`.

### Feature area → doc mapping

Before working in a feature area, read the corresponding doc for context. If your change alters behaviour, configuration, or APIs, update the doc in the same commit.

| When touching… | Read/update… |
|----------------|--------------|
| `internal/auth/`, OAuth flow, sessions | `docs/authentication.md` |
| `internal/handlers/admin*`, `cmd/admin/` | `docs/admin.md` |
| `internal/services/subscription*`, Stripe webhooks | `docs/stripe.md`, `docs/feature-flags.md` |
| `internal/cache/`, HTTP cache headers | `docs/caching.md` |
| `internal/database/`, schema migrations | `docs/setup.md` (schema section) |
| API endpoints (`internal/handlers/`) | `docs/api.md` |
| Deployment config (`app.yaml`, `cron.yaml`, secrets) | `docs/deployment.md` |
| Monitoring (`monitoring/`) | `docs/monitoring.md` |
| Performance, query optimisation | `docs/performance.md` |
| Security controls, input validation | `docs/security.md` |
| Test infrastructure, `test.sh` | `docs/testing.md` |
| User-facing features (UI, keyboard shortcuts) | `docs/features.md` |

## Commit Conventions

Every commit message must include both trailers:

```
Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-Authored-By: Jeffrey Pratt <jeffrey@jeffreypratt.org>
```

## Issue Tracking: bd (beads)

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

- Use bd for ALL task tracking, never markdown TODO lists or external trackers
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

Never stop before pushing, since that leaves work stranded locally. If push fails, resolve and retry.

## Ephemeral Planning Documents

Do NOT create planning or design documents (PLAN.md, DESIGN.md, ARCHITECTURE.md, etc.) in the repo root. If you need to create one, place it in `history/` which is gitignored.

## Multi-Agent Orchestration (CAO)

When `CAO_TERMINAL_ID` is set, you are in a multi-agent session:

- **Supervisor**: Coordinates work via beads issues, never writes code directly
- **Developer**: Implements tasks from `bd show <id>`; iterates on reviewer feedback
- **Reviewer**: Reviews diffs; approves or requests changes

Beads issues carry all task context (title, description, acceptance criteria, design notes). Pass the issue ID between agents; they fetch details via `bd show <id>`.


<!-- BEGIN BEADS INTEGRATION v:1 profile:minimal hash:7510c1e2 -->
## Beads Issue Tracker

This project uses **bd (beads)** for issue tracking. Run `bd prime` to see full workflow context and commands.

### Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --claim  # Claim work
bd close <id>         # Complete work
```

### Rules

- Use `bd` for ALL task tracking; do NOT use TodoWrite, TaskCreate, or markdown TODO lists
- Run `bd prime` for detailed command reference and session close protocol
- Use `bd remember` for persistent knowledge; do NOT use MEMORY.md files

**Architecture in one line:** issues live in a local Dolt DB; sync uses `refs/dolt/data` on your git remote; `.beads/issues.jsonl` is a passive export. See https://github.com/gastownhall/beads/blob/main/docs/SYNC_CONCEPTS.md for details and anti-patterns.

## Session Completion

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
<!-- END BEADS INTEGRATION -->
