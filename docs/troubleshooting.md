# Troubleshooting Guide

Common issues and solutions for GoRead2, covering authentication, feeds, database, subscriptions, admin, performance, deployment, monitoring, and development.

## Table of Contents

- [Authentication Issues](#authentication-issues)
- [Feed Issues](#feed-issues)
- [Database Issues](#database-issues)
- [Subscription Issues](#subscription-issues)
- [Admin Issues](#admin-issues)
- [Performance Issues](#performance-issues)
- [Deployment Issues](#deployment-issues)
- [Monitoring Issues](#monitoring-issues)
- [Development Issues](#development-issues)
- [Logging and Debugging](#logging-and-debugging)
- [Getting Help](#getting-help)
- [Related Documentation](#related-documentation)

## Authentication Issues

### OAuth Configuration Errors

**Problem**: "OAuth configuration error" or "Invalid client ID"

**Solutions**:
```bash
# Verify environment variables are set
echo $GOOGLE_CLIENT_ID
echo $GOOGLE_CLIENT_SECRET
echo $GOOGLE_REDIRECT_URL

# Check redirect URL matches exactly (no trailing slash)
# Development: http://localhost:8080/auth/callback
# Production: https://your-domain.com/auth/callback
```

**Common Causes**:
- Missing or incorrect environment variables
- Redirect URL mismatch in Google Cloud Console
- OAuth consent screen not configured

### Session Problems

**Problem**: Users logged out frequently or sessions not persisting

**Solutions**:
```bash
# Verify HTTPS in production (required for secure cookies)
curl -I https://your-domain.com/

# Clear browser cookies and retry
# In browser: DevTools → Storage → Cookies → Clear
```

**Common Causes**:
- HTTP instead of HTTPS in production
- Incorrect cookie domain settings
- Session cleanup too aggressive

See [authentication.md](authentication.md#session-management) for how sessions are created, stored, and cleaned up.

### Login Redirect Loops

**Problem**: Continuous redirects during login process

**Solutions**:
- Check OAuth consent screen is published (not in testing)
- Verify user is added to test users (if in testing mode)
- Check browser blocks third-party cookies
- Ensure redirect URL protocol matches (HTTP vs HTTPS)

### "The OAuth State Parameter Is Not Valid" Error

**Common Causes**:
- Cookie was deleted between login initiation and callback
- Request came from a different browser/incognito window
- OAuth state cookie expired (10 minute timeout)

**Solution**: Try logging in again.

### Logged Out When Switching Between Local and Production

**Common Causes**:
- Using an old build predating environment-specific cookie names (see [authentication.md](authentication.md#environment-isolation))
- Stale cookies from a previous session

**Solution**: Clear cookies and log in again; verify environment variables are set correctly.

### Session Expires Too Quickly

Sessions last 7 days by default.

**Common Causes**:
- Server clock is incorrect
- Sessions aren't persisting in the database
- Browser cookie settings are clearing cookies early

### Can't Stay Logged In

**Common Causes**:
- Browser blocking cookies
- Incognito/private browsing mode
- Cookie settings too restrictive

**Solution**: Allow cookies for the application domain.

## Feed Issues

### Feed Discovery Failures

**Problem**: "Failed to fetch feed" or "Invalid feed URL"

**Solutions**:
```bash
# Test feed URL manually
curl -I "https://example.com/feed.xml"

# Check common feed locations
curl -I "https://example.com/rss"
curl -I "https://example.com/atom.xml"
curl -I "https://example.com/index.xml"

# Verify feed format
curl "https://example.com/feed.xml" | head -20
```

**Common Causes**:
- Invalid or moved feed URL
- Feed requires authentication
- Server blocks automated requests
- Malformed RSS/Atom XML

### Feeds Not Updating

**Problem**: Articles not appearing despite feed updates

**Solutions**:
```bash
# Check manual refresh
curl -X POST "http://localhost:8080/api/feeds/refresh" \
  -H "Cookie: session=..."

# Check cron job (App Engine)
gcloud app logs tail -s default | grep refresh

# Verify feed last_fetch timestamp
sqlite3 goread2.db "SELECT title, last_fetch FROM feeds;"
```

**Common Causes**:
- Background refresh not configured
- Feed URL changed or blocked
- Server timezone issues
- Network connectivity problems

### OPML Import Issues

**Problem**: OPML import fails or imports incomplete

**Solutions**:
```bash
# Validate OPML file format
xmllint --format subscriptions.opml

# Check file size (max 10MB)
ls -lh subscriptions.opml

# Test with smaller OPML subset
head -50 subscriptions.opml > test.opml
```

**Common Causes**:
- Malformed OPML XML
- File too large
- Feed limit reached during import
- Nested folder structure unsupported

## Database Issues

Local development uses SQLite (`goread2.db`); production uses Google Cloud Datastore. See [setup.md](setup.md#database-configuration) for how the database is selected.

### SQLite Database Locked

**Problem**: "Database is locked" errors (local development only)

**Solutions**:
```bash
# Stop all GoRead2 instances
pkill goread2

# Check for zombie processes
ps aux | grep goread2

# Fix permissions
sudo chown $USER:$USER goread2.db
chmod 644 goread2.db

# Remove lock file if exists
rm -f goread2.db.lock
```

**Common Causes**:
- Multiple GoRead2 instances running
- Improper shutdown leaving connections open
- File permission issues

### User Data Isolation Failures

**Problem**: Users seeing other users' data

**Solutions**:
```bash
# Run isolation tests
go test ./test/integration/... -v -run TestUserIsolation

# Check user session
curl "http://localhost:8080/auth/me" -H "Cookie: session=..."

# Verify database queries include user filtering
grep -r "user_id" internal/database/
```

**Common Causes**:
- Missing user_id filters in database queries
- Session middleware not working
- Database schema issues

### Database Corruption

**Problem**: Database errors or inconsistent data (local SQLite only)

**Solutions**:
```bash
# Check database integrity
sqlite3 goread2.db "PRAGMA integrity_check;"

# Backup and recreate (DESTRUCTIVE)
cp goread2.db goread2_backup.db
rm goread2.db
# Restart application to recreate schema
```

## Subscription Issues

### Subscription Routes Return 404

**Problem**: Subscription-related API routes return 404

**Solutions**:
- Check `SUBSCRIPTION_ENABLED=true` is set
- Verify Stripe configuration (see below)

**Common Causes**:
- `SUBSCRIPTION_ENABLED` unset or `false`; routes are only registered when it's enabled, see [feature-flags.md](feature-flags.md)

### Stripe Configuration Problems

**Problem**: "Stripe not configured" or payment failures

**Solutions**:
```bash
# Validate Stripe configuration
go run cmd/setup-stripe/main.go validate

# Check environment variables
echo $SUBSCRIPTION_ENABLED
echo $STRIPE_SECRET_KEY
echo $STRIPE_PUBLISHABLE_KEY
```

**Common Causes**:
- Missing Stripe environment variables
- Using test keys in production (or vice versa)
- Product/price not created
- Webhook secret mismatch

### Feed Limit Not Enforced

**Problem**: Users can add unlimited feeds when they shouldn't

**Solutions**:
```bash
# Check subscription system is enabled
echo $SUBSCRIPTION_ENABLED

# Verify user subscription status
./admin.sh info user@example.com

# Check subscription service logic
grep -r "CanUserAddFeed" internal/services/
```

**Common Causes**:
- `SUBSCRIPTION_ENABLED=false`
- User has admin privileges
- User has free months remaining
- Subscription check bypassed

### Webhook Failures

**Problem**: Stripe webhooks returning errors

**Solutions**:
```bash
# Check webhook endpoint
curl -X POST https://your-domain.com/webhooks/stripe \
  -H "Content-Type: application/json" \
  -d '{"type": "ping"}'

# Verify webhook secret
echo $STRIPE_WEBHOOK_SECRET

# Check webhook logs in Stripe Dashboard
# Go to Developers → Webhooks → View logs
```

**Common Causes**:
- Webhook endpoint not publicly accessible
- Incorrect webhook secret
- HTTPS required for production
- Webhook signature verification failing

### Product/Price Not Found

**Problem**: Checkout fails to load, or logs show "Price not found"

**Solutions**:
```bash
# Recreate product and price
go run cmd/setup-stripe/main.go create-product

# Update STRIPE_PRICE_ID with the new price ID
export STRIPE_PRICE_ID=price_new_id_here
```

### Payment Fails in Production

**Problem**: Stripe test cards work, but real cards are declined or checkout errors in production

**Solutions**:
- Switch to live API keys (`sk_live_`/`pk_live_`) in production
- Ensure the webhook endpoint uses HTTPS
- Check the Stripe Dashboard for detailed error messages
- Verify business information is complete in the Stripe account

See [stripe.md](stripe.md) for the full Stripe setup and webhook configuration.

## Admin Issues

### User Not Found

**Solutions**:
```bash
# Check email spelling and case sensitivity
./admin.sh info user@example.com

# Verify user has logged in at least once
./admin.sh list | grep user@example.com
```

**Common Causes**:
- User must have logged in at least once (account must exist)
- Email spelling or case mismatch

### Database Locked When Running Admin Commands

**Solutions**:
```bash
# Stop the local GoRead2 process before running admin commands
pkill goread2
./admin.sh admin user@example.com on
```

**Common Causes**:
- The application is still running locally against the same SQLite file
- Only one process can access the local SQLite database at a time

### Admin Changes Not Reflected

**Solutions**:
- Restart the local GoRead2 process (or redeploy, in production)
- Have the user log out and log back in
- Clear browser cache/cookies

### Stripe Integration Conflicts

Admin and free-month users may still see Stripe UI elements. This is by design: admin status overrides subscription requirements, so any payment prompts can be ignored.

### Admin Token Error Messages

- **"ADMIN_TOKEN must be exactly 64 characters (32 bytes as hex)"**: Token format is invalid. Use `create-token` to generate a valid one.
- **"Invalid ADMIN_TOKEN - token not found in database or inactive"**: Token doesn't exist or was revoked. Generate a new one with `create-token`.
- **"ADMIN_TOKEN environment variable must be set"**: Set `ADMIN_TOKEN` to a valid 64-character token.
- **"No admin users found in database"** (bootstrap): create a user account and set it as admin in the database first.
- **"Admin tokens already exist"**: shown when creating additional tokens; confirm with `y` if authorized.

See [admin.md](admin.md) for the full admin token and CLI reference.

## Performance Issues

### Slow Page Loading

**Problem**: Application takes 5+ seconds to load

**Solutions**:
```bash
# Check database query performance (local SQLite)
sqlite3 goread2.db ".timer on"
sqlite3 goread2.db "SELECT * FROM feeds LIMIT 10;"

# Monitor network requests in browser DevTools
# Look for slow API calls

# Check for database indexes (local SQLite)
sqlite3 goread2.db ".schema" | grep INDEX
```

**Common Causes**:
- Missing database indexes
- Large number of feeds/articles
- Slow external feed fetching
- Inefficient database queries

See [performance.md](performance.md) for the optimizations already in place.

### High Memory Usage

**Problem**: GoRead2 consuming excessive memory

**Solutions**:
```bash
# Monitor memory usage
top -p $(pgrep goread2)

# Check for memory leaks
go tool pprof http://localhost:8080/debug/pprof/heap
```

**Common Causes**:
- Database connection leaks
- Session cleanup not running
- Large feed content cached in memory
- Background tasks accumulating

### Feed Refresh Timeouts

**Problem**: Feed refresh takes too long or times out

**Solutions**:
```bash
# Check feed fetch timeout settings
grep -r "timeout" internal/services/feed_*

# Test slow feeds manually
time curl "https://slow-feed.example.com/rss"
```

**Common Causes**:
- Slow external RSS servers
- Too many feeds refreshed simultaneously
- Network connectivity issues
- Large feed content

## Deployment Issues

GoRead2 deploys exclusively to Google App Engine; see [deployment.md](deployment.md) for the full pipeline.

### App Engine Deployment Failures

**Problem**: Deployment fails

**Solutions**:
```bash
# Deploys are automated via GitHub Actions (see docs/deployment.md):
# push to main for staging, or trigger production manually:
gh workflow run deploy-prod.yml --repo jeffreyp/goread2

# Manual deployment debugging
gcloud app deploy --dry-run

# Verify Go version compatibility
grep runtime app.yaml

# Check environment variable format
grep -A 20 env_variables app.yaml

# Review deployment logs
gcloud app logs tail -s default
```

**Common Causes**:
- Invalid app.yaml syntax
- Missing required environment variables
- Go version incompatibility
- Resource limits exceeded

## Monitoring Issues

### Dashboard Not Showing Data

**Solutions**:
- Verify that the app is deployed and receiving traffic
- Check that metrics are being generated in Metrics Explorer
- Datastore metrics may take a few minutes to appear after deployment

### Alerts Not Firing

**Solutions**:
- Confirm notification channels are configured
- Check alert policy status in Cloud Console
- Verify that thresholds are being exceeded using Metrics Explorer

### Permission Errors

**Required IAM roles**:
- `roles/monitoring.dashboardEditor` - Create/edit dashboards
- `roles/monitoring.alertPolicyEditor` - Create/edit alerting policies

See [monitoring.md](monitoring.md) for the full dashboard and alerting setup.

## Development Issues

### Build Failures

**Problem**: Build fails

**Solutions**:
```bash
# Use Makefile for complete build (recommended)
make all

# Build specific components
make build          # Build Go application
make build-frontend # Build JS/CSS assets
make test           # Run test suite

# Manual troubleshooting
go mod tidy
go clean -modcache
go version
```

**Common Causes**:
- Outdated dependencies
- Go version incompatibility
- Missing environment variables
- Import path issues

### Test Failures

**Problem**: Tests failing unexpectedly

**Solutions**:
```bash
# Use Makefile for testing (recommended)
make test

# Manual test troubleshooting
go test -v ./test/...
go test -v -run TestSpecificFunction ./test/unit/

# Check test environment
env | grep -E "(GOOGLE|STRIPE)"

# Clean test databases
rm -f test_*.db
```

**Common Causes**:
- Missing test dependencies
- Environment variable conflicts
- Test data isolation issues
- Timing-dependent test failures

**CI-specific**: a red `Tests` run on GitHub covers six separate jobs in `.github/workflows/test.yml`: `test` (Go unit/integration), `lint` (golangci-lint), `frontend-build` (ESLint + Jest + `make build-frontend`), `benchmark` (regression-gated Go benchmarks), `security` (`test/security/` + `govulncheck`), and `build`. Check which job actually failed in the Actions tab before assuming it's a test logic problem; see [testing.md](testing.md#cicd-integration) for what each job runs.

### Frontend Issues

**Problem**: JavaScript errors or UI not working

**Solutions**:
```bash
# Check browser console for errors
# Open DevTools → Console

# Verify static files served correctly
curl -I http://localhost:8080/static/js/app.min.js

# Check for missing dependencies
ls web/static/js/
ls web/static/css/

# Test frontend in isolation
npm test
```

**Common Causes**:
- JavaScript syntax errors
- Missing static files
- CSS/JavaScript not minified
- Browser caching old files

## Logging and Debugging

### Enable Debug Logging

```bash
# Development mode with verbose logging
export GIN_MODE=debug
go run main.go
```

GoRead2 logs to stdout; there is no log file. Locally that means the terminal running the
process. In production, use `gcloud app logs tail -s default` (see below).

### Common Log Locations

```bash
# Local development: stdout of the running process (go run main.go / make dev)

# App Engine
gcloud app logs tail -s default
```

### Debug Commands

```bash
# SECURITY: Set admin token first
export ADMIN_TOKEN="$(openssl rand -hex 32)"

# Check system status
go run cmd/admin/main.go system-info

# Validate configuration
go run cmd/setup-stripe/main.go validate

# Test database connection
go run cmd/admin/main.go list-users

# Check feed fetching
curl -v "https://example.com/feed.xml"
```

## Getting Help

### Collect Debug Information

When reporting issues, include:

```bash
# System information
go version
cat app.yaml
env | grep -E "(GOOGLE|STRIPE|SUBSCRIPTION)"

# Database state
./admin.sh list
./admin.sh info your-email@example.com

# Test results
./test.sh 2>&1 | tail -50
```

### Report Issues

- **GitHub Issues**: [goread2/issues](https://github.com/jeffreyp/goread2/issues)
- **Include**: Error messages, logs, configuration, steps to reproduce
- **Environment**: Local development or App Engine (staging/production)

## Related Documentation

- [Setup Guide](setup.md) - Installation and configuration
- [Deployment Guide](deployment.md) - Production deployment
- [Admin Guide](admin.md) - User management
- [API Reference](api.md) - API documentation
