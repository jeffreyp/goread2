# Setup Guide

Complete installation and configuration guide for GoRead2.

## Prerequisites

- **Go 1.23 or later**
- **Google Cloud Project** (for OAuth authentication)
- **SQLite3** (automatically included with go-sqlite3)
- **Stripe Account** (optional, for subscription features)

## Google OAuth Setup

### 1. Create Google Cloud Project

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select existing one
3. Enable the Google+ API (for user profile information)

### 2. Configure OAuth Consent Screen

1. Navigate to APIs & Services → OAuth consent screen
2. Configure the consent screen with your application details
3. Add test users if in development mode

### 3. Create OAuth Credentials

1. Go to APIs & Services → Credentials
2. Create OAuth 2.0 Client ID
3. Choose "Web application"
4. Set authorized redirect URIs:
   - Development: `http://localhost:8080/auth/callback`
   - Production: `https://your-domain.com/auth/callback`

### 4. Set Environment Variables

```bash
export GOOGLE_CLIENT_ID="your-client-id"
export GOOGLE_CLIENT_SECRET="your-client-secret"
export GOOGLE_REDIRECT_URL="http://localhost:8080/auth/callback"
```

## Stripe Setup (Optional)

For subscription features, configure Stripe:

### 1. Create Stripe Account

1. Go to [stripe.com](https://stripe.com) and create an account
2. Get your API keys from the Stripe Dashboard

### 2. Set Stripe Environment Variables

```bash
export STRIPE_SECRET_KEY="sk_test_your-secret-key"
export STRIPE_PUBLISHABLE_KEY="pk_test_your-publishable-key"
export STRIPE_WEBHOOK_SECRET="whsec_your-webhook-secret"
export STRIPE_PRICE_ID="price_your-price-id"
```

### 3. Create Product and Price

```bash
go run cmd/setup-stripe/main.go create-product
```

For detailed Stripe setup, see [Stripe Setup Guide](STRIPE.md).

## Installation

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
   
   # Optional: For subscription features
   export STRIPE_SECRET_KEY="sk_test_your-secret-key"
   export STRIPE_PUBLISHABLE_KEY="pk_test_your-publishable-key"
   export STRIPE_WEBHOOK_SECRET="whsec_your-webhook-secret"
   export STRIPE_PRICE_ID="price_your-price-id"
   ```

4. **Build and run:**
   ```bash
   # Build with the Makefile (recommended)
   make build
   ./goread2

   # Or build manually
   go build -o goread2 .
   ./goread2

   # For development with validation
   make dev
   ```

5. **Access the application:**
   Navigate to `http://localhost:8080` and sign in with Google

## Configuration

### Environment Variables

**Required:**
- `GOOGLE_CLIENT_ID` - Google OAuth client ID
- `GOOGLE_CLIENT_SECRET` - Google OAuth client secret  
- `GOOGLE_REDIRECT_URL` - OAuth redirect URL

**Optional:**
- `GOOGLE_CLOUD_PROJECT` - Use Google Cloud Datastore (if set)
- `PORT` - Server port (default: 8080)
- `STRIPE_SECRET_KEY` - Stripe secret key for payment processing
- `STRIPE_PUBLISHABLE_KEY` - Stripe publishable key for frontend
- `STRIPE_WEBHOOK_SECRET` - Stripe webhook endpoint secret
- `STRIPE_PRICE_ID` - Stripe price ID for Pro subscription
- `SUBSCRIPTION_ENABLED` - Enable/disable subscription system (default: false)

### Database Configuration

- **Local Development**: SQLite database (`goread2.db`)
- **Production**: Google Cloud Datastore (when `GOOGLE_CLOUD_PROJECT` is set)

### Session Configuration

- **Security**: HTTP-only cookies with secure flags in production
- **Expiration**: 24-hour session lifetime
- **Cleanup**: Automatic cleanup of expired sessions

## Feature Flags

### Subscription System Toggle

Control the subscription system using the `SUBSCRIPTION_ENABLED` environment variable:

```bash
# Enable subscription system
export SUBSCRIPTION_ENABLED=true

# Disable subscription system (default)
export SUBSCRIPTION_ENABLED=false
```

When disabled:
- **Unlimited feeds** for all users
- **No billing** or payment processing
- **Simplified UI** without upgrade prompts

See [Feature Flags Guide](FEATURE-FLAGS.md) for complete details.

## First Run

### 1. Authentication

1. Navigate to the application URL
2. Click "Login with Google" to authenticate
3. Grant necessary permissions
4. You'll be redirected to your personal dashboard

### 2. Managing Feeds

1. **Adding feeds**: Click "Add Feed" and enter RSS/Atom URL
2. **OPML import**: Click "Import OPML" to import feeds from other RSS readers
3. **Feed discovery**: Supports both RSS and Atom formats

### 3. Admin Setup (Optional)

Make yourself an admin user:

```bash
# Replace with your email
./admin.sh admin your-email@gmail.com on
```

See [Admin Guide](admin.md) for complete user management.

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

## Development Setup

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

## Next Steps

- Read the [Deployment Guide](DEPLOYMENT.md) for production setup
- Configure [Stripe payments](STRIPE.md) for subscriptions
- Set up [admin access](ADMIN.md) for user management
- Review the [API documentation](API.md) for integration