# Testing Guide

Testing guide for GoRead2's multi-user RSS reader application.

## Table of Contents

- [Overview](#overview)
- [Current Test Structure](#current-test-structure)
- [Running Tests](#running-tests)
- [Test Categories](#test-categories)
- [Test Environment](#test-environment)
- [Coverage Goals](#coverage-goals)
- [Test Data and Fixtures](#test-data-and-fixtures)
- [Security Testing](#security-testing)
- [CI/CD Integration](#cicd-integration)
- [Writing New Tests](#writing-new-tests)
- [Test Performance](#test-performance)
- [Performance Testing](#performance-testing)
- [Debugging Tests](#debugging-tests)
- [Troubleshooting](#troubleshooting)
- [Related Documentation](#related-documentation)

## Overview

GoRead2's testing infrastructure includes:
- **Package-level unit tests** across nine `internal/` packages plus `cmd/admin`; coverage drifts often enough that no fixed numbers are kept here, regenerate with `go test ./internal/... -cover` for the current figures (see [Coverage Goals](#coverage-goals) below)
- **Integration tests** for end-to-end API validation (`test/integration/`)
- **A dedicated security regression suite** (`test/security/`)
- **Frontend tests** with Jest and jsdom (149 tests across 8 files)
- **CI/CD integration** with GitHub Actions

## Current Test Structure

```
internal/           # Unit tests, one or more <name>_test.go per package
├── assets/
├── auth/
├── cache/
├── config/
├── database/        # datastore_test.go and datastore_user_article_test.go are emulator-gated
├── handlers/
├── middleware/
├── secrets/
└── services/
cmd/
└── admin/           # Admin CLI command tests, including fatal-path tests via subprocess re-exec
test/
├── integration/     # End-to-end API and workflow tests
├── security/        # Consolidated security regression suite, run as a dedicated CI job
├── helpers/         # Shared test database, server, and HTTP helpers (see Test Helpers below)
└── fixtures/        # Sample feed data and on-disk XML fixtures
web/tests/           # Jest/jsdom frontend tests (149 tests across 8 files)
```

Every package's actual test files are listed next to its source under `internal/`, `cmd/admin/`, and `test/`; this doc doesn't keep its own copy of that list, since it goes stale every time a test file is added, renamed, or split.

## Running Tests

### All Tests (Recommended)

```bash
# Use the test script directly
./test.sh

# Or use the Makefile target
make test
```

The test script runs:
- Backend unit and integration tests
- Frontend JavaScript tests
- Coverage report generation
- Code quality checks (`make lint` remains a separate required gate; the script only warns on a lint failure)
- Build verification
- Colored output for better readability

### Complete Build and Test

```bash
# Run complete build with frontend assets, build, and tests (uses test-quick)
make all
```

### Backend Tests Only

```bash
# All package-level unit tests
go test ./internal/...

# Specific package tests
go test ./internal/config/
go test ./internal/auth/
go test ./internal/services/
go test ./internal/handlers/

# Integration tests
go test ./test/integration/...

# With coverage
go test -coverprofile=coverage.out ./internal/...
go tool cover -html=coverage.out -o coverage.html

# Verbose output with race detection
go test -v -race ./internal/... ./test/integration/...
```

### Build System Commands

```bash
make build            # Build the Go application
make build-frontend   # Build minified JS/CSS assets
make validate-config  # Validate application configuration
make clean            # Remove build artifacts
make test-quick       # Run tests using Go cache, fast for dev iteration
make test             # Run full test suite with coverage (CI/pre-deploy)
make test-race        # Run Go tests with race detector (CI also runs this automatically)
```

### Frontend Tests Only

```bash
# Navigate to project root and run
npm test

# With coverage
npm run test:coverage

# Watch mode for development
npm run test:watch

# Specific test file
npm test -- app-core.test.js
```

## Test Categories

### 1. Package-Level Unit Tests

Each `internal/` package is tested in isolation, mostly without a database or HTTP server. A few patterns worth knowing before reading a package's test files:

- **Handlers** (`internal/handlers/`): request and error-path tests using `httptest.ResponseRecorder` and a per-file mock `database.Database` (for example `mockDBFeedHandler`, `mockDBAdminHandler`), not a full integration setup. `Login`/`Callback` are not unit-tested, since they call out to Google's real OAuth endpoints; they're exercised instead by `test/integration` and manual verification. The steps the Google callback shares with the dev login (state consumption, session creation, and the native code handoff) are covered by the `devlogin`-tagged tests in `dev_login_test.go`, which `test.sh` runs with `go test -tags devlogin`.
- **Services** (`internal/services/`): Stripe calls are served by `fakeStripe`, an `httptest` server installed with `stripe.SetBackend` that returns canned responses keyed by method and path and records each request. This exercises customer deletion scoping by `user_id` metadata, tolerance of `resource_missing`, search query escaping, and the subscription webhook's handling of deleted users, without a real Stripe account. The admin token system (cryptographic generation, database-backed validation, lifecycle, and bootstrap protection) is tested against both SQLite (`admin_token_test.go`) and the Datastore emulator (`admin_token_datastore_test.go`); see [Datastore Emulator](#datastore-emulator-internalservicesadmin_token_datastore_testgo) below.
- **Database** (`internal/database/`): `datastore_test.go` and `datastore_user_article_test.go` exercise `DatastoreDB`'s implementation of the `database.Database` interface against the CI emulator, mirroring the SQLite `schema_test.go` suite.
- **Secrets** (`internal/secrets/`): `GetOAuthCredentials` and `GetStripeCredentials` cover environment-variable retrieval, the `_secret:` prefix for Secret Manager references, and fallback priority. `GetSecret` itself is the hardest of the three to cover fully, since the missing-credential and Secret-Manager-error paths need the Secret Manager API mocked.

### 2. Backend Integration Tests

#### API Endpoints (`test/integration/api_test.go`)

End-to-end tests against the real HTTP handlers: authenticated feed and article operations, user data isolation between two accounts, and status codes for unauthenticated and malformed requests.

#### User Workflow Tests (`test/integration/workflow_test.go`, `critical_workflows_test.go`)

Complete user journeys through the application over real HTTP: sign-up through feed subscription, article read/star, OPML import and export, and admin privilege grant/verify/revoke.

#### Account Deletion Trial Test (`test/integration/account_deletion_trial_test.go`)

`TestSignInAfterAccountDeletionStartsWithTrialEnded` covers the anti-abuse path for deleted accounts. It signs a Google ID in through `AuthService.SignInGoogleUser`, the layer below the Google token exchange, deletes the account through `DELETE /api/account`, and signs the same Google ID in again. The new account must have a different user ID, a `TrialEndsAt` equal to its `CreatedAt`, and `IsUserSubscriptionActive` false. A Google ID that never deleted an account must still get the 30-day trial.

The test sets `ACCOUNT_TOMBSTONE_KEY`, since deletion writes no tombstone without it. It runs once on SQLite and once on the Datastore emulator, because both `CreateUser` implementations must keep a preset `TrialEndsAt` rather than apply the 30-day default. The Datastore case skips unless `DATASTORE_EMULATOR_HOST` is set.

#### Performance Benchmarks (`test/integration/performance_test.go`)

Timing is measured with `testing.B` benchmarks, run by a dedicated `benchmark` CI job rather than `go test`'s normal test run (see [CI regression gate](#ci-benchmark-regression-gate-scriptscheck-benchmark-regressionsh) below):

```go
func BenchmarkGetUserFeeds100(b *testing.B)           // Query all feeds for a user subscribed to 100
func BenchmarkGetUserArticlesPaginated(b *testing.B)  // Paginated query (50/page) against 1000 articles
func BenchmarkGetUserUnreadCounts(b *testing.B)       // Per-feed unread counts across 100 feeds
func BenchmarkConcurrentReads(b *testing.B)           // 10 users concurrently reading a shared feed
```

Fixture setup (creating the feeds/articles/users) happens once per `-count` repetition via `b.ResetTimer()`, so only the operation named is measured, not the setup cost. `TestConcurrentUserOperations` covers correctness of concurrent reads *and* writes under `-race` separately, as a concurrency-safety test rather than a timing one.

Run locally:

```bash
go test -run='^$' -bench=. -benchmem ./test/integration/...
```

#### CI Benchmark Regression Gate (`scripts/check-benchmark-regression.sh`)

The `benchmark` job in `test.yml` runs the four benchmarks above with `-benchtime=50x -count=10` and compares them against a baseline using [`benchstat`](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat). No baseline file is committed to the repo: raw timings are hardware-dependent, so a baseline is only meaningful compared against a run on the *same* runner type it was recorded on. Instead:

- The baseline is the most recent successful main-branch run's benchmark output, persisted via `actions/cache` under a rolling key (`bench-baseline-<run-id>`, restored by the `bench-baseline-` prefix).
- Every push to `main` whose benchmarks *don't* regress rolls the cache forward to that run's numbers, so the baseline tracks main over time instead of drifting stale.
- `scripts/check-benchmark-regression.sh <baseline> <current> [threshold-pct]` (default threshold 20%) parses `benchstat -format csv` output and fails only on sec/op deltas that are both `>threshold%` *and* statistically significant per benchstat's own test; a change benchstat marks `~` is noise and is ignored regardless of the raw percentage. `B/op`/`allocs/op` deltas are printed but not gated on.
- A regression fails the `benchmark` job, which fails the whole `test.yml` workflow, which blocks `deploy-staging.yml` (gated on `workflow_run.conclusion == 'success'`, the same mechanism as every other required job). On a pull request, a regression additionally posts the benchstat report as a PR comment via `gh pr comment`.
- If no cache is available to restore (for example, after cache eviction), the job skips the comparison and bootstraps a fresh baseline.

### 3. Frontend Tests

Jest and jsdom cover the frontend across 8 files (149 tests total): `app-core.test.js` (DOM manipulation, event handling, API mocking, form validation, UI state), `error-handler.test.js` (connection indicator, error display by type, toast notifications, HTTP status classification), `pagination.test.js` (Load More button state, cursor-based pagination, article rendering and indexing across pages; Load More appears whenever the server returns a next cursor, for any feed view, not only "all"), `accessibility.test.js` (ARIA, focus, keyboard navigation), `account-app.test.js`, `article-scroll.test.js`, `goread-app.test.js`, and `integration.test.js`. `web/tests/utils.js` and `setup.js` hold shared test utilities and DOM/fetch setup rather than tests themselves.

## Test Environment

### Environment Variables

Tests use these environment variables:

```bash
export GOOGLE_CLIENT_ID=test_client_id
export GOOGLE_CLIENT_SECRET=test_client_secret
export GOOGLE_REDIRECT_URL=http://localhost:8080/auth/callback
export SUBSCRIPTION_ENABLED=false  # Disable for most tests
```

### Database Setup

Tests use in-memory SQLite (`file::memory:` or `:memory:`) rather than a file on disk, which eliminates disk I/O and lets each test start from a clean, empty schema. `test/helpers.CreateTestDB(t)` creates one and applies the schema; package-level tests inside `internal/database` and `internal/services` open their own in most cases, since they test that package directly rather than through the shared helper.

### Datastore Emulator (`internal/services/admin_token_datastore_test.go`)

Tests gated on `DATASTORE_EMULATOR_HOST` exercise `DatastoreDB` (the production database implementation) against a real (local) Datastore emulator instead of mocks. They `t.Skip` when the env var is unset, so they run silently skipped in a plain local `make test`/`go test ./...`. The same gating is used by `internal/database/datastore_test.go` and `internal/database/datastore_user_article_test.go`, which exercise `DatastoreDB` through the `database.Database` interface (see below).

**In CI** (`.github/workflows/test.yml`, `test` job): a Cloud Datastore emulator is started before the unit test step:
1. `actions/setup-java` installs a Java 21+ JRE (the emulator is a Java process; the runner's default `java` isn't guaranteed to meet the minimum version).
2. `google-github-actions/setup-gcloud` provisions a standalone gcloud SDK with the `beta` and `cloud-datastore-emulator` components (the runner's preinstalled `gcloud` is apt-managed and refuses `gcloud components install`).
3. `gcloud beta emulators datastore start --host-port=localhost:8081 --no-store-on-disk --consistency=1.0` runs in the background; the step polls `http://localhost:8081/` until it responds before continuing.
4. `DATASTORE_EMULATOR_HOST=localhost:8081` is exported for the rest of the job, so `go test ./internal/...` picks up the gated tests instead of skipping them.

**`--consistency=1.0` is required, not cosmetic**: the emulator defaults to simulating Datastore's eventual consistency (~0.9), which makes `GetAll` queries in `ListAdminTokens`/`TestDatastoreAdminTokenCompatibility` intermittently miss entities written moments earlier.

**Aggregation queries are not exercised**: the legacy emulator does not implement `RunAggregationQuery`. `DatastoreDB` counts articles with aggregation count queries in production and falls back to a keys-only query when the server returns `Unimplemented`, so emulator runs cover only the fallback path.

**Local setup** (to run these tests outside CI): `scripts/with-datastore-emulator.sh` starts an emulator with the CI flags, sets `DATASTORE_EMULATOR_HOST`, runs the given command, and shuts the emulator down when the command exits, fails, or is interrupted. `make test-datastore` runs the full `test.sh` suite through it.
```bash
gcloud components install beta cloud-datastore-emulator   # once
make test-datastore                                        # full suite, Datastore tests included
./scripts/with-datastore-emulator.sh go test ./internal/database/... -run TestDatastore
```
The wrapper listens on `localhost:8081` unless `DATASTORE_EMULATOR_PORT` is set, and refuses to start if the port is already in use, which also exposes an emulator left running from an earlier session. The emulator's Java process requires a Java 21+ JRE; the wrapper puts Homebrew's keg-only `openjdk` on `PATH` when it is installed.

Do not start the emulator directly with `gcloud beta emulators datastore start ... &`. A background emulator keeps running and holds its port until it is stopped by hand.

`AdminToken`/`User` entity operations (via `SubscriptionService`, using the raw `*datastore.Client` from `DatastoreDB.GetClient()`) are covered in `internal/services/admin_token_datastore_test.go`. `DatastoreDB`'s own methods implementing the `database.Database` interface (`AddFeed`, `GetUserFeedArticles`, `MarkUserArticleRead`, `ToggleUserArticleStar`, `GetUserUnreadCounts`, `CreateUser`, `GetAccountStats`, `CleanupOrphanedUserArticles`, sessions, audit logs, and the rest) are covered directly (through the `Database` interface, mirroring the SQLite `schema_test.go` suite) in `internal/database/datastore_test.go` and `internal/database/datastore_user_article_test.go`.

### Local Dev Login (`make dev-login`)

`make dev-login` starts the server built with the `devlogin` tag, in which `/auth/login` opens a local form instead of Google's consent screen (see [authentication.md](authentication.md#local-dev-login)). The form pre-fills a fresh Google ID such as `dev-1a2b3c4d`, with an `@example.test` email, so submitting it unchanged creates a new disposable account. Typing a Google ID that was used before signs in to that account, or, after the account has been deleted, creates a new account with the trial already ended.

The target fills in defaults for any of these variables that are not already set:

| Variable | Default | Purpose |
|----------|---------|---------|
| `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET` | `dev-login` | Placeholders that satisfy the startup check; Google is never contacted |
| `GOOGLE_REDIRECT_URL` | `http://localhost:8080/auth/callback` | Same |
| `ACCOUNT_TOMBSTONE_KEY` | `dev-login-tombstone-key` | Enables tombstones, so re-sign-in after deletion is testable |
| `SUBSCRIPTION_ENABLED` | `false` | Account deletion skips Stripe |

Without `GOOGLE_CLOUD_PROJECT`, accounts are stored in the local SQLite file, `goread2.db`. To use the Datastore implementation instead, run the server through the emulator wrapper, which sets `DATASTORE_EMULATOR_HOST` and stops the emulator when the server exits. The server refuses a `GOOGLE_CLOUD_PROJECT` without `DATASTORE_EMULATOR_HOST`:

```bash
GOOGLE_CLOUD_PROJECT=goread2-local-test ./scripts/with-datastore-emulator.sh make dev-login
```

Without Cloud Tasks, `DELETE /api/account` deletes the account in-process and responds `200` instead of queueing it. The iOS and macOS apps use the dev login through their usual sign-in button when the Debug scheme points at the local server (see [apple-apps.md](apple-apps.md#schemes-and-the-api-base-url)).

### Test Helpers (`test/helpers/`)

```go
// HTTP mocking
helpers.NewMockFeedServer(t, feedXML)                    // Single feed server
helpers.NewMockFeedServerWithStatus(t, statusCode, body) // Custom status
helpers.NewMockMultiFeedServer(t, map[string]string)     // Multiple feeds, keyed by path
helpers.NewMockHTTPClient(server)                        // Wraps a mock server as an HTTPClient

// Database setup
helpers.CreateTestDB(t)              // In-memory SQLite database.Database
helpers.CreateTestDatastoreDB(t)     // Datastore database on the emulator; skips unless DATASTORE_EMULATOR_HOST is set
helpers.CreateTestUser(t, db, ...)   // Test user creation
helpers.CreateTestFeed(t, db, ...)   // Test feed creation
helpers.CreateTestArticle(t, db, ...) // Test article creation

// Server setup
helpers.SetupTestServer(t)           // Router, handlers, and services over CreateTestDB
helpers.SetupTestServerWithDB(t, db) // The same server over any database.Database

// Requests against a TestServer
ts.CreateAuthenticatedRequest(t, method, url, body, user) // Session cookie + CSRF token set
helpers.CreateUnauthenticatedRequest(t, method, url, body)
ts.ExecuteRequest(req)                                     // Returns an httptest.ResponseRecorder
helpers.AssertJSONResponse(t, rr, expectedStatus, expectedBody)
```

A mock HTTP client replaces real network calls in feed-fetching tests (feed service's `SetHTTPClient`, injected with `helpers.NewMockHTTPClient`); production code is unaffected, since a nil client falls back to real HTTP. SSRF validation does not run against mock clients, since the mock server's URL is itself a test-only bypass of what would otherwise be blocked.

## Coverage Goals

Package-level coverage is not tracked as a fixed number in this doc, since it drifts with every test added or removed; a stale percentage is worse than none. Regenerate it directly:

```bash
# Generate overall coverage report
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | tail -1  # See total percentage

# Generate package-specific coverage
go test -coverprofile=coverage.out ./internal/...
go tool cover -html=coverage.out -o coverage.html

# View detailed function-level coverage
go tool cover -func=coverage.out

# Generate frontend coverage
npm run test:coverage

# View coverage in browser
open coverage.html                    # Backend HTML report
open web/coverage/index.html          # Frontend coverage
```

The Datastore-backed tests only run, and only count toward `internal/database` and `internal/services` coverage, with `DATASTORE_EMULATOR_HOST` set (see [Datastore Emulator](#datastore-emulator-internalservicesadmin_token_datastore_testgo) above); `make test-datastore` or `./scripts/with-datastore-emulator.sh go test -coverprofile=coverage.out ./internal/...` includes them.

## Test Data and Fixtures

### Sample Data (`test/fixtures/sample_feeds.go`)

Provides consistent test data:

```go
var SampleUsers = []User{
    {Email: "user1@example.com", Name: "Test User 1"},
    {Email: "user2@example.com", Name: "Test User 2"},
    {Email: "admin@example.com", Name: "Admin User", IsAdmin: true},
}

var SampleFeeds = []Feed{
    {Title: "Test Feed 1", URL: "https://example.com/feed1.xml"},
    {Title: "Test Feed 2", URL: "https://example.com/feed2.xml"},
}
```

### Feed Format Fixtures (`test/fixtures/feeds/`)

On-disk XML fixtures exercised by the table-driven suite in `internal/services/feed_fixtures_test.go` (`TestParseFeedFixtures`):

- `rss2_standard.xml`, `atom_standard.xml`, `rdf_standard.xml`: one representative document per supported format (RSS 2.0, Atom 1.0, RSS 1.0/RDF).
- `rss2_relative_urls.xml`: item `<link>` values are relative/scheme-relative paths. The parser copies `<link>` verbatim into `ArticleData.Link` with no `url.Parse`/`ResolveReference` step, so this fixture documents that pass-through behavior rather than testing normalization that doesn't exist.
- `rss2_missing_fields.xml`: empty channel title/description and items missing `<title>`/`<description>`, verifying the parser substitutes fallback titles per-item instead of erroring or skipping items.
- `malformed.xml`: unclosed tags; the parser must fail all three (RSS/RDF/Atom) unmarshal attempts and return an error rather than partial data.
- `rss2_large_item_count.xml`: 3,000 items, guarding against silent truncation. There is no per-item cap during parsing itself; the only feed-fetch DoS control is the 10MB `maxFeedBodySize` body-size cap enforced in `fetchFeed` before parsing (see `TestFetchFeed_SizeLimit`).

### Generating Test Articles

Use the `generate-test-articles` utility to create test data in a local SQLite database for testing pagination, feed loading, and article navigation:

```bash
# Generate test articles for a user and feed
go run cmd/generate-test-articles/main.go <user_id> <feed_id> <num_articles>

# Example: Create 150 test articles for user 91 in feed 1
go run cmd/generate-test-articles/main.go 91 1 150
```

**What it does:**
- Creates the specified number of test articles for a given feed
- Automatically subscribes the user to the feed if not already subscribed
- Marks all generated articles as unread for the specified user
- Generates realistic article data with timestamps, titles, content, and descriptions
- Each article has a unique URL and is timestamped 1 minute older than the previous

**Limits:**
- Between 1 and 1000 articles per run
- Only works with the local SQLite database (not Datastore)

**Example output:**
```
Generating 150 test articles for feed 1...
Found user: Test User (user91@example.com)
Found feed: Test Feed 1
Created 50/150 articles...
Created 100/150 articles...
Created 150/150 articles...
Successfully created 150 articles
Ensuring articles are unread for user 91...

Success. Created 150 test articles for feed Test Feed 1 (ID: 1).
All articles are unread and ready for testing.
```

## Security Testing

`test/security/` is a consolidated regression suite, run as its own blocking CI job separate from the main test job (see [CI/CD Integration](#cicd-integration) below):

- `auth_test.go`: every `RequireAuth` route rejects a request with no session cookie
- `csrf_test.go`: CSRF token enforcement on every mutating endpoint
- `feed_limit_test.go`: `POST /api/feeds` returns 402 once a trial user hits `FreeTrialFeedLimit`
- `ssrf_test.go`: `POST /api/feeds` rejects loopback, link-local, RFC 1918, and metadata-endpoint URLs

The admin token system has its own, more detailed coverage in `internal/services/admin_token_test.go` and `admin_token_datastore_test.go`: cryptographically secure generation (64-character hex, SHA-256 hashing), database-backed validation against both SQLite and Datastore, the full create/validate/list/revoke lifecycle, and bootstrap protection that refuses to create a token when no admin user exists yet.

## CI/CD Integration

### GitHub Actions (`.github/workflows/test.yml`)

The CI pipeline takes its Go release from `go.mod`: every `actions/setup-go` step uses `go-version-file: go.mod`, which reads the `toolchain` directive (for example `go1.26.9`). It runs on every push to `main` and every pull request targeting `main`, with six jobs:

```yaml
jobs:
  test:            # unit, devlogin-tagged, and integration tests, coverage upload to Codecov
  lint:            # golangci-lint, default and devlogin build tags
  frontend-build:  # npm ci + npm run lint:js + npm run test:ci + make build-frontend
  benchmark:       # go test -bench + benchstat vs cached main baseline; fails on >20% regression
  security:        # go test ./test/security/... (blocking regression gate) + govulncheck (continue-on-error: true, reports only)
  build:           # needs: [test, lint, frontend-build]; go build ./... + make build
```

**Pipeline features:**
- Go release pinned by the `toolchain` directive in `go.mod`. `setup-go` sets `GOTOOLCHAIN=local`, so the directive takes effect only because each step reads `go.mod` directly; a bare major version such as `go-version: '1.26'` can resolve to an older patch release cached on the runner
- Package-level unit tests, including the root `main` package (`go test -short -race -coverprofile=coverage.out . ./internal/...`)
- Dev login tests (`go test -race -tags devlogin -run DevLogin ./internal/handlers/...`), which the plain unit test run skips because `dev_login.go` is compiled only with the `devlogin` tag
- Integration tests (`./test/integration/...`)
- Coverage reporting to Codecov, authenticated by the `CODECOV_TOKEN` repository secret. The upload is non-blocking (`fail_ci_if_error: false`), so a missing or invalid token leaves CI green and must be diagnosed from the step's log
- Linting with golangci-lint, run a second time with `--build-tags devlogin` over `internal/handlers/`. The gofmt formatter is enabled in `.golangci.yml`, so a file that `gofmt -l` reports fails the lint step
- ESLint static analysis (`npm run lint:js`, flat config in `eslint.config.js`) against `web/static/js/*.js` (excluding `*.min.js`), catching undefined variables (`no-undef`) and unreachable code (`no-unreachable`) before the Jest step runs; browser/library globals (`window`, `DOMPurify`, `marked`, etc.) are declared explicitly since there's no `eslint-plugin-browser` env package installed
- Frontend tests (`npm run test:ci`, the same 149 Jest tests `make test` runs locally) followed by frontend build verification (`make build-frontend`); a broken Jest suite or broken JS/CSS build fails CI
- Benchmark regression gate (`benchmark` job): see [CI Benchmark Regression Gate](#ci-benchmark-regression-gate-scriptscheck-benchmark-regressionsh) above
- Security regression suite (`./test/security/...`): blocking gate covering CSRF enforcement, auth-bypass, SSRF protection on `POST /api/feeds`, and `FreeTrialFeedLimit` enforcement, all exercised through the real HTTP handlers
- `govulncheck` as a non-blocking reporting job, run in the same `security` job after the regression suite
- Single-platform build artifact (`goread2` binary): does not include darwin/windows builds, since deployment is GAE-only
- All actions are pinned to commit SHA (not floating tags like `@v4`) for supply-chain hardening, matching the deploy workflows; see the workflow file's inline `# vX` comments for the corresponding version

## Writing New Tests

A package-level unit test typically only needs the package's own types; an HTTP-level test typically needs `test/helpers` (see [Test Helpers](#test-helpers-testhelpers) above). This example shows the latter: a handler test that creates a user, signs them in, and asserts on the response.

```go
func TestGetFeeds_ReturnsSubscribedFeeds(t *testing.T) {
    db := helpers.CreateTestDB(t)
    user := helpers.CreateTestUser(t, db, "google-id-1", "user1@example.com", "Test User")
    feed := helpers.CreateTestFeed(t, db, "Test Feed", "https://example.com/feed.xml", "")
    if err := db.SubscribeUserToFeed(user.ID, feed.ID); err != nil {
        t.Fatalf("failed to subscribe: %v", err)
    }

    ts := helpers.SetupTestServerWithDB(t, db)
    req := ts.CreateAuthenticatedRequest(t, "GET", "/api/feeds", nil, user)
    rr := ts.ExecuteRequest(req)

    if rr.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
    }
}
```

Frontend tests follow the existing files in `web/tests/`: set up `document.body.innerHTML` and any needed globals (`global.fetch = jest.fn()`, etc.) in `beforeEach`, then assert on DOM state or mock calls after driving the app through its public methods.

## Test Performance

### In-Memory Database

Tests use in-memory SQLite with a shared cache rather than file-based databases:
```go
db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
```
This eliminates disk I/O and temporary file creation/deletion, and supports concurrent access via the shared cache.

### Mock HTTP Clients

Feed fetching tests inject a mock HTTP client instead of making real network calls (see [Test Helpers](#test-helpers-testhelpers) above for `NewMockFeedServer`/`NewMockHTTPClient`). This eliminates network latency and gives deterministic, controlled responses; production code is unaffected, since a nil client falls back to real HTTP.

## Performance Testing

### Benchmark Tests

Benchmarks for the three most-queried database operations live in
`internal/database/schema_bench_test.go`:

```bash
# Run all database benchmarks (3 seconds each, single run)
go test ./internal/database/ -bench=. -benchtime=3s -count=1

# Run a specific benchmark
go test ./internal/database/ -bench=BenchmarkGetUserArticlesPaginated -benchtime=5s

# Compare two revisions (requires benchstat)
go test ./internal/database/ -bench=. -count=5 | tee new.txt
benchstat old.txt new.txt
```

Benchmarks seeded with realistic data (5–10 feeds × 30–50 articles):

| Benchmark | What it measures |
|-----------|-----------------|
| `BenchmarkGetUserArticlesPaginatedFirstPage` | First-page query with no cursor |
| `BenchmarkGetUserArticlesPaginatedWithCursor` | Subsequent-page query using a real cursor |
| `BenchmarkGetUserArticlesPaginatedUnreadOnly` | Filtered query (unread articles only) |
| `BenchmarkGetUserUnreadCounts` | Per-feed unread counts for a user |
| `BenchmarkGetAccountStats` | Aggregated stats (total articles, unread, active feeds) |

The same file contains property-based tests (`TestCursorRoundTrip`) that verify the
cursor encode/decode round-trip holds for 1,000 randomly generated inputs via
`testing/quick`, and `TestDecodeCursorInvalidInputs` for malformed cursor rejection.

See [Performance Benchmarks](#performance-benchmarks-testintegrationperformance_testgo) above for the separate, timing-focused benchmarks in `test/integration/` and the CI regression gate that watches them.

## Debugging Tests

### Verbose Output

```bash
# Backend tests with verbose output
go test -v ./test/...

# Frontend tests with debug info
npm test -- --verbose

# Specific test with debugging
go test -v -run TestSpecificFunction ./test/integration/
```

### Debug Commands

```bash
# Run specific package tests with verbose output
go test -v ./internal/config/
go test -v ./test/integration/

# Run tests with race detection (or use: make test-race)
go test -race ./internal/... ./test/integration/...

# Frontend tests with coverage and debugging
npm test -- --coverage --verbose

# Check dependencies
go mod verify
go mod tidy
npm audit
```

## Troubleshooting

### Common Issues

**Package-Level Unit Test Issues:**
- Ensure dependencies are available: `go mod tidy`
- Check environment variables for config tests
- Verify test isolation and cleanup

**Integration Test Issues:**
- Check test database setup and permissions
- Verify environment variables are set
- Ensure test server starts correctly
- Check for port conflicts

**Frontend Test Issues:**
- Verify Jest and jsdom are installed: `npm install`
- Check DOM setup in test files
- Ensure fetch mocking is configured properly

See the [Troubleshooting Guide](troubleshooting.md) for build and deploy failures outside the test suite itself.

## Related Documentation

- [Security Guidelines](security.md) - Regression suite covered by the `security` CI job
- [Performance & Cost Optimization](performance.md) - Benchmark regression gate
- [CONTRIBUTING.md](../CONTRIBUTING.md) - Pull request requirements and test conventions
- [Troubleshooting Guide](troubleshooting.md) - Build and test failures
