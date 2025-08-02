# GoRead2 Testing Guide

This document describes the comprehensive testing suite for GoRead2's multi-user RSS reader application.

## Test Structure

```
test/
├── helpers/           # Test utilities and setup functions
│   ├── database.go    # Database test helpers
│   └── http.go        # HTTP test helpers
├── unit/              # Unit tests
│   ├── auth_test.go   # Authentication service tests
│   ├── database_test.go # Database layer tests
│   └── feed_service_test.go # Feed service tests
├── integration/       # Integration tests
│   └── api_test.go    # API endpoint tests
└── fixtures/          # Test data and sample feeds
    └── sample_feeds.go # Sample data for tests
```

## Test Categories

### 1. Unit Tests

**Database Layer (`test/unit/database_test.go`)**
- User CRUD operations
- Feed management
- Article storage and retrieval
- User-feed subscriptions
- User-specific read/starred status
- Data isolation between users

**Authentication (`test/unit/auth_test.go`)**
- OAuth configuration validation
- Session creation and management
- Session expiration handling
- Authentication middleware
- User context extraction

**Feed Service (`test/unit/feed_service_test.go`)**
- User-specific feed operations
- Multi-user data isolation
- Article status management
- Feed subscription logic

### 2. Integration Tests

**API Endpoints (`test/integration/api_test.go`)**
- Authentication requirements
- Feed CRUD operations
- Article operations (read/star)
- User isolation verification
- Error handling and status codes

### 3. Test Infrastructure

**Database Helpers (`test/helpers/database.go`)**
- In-memory SQLite database creation
- Test user/feed/article creation
- Environment setup/cleanup

**HTTP Helpers (`test/helpers/http.go`)**
- Test server setup with full middleware stack
- Authenticated request creation
- Response assertion utilities

## Running Tests

### Quick Test Run
```bash
go test ./test/unit/... ./test/integration/...
```

### With Coverage
```bash
go test -coverprofile=coverage.out ./test/...
go tool cover -html=coverage.out -o coverage.html
```

### Using Test Script
```bash
./test.sh
```

The test script provides:
- Colored output for better readability
- Coverage report generation
- Linting (if golangci-lint is available)
- Test summary and failure reporting

## CI/CD Integration

### GitHub Actions (`.github/workflows/test.yml`)

The CI pipeline runs:
- **Multi-version testing**: Go 1.21, 1.22, 1.23
- **Unit and integration tests**
- **Code coverage reporting**
- **Linting with golangci-lint**
- **Security scanning with gosec**
- **Multi-platform builds**
- **Artifact uploads**

### Key Features
- Parallel job execution for faster feedback
- Coverage reports uploaded to Codecov
- Security vulnerability detection
- Build artifact generation for releases

## Test Environment

### Environment Variables
Tests use these environment variables:
```bash
GOOGLE_CLIENT_ID=test_client_id
GOOGLE_CLIENT_SECRET=test_client_secret
GOOGLE_REDIRECT_URL=http://localhost:8080/auth/callback
```

### Database
- Tests use in-memory SQLite databases
- Each test gets a fresh, isolated database
- No external dependencies required

### Mocking Strategy
- HTTP requests to external RSS feeds are not mocked in basic tests
- Real integration tests would benefit from HTTP client mocking
- Database operations use real SQLite for integration accuracy

## Coverage Goals

Current test coverage focuses on:
- **Database operations**: 95%+ coverage
- **Authentication logic**: 90%+ coverage  
- **API endpoints**: 85%+ coverage
- **User isolation**: 100% coverage (critical for security)

## Test Data

### Fixtures (`test/fixtures/sample_feeds.go`)
- Sample users with different roles
- Example RSS and Atom feeds
- Sample articles with various states
- XML feed examples for parser testing

### Test Users
- `user1@example.com`: Standard user
- `user2@example.com`: Second user for isolation testing
- `admin@example.com`: Admin user (future use)

## Security Testing

Tests verify:
- **User data isolation**: Users cannot access other users' data
- **Authentication requirements**: All protected endpoints require auth
- **Session security**: Session creation, validation, and cleanup
- **Input validation**: Malformed requests are handled properly

## Performance Considerations

- Tests use in-memory databases for speed
- Parallel test execution where possible
- Minimal external dependencies
- Fast setup/teardown for quick feedback

## Future Enhancements

1. **Mock HTTP client** for feed fetching tests
2. **Load testing** for multi-user scenarios  
3. **Database migration tests**
4. **End-to-end browser tests** with Selenium
5. **Performance benchmarks**
6. **Chaos engineering** tests

## Troubleshooting

### Common Issues

**Test Database Errors**
- Ensure SQLite driver is available
- Check file permissions in test directory

**Authentication Test Failures**
- Verify environment variables are set
- Check Google OAuth configuration

**Integration Test Timeouts**
- Increase timeout for network-dependent tests
- Consider mocking external HTTP calls

### Debug Mode
Run tests with verbose output:
```bash
go test -v -timeout=30s ./test/...
```

## Contributing

When adding new features:

1. **Write tests first** (TDD approach)
2. **Ensure data isolation** for multi-user features
3. **Add integration tests** for new API endpoints
4. **Update test documentation** for new test categories
5. **Maintain coverage goals** (aim for >90% for new code)

## Test Philosophy

Our testing approach prioritizes:
- **User data security**: Rigorous isolation testing
- **Authentication integrity**: Comprehensive auth testing
- **Real-world scenarios**: Integration over unit mocking
- **Fast feedback**: Quick test execution
- **Maintainability**: Clear, well-documented tests