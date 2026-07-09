# Testing Guide

Testing guide for GoRead2's multi-user RSS reader application.

## Current Status

✅ **All tests are passing successfully** - The testing infrastructure is fully functional with no interface compatibility issues.

## Overview

GoRead2's testing infrastructure includes:
- **Package-level unit tests** across ten `internal/` packages plus `cmd/admin`, with per-package coverage ranging from ~65% to ~85% (see [Test Coverage](#test-coverage) below); coverage drifts often enough that the numbers there are a snapshot, not a guarantee; regenerate with `go test ./internal/... -cover` for the current figures
- **Integration tests** for end-to-end API validation
- **Frontend tests** with Jest and jsdom (140 tests across 7 files)
- **CI/CD integration** with GitHub Actions

## Current Test Structure

```
internal/
├── auth/
│   ├── auth_test.go          # Authentication service tests
│   ├── client_ip_test.go     # Client IP extraction tests
│   ├── csrf_test.go          # CSRF token generation/validation tests
│   ├── middleware_test.go    # Authentication middleware tests
│   ├── rate_limiter_test.go  # Rate limiter unit tests
│   └── session_test.go       # Session management tests
│                              # (auth package: ~70% coverage)
├── cache/
│   ├── feed_list_cache_test.go  # Feed list cache tests
│   └── unread_cache_test.go     # Unread count cache tests
│                                 # (cache package: ~82% coverage)
├── config/
│   ├── config_test.go        # Configuration management tests
│   └── validation_test.go    # Config validation tests
│                              # (config package: ~85% coverage)
├── database/
│   ├── schema_test.go               # Core schema and CRUD tests
│   ├── schema_extended_test.go      # Pagination edge cases and ordering
│   ├── schema_errors_test.go        # Error paths and boundary conditions
│   ├── schema_user_article_test.go  # User-article status (read/star) tests
│   ├── pagination_duplicate_test.go # Duplicate-cursor pagination tests
│   ├── datastore_test.go            # DatastoreDB interface tests (emulator-gated)
│   ├── datastore_user_article_test.go # DatastoreDB user-article tests (emulator-gated)
│   └── schema_bench_test.go         # Benchmarks: BenchmarkGetUserArticlesPaginatedFirstPage,
│                                    #   ...WithCursor, ...UnreadOnly, BenchmarkGetUserUnreadCounts,
│                                    #   BenchmarkGetAccountStats + property tests for cursor encode/decode
├── handlers/
│   ├── admin_handler_test.go    # Admin handler request/error-path tests
│   ├── article_handler_test.go  # Article handler tests (GetArticle)
│   ├── auth_handler_test.go     # Auth handler constructor tests
│   ├── feed_handler_test.go     # Feed handler request/error-path tests (AddFeed, ImportOPML,
│   │                            #   GetArticles, RefreshFeeds, DebugAllSubscriptions, etc.)
│   └── payment_handler_test.go  # Payment handler tests incl. signed Stripe webhook payloads
│                                # (handlers package: ~79% coverage)
├── middleware/
│   ├── body_limit_test.go        # Request body size limit tests
│   ├── cors_test.go              # CORS middleware tests
│   ├── request_cache_test.go     # Request-scoped cache invalidation tests
│   └── security_headers_test.go  # Security header middleware tests
│                                  # (middleware package: ~84% coverage)
├── secrets/
│   └── secrets_test.go          # Secrets manager tests (~65% coverage)
├── services/
│   ├── admin_token_test.go               # SQLite admin token tests (comprehensive)
│   ├── admin_token_datastore_test.go     # Datastore admin token tests
│   ├── audit_service_test.go             # Audit logging tests
│   ├── edge_cases_test.go                # Cross-cutting edge-case tests
│   ├── feed_discovery_test.go            # Feed discovery and URL normalization tests
│   ├── feed_fixtures_test.go             # Contract/fixture tests for RSS 2.0, Atom, RDF feeds
│   ├── feed_scheduler_test.go            # Feed scheduler concurrency/stress tests
│   ├── feed_service_test.go              # Feed service core logic tests
│   ├── feed_service_coverage_test.go     # Additional feed service coverage tests
│   ├── payment_service_test.go           # Payment service logic tests
│   ├── rate_limiter_test.go              # Concurrency stress tests for the rate limiter
│   ├── subscription_service_test.go      # Subscription service logic tests
│   └── url_validator_test.go             # SSRF/URL validation tests
│                                          # (services package: ~67% coverage)
cmd/
└── admin/
    └── main_test.go      # Admin CLI command handler tests (~52% coverage):
                           #   set-admin, grant-months, user-info, list-users,
                           #   fix-subscription, set-subscription-id (Stripe backend
                           #   mocked via stripe.SetBackend), create-token, list/revoke
                           #   token, audit-logs; fatal (log.Fatal/os.Exit) error paths
                           #   verified via subprocess re-exec
test/
├── integration/                          # Backend integration tests
│   ├── admin_integration_test.go         # Admin command integration tests
│   ├── admin_security_test.go            # Admin security and bootstrap tests
│   ├── api_test.go                       # End-to-end API testing
│   ├── cache_test.go                     # Cache integration tests
│   ├── critical_workflows_test.go        # Critical user-workflow regression tests
│   ├── feature_flag_test.go              # SUBSCRIPTION_ENABLED toggle behavior tests
│   ├── main_test.go                      # Shared integration test setup
│   ├── subscription_integration_test.go  # Subscription lifecycle integration tests
│   └── workflow_test.go                  # User workflow integration tests
├── security/                       # Consolidated security regression suite (dedicated CI job, gr-rrt)
│   ├── auth_test.go                # Every RequireAuth route rejects requests with no session cookie
│   ├── csrf_test.go                # CSRF token enforcement on mutating endpoints (moved from test/integration/auth_csrf_test.go)
│   ├── feed_limit_test.go          # POST /api/feeds returns 402 once a trial user hits FreeTrialFeedLimit
│   └── ssrf_test.go                # POST /api/feeds rejects loopback/link-local/RFC1918/metadata URLs
└── fixtures/            # Test data and sample feeds
    └── sample_feeds.go  # Sample data for tests
web/tests/                     # Frontend tests (140 tests across 7 files)
├── accessibility.test.js      # ARIA/focus/keyboard accessibility assertions
├── account-app.test.js        # Account page app tests
├── app-core.test.js           # Core frontend functionality
├── error-handler.test.js      # Error handling and toast notifications
├── goread-app.test.js         # Main app class tests
├── integration.test.js        # Frontend integration tests
├── pagination.test.js         # Pagination and Load More functionality
├── utils.js                   # Frontend test utilities
├── setup.js                   # Test environment setup
└── README.md                  # Frontend testing documentation
```

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
- Code quality checks (linting)
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
go test ./internal/config/        # Configuration tests
go test ./internal/auth/          # Authentication tests  
go test ./internal/services/      # Service layer tests
go test ./internal/handlers/      # Handler constructor tests

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
make clean           # Remove build artifacts
make test-quick      # Run tests using Go cache, fast for dev iteration
make test            # Run full test suite with coverage (CI/pre-deploy)
make test-race       # Run Go tests with race detector (CI also runs this automatically)
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

#### Config Package (`internal/config/config_test.go`)

Tests configuration management with 85.2% coverage:

```go
func TestConfigLoad(t *testing.T) {
    // Test configuration loading with different environment scenarios
}

func TestConfigParseBool(t *testing.T) {
    // Test boolean parsing with various string formats
}

func TestParseEmailList(t *testing.T) {
    // Test email list parsing and validation
}
```

**Coverage includes:**
- Environment variable loading and defaults
- Boolean parsing (true/false, 1/0, yes/no, etc.)
- Email list parsing from comma-separated strings
- Configuration singleton behavior
- Input validation and sanitization

#### Auth Package (`internal/auth/`)

Tests authentication system with 69.6% coverage:

```go
func TestNewAuthService(t *testing.T) {
    // Test OAuth service initialization
}

func TestSessionManager(t *testing.T) {
    // Test session creation, retrieval, and cleanup
}

func TestRequireAuth(t *testing.T) {
    // Test authentication middleware (JSON API endpoints)
}

func TestOptionalAuth(t *testing.T) {
    // Test optional authentication middleware
}

func TestRequireAdmin(t *testing.T) {
    // Test admin privilege enforcement
}

func TestCSRFMiddleware(t *testing.T) {
    // Test CSRF protection middleware
}

func TestRateLimitMiddleware(t *testing.T) {
    // Test rate limiting middleware
}

func TestCSRFConcurrentGeneration(t *testing.T) {
    // Test concurrent CSRF token generation
}

func TestRateLimiterConcurrentAccess(t *testing.T) {
    // Test concurrent rate limiter access
}
```

**Coverage includes:**
- OAuth service configuration and validation
- Session management (create, get, delete, expiration)
- Cookie handling and HTTP request/response
- Context user extraction for Gin and standard contexts
- Admin user initialization from environment
- Session cleanup and security
- **Middleware error paths** (100% coverage):
  - RequireAuth with no session (JSON error)
  - RequireAuth with valid session
  - OptionalAuth with and without session
  - RequireAdmin with no session, non-admin user, and admin user
- **CSRF protection** (100% middleware coverage):
  - Safe methods bypass (GET, HEAD, OPTIONS)
  - POST without session returns unauthorized
  - POST without CSRF token returns forbidden
  - POST with invalid CSRF token returns forbidden
  - POST with valid CSRF token succeeds
  - Concurrent token generation safety
- **Rate limiting** (100% middleware coverage):
  - Requests within limit allowed
  - Requests exceeding limit blocked (429 status)
  - Independent limits per IP address
  - Concurrent access safety
  - Cleanup mechanism verification

#### Secrets Package (`internal/secrets/`)

Tests secrets management system with 64.6% coverage:

```go
func TestGetOAuthCredentials_FromEnvironment(t *testing.T) {
    // Test OAuth credential retrieval from environment variables
}

func TestGetStripeCredentials_FromEnvironment(t *testing.T) {
    // Test Stripe credential retrieval from environment variables
}

func TestGetSecret_MissingProjectID(t *testing.T) {
    // Test error handling when GOOGLE_CLOUD_PROJECT is missing
}
```

**Coverage includes:**
- **GetOAuthCredentials** (100% coverage):
  - Retrieval from environment variables
  - Secret reference detection (`_secret:` prefix)
  - Empty credential handling
  - GOOGLE_CLOUD_PROJECT validation
  - Custom secret name support
- **GetStripeCredentials** (100% coverage):
  - All four Stripe credentials (secret key, publishable key, webhook secret, price ID)
  - Placeholder value detection
  - Fallback to Secret Manager when needed
  - Environment variable priority
- **GetSecret** (23.1% coverage):
  - GOOGLE_CLOUD_PROJECT requirement validation
  - Error handling (requires Secret Manager API mocking for full coverage)
- **Security validation**:
  - Missing credential detection
  - Invalid configuration error messages
  - Environment-based fallback logic

#### Services Package (`internal/services/`)

Tests service layer logic with 67.2% coverage:

```go
func TestSubscriptionService(t *testing.T) {
    // Test feed limits, trial logic, admin privileges
}

func TestFeedDiscovery(t *testing.T) {
    // Test URL normalization and validation
}

func TestGenerateAdminToken(t *testing.T) {
    // Test cryptographic token generation
}

func TestValidateAdminToken(t *testing.T) {
    // Test database-based token validation
}

func TestListAdminTokens(t *testing.T) {
    // Test token listing and metadata
}

func TestRevokeAdminToken(t *testing.T) {
    // Test token revocation and lifecycle
}

func TestHasAdminTokens(t *testing.T) {
    // Test active token detection
}

func TestDatastoreAdminTokens(t *testing.T) {
    // Test Google Datastore compatibility
}
```

**Coverage includes:**
- **Admin Token Security System**: Comprehensive testing of the new secure admin authentication
  - Cryptographic token generation (64-char hex, SHA-256 hashing)
  - Database-based token validation (SQLite + Google Datastore)
  - Token lifecycle management (create, validate, list, revoke)
  - Bootstrap security protection requiring existing admin users
  - Token uniqueness and format validation
  - Last-used timestamp tracking and proper error handling
- Feed limit enforcement for trial users
- Subscription status validation
- Admin privilege checking
- URL normalization (protocol addition, validation)
- Service constructors and dependencies

#### Handlers Package (`internal/handlers/`)

Tests handler request and error paths with 79.3% coverage, using `httptest.ResponseRecorder`
and a mock `database.Database` per test file (`mockDBFeedHandler`, `mockDBAdminHandler`) rather
than a full integration setup:

```go
func TestAddFeed(t *testing.T) {
    // unauthenticated, invalid JSON, invalid/SSRF-blocked URL,
    // CanUserAddFeed DB error, trial expired, feed limit reached
}

func TestWebhookHandler_SubscriptionCreatedOrUpdated(t *testing.T) {
    // valid Stripe-signed payloads built with webhook.GenerateTestSignedPayload,
    // success, service error (500), and malformed payload (400) paths
}
```

**Coverage includes:**
- Handler constructor functions and service dependency injection
- Auth/validation error paths (401 unauthenticated, 400 malformed body/params)
- Feed handlers: `AddFeed`, `ImportOPML` (missing file, oversized file, malformed XML,
  subscription-limit errors), `GetArticles` (both the `all` and single-feed branches),
  `RefreshFeeds` (manual and cron paths), `DebugAllSubscriptions`, `DebugArticleByURL`
- Admin handlers: `SetAdminStatus` and `GrantFreeMonths` error paths (missing param,
  invalid body, unauthenticated, user not found, DB error), `GetAuditLogs` DB error
- Payment handlers: Stripe webhook signature verification, all handled event types
  (`checkout.session.completed`, `customer.subscription.created/updated/deleted`,
  unhandled event types), and the `subscription_success`/`subscription_cancel` HTML views
- OAuth login/callback flows (`Login`, `Callback`) remain untested at the unit level:
  they call out to Google's real OAuth endpoints and are exercised instead by
  `test/integration` and manual verification

### 2. Backend Integration Tests

#### API Endpoints (`test/integration/api_test.go`)

Tests end-to-end API functionality:

```go
func TestFeedAPIWithAuth(t *testing.T) {
    // Test authenticated feed operations
}

func TestUserIsolationInAPI(t *testing.T) {
    // Verify API enforces user data isolation
}
```

**Coverage includes:**
- Authentication requirements
- Feed CRUD operations
- Article operations (read/star)
- User isolation verification
- Error handling and status codes
- Full request/response cycle testing

#### User Workflow Tests (`test/integration/workflow_test.go`)

Tests complete end-to-end user workflows through the application:

```go
func TestUserWorkflowEndToEnd(t *testing.T) {
    // 11-step complete user journey:
    // 1. Create user (simulating registration)
    // 2. Verify no feeds initially
    // 3. Subscribe to feed
    // 4. Verify has 1 feed
    // 5. Add articles to feed
    // 6. Get user's articles
    // 7. Mark article as read
    // 8. Star article
    // 9. Verify article statuses updated
    // 10. Delete feed
    // 11. Verify feed gone
}

func TestOPMLImportWorkflow(t *testing.T) {
    // Test OPML import endpoint validation
}

func TestOPMLExportWorkflow(t *testing.T) {
    // Test OPML export and verification
    // Verifies exported OPML contains correct feed URLs
}

func TestAdminWorkflow(t *testing.T) {
    // Test admin operations:
    // - Grant admin privileges
    // - Verify admin status
    // - Revoke admin privileges
}
```

**Coverage includes:**
- Complete user registration → feeds → articles → actions flow
- OPML import/export functionality
- Admin privilege management
- Feed subscription lifecycle
- Article status updates (read/starred)
- Feed deletion and cleanup
- API endpoint authentication and CSRF protection

#### Performance Benchmarks (`test/integration/performance_test.go`)

`TestPerformanceBaseline` used to record timing via `t.Logf` with nothing checked against it: no pass/fail signal, just numbers in the log. It's been replaced with real `testing.B` benchmarks, run by a dedicated `benchmark` CI job rather than `go test`'s normal test run (see [CI regression gate](#ci-benchmark-regression-gate-scriptscheck-benchmark-regressionsh) below):

```go
func BenchmarkGetUserFeeds100(b *testing.B)           // Query all feeds for a user subscribed to 100
func BenchmarkGetUserArticlesPaginated(b *testing.B)  // Paginated query (50/page) against 1000 articles
func BenchmarkGetUserUnreadCounts(b *testing.B)       // Per-feed unread counts across 100 feeds
func BenchmarkConcurrentReads(b *testing.B)           // 10 users concurrently reading a shared feed
```

Fixture setup (creating the feeds/articles/users) happens once per `-count` repetition via `b.ResetTimer()`, so only the operation named is measured, not the setup cost. `TestConcurrentUserOperations` (correctness of concurrent reads *and* writes under `-race`) is unchanged, since it's a concurrency-safety test, not a timing one.

Run locally:

```bash
go test -run='^$' -bench=. -benchmem ./test/integration/...
```

#### CI Benchmark Regression Gate (`scripts/check-benchmark-regression.sh`)

The `benchmark` job in `test.yml` runs the four benchmarks above with `-benchtime=50x -count=10` and compares them against a baseline using [`benchstat`](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat). No baseline file is committed to the repo: raw timings are hardware-dependent, so a baseline is only meaningful compared against a run on the *same* runner type it was recorded on. Instead:

- The baseline is the most recent successful main-branch run's benchmark output, persisted via `actions/cache` under a rolling key (`bench-baseline-<run-id>`, restored by the `bench-baseline-` prefix).
- Every push to `main` whose benchmarks *don't* regress rolls the cache forward to that run's numbers, so the baseline tracks main over time instead of drifting stale.
- `scripts/check-benchmark-regression.sh <baseline> <current> [threshold-pct]` (default threshold 20%) parses `benchstat -format csv` output and fails only on sec/op deltas that are both `>threshold%` *and* statistically significant per benchstat's own test; a change benchstat marks `~` is noise and is ignored regardless of the raw percentage. `B/op`/`allocs/op` deltas are printed but not gated on.
- A regression fails the `benchmark` job, which fails the whole `test.yml` workflow, which blocks `deploy-staging.yml` (gated on `workflow_run.conclusion == 'success'`, the same mechanism as every other required job; see gr-vu1d's neighbor jobs). On a pull request, a regression additionally posts the benchstat report as a PR comment via `gh pr comment`.
- The very first run after this job shipped has no cache to restore, so it skips the comparison and just bootstraps the baseline.

Part of epic gr-f6v (gr-4o2f).

### 3. Frontend Tests

#### Error Handling and Toast Notifications (`web/tests/error-handler.test.js`)

Tests error handling system and user notifications with 18 comprehensive tests:

```javascript
describe('Error Handling and Toast Notifications', () => {
  test('displays connection indicator for online/offline states', () => {
    // Test connection status indicator UI
  });

  test('shows error messages with appropriate types and icons', () => {
    // Test error display (network, auth, validation, server, unknown)
  });

  test('handles toast notifications with auto-dismiss', () => {
    // Test toast creation and lifecycle
  });
});
```

**Coverage includes:**
- **Connection Indicator UI**: Online/offline state display and transitions
- **Error Message Display**:
  - 5 error types with type-specific icons (📡 network, 🔒 auth, ⚠️ validation, 🔧 server, ❌ unknown)
  - Retry and dismiss button functionality
  - Error message replacement logic
  - User-friendly error messages for each type
- **Toast Notifications**:
  - 4 toast types with icons (ℹ️ info, ✓ success, ⚠️ warning, ✕ error)
  - Toast container management
  - Multiple concurrent toasts support
  - Animation classes and auto-removal
- **Error Classification**: HTTP status code mapping (401/403 → auth, 4xx → validation, 5xx → server)
- **Button Interactions**: Click handlers for retry and dismiss actions
- **DOM Structure**: Proper HTML element creation and class management

#### Pagination and Load More (`web/tests/pagination.test.js`)

Tests cursor-based pagination and Load More button with 18 comprehensive tests:

```javascript
describe('Pagination Functionality', () => {
  test('creates Load More button when more articles available', () => {
    // Test Load More button creation and styling
  });

  test('handles button click and loading states', () => {
    // Test button state transitions (normal → loading → restored)
  });

  test('manages cursor-based pagination correctly', () => {
    // Test next_cursor handling and page requests
  });
});
```

**Coverage includes:**
- **Load More Button**: Creation, removal, styling, and state management
- **Button States**: Loading text, disabled state, error recovery
- **Article Rendering**:
  - Empty state placeholder
  - Multiple article rendering
  - Appending new articles on load more
  - Article index assignment and preservation
- **Pagination State**: hasMoreArticles flag tracking based on cursor
- **Cursor Logic**: Using next_cursor for subsequent page requests
- **Feed-Specific Behavior**: Load More only appears for 'all' feed view
- **Error Handling**: Button state restoration on loading errors
- **Index Management**: Sequential article indexing across multiple pages

#### Core Functionality (`web/tests/app-core.test.js`)

Tests frontend application logic with 28 comprehensive tests:

```javascript
describe('GoReadApp', () => {
  test('initializes with correct state', () => {
    // Test app initialization
  });

  test('handles authentication flow', () => {
    // Test login/logout functionality
  });

  test('manages feeds and articles', () => {
    // Test CRUD operations
  });
});
```

**Coverage includes:**
- DOM manipulation and rendering
- Event handling and user interactions
- API mocking and error scenarios
- Form validation and submission
- UI state management
- Utility function validation
- Modal and dialog interactions
- Keyboard navigation
- Error handling and display


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

Tests use in-memory SQLite databases for faster execution:

```go
// Tests use in-memory databases with shared cache for concurrent access
func setupTestDB(t *testing.T) *DB {
    // Creates in-memory database with shared cache
    db, err := sql.Open("sqlite3", "file::memory:?cache=shared")

    // Enable foreign key constraints
    _, err = db.Exec("PRAGMA foreign_keys = ON")

    // Each test gets clean database state
    // In-memory eliminates disk I/O overhead (2-3x faster)
}
```

**Performance Benefits:**
- **2-3x faster** database operations by eliminating disk I/O
- **Shared cache mode** (`?cache=shared`) enables concurrent access
- **Automatic cleanup** when database connection closes
- **Consistent behavior** with production SQLite schema

### Datastore Emulator (`internal/services/admin_token_datastore_test.go`)

Tests gated on `DATASTORE_EMULATOR_HOST` exercise `DatastoreDB` (the production database implementation) against a real (local) Datastore emulator instead of mocks. They `t.Skip` when the env var is unset, so they run silently skipped in a plain local `make test`/`go test ./...`. The same gating is used by `internal/database/datastore_test.go` and `internal/database/datastore_user_article_test.go`, which exercise `DatastoreDB` through the `database.Database` interface (see below).

**In CI** (`.github/workflows/test.yml`, `test` job): a Cloud Datastore emulator is started before the unit test step:
1. `actions/setup-java` installs a Java 21+ JRE (the emulator is a Java process; the runner's default `java` isn't guaranteed to meet the minimum version).
2. `google-github-actions/setup-gcloud` provisions a standalone gcloud SDK with the `beta` and `cloud-datastore-emulator` components (the runner's preinstalled `gcloud` is apt-managed and refuses `gcloud components install`).
3. `gcloud beta emulators datastore start --host-port=localhost:8081 --no-store-on-disk --consistency=1.0` runs in the background; the step polls `http://localhost:8081/` until it responds before continuing.
4. `DATASTORE_EMULATOR_HOST=localhost:8081` is exported for the rest of the job, so `go test ./internal/...` picks up the gated tests instead of skipping them.

**`--consistency=1.0` is required, not cosmetic**: the emulator defaults to simulating Datastore's eventual consistency (~0.9), which made `GetAll` queries in `ListAdminTokens`/`TestDatastoreAdminTokenCompatibility` intermittently miss entities written moments earlier: a real flake, not a fluke, reproduced locally before this was pinned to full consistency.

**Local setup** (to run these tests outside CI):
```bash
gcloud components install beta cloud-datastore-emulator
gcloud beta emulators datastore start --host-port=localhost:8081 --no-store-on-disk --consistency=1.0 &
DATASTORE_EMULATOR_HOST=localhost:8081 go test ./internal/services/... -run TestDatastore
DATASTORE_EMULATOR_HOST=localhost:8081 go test ./internal/database/... -run TestDatastore
```
The emulator's Java process requires a Java 21+ JRE on `PATH`; on macOS with Homebrew's `openjdk` installed but not linked, prefix the `gcloud emulators` command with `PATH="$(brew --prefix openjdk)/bin:$PATH"`.

`AdminToken`/`User` entity operations (via `SubscriptionService`, using the raw `*datastore.Client` from `DatastoreDB.GetClient()`) are covered in `internal/services/admin_token_datastore_test.go`. `DatastoreDB`'s own methods implementing the `database.Database` interface (`AddFeed`, `GetUserFeedArticles`, `MarkUserArticleRead`, `ToggleUserArticleStar`, `GetUserUnreadCounts`, `CreateUser`, `GetAccountStats`, `CleanupOrphanedUserArticles`, sessions, audit logs, and the rest) are covered directly (through the `Database` interface, mirroring the SQLite `schema_test.go` suite) in `internal/database/datastore_test.go` and `internal/database/datastore_user_article_test.go`, bringing `internal/database` to 80%+ coverage when run against the emulator.

### HTTP Test Setup

Integration tests use test servers and mock HTTP clients:

```go
// Integration tests setup full HTTP stack
func setupTestServer(t *testing.T) *httptest.Server {
    // Creates test server with middleware
    // Includes authentication and session handling
}

// Mock HTTP servers for feed fetching tests
func TestAddFeedWithMockHTTP(t *testing.T) {
    // Create mock RSS feed server
    mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/xml")
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte(sampleRSS))
    }))
    defer mockServer.Close()

    // Inject mock HTTP client into FeedService
    fs.SetHTTPClient(&mockHTTPClient{Server: mockServer})

    // Test feed operations without real network calls
    feed, err := fs.AddFeed(mockServer.URL)
}
```

**Mock HTTP Infrastructure:**
- **HTTPClient interface** for dependency injection
- **Mock feed servers** using `httptest.NewServer`
- **5-10x faster** feed tests by eliminating network latency
- **Deterministic testing** with controlled RSS/Atom responses
- **No SSRF validation** when using mock clients (test-only bypass)

## Coverage Goals

Current test coverage status and targets:

### ✅ Achieved Coverage
- **Overall project**: Coverage across all packages with significant improvements
- **Config package**: 85.2% coverage (comprehensive unit tests)
- **Auth package**: 69.6% coverage (session, middleware, CSRF, rate limiting, OAuth service)
  - **Middleware**: 100% coverage for RequireAuth, OptionalAuth, RequireAdmin
  - **CSRF Manager**: 100% coverage for CSRFMiddleware, concurrent token generation
  - **Rate Limiter**: 100% coverage for RateLimitMiddleware, concurrent access, cleanup
- **Secrets package**: 64.6% coverage (OAuth and Stripe credential management)
  - **GetOAuthCredentials**: 100% coverage for environment variable handling
  - **GetStripeCredentials**: 100% coverage for all Stripe credentials
  - **GetSecret**: 23.1% coverage (environment validation, requires Secret Manager API mocking for full coverage)
- **Services package**: 67.2% coverage (subscription logic, feed discovery, feed scheduling, **comprehensive admin token security**)
- **Handlers package**: 79.3% coverage (request/error-path tests for feed, admin, and payment
  handlers; OAuth `Login`/`Callback` remain unit-untested; see Handlers Package section above)
- **Integration tests**: Full end-to-end API validation with user isolation testing, plus admin security testing
- **Frontend**: 140 tests across 7 files covering core functionality, error handling, accessibility, pagination, and account-page behavior
  - **Core frontend tests**: 28 tests for DOM manipulation, events, forms, utilities
  - **Error handler tests**: 18 tests for connection monitoring, error display, toast notifications
  - **Pagination tests**: 18 tests for Load More button, cursor-based pagination, article rendering
- **Admin Token System**: Comprehensive test coverage for the new secure authentication system
  - 6 SQLite backend test suites with 20+ individual test cases
  - 6 Datastore backend test suites, run for real against a Cloud Datastore emulator in CI (see [Datastore Emulator](#datastore-emulator-internalservicesadmin_token_datastore_testgo) above); skip locally unless `DATASTORE_EMULATOR_HOST` is set
  - Security integration tests for bootstrap protection and token lifecycle
  - Edge case and error handling tests
- **Overall system**: All core tests passing successfully
- **DatastoreDB interface coverage**: `internal/database/datastore_test.go` and `internal/database/datastore_user_article_test.go` exercise `DatastoreDB`'s ~30 `database.Database` interface methods (feeds, articles, users, subscriptions/admin, sessions, audit logs) against the CI emulator, mirroring the SQLite `schema_test.go` suite; `internal/database` reaches 80%+ coverage when run with `DATASTORE_EMULATOR_HOST` set (see [Datastore Emulator](#datastore-emulator-internalservicesadmin_token_datastore_testgo) above)

### 🎯 Future Coverage Targets
- **Feed service operations**: Target 60%+ coverage (HTTP dependency mocking needed)
- **Handler HTTP logic**: Target 50%+ coverage (requires Gin test setup)  
- **Payment service**: Target 40%+ coverage (Stripe API mocking needed)
- **Overall project**: Target 80%+ coverage (significant infrastructure needed)

### Viewing Coverage

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

### Test Feed Examples

RSS and Atom feed examples for parser testing:

```xml
<!-- RSS 2.0 Sample -->
<rss version="2.0">
  <channel>
    <title>Test RSS Feed</title>
    <item>
      <title>Test Article</title>
      <description>Test content</description>
    </item>
  </channel>
</rss>

<!-- Atom Sample -->
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Test Atom Feed</title>
  <entry>
    <title>Test Article</title>
    <content>Test content</content>
  </entry>
</feed>
```

### Feed Format Fixtures (`test/fixtures/feeds/`)

On-disk XML fixtures (as opposed to the inline string constants above) exercised by the table-driven suite in `internal/services/feed_fixtures_test.go` (`TestParseFeedFixtures`):

- `rss2_standard.xml`, `atom_standard.xml`, `rdf_standard.xml`: one representative document per supported format (RSS 2.0, Atom 1.0, RSS 1.0/RDF).
- `rss2_relative_urls.xml`: item `<link>` values are relative/scheme-relative paths. The parser copies `<link>` verbatim into `ArticleData.Link` with no `url.Parse`/`ResolveReference` step, so this fixture documents that pass-through behavior rather than testing normalization that doesn't exist.
- `rss2_missing_fields.xml`: empty channel title/description and items missing `<title>`/`<description>`, verifying the parser substitutes fallback titles per-item instead of erroring or skipping items.
- `malformed.xml`: unclosed tags; the parser must fail all three (RSS/RDF/Atom) unmarshal attempts and return an error rather than partial data.
- `rss2_large_item_count.xml`: 3,000 items, guarding against silent truncation. There is no per-item cap during parsing itself; the only feed-fetch DoS control is the 10MB `maxFeedBodySize` body-size cap enforced in `fetchFeed` before parsing (see `TestFetchFeed_SizeLimit`).

### Generating Test Articles

For testing pagination, feed loading, and article navigation, use the `generate-test-articles` utility to create test data in your local SQLite database:

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
- Minimum: 1 article
- Maximum: 1000 articles per run
- Only works with local SQLite database (not Datastore)

**Use cases:**
- Testing pagination with >50 articles (Load More button)
- Testing article navigation and scrolling behavior
- Testing unread count display and updates
- Testing bulk mark-as-read operations
- Verifying performance with large article lists

**Example output:**
```
Generating 150 test articles for feed 1...
Found user: Jeffrey Pratt (jeffreyp07@gmail.com)
Found feed: Test Feed 1
Created 50/150 articles...
Created 100/150 articles...
Created 150/150 articles...
Successfully created 150 articles
Ensuring articles are unread for user 91...

✅ Success!
   Created: 150 test articles
   Feed: Test Feed 1 (ID: 1)
   User: Jeffrey Pratt (jeffreyp07@gmail.com)
   All articles are unread and ready for testing.
```

## Security Testing

Critical tests for multi-user security and admin authentication:

### User Data Isolation

```go
func TestUserDataIsolation(t *testing.T) {
    // Create two users
    user1 := createTestUser(t, "user1@example.com")
    user2 := createTestUser(t, "user2@example.com")
    
    // User1 subscribes to feed
    feed := createTestFeed(t)
    subscribeUserToFeed(t, user1.ID, feed.ID)
    
    // Verify user2 cannot see user1's feed
    user2Feeds := getUserFeeds(t, user2.ID)
    assert.Empty(t, user2Feeds)
    
    // Verify user1 can see their feed
    user1Feeds := getUserFeeds(t, user1.ID)
    assert.Len(t, user1Feeds, 1)
}
```

### Authentication Requirements

```go
func TestAPIRequiresAuthentication(t *testing.T) {
    // Test all protected endpoints require authentication
    endpoints := []string{
        "/api/feeds",
        "/api/feeds/1/articles",
        "/api/articles/1/read",
    }
    
    for _, endpoint := range endpoints {
        resp := makeUnauthenticatedRequest(t, endpoint)
        assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
    }
}
```

### Admin Token Security Testing

Critical tests for the new secure admin authentication system:

```go
func TestAdminTokenGeneration(t *testing.T) {
    // Test cryptographically secure token generation
    service := NewSubscriptionService(db)
    token, err := service.GenerateAdminToken("Test token")
    
    // Verify 64-character hex format
    assert.Len(t, token, 64)
    assert.Regexp(t, `^[0-9a-fA-F]{64}$`, token)
}

func TestAdminTokenValidation(t *testing.T) {
    // Test database-based token validation
    service := NewSubscriptionService(db)
    token, _ := service.GenerateAdminToken("Test token")
    
    // Valid token should pass
    valid, err := service.ValidateAdminToken(token)
    assert.NoError(t, err)
    assert.True(t, valid)
    
    // Invalid token should fail
    valid, err = service.ValidateAdminToken("invalid")
    assert.NoError(t, err)
    assert.False(t, valid)
}

func TestBootstrapProtection(t *testing.T) {
    // Test bootstrap security requiring existing admin users
    cleanDB := setupEmptyTestDB(t)
    
    // Should fail without admin users
    cmd := exec.Command("go", "run", "cmd/admin/main.go", "create-token", "test")
    cmd.Env = append(os.Environ(), "ADMIN_TOKEN=bootstrap")
    output, err := cmd.CombinedOutput()
    
    assert.Error(t, err)
    assert.Contains(t, string(output), "No admin users found in database")
}

func TestTokenLifecycle(t *testing.T) {
    // Test complete token lifecycle (create, validate, revoke)
    service := NewSubscriptionService(db)
    
    // Create token
    token, err := service.GenerateAdminToken("Lifecycle test")
    assert.NoError(t, err)
    
    // Validate token works
    valid, err := service.ValidateAdminToken(token)
    assert.True(t, valid)
    
    // Get token ID and revoke
    tokens, _ := service.ListAdminTokens()
    tokenID := tokens[0].ID
    err = service.RevokeAdminToken(tokenID)
    assert.NoError(t, err)
    
    // Token should no longer work
    valid, err = service.ValidateAdminToken(token)
    assert.False(t, valid)
}
```

**Admin Security Test Coverage:**
- **Cryptographic Security**: 64-character hex tokens with SHA-256 hashing
- **Bootstrap Protection**: Prevents unauthorized token creation without existing admin users  
- **Database Validation**: All tokens validated against database, not environment variables
- **Dual Database Support**: Tests work with both SQLite (local) and Google Datastore (GAE)
- **Token Lifecycle**: Complete create → validate → list → revoke → invalidate cycle
- **Edge Cases**: Invalid formats, non-existent tokens, already-revoked tokens
- **Security Warnings**: Prompts when creating additional tokens

## CI/CD Integration

### GitHub Actions (`.github/workflows/test.yml`)

The CI pipeline pins Go via a single `GO_VERSION` env var (currently `1.25`, matching `go.mod`) and runs six jobs:

```yaml
name: Tests
on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

env:
  GO_VERSION: '1.25'

jobs:
  test:            # unit + integration tests, coverage upload to Codecov
  lint:            # golangci-lint
  frontend-build:  # npm ci + npm run lint:js + npm run test:ci + make build-frontend
  benchmark:       # go test -bench + benchstat vs cached main baseline; fails on >20% regression
  security:        # go test ./test/security/... (blocking regression gate) + govulncheck (continue-on-error: true, reports only)
  build:           # needs: [test, lint, frontend-build]; go build ./... + make build
```

**Pipeline features:**
- Go 1.25 testing, pinned to match `go.mod` (bumped from 1.24 to clear govulncheck findings GO-2026-5039/GO-2026-5037, both fixed in go1.25.11; see gr-5ar0)
- Package-level unit tests (`go test -short -race -coverprofile=coverage.out ./internal/...`)
- Integration tests (`./test/integration/...`)
- Coverage reporting to Codecov
- Linting with golangci-lint
- ESLint static analysis (`npm run lint:js`, flat config in `eslint.config.js`) against `web/static/js/*.js` (excluding `*.min.js`), catching undefined variables (`no-undef`) and unreachable code (`no-unreachable`) before the Jest step runs; browser/library globals (`window`, `DOMPurify`, `marked`, etc.) are declared explicitly since there's no `eslint-plugin-browser` env package installed (gr-il9c)
- Frontend tests (`npm run test:ci`, the same 140 Jest tests `make test` runs locally) followed by frontend build verification (`make build-frontend`); a broken Jest suite or broken JS/CSS build now fails CI instead of shipping silently (gr-v9ki)
- Benchmark regression gate (`benchmark` job): see [CI Benchmark Regression Gate](#ci-benchmark-regression-gate-scriptscheck-benchmark-regressionsh) above (gr-4o2f)
- Security regression suite (`./test/security/...`): blocking gate covering CSRF enforcement, auth-bypass (every `RequireAuth` route rejects a request with no session cookie), SSRF protection on `POST /api/feeds`, and `FreeTrialFeedLimit` enforcement, all exercised through the real HTTP handlers rather than scattered across `test/integration` and package-level unit tests with no dedicated CI signal (gr-rrt)
- `govulncheck` as a non-blocking reporting job, run in the same `security` job after the regression suite
- Single-platform build artifact (`goread2` binary): dropped darwin/windows builds, since deployment is GAE-only and those artifacts served no purpose
- All actions are pinned to commit SHA (not floating tags like `@v4`) per supply-chain hardening feedback from a security review, matching the deploy workflows; see the workflow file's inline `# vX` comments for the corresponding version (gr-3ls6)

### Post-Deploy Smoke Check (`scripts/smoke-check.sh`)

Separate from the `test.yml` CI pipeline above, `scripts/smoke-check.sh <base-url>` runs unauthenticated HTTP assertions against a live, already-deployed App Engine version. It verifies the deploy itself (app started, static assets built, OAuth config loaded, security headers present, no backdoor auth endpoint), not application logic. Called by both `deploy-staging.yml` (against the new `staging-<sha>` URL) and `deploy-prod.yml` (against `https://goreadapp.com` after promotion); see [deployment.md](deployment.md#post-deploy-smoke-check-scriptssmoke-checksh) for the full assertion list.

## Writing New Tests

### Package-Level Unit Test Example

```go
func TestNewConfigFeature(t *testing.T) {
    // Setup - create test environment
    os.Setenv("TEST_VAR", "test_value")
    defer os.Unsetenv("TEST_VAR")
    
    // Test
    config := LoadConfig()
    result := config.GetTestValue()
    
    // Assert
    if result != "test_value" {
        t.Errorf("expected 'test_value', got %s", result)
    }
}
```

### Frontend Test Example

```javascript
describe('New Feature', () => {
  beforeEach(() => {
    // Setup DOM and mocks
    document.body.innerHTML = '<div id="app"></div>';
    global.fetch = jest.fn();
  });

  test('handles user interaction', async () => {
    // Setup
    const app = new GoReadApp();
    
    // Simulate user action
    const button = document.querySelector('#test-button');
    button.click();
    
    // Assert
    expect(fetch).toHaveBeenCalledWith('/api/test');
  });
});
```

### Integration Test Example

```go
func TestAPIIntegration(t *testing.T) {
    // Setup test server and database
    db := setupTestDB(t)
    server := setupTestServer(t, db)
    defer server.Close()
    
    // Create test user and session
    user := createTestUser(t, db, "test@example.com")
    session := createAuthenticatedSession(t, user)
    
    // Make authenticated API request
    resp := makeAuthenticatedRequest(t, server, session, "GET", "/api/feeds")
    
    // Assert response
    if resp.StatusCode != http.StatusOK {
        t.Errorf("expected status 200, got %d", resp.StatusCode)
    }
    
    var feeds []Feed
    json.NewDecoder(resp.Body).Decode(&feeds)
    // Verify feed data structure
}
```

## Test Performance Optimizations

GoRead2's test suite has been optimized for faster execution:

### In-Memory Database (2-3x Faster)

**Before:** File-based SQLite databases with temporary files
```go
tmpFile := fmt.Sprintf("/tmp/goread2_test_%d.db", time.Now().UnixNano())
db, err := sql.Open("sqlite3", tmpFile)
// Cleanup: os.Remove(tmpFile)
```

**After:** In-memory SQLite with shared cache
```go
db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
// No file I/O overhead, automatic cleanup
```

**Performance Impact:**
- ✅ Eliminates disk I/O completely
- ✅ No temporary file creation/deletion
- ✅ 2-3x faster database operations
- ✅ Supports concurrent access via shared cache

### Mock HTTP Clients (5-10x Faster)

**Before:** Real HTTP calls or no feed fetching tests

**After:** Mock HTTP servers with dependency injection
```go
// Define HTTP client interface
type HTTPClient interface {
    Do(req *http.Request) (*http.Response, error)
}

// Inject mock client in tests
mockServer := httptest.NewServer(...)
fs.SetHTTPClient(&mockHTTPClient{Server: mockServer})
```

**Performance Impact:**
- ✅ Eliminates network latency (0.3s+ per fetch)
- ✅ Deterministic, controlled responses
- ✅ 5-10x faster network-dependent tests
- ✅ Production code unchanged (nil client = real HTTP)

### Test Helper Improvements

New test helpers in `test/helpers/`:
```go
// HTTP mocking
NewMockFeedServer(t, feedXML)                    // Single feed server
NewMockFeedServerWithStatus(t, statusCode, body) // Custom status
NewMockMultiFeedServer(t, map[string]string)     // Multiple feeds
NewMockHTTPClient(server)                        // HTTP client wrapper

// Database setup
CreateTestDB(t)              // In-memory SQLite database
CreateTestUser(t, db, ...)   // Test user creation
CreateTestFeed(t, db, ...)   // Test feed creation
```

### Performance Results

Total test execution time improvements:
- **Database tests:** ~40% faster (file I/O eliminated)
- **Feed service tests:** ~80% faster (network calls mocked)
- **Integration tests:** ~75% faster (combined optimizations)

**Overall:** Test suite runs ~80% faster (75s → 15s)

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

### Load Testing

```bash
# Install artillery for load testing
npm install -g artillery

# Run load test
artillery quick --count 10 --num 50 http://localhost:8080
```

## Debugging Tests

### Verbose Output

```bash
# Backend tests with verbose output
go test -v ./test/...

# Frontend tests with debug info
npm test -- --verbose

# Specific test with debugging
go test -v -run TestSpecificFunction ./test/unit/
```

### Test Debugging Tools

```go
// Add debugging to tests
func TestWithDebugging(t *testing.T) {
    if testing.Verbose() {
        log.SetOutput(os.Stdout)
    }
    
    // Test code with log.Printf() statements
}
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

**Interface Mismatch Issues:**
- Cannot add unit tests for packages with interface inconsistencies
- Need to reconcile pointer vs value type mismatches
- Method signatures must match between interface and implementations

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

## Best Practices

### Test Organization

- **Arrange-Act-Assert**: Structure tests clearly
- **Single responsibility**: One test per function/behavior
- **Descriptive names**: Test names should explain what they verify
- **Independent tests**: Tests shouldn't depend on each other

### Test Data

- **Use fixtures**: Consistent test data across tests
- **Clean slate**: Each test starts with fresh data
- **Realistic data**: Test data should mirror production scenarios
- **Edge cases**: Test boundary conditions and error states

### Security Testing

- **User isolation**: Always test multi-user scenarios
- **Authentication**: Verify all protected endpoints
- **Input validation**: Test malformed inputs
- **Session security**: Test session creation and expiration

### Performance

- **Fast tests**: Keep unit tests under 100ms
- **Parallel execution**: Use `t.Parallel()` where safe
- **Minimal setup**: Only create what's needed for each test
- **Proper cleanup**: Always clean up resources

## Contributing

When adding new features:

1. **Write package-level tests** for new functionality
2. **Add integration tests** for new API endpoints  
3. **Ensure interface consistency** before adding unit tests
4. **Maintain current coverage** (don't decrease existing coverage)
5. **Follow Go testing conventions** (tests in same package as code)

### Pull Request Requirements

- All tests must pass: `make test` (full coverage run) or `./test.sh`
- Use `make test-quick` during development for fast cached feedback
- Package-level coverage should not decrease
- Include integration tests for API changes
- Update documentation for user-facing changes
- Follow existing test patterns and conventions

### Future Testing Roadmap

1. **Fix interface mismatches** to enable more unit tests
2. **Add package-level tests** for database, auth, and admin packages
3. **Achieve target coverage goals** (80-90% for new packages)
4. **Maintain security testing** for multi-user isolation

The testing infrastructure follows Go conventions with package-level tests co-located with source code, comprehensive integration testing, and frontend validation to ensure GoRead2 maintains quality and reliability.
