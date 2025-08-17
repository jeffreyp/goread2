# Testing Guide

Comprehensive testing guide for GoRead2's multi-user RSS reader application.

## Overview

GoRead2 includes a robust testing suite covering:
- **Backend unit tests** for core functionality
- **Integration tests** for API endpoints and user isolation
- **Frontend tests** with Jest and jsdom
- **Security tests** for multi-user data isolation
- **CI/CD integration** with GitHub Actions

## Test Structure

```
test/
├── helpers/              # Backend test utilities
│   ├── database.go       # Database test helpers
│   └── http.go          # HTTP test helpers
├── unit/                # Backend unit tests
│   ├── auth_test.go     # Authentication service tests
│   ├── database_test.go # Database layer tests
│   └── admin_test.go    # Admin functionality tests
├── integration/         # Backend integration tests
│   ├── api_test.go      # API endpoint tests
│   └── admin_integration_test.go # Admin CLI tests
├── fixtures/            # Test data and sample feeds
│   └── sample_feeds.go  # Sample data for tests
└── web/tests/           # Frontend tests
    ├── app-core.test.js # Core frontend functionality
    ├── utils.js         # Frontend test utilities
    ├── setup.js         # Test environment setup
    └── README.md        # Frontend testing documentation
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
# Quick test run
go test ./test/unit/... ./test/integration/...

# With coverage
go test -coverprofile=coverage.out ./test/...
go tool cover -html=coverage.out -o coverage.html

# Verbose output
go test -v -timeout=30s ./test/...

# Specific test package
go test ./test/unit/database_test.go
go test ./test/integration/api_test.go
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

### 1. Backend Unit Tests

#### Database Layer (`test/unit/database_test.go`)

Tests core database operations:

```go
func TestUserCRUD(t *testing.T) {
    // Test user creation, retrieval, updates
}

func TestUserDataIsolation(t *testing.T) {
    // Verify users cannot access each other's data
}

func TestFeedSubscriptions(t *testing.T) {
    // Test user-specific feed subscriptions
}
```

**Coverage includes:**
- User CRUD operations
- Feed management
- Article storage and retrieval
- User-feed subscriptions
- User-specific read/starred status
- Data isolation between users

#### Authentication (`test/unit/auth_test.go`)

Tests authentication system:

```go
func TestOAuthConfiguration(t *testing.T) {
    // Verify OAuth setup and configuration
}

func TestSessionManagement(t *testing.T) {
    // Test session creation, validation, expiration
}

func TestAuthenticationMiddleware(t *testing.T) {
    // Verify middleware protects endpoints
}
```

**Coverage includes:**
- OAuth configuration validation
- Session creation and management
- Session expiration handling
- Authentication middleware
- User context extraction

#### Admin Functionality (`test/unit/admin_test.go`)

Tests admin operations:

```go
func TestAdminUserOperations(t *testing.T) {
    // Test admin privilege management
}

func TestSubscriptionService(t *testing.T) {
    // Test subscription logic and limits
}
```

**Coverage includes:**
- User admin operations
- Free months granting
- Subscription service methods
- Permission management

### 2. Backend Integration Tests

#### API Endpoints (`test/integration/api_test.go`)

Tests complete API functionality:

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

#### Admin CLI (`test/integration/admin_integration_test.go`)

Tests command-line admin operations:

```go
func TestAdminCommands(t *testing.T) {
    // Test CLI admin functionality
}
```

**Coverage includes:**
- Command-line admin operations
- User management commands
- Database integration testing
- Error handling validation

### 3. Frontend Tests

#### Core Functionality (`web/tests/app-core.test.js`)

Tests frontend application logic:

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

Tests use isolated databases:

```go
// test/helpers/database.go
func SetupTestDB(t *testing.T) Database {
    // Creates fresh in-memory SQLite database
    // Each test gets isolated database
}
```

### HTTP Test Setup

Tests use test servers:

```go
// test/helpers/http.go
func SetupTestServer(t *testing.T) *httptest.Server {
    // Creates test server with full middleware stack
    // Includes authentication and session handling
}
```

## Coverage Goals

Current test coverage targets:

- **Database operations**: 95%+ coverage
- **Authentication logic**: 90%+ coverage  
- **API endpoints**: 85%+ coverage
- **User isolation**: 100% coverage (critical for security)
- **Admin functions**: 90%+ coverage
- **Frontend core**: 80%+ coverage

### Viewing Coverage

```bash
# Generate backend coverage
go test -coverprofile=coverage.out ./test/...
go tool cover -html=coverage.out

# Generate frontend coverage
npm run test:coverage

# View coverage in browser
open coverage.html                    # Backend
open coverage/lcov-report/index.html  # Frontend
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
on: [push, pull_request]

jobs:
  test:
    strategy:
      matrix:
        go-version: [1.21, 1.22, 1.23]
        
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with:
          go-version: ${{ matrix.go-version }}
          
      - name: Run tests
        run: ./test.sh
        
      - name: Upload coverage
        uses: codecov/codecov-action@v3
```

**Pipeline features:**
- Multi-version Go testing
- Code coverage reporting
- Linting with golangci-lint
- Security scanning with gosec
- Build artifact generation
- Parallel job execution

## Writing New Tests

### Backend Test Example

```go
func TestNewFeature(t *testing.T) {
    // Setup
    db := helpers.SetupTestDB(t)
    defer db.Close()
    
    user := helpers.CreateTestUser(t, db, "test@example.com")
    
    // Test
    result := newFeatureFunction(user.ID)
    
    // Assert
    assert.NotNil(t, result)
    assert.Equal(t, expectedValue, result.Value)
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
    // Setup test server
    server := helpers.SetupTestServer(t)
    defer server.Close()
    
    // Create authenticated session
    session := helpers.CreateAuthenticatedSession(t, "test@example.com")
    
    // Make authenticated request
    resp := helpers.MakeAuthenticatedRequest(t, server, session, "GET", "/api/feeds")
    
    // Assert
    assert.Equal(t, http.StatusOK, resp.StatusCode)
    
    var feeds []Feed
    json.NewDecoder(resp.Body).Decode(&feeds)
    assert.IsType(t, []Feed{}, feeds)
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

**Test Database Errors:**
- Ensure SQLite driver is available: `go mod tidy`
- Check file permissions in test directory
- Verify tests clean up properly

**Authentication Test Failures:**
- Check environment variables are set correctly
- Verify Google OAuth configuration in tests
- Ensure session handling works in test environment

**Frontend Test Issues:**
- Verify Jest and jsdom are installed: `npm install`
- Check DOM setup in test files
- Ensure fetch mocking is configured properly

**Integration Test Timeouts:**
- Increase timeout for network-dependent tests
- Consider mocking external HTTP calls
- Check test server startup time

### Debug Commands

```bash
# Run specific test with timing
go test -v -timeout=60s -run TestSlowFunction ./test/...

# Frontend tests with coverage and debugging
npm test -- --coverage --verbose

# Check test dependencies
go mod verify
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

1. **Write tests first** (TDD approach)
2. **Ensure data isolation** for multi-user features
3. **Add integration tests** for new API endpoints
4. **Update test documentation** for new test categories
5. **Maintain coverage goals** (aim for >90% for new code)

### Pull Request Requirements

- All tests must pass: `./test.sh`
- Code coverage must not decrease
- Include both unit and integration tests
- Update documentation for user-facing changes
- Follow existing test patterns and conventions

This comprehensive testing suite ensures GoRead2 maintains high quality, security, and reliability across all user scenarios.