# Contributing Guide

Welcome to GoRead2! This guide explains how to contribute to the project.

## Getting Started

### Prerequisites

- **Go 1.23+** for backend development
- **Node.js 16+** for frontend testing
- **Git** for version control
- **Google Cloud Project** for OAuth setup (development)

### Development Setup

1. **Fork and clone**:
   ```bash
   git clone https://github.com/your-username/goread2.git
   cd goread2
   ```

2. **Install dependencies**:
   ```bash
   go mod tidy
   npm install  # For frontend testing
   ```

3. **Set up environment**:
   ```bash
   # Copy example environment
   cp .env.example .env
   
   # Edit .env with your OAuth credentials
   export GOOGLE_CLIENT_ID="your-client-id"
   export GOOGLE_CLIENT_SECRET="your-client-secret"
   export GOOGLE_REDIRECT_URL="http://localhost:8080/auth/callback"
   ```

4. **Run tests**:
   ```bash
   ./test.sh
   ```

5. **Start development server**:
   ```bash
   go run main.go
   ```

## Development Workflow

### Making Changes

1. **Create a feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** following the coding standards below

3. **Write tests** for new functionality:
   ```bash
   # Unit tests
   touch test/unit/your_feature_test.go
   
   # Integration tests
   touch test/integration/your_feature_integration_test.go
   ```

4. **Run tests frequently**:
   ```bash
   # Quick test run
   go test ./test/...

   # Full test suite using Makefile (recommended)
   make test

   # Or use the test script directly
   ./test.sh
   ```

5. **Commit with clear messages**:
   ```bash
   git add .
   git commit -m "feat: add RSS feed auto-discovery
   
   - Implement feed URL detection from HTML pages
   - Add support for <link rel='alternate'> discovery
   - Include tests for common blog platforms
   
   Closes #123"
   ```

### Pull Request Process

1. **Push your branch**:
   ```bash
   git push origin feature/your-feature-name
   ```

2. **Create pull request** on GitHub with:
   - Clear title and description
   - Reference to any related issues
   - Screenshot/demo if UI changes
   - Test results

3. **Address review feedback**:
   - Make requested changes
   - Push updates to the same branch
   - Respond to comments

4. **Merge requirements**:
   - All CI checks pass
   - Code review approval
   - No merge conflicts
   - Documentation updated if needed

## Coding Standards

### Go Code Style

Follow standard Go conventions:

```go
// Good: Clear function documentation
// GetUserFeeds retrieves all feeds subscribed by the specified user.
// Returns empty slice if user has no subscriptions.
func (db *DB) GetUserFeeds(userID int) ([]Feed, error) {
    // Always include user filtering for security
    query := `SELECT * FROM feeds f 
              JOIN user_feeds uf ON f.id = uf.feed_id 
              WHERE uf.user_id = ?`
    
    // Handle errors explicitly
    rows, err := db.Query(query, userID)
    if err != nil {
        return nil, fmt.Errorf("failed to query user feeds: %w", err)
    }
    defer rows.Close()
    
    // ... rest of implementation
}
```

**Key principles**:
- **User isolation**: Always filter by user ID in database queries
- **Error handling**: Wrap errors with context using `fmt.Errorf`
- **Security**: Validate all inputs, escape SQL parameters
- **Documentation**: Document public functions and complex logic
- **Testing**: Include both unit and integration tests

### Database Operations

Always consider multi-user isolation:

```go
// Good: User-specific query
func (db *DB) GetUserArticles(userID int) ([]Article, error) {
    query := `SELECT a.* FROM articles a
              JOIN user_feeds uf ON a.feed_id = uf.feed_id
              WHERE uf.user_id = ?`
    // ...
}

// Bad: No user filtering (security risk)
func (db *DB) GetAllArticles() ([]Article, error) {
    query := `SELECT * FROM articles`  // Exposes all user data!
    // ...
}
```

### API Handlers

Require authentication and include proper error handling:

```go
func (h *FeedHandler) AddFeed(c *gin.Context) {
    // Always get authenticated user
    user, exists := auth.GetUserFromContext(c)
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
        return
    }

    // Validate input
    var req struct {
        URL string `json:"url" binding:"required,url"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Check permissions (subscription limits, etc.)
    if err := h.subscriptionService.CanUserAddFeed(user.ID); err != nil {
        c.JSON(http.StatusPaymentRequired, gin.H{"error": err.Error()})
        return
    }

    // Perform operation with user context
    feed, err := h.feedService.AddFeedForUser(user.ID, req.URL)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusCreated, feed)
}
```

### Frontend Code

Keep JavaScript clean and well-structured:

```javascript
class GoReadApp {
    async addFeed(url) {
        try {
            // Validate input
            if (!url || !this.isValidURL(url)) {
                throw new Error('Please enter a valid RSS feed URL');
            }

            // Show loading state
            this.showLoading('Adding feed...');

            // Make API call
            const response = await fetch('/api/feeds', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                credentials: 'include',
                body: JSON.stringify({ url })
            });

            // Handle response
            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.error || 'Failed to add feed');
            }

            const feed = await response.json();

            // Update UI
            this.addFeedToUI(feed);
            this.hideLoading();
            this.showSuccess(`Added feed: ${feed.title}`);

        } catch (error) {
            this.hideLoading();
            this.showError(error.message);
        }
    }
}
```

### Frontend Architecture

GoRead2 uses **bundle splitting with lazy loading** to optimize performance:

#### Bundle Structure

- **`app.js`** (40KB minified): Core application bundle with feed management, article viewing, and navigation
- **`modals.js`** (2.5KB minified): Modal interactions lazy-loaded on first use
- **`account.js`** (15KB minified): Account page functionality

#### Lazy Loading

Modal code is lazy-loaded using native ES6 dynamic imports:

```javascript
// In app.js - modal methods delegate to lazy-loaded module
async showAddFeedModal() {
    const manager = await this.loadModalManager();
    manager.showAddFeedModal();
}

// Lazy load helper (in app.js)
async loadModalManager() {
    if (this.modalManager) return this.modalManager;

    console.log('Loading modal module...');
    const module = await import('./modals.js');
    this.modalManager = new module.ModalManager(this);
    this.modalManager.init();
    console.log('Modal module loaded');

    return this.modalManager;
}
```

#### Performance Benefits

- **Initial bundle**: 40KB → ~14KB gzipped (~33% reduction)
- **Time to Interactive**: ~200ms faster on 3G
- **Parse time**: ~35% reduction on mobile
- **First modal open**: +100-150ms one-time delay (acceptable)
- **Browser support**: Chrome 63+, Firefox 67+, Safari 11.1+, Edge 79+ (95%+ coverage)

#### Adding Lazy-Loaded Modules

When adding features that aren't needed immediately:

1. Create a new module file (e.g., `feature.js`)
2. Export a class or functions from the module
3. Add lazy load helper to `app.js`
4. Update `package.json` and `Makefile` build scripts
5. Update this documentation

Example:
```javascript
// feature.js
export class FeatureManager {
    constructor(app) {
        this.app = app;
    }
    // ... feature methods
}

// app.js
async loadFeatureManager() {
    if (this.featureManager) return this.featureManager;
    const module = await import('./feature.js');
    this.featureManager = new module.FeatureManager(this);
    return this.featureManager;
}
```

### Testing Requirements

#### Backend Tests

Write comprehensive tests for all new functionality:

```go
func TestAddFeedForUser(t *testing.T) {
    // Setup
    db := helpers.SetupTestDB(t)
    defer db.Close()
    
    user := helpers.CreateTestUser(t, db, "test@example.com")
    
    tests := []struct {
        name        string
        url         string
        expectError bool
    }{
        {"valid RSS feed", "https://example.com/feed.xml", false},
        {"invalid URL", "not-a-url", true},
        {"duplicate feed", "https://example.com/feed.xml", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := addFeedForUser(db, user.ID, tt.url)
            
            if tt.expectError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                
                // Verify user can access the feed
                feeds, err := db.GetUserFeeds(user.ID)
                assert.NoError(t, err)
                assert.Len(t, feeds, 1)
            }
        })
    }
}
```

#### Integration Tests

Test complete workflows:

```go
func TestFeedAPIWorkflow(t *testing.T) {
    server := helpers.SetupTestServer(t)
    defer server.Close()
    
    // Create authenticated session
    session := helpers.CreateAuthenticatedSession(t, "test@example.com")
    
    // Add feed
    resp := helpers.MakeAuthenticatedRequest(t, server, session, 
        "POST", "/api/feeds", `{"url": "https://example.com/feed.xml"}`)
    assert.Equal(t, http.StatusCreated, resp.StatusCode)
    
    // Verify feed appears in user's list
    resp = helpers.MakeAuthenticatedRequest(t, server, session, 
        "GET", "/api/feeds", "")
    assert.Equal(t, http.StatusOK, resp.StatusCode)
    
    var feeds []Feed
    json.NewDecoder(resp.Body).Decode(&feeds)
    assert.Len(t, feeds, 1)
    assert.Equal(t, "https://example.com/feed.xml", feeds[0].URL)
}
```

#### Frontend Tests

Test UI interactions and API integration:

```javascript
describe('Feed Management', () => {
    beforeEach(() => {
        document.body.innerHTML = '<div id="app"></div>';
        global.fetch = jest.fn();
    });

    test('adds feed successfully', async () => {
        // Setup
        const app = new GoReadApp();
        fetch.mockResolvedValueOnce({
            ok: true,
            json: () => Promise.resolve({
                id: 1,
                title: 'Test Feed',
                url: 'https://example.com/feed.xml'
            })
        });

        // Execute
        await app.addFeed('https://example.com/feed.xml');

        // Verify
        expect(fetch).toHaveBeenCalledWith('/api/feeds', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            credentials: 'include',
            body: JSON.stringify({ url: 'https://example.com/feed.xml' })
        });
    });
});
```

## Security Guidelines

### User Data Protection

- **Always filter by user ID** in database queries
- **Validate all inputs** to prevent injection attacks
- **Use parameterized queries** for SQL operations
- **Escape HTML output** to prevent XSS

### Authentication

- **Require authentication** for all user-specific operations
- **Validate session tokens** on every request
- **Use HTTPS** in production for secure cookie transmission
- **Implement proper session cleanup**

### Input Validation

```go
// Good: Comprehensive input validation
func validateFeedURL(url string) error {
    if url == "" {
        return errors.New("URL cannot be empty")
    }
    
    parsed, err := url.Parse(url)
    if err != nil {
        return fmt.Errorf("invalid URL format: %w", err)
    }
    
    if parsed.Scheme != "http" && parsed.Scheme != "https" {
        return errors.New("URL must use HTTP or HTTPS")
    }
    
    // Additional validation...
    return nil
}
```

## Documentation

### Code Documentation

- **Document public APIs** with clear descriptions
- **Include examples** for complex functions
- **Document security considerations** for user-facing code
- **Keep comments up-to-date** with code changes

### User Documentation

When adding user-facing features:

- **Update README.md** with new capabilities
- **Add API documentation** to `docs/API.md`
- **Include troubleshooting** in `docs/TROUBLESHOOTING.md`
- **Provide examples** and use cases

## Issue Tracking with Beads

GoRead2 uses [Beads](https://github.com/steveyegge/beads) for issue tracking, as specified in `CLAUDE.md`. Beads provides a lightweight, command-line issue tracker that stores issues in the `.beads/` directory.

### Getting Started with Beads

1. **Install Beads**:
   ```bash
   # Install via pip
   pip install beads-cli

   # Or run from source
   git clone https://github.com/steveyegge/beads
   cd beads && pip install -e .
   ```

2. **Quick start**:
   ```bash
   # See quick tutorial and examples
   bd quickstart
   ```

### Common Beads Commands

```bash
# List all issues
bd list

# List only open issues
bd list --open

# Create a new bug
bd new "Description of the bug" --type bug --priority P2

# Create a new feature request
bd new "Feature description" --type feature --priority P3

# Show issue details
bd show goread2-1

# Close an issue
bd close goread2-1

# Search for issues
bd search "keyword"
```

### Issue Types and Priorities

**Issue Types:**
- `bug` - Something isn't working correctly
- `feature` - New functionality or enhancement
- `task` - Maintenance, refactoring, or documentation
- `question` - Need clarification or discussion

**Priority Levels:**
- `P0` - Critical (blocking production)
- `P1` - High (important bugs or features)
- `P2` - Medium (normal priority)
- `P3` - Low (nice to have)

### Beads Directory Structure

```
.beads/
├── issues.jsonl      # Issue tracking data (committed to git)
└── goread2.db        # SQLite database (not committed)
```

The `issues.jsonl` file is committed to version control, allowing issue history to be tracked alongside code changes. The database file is gitignored.

### Workflow Example

```bash
# Create an issue for a bug you found
bd new "Load more articles pagination is broken" --type bug --priority P2

# Work on the fix...
git checkout -b fix/pagination-bug

# Make your changes, commit, and reference the issue
git commit -m "Fix Load More pagination with offset adjustment

Resolves: goread2-3"

# Close the issue when fixed
bd close goread2-3
```

### Integration with Git Workflow

When referencing Beads issues in commits:

```bash
# Reference an issue
git commit -m "Add feature X

Related to: goread2-5"

# Close an issue
git commit -m "Fix bug Y

Resolves: goread2-12
Closes: goread2-12"
```

### Why Beads?

- **Lightweight** - No external services required
- **Git-native** - Issues tracked alongside code
- **CLI-friendly** - Fast and scriptable
- **Developer-focused** - Built for command-line workflows
- **Privacy** - All data stays in your repository

## Issue Reporting

For external users who may not have Beads installed:

### Bug Reports

Include these details:

- **Environment**: Local, Docker, App Engine, etc.
- **Steps to reproduce**: Exact sequence of actions
- **Expected behavior**: What should happen
- **Actual behavior**: What actually happens
- **Logs**: Relevant error messages and stack traces
- **Configuration**: Environment variables (redact secrets)

### Feature Requests

Describe:

- **Use case**: Why is this feature needed?
- **Proposed solution**: How should it work?
- **Alternatives**: Other ways to solve the problem
- **Impact**: Who would benefit from this feature?

## Code Review Process

### For Contributors

- **Self-review** your code before submitting
- **Write clear commit messages** explaining the changes
- **Include tests** for all new functionality
- **Update documentation** for user-facing changes
- **Be responsive** to review feedback

### For Reviewers

Focus on:

- **Security**: User data isolation, input validation
- **Performance**: Database query efficiency, memory usage
- **Maintainability**: Code clarity, documentation
- **Testing**: Adequate test coverage
- **User experience**: API design, error handling

## Release Process

### Version Numbering

Using Semantic Versioning (SemVer):

- **Major (X.0.0)**: Breaking changes, major new features
- **Minor (0.X.0)**: New features, backwards compatible
- **Patch (0.0.X)**: Bug fixes, security updates

### Release Checklist

Before releasing:

- [ ] All tests pass (`./test.sh`)
- [ ] Documentation updated
- [ ] Security review completed
- [ ] Performance testing (if significant changes)
- [ ] Database migration tested (if schema changes)
- [ ] Deployment tested in staging environment

## Community

### Communication

- **GitHub Issues**: Bug reports and feature requests
- **GitHub Discussions**: Questions and general discussion
- **Pull Requests**: Code review and collaboration

### Code of Conduct

- **Be respectful**: Treat all contributors with respect
- **Be inclusive**: Welcome contributors of all backgrounds
- **Be constructive**: Provide helpful feedback and suggestions
- **Be patient**: Help newcomers learn the codebase

### Recognition

Contributors are recognized through:

- **Git commit attribution**: Your commits show your contributions
- **Release notes**: Significant contributions mentioned
- **CONTRIBUTORS.md**: List of all project contributors

## Getting Help

### Resources

- **Documentation**: Start with [Setup Guide](SETUP.md)
- **API Reference**: Complete API documentation in [API Guide](API.md)
- **Testing**: How to run and write tests in [Testing Guide](TESTING.md)
- **Troubleshooting**: Common issues in [Troubleshooting Guide](TROUBLESHOOTING.md)

### Ask for Help

Don't hesitate to ask:

- **GitHub Discussions**: For questions about contributing
- **Issue comments**: For clarification on specific issues
- **Pull request reviews**: For feedback on your code

Thank you for contributing to GoRead2! Your help makes this project better for everyone.