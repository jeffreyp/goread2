# GoRead2 - Multi-User RSS Reader

[![Tests](https://github.com/jeffreyp/goread2/actions/workflows/test.yml/badge.svg)](https://github.com/jeffreyp/goread2/actions/workflows/test.yml)

A modern, multi-user RSS reader inspired by Google Reader, built with Go and featuring Google OAuth authentication, a clean three-pane interface, and comprehensive user data isolation.

## Features

- **Multi-user support**: Google OAuth authentication with secure user data isolation
- **Three-pane layout**: Feed list → Article list → Article content (just like Google Reader)
- **RSS/Atom feed support**: Add and manage multiple RSS and Atom feeds
- **OPML import**: Import feed subscriptions from other RSS readers (Feedly, Inoreader, etc.)
- **Real-time updates**: Background polling for new articles every 30 minutes
- **Per-user article management**: Mark articles as read/unread, star favorites - all personal to each user
- **Keyboard shortcuts**: Navigate efficiently with vim-like shortcuts
- **Clean UI**: Fast, responsive interface with modern design
- **Self-hosted**: Can run locally or deploy to cloud platforms
- **Comprehensive testing**: Full test suite with unit and integration tests

## Architecture

- **Multi-user authentication**: Google OAuth 2.0 integration
- **User data isolation**: Each user has their own read/star status and feed subscriptions
- **Secure sessions**: HTTP-only cookies with proper session management
- **Database flexibility**: SQLite for local development, Google Cloud Datastore for production
- **Test coverage**: 90%+ test coverage with automated CI/CD pipeline

## Screenshot

The interface features:
- **Authentication**: Google OAuth login/logout in the header
- **Left pane**: Personal feed subscriptions with unread counts
- **Center pane**: Article list with personal read/unread status
- **Right pane**: Full article content with original formatting

## Keyboard Shortcuts

- `j` - Next article
- `k` - Previous article  
- `o` / `Enter` - Open article in new tab
- `m` - Mark current article as read/unread
- `s` - Star/unstar current article
- `r` - Refresh all feeds

## Installation

### Prerequisites

- Go 1.21 or later
- Google Cloud Project (for OAuth)
- SQLite3 (automatically included with go-sqlite3)

### Google OAuth Setup

1. **Create Google Cloud Project:**
   - Go to [Google Cloud Console](https://console.cloud.google.com/)
   - Create a new project or select existing one

2. **Configure OAuth Consent Screen:**
   - Navigate to APIs & Services → OAuth consent screen
   - Configure the consent screen with your application details

3. **Create OAuth Credentials:**
   - Go to APIs & Services → Credentials
   - Create OAuth 2.0 Client ID
   - Set authorized redirect URI: `http://localhost:8080/auth/callback` (for local dev)

4. **Set Environment Variables:**
   ```bash
   export GOOGLE_CLIENT_ID="your-client-id"
   export GOOGLE_CLIENT_SECRET="your-client-secret"
   export GOOGLE_REDIRECT_URL="http://localhost:8080/auth/callback"
   ```

### Local Development

1. **Clone the project:**
   ```bash
   git clone https://github.com/jeffreyp/goread2.git
   cd goread2
   ```

2. **Install dependencies:**
   ```bash
   go mod tidy
   ```

3. **Set up environment:**
   ```bash
   # Create .env file or export variables
   export GOOGLE_CLIENT_ID="your-client-id"
   export GOOGLE_CLIENT_SECRET="your-client-secret"
   export GOOGLE_REDIRECT_URL="http://localhost:8080/auth/callback"
   ```

4. **Build and run:**
   ```bash
   go build -o goread2 .
   ./goread2
   ```

5. **Access the application:**
   Navigate to `http://localhost:8080` and sign in with Google

### Production Deployment

See [DEPLOYMENT.md](DEPLOYMENT.md) for detailed production deployment instructions including:
- Google App Engine deployment with Cloud Datastore
- Docker containerization
- Environment variable configuration
- SSL/TLS setup

## Usage

### Authentication

1. Navigate to the application URL
2. Click "Login with Google" to authenticate
3. Grant necessary permissions
4. You'll be redirected to your personal dashboard

### Managing Feeds

1. **Adding feeds**: Click "Add Feed" and enter RSS/Atom URL
2. **OPML import**: Click "Import OPML" to import feeds from other RSS readers
   - Supports exports from Feedly, Inoreader, NewsBlur, and other RSS readers
   - Handles nested folders and complex OPML structures
   - Shows import progress and success count
3. **Subscribing**: Feeds are automatically subscribed for your user
4. **Unsubscribing**: Click the × next to any feed in your list
5. **Feed discovery**: Supports both RSS and Atom formats

### Reading Articles

1. **Personal feed list**: Left panel shows only your subscribed feeds
2. **Article status**: Read/unread and starred status is personal to you
3. **All articles view**: See articles from all your subscribed feeds
4. **Article content**: Right panel displays full article with original formatting

### Multi-User Features

- **User isolation**: Each user sees only their own subscriptions and article status
- **Shared feeds**: Multiple users can subscribe to the same feed independently
- **Personal article status**: Read/starred status is unique per user
- **Session management**: Secure login/logout with proper session cleanup

## API Endpoints

All API endpoints require authentication via session cookies:

### Authentication
- `GET /auth/login` - Initiate Google OAuth flow
- `GET /auth/callback` - OAuth callback handler
- `POST /auth/logout` - Logout and clear session
- `GET /auth/me` - Get current user information

### Feeds (User-specific)
- `GET /api/feeds` - List user's subscribed feeds
- `POST /api/feeds` - Subscribe user to new feed
- `POST /api/feeds/import` - Import feeds from OPML file
- `DELETE /api/feeds/:id` - Unsubscribe user from feed
- `POST /api/feeds/refresh` - Refresh all user's feeds

### Articles (User-specific)
- `GET /api/feeds/:id/articles` - Get articles for user's feed
- `GET /api/feeds/all/articles` - Get all articles from user's feeds
- `POST /api/articles/:id/read` - Mark article read/unread for user
- `POST /api/articles/:id/star` - Toggle article star for user

## Configuration

### Environment Variables

**Required:**
- `GOOGLE_CLIENT_ID` - Google OAuth client ID
- `GOOGLE_CLIENT_SECRET` - Google OAuth client secret  
- `GOOGLE_REDIRECT_URL` - OAuth redirect URL

**Optional:**
- `GOOGLE_CLOUD_PROJECT` - Use Google Cloud Datastore (if set)
- `PORT` - Server port (default: 8080)

### Database Configuration

- **Local Development**: SQLite database (`goread2.db`)
- **Production**: Google Cloud Datastore (when `GOOGLE_CLOUD_PROJECT` is set)

### Session Configuration

- **Security**: HTTP-only cookies with secure flags in production
- **Expiration**: 24-hour session lifetime
- **Cleanup**: Automatic cleanup of expired sessions

## Testing

The project includes a comprehensive test suite covering all functionality:

### Running Tests

```bash
# Run all tests
./test.sh

# Run specific test categories
go test ./test/unit/...          # Unit tests
go test ./test/integration/...   # Integration tests

# Run with coverage
go test -coverprofile=coverage.out ./test/...
go tool cover -html=coverage.out
```

### Test Categories

- **Unit Tests**: Database operations, authentication, business logic
- **Integration Tests**: API endpoints, user isolation, authentication flow
- **Security Tests**: User data isolation, session management
- **Feed Parser Tests**: RSS and Atom feed processing

### CI/CD Pipeline

GitHub Actions workflow automatically:
- Tests on Go 1.21, 1.22, 1.23
- Runs linting and security scanning
- Generates coverage reports
- Builds artifacts for deployment

## Project Structure

```
goread2/
├── main.go                     # Application entry point with auth setup
├── internal/
│   ├── auth/                   # Authentication system
│   │   ├── auth.go            # Google OAuth integration
│   │   ├── middleware.go      # Authentication middleware
│   │   └── session.go         # Session management
│   ├── database/
│   │   ├── schema.go          # Multi-user database models
│   │   └── datastore.go       # Google Cloud Datastore implementation
│   ├── handlers/
│   │   ├── feed_handler.go    # Feed API handlers
│   │   └── auth_handler.go    # Authentication handlers
│   └── services/
│       └── feed_service.go    # Multi-user business logic
├── test/                      # Comprehensive test suite
│   ├── unit/                  # Unit tests
│   ├── integration/           # Integration tests
│   ├── helpers/               # Test utilities
│   └── fixtures/              # Test data
├── web/
│   ├── templates/
│   │   └── index.html         # Main application template
│   └── static/
│       ├── css/
│       │   └── styles.css     # Google Reader-inspired styles
│       └── js/
│           └── app.js         # Frontend with auth integration
├── .github/
│   └── workflows/
│       └── test.yml           # CI/CD pipeline
├── README_TESTING.md          # Testing documentation
├── DEPLOYMENT.md              # Deployment guide
└── test.sh                    # Test runner script
```

## Security

### Authentication Security
- **OAuth 2.0**: Industry-standard Google OAuth integration
- **Secure sessions**: HTTP-only cookies with CSRF protection
- **Session expiration**: Automatic cleanup of expired sessions
- **No password storage**: Leverages Google's authentication

### Data Isolation
- **User separation**: Complete isolation of user data in database
- **Feed subscriptions**: Per-user feed subscription management  
- **Article status**: Read/starred status isolated per user
- **Database queries**: All queries filtered by user ID

### Input Validation
- **Feed URLs**: Validation and sanitization of RSS/Atom URLs
- **User input**: Proper escaping and validation
- **SQL injection**: Parameterized queries throughout
- **XSS protection**: Content sanitization in templates

## Development

### Setting Up Development Environment

1. **Clone and setup:**
   ```bash
   git clone https://github.com/jeffreyp/goread2.git
   cd goread2
   go mod tidy
   ```

2. **Configure OAuth:**
   - Set up Google Cloud project and OAuth credentials
   - Export environment variables
   - Update redirect URLs for local development

3. **Run tests:**
   ```bash
   ./test.sh
   ```

4. **Start development server:**
   ```bash
   go run main.go
   ```

### Adding New Features

1. **Database changes**: Update multi-user schema in `internal/database/schema.go`
2. **Authentication**: Modify middleware in `internal/auth/`
3. **API endpoints**: Add user-aware handlers in `internal/handlers/`
4. **Business logic**: Extend multi-user services in `internal/services/`
5. **Frontend**: Update authentication flow in `web/static/js/app.js`
6. **Tests**: Add comprehensive tests for new functionality

### Code Quality

- **Linting**: Use `golangci-lint` for code quality
- **Testing**: Maintain 90%+ test coverage
- **Documentation**: Update README and code comments
- **Security**: Follow security best practices

## Troubleshooting

### Authentication Issues

**OAuth errors:**
- Verify Google Cloud project configuration
- Check OAuth client ID and secret
- Ensure redirect URLs match exactly
- Verify OAuth consent screen setup

**Session problems:**
- Clear browser cookies and retry
- Check server logs for session errors
- Verify environment variables are set

### Feed Issues

**"Failed to fetch feed" error:**
- Verify RSS/Atom feed URL is valid and accessible
- Check server logs for specific HTTP errors
- Some feeds may require User-Agent headers

**Feed not updating:**
- Check feed refresh cron job/background task
- Verify feed URL hasn't changed
- Look for HTTP status errors in logs

### Database Issues

**Local SQLite problems:**
- Stop all running instances
- Check `goread2.db` file permissions
- Delete database file to reset (loses data)

**User data isolation:**
- Verify user ID is properly set in session
- Check database queries include user filtering
- Review test results for isolation verification

### Performance

**Slow article loading:**
- Check database indexes
- Monitor feed fetch times
- Consider caching strategies

**Memory usage:**
- Monitor session cleanup
- Check for database connection leaks
- Review background task efficiency

## Production Considerations

### Monitoring
- Set up application logging
- Monitor authentication success/failure rates
- Track feed fetch performance
- Monitor user session metrics

### Scaling
- Consider database connection pooling
- Implement feed fetch queuing for many users
- Add caching layers for frequently accessed data
- Monitor memory usage and optimize accordingly

### Backup
- Regular database backups (SQLite or Cloud Datastore)
- User data export capabilities
- Session data recovery procedures

## Contributing

1. Fork the repository
2. Create a feature branch
3. Implement changes with tests
4. Ensure all tests pass (`./test.sh`)
5. Update documentation
6. Submit a pull request

### Pull Request Requirements
- All tests must pass
- Code coverage must not decrease
- Include integration tests for new features
- Update documentation for user-facing changes
- Follow existing code style and patterns

## License

This project is licensed under the [MIT License](LICENSE). You are free to use, modify, and distribute this software for any purpose, including commercial use, as long as you include the original license and copyright notice.