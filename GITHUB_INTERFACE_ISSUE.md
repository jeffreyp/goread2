# Database Interface Inconsistencies Preventing Unit Tests

## Problem Description

The current `database.Database` interface has inconsistencies between its definition and implementations, preventing proper unit testing of individual packages. While integration tests work well, package-level unit tests fail due to interface mismatches.

## Current Status

✅ **Working Tests:**
- Config package: 85.7% coverage 
- Integration tests: Full end-to-end validation
- Frontend tests: Complete UI coverage

❌ **Blocked Tests:**
- Auth package unit tests
- Handlers package unit tests  
- Services package unit tests
- Database package unit tests

## Interface Issues

### 1. Return Type Mismatches (Value vs Pointer Types)

The interface expects value slices but implementations return pointer slices:

```go
// Interface definition (schema.go:52)
GetAllArticles() ([]Article, error)

// Actual implementations return
GetAllArticles() ([]*Article, error)
```

**Affected Methods:**
- `GetAllArticles() ([]Article, error)` vs `([]*Article, error)`
- `GetFeeds() ([]Feed, error)` vs `([]*Feed, error)`
- `GetUserFeeds(userID int) ([]Feed, error)` vs `([]*Feed, error)`
- `GetUserArticles(userID int) ([]Article, error)` vs `([]*Article, error)`

### 2. Method Signature Inconsistencies

#### UpdateUserSubscription Parameters
```go
// Interface expectation
UpdateUserSubscription(userID int, status, subscriptionID string, lastPaymentDate time.Time) error

// Some implementations have
UpdateUserSubscription(userID int, isActive bool, expiresAt *int64) error
```

#### MarkUserArticleRead Parameters  
```go
// Interface has
MarkUserArticleRead(userID, articleID int, isRead bool) error

// Some implementations expect
MarkUserArticleRead(userID, articleID int) error
```

#### UpdateFeedLastFetch Parameters
```go
// Interface expects
UpdateFeedLastFetch(feedID int, lastFetch time.Time) error

// Some implementations have
UpdateFeedLastFetch(feedID int) error
```

### 3. Method Parameter Count Mismatches

```go
// Interface definition
GetArticles(feedID int) ([]Article, error)

// Some implementations expect
GetArticles(feedID int, limit int, offset int) ([]*Article, error)
```

## Impact

These inconsistencies prevent:
1. Creation of proper mock implementations for testing
2. Package-level unit test development
3. Full code coverage measurement
4. Test-driven development practices

## Proposed Solution

### Phase 1: Standardize Interface Definition
1. **Decide on return types**: Choose between value types (`[]Article`) or pointer types (`[]*Article`)
2. **Standardize parameters**: Ensure all implementations match the interface exactly
3. **Update interface documentation**: Add clear parameter descriptions

### Phase 2: Update Implementations
1. **Update SQLite implementation** (`schema.go`) to match interface
2. **Update Datastore implementation** (`datastore.go`) to match interface  
3. **Ensure backward compatibility** during transition

### Phase 3: Add Unit Tests
1. Create proper mock implementations that match the interface
2. Add comprehensive unit tests for each package
3. Achieve target coverage levels

## Recommendations

### Option A: Use Pointer Types (Recommended)
Change interface to use pointer types since:
- Most Go database libraries return pointers
- Avoids unnecessary copying of large structs
- Consistent with existing codebase patterns

```go
// Update interface to:
GetAllArticles() ([]*Article, error)
GetFeeds() ([]*Feed, error) 
GetUserFeeds(userID int) ([]*Feed, error)
```

### Option B: Use Value Types
Keep current interface and update implementations:
- More memory-efficient for small structs
- Prevents accidental mutation
- Requires updating all database implementations

## Files Requiring Updates

### Interface Definition
- `internal/database/schema.go` - Main interface definition

### Implementations  
- `internal/database/datastore.go` - Datastore implementation
- `internal/database/schema.go` - SQLite implementation

### Tests (After Interface Fix)
- `internal/auth/auth_test.go` - Auth service unit tests
- `internal/handlers/*_test.go` - HTTP handler unit tests
- `internal/services/*_test.go` - Business logic unit tests
- `internal/database/database_test.go` - Database unit tests

## Success Criteria

- [ ] All interface methods have consistent signatures
- [ ] Mock implementations can be created without compilation errors
- [ ] Unit tests can be written for all packages
- [ ] Overall test coverage increases significantly
- [ ] Integration tests continue to pass

## Additional Context

The current test infrastructure is solid:
- Test script (`test.sh`) works properly
- Coverage reporting is functional  
- CI/CD pipeline structure is in place
- Frontend tests are comprehensive

This issue focuses specifically on the backend Go interface consistency needed to unlock comprehensive unit testing capabilities.

## Labels

- `bug` - Interface inconsistencies
- `testing` - Related to test infrastructure
- `good first issue` - Well-defined scope
- `priority: medium` - Important for code quality but not blocking