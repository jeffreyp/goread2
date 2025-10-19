# Testing Guide

Testing guide for GoRead2's multi-user RSS reader application.

## Current Status

✅ **All tests are passing successfully** - The testing infrastructure is fully functional with no interface compatibility issues.

## Overview

GoRead2's testing infrastructure includes:
- **Package-level unit tests** with 8.0% overall coverage across multiple packages
- **Integration tests** for end-to-end API validation
- **Frontend tests** with Jest and jsdom (26+ tests)
- **CI/CD integration** with GitHub Actions

## Current Test Structure

```
internal/
├── auth/
│   ├── auth_test.go         # Authentication service tests
│   ├── middleware_test.go   # Authentication middleware tests
│   └── session_test.go      # Session management tests (59.7% coverage)
├── config/
│   └── config_test.go       # Configuration management tests (96.7% coverage)
├── handlers/
│   ├── admin_handler_test.go    # Admin handler constructor tests
│   ├── auth_handler_test.go     # Auth handler constructor tests  
│   ├── feed_handler_test.go     # Feed handler constructor tests
│   └── payment_handler_test.go  # Payment handler constructor tests (1.0% coverage)
├── services/
│   ├── admin_token_test.go               # SQLite admin token tests (comprehensive)
│   ├── admin_token_datastore_test.go     # Datastore admin token tests 
│   ├── feed_discovery_test.go            # Feed discovery and URL normalization tests
│   └── subscription_service_test.go      # Subscription service logic tests (20.1% coverage)
test/
├── integration/                    # Backend integration tests
│   ├── admin_integration_test.go   # Admin command integration tests
│   ├── admin_security_test.go      # Admin security and bootstrap tests
│   └── api_test.go                 # End-to-end API testing
└── fixtures/            # Test data and sample feeds
    └── sample_feeds.go  # Sample data for tests
web/tests/               # Frontend tests
├── app-core.test.js     # Core frontend functionality (26+ tests)
├── font-choice.test.js  # Font choice feature tests (comprehensive)
├── utils.js             # Frontend test utilities
├── setup.js             # Test environment setup
└── README.md            # Frontend testing documentation
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
# Run complete build with frontend assets, build, and tests
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

Tests configuration management with 96.7% coverage:

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

Tests authentication system with 59.7% coverage:

```go
func TestNewAuthService(t *testing.T) {
    // Test OAuth service initialization
}

func TestSessionManager(t *testing.T) {
    // Test session creation, retrieval, and cleanup
}

func TestMiddleware(t *testing.T) {
    // Test authentication middleware functions
}
```

**Coverage includes:**
- OAuth service configuration and validation
- Session management (create, get, delete, expiration)
- Cookie handling and HTTP request/response
- Context user extraction for Gin and standard contexts
- Admin user initialization from environment
- Session cleanup and security

#### Services Package (`internal/services/`)

Tests service layer logic with 20.1% coverage:

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

Tests handler constructors with 1.0% coverage:

```go
func TestNewAdminHandler(t *testing.T) {
    // Test admin handler initialization
}

func TestNewFeedHandler(t *testing.T) {
    // Test feed handler initialization
}
```

**Coverage includes:**
- Handler constructor functions
- Service dependency injection
- Basic handler structure validation

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

### 3. Frontend Tests

#### Core Functionality (`web/tests/app-core.test.js`)

Tests frontend application logic with 26 comprehensive tests:

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

#### Font Choice Feature (`web/tests/font-choice.test.js`)

Tests comprehensive font choice functionality:

```javascript
describe('Font Choice Feature', () => {
  test('toggles between sans-serif and serif fonts', () => {
    // Test font switching functionality
  });

  test('persists font preference in localStorage', () => {
    // Test preference storage and retrieval
  });

  test('applies CSS custom properties correctly', () => {
    // Test CSS variable management
  });
});
```

**Coverage includes:**
- CSS custom properties and font variables
- Font preference initialization and persistence
- Toggle button functionality and UI updates
- Keyboard shortcut integration (f key)
- LocalStorage integration and error handling
- CSS class management (font-serif)
- Accessibility features and ARIA attributes
- Integration with existing UI components
- Error handling for corrupted preferences
- Visual regression prevention
- Cross-session preference persistence

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
- **Config package**: 96.7% coverage (comprehensive unit tests)  
- **Auth package**: 59.7% coverage (session, middleware, OAuth service)
- **Services package**: 20.1% coverage (subscription logic, feed discovery, **comprehensive admin token security**)
- **Handlers package**: 1.0% coverage (constructor functions)
- **Integration tests**: Full end-to-end API validation with user isolation testing, plus admin security testing
- **Frontend**: 26 tests covering core functionality
- **Admin Token System**: Comprehensive test coverage for the new secure authentication system
  - 6 SQLite backend test suites with 20+ individual test cases
  - 6 Datastore backend test suites (skip when emulator unavailable)
  - Security integration tests for bootstrap protection and token lifecycle
  - Edge case and error handling tests
- **Overall system**: All core tests passing successfully

### 🎯 Future Coverage Targets
- **Database operations**: Target 80%+ coverage (requires complex mocking)
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

The CI pipeline includes:

```yaml
name: Tests
on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    strategy:
      matrix:
        go-version: [1.23]
        
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v4
        with:
          go-version: ${{ matrix.go-version }}
          
      - name: Run unit tests
        run: go test -short -race -coverprofile=coverage.out ./internal/...
        
      - name: Run integration tests
        run: go test -race ./test/integration/...
        
      - name: Upload coverage
        uses: codecov/codecov-action@v4
```

**Pipeline features:**
- Go 1.23 testing
- Package-level unit tests (`./internal/...`)
- Integration tests (`./test/integration/...`)
- Coverage reporting to Codecov
- Linting with golangci-lint
- Multi-platform build artifacts
- Separate test, lint, and build jobs

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

```go
func BenchmarkFeedProcessing(b *testing.B) {
    db := setupBenchmarkDB(b)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        processFeed(db, sampleFeed)
    }
}
```

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

# Run tests with race detection
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

- All tests must pass: `./test.sh`
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