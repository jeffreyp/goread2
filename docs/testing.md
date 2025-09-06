# Testing Guide

Testing guide for GoRead2's multi-user RSS reader application.

## Current Status

✅ **All tests are passing successfully** - The testing infrastructure is fully functional with no interface compatibility issues.

## Overview

GoRead2's testing infrastructure includes:
- **Package-level unit tests** (currently config package with 66.7% coverage)
- **Integration tests** for end-to-end API validation
- **Frontend tests** with Jest and jsdom (26 tests)
- **CI/CD integration** with GitHub Actions

## Current Test Structure

```
internal/
├── config/
│   └── config_test.go   # Config package unit tests (66.7% coverage)
test/
├── integration/         # Backend integration tests
│   └── api_test.go      # End-to-end API testing
└── fixtures/            # Test data and sample feeds
    └── sample_feeds.go  # Sample data for tests
web/tests/               # Frontend tests
├── app-core.test.js     # Core frontend functionality (26 tests)
├── utils.js             # Frontend test utilities  
├── setup.js             # Test environment setup
└── README.md            # Frontend testing documentation
```

## Running Tests

### All Tests (Recommended)

```bash
./test.sh
```

The test script runs:
- Backend unit and integration tests
- Frontend JavaScript tests  
- Coverage report generation
- Code quality checks (linting)
- Build verification
- Colored output for better readability

### Backend Tests Only

```bash
# Package-level unit tests (currently config package)
go test ./internal/config/...

# Integration tests
go test ./test/integration/...

# With coverage
go test -coverprofile=coverage.out ./internal/...
go tool cover -html=coverage.out -o coverage.html

# Verbose output
go test -v ./internal/... ./test/integration/...

# Specific test package
go test ./internal/config/
go test ./test/integration/
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

Tests configuration management with 66.7% coverage:

```go
func TestLoadConfig(t *testing.T) {
    // Test configuration loading from environment
}

func TestConfigValidation(t *testing.T) {
    // Test configuration parameter validation
}

func TestDatabaseConfig(t *testing.T) {
    // Test database connection configuration
}
```

**Coverage includes:**
- Environment variable loading
- Configuration validation
- Database connection strings
- OAuth configuration parameters
- Default value handling

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

Tests use isolated test databases:

```go
// Integration tests create isolated database instances
func setupTestDB(t *testing.T) *sql.DB {
    // Creates fresh test database
    // Each test gets clean database state
}
```

### HTTP Test Setup

Integration tests use test servers:

```go
// Integration tests setup full HTTP stack
func setupTestServer(t *testing.T) *httptest.Server {
    // Creates test server with middleware
    // Includes authentication and session handling
}
```

## Coverage Goals

Current test coverage status and targets:

### ✅ Achieved Coverage
- **Config package**: 66.7% coverage (package-level unit tests)
- **Integration tests**: Full end-to-end API validation with user isolation testing
- **Frontend**: 26 tests covering core functionality
- **Overall system**: All tests passing successfully
- **User isolation**: Verified through comprehensive integration tests

### 🎯 Future Coverage Targets
- **Database operations**: 90%+ coverage goal (interfaces are now compatible)
- **Authentication logic**: 85%+ coverage goal  
- **Additional packages**: 80%+ coverage goal

### Viewing Coverage

```bash
# Generate package-level coverage
go test -coverprofile=coverage.out ./internal/...
go tool cover -html=coverage.out -o coverage.html

# Generate frontend coverage
npm run test:coverage

# View coverage in browser
open coverage.html                    # Backend
open web/coverage/index.html          # Frontend
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

## Security Testing

Critical tests for multi-user security:

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