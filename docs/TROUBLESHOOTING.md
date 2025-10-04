# Troubleshooting Guide

Common issues and solutions for GoRead2.

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
# Check session configuration
grep -i session logs/goread2.log

# Verify HTTPS in production (required for secure cookies)
curl -I https://your-domain.com/

# Clear browser cookies and retry
# In browser: DevTools → Storage → Cookies → Clear
```

**Common Causes**:
- HTTP instead of HTTPS in production
- Incorrect cookie domain settings
- Session cleanup too aggressive

### Login Redirect Loops

**Problem**: Continuous redirects during login process

**Solutions**:
- Check OAuth consent screen is published (not in testing)
- Verify user is added to test users (if in testing mode)
- Check browser blocks third-party cookies
- Ensure redirect URL protocol matches (HTTP vs HTTPS)

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

### SQLite Database Locked

**Problem**: "Database is locked" errors

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
- Database corruption

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

**Problem**: Database errors or inconsistent data

**Solutions**:
```bash
# Check database integrity
sqlite3 goread2.db "PRAGMA integrity_check;"

# Backup and recreate (DESTRUCTIVE)
cp goread2.db goread2_backup.db
rm goread2.db
# Restart application to recreate schema

# Restore from backup if available
```

## Subscription Issues

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

# Test product/price exists
go run cmd/setup-stripe/main.go list-products
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

## Performance Issues

### Slow Page Loading

**Problem**: Application takes 5+ seconds to load

**Solutions**:
```bash
# Check database query performance
sqlite3 goread2.db ".timer on"
sqlite3 goread2.db "SELECT * FROM feeds LIMIT 10;"

# Monitor network requests in browser DevTools
# Look for slow API calls

# Check for database indexes
sqlite3 goread2.db ".schema" | grep INDEX
```

**Common Causes**:
- Missing database indexes
- Large number of feeds/articles
- Slow external feed fetching
- Inefficient database queries

### High Memory Usage

**Problem**: GoRead2 consuming excessive memory

**Solutions**:
```bash
# Monitor memory usage
top -p $(pgrep goread2)

# Check for memory leaks
go tool pprof http://localhost:8080/debug/pprof/heap

# Review session cleanup
grep -i "session.*cleanup" logs/goread2.log
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

# Consider reducing concurrent fetches
```

**Common Causes**:
- Slow external RSS servers
- Too many feeds refreshed simultaneously
- Network connectivity issues
- Large feed content

## Deployment Issues

### App Engine Deployment Failures

**Problem**: Deployment fails

**Solutions**:
```bash
# Use Makefile for deployment (recommended)
# Development deployment with validation
make deploy-dev

# Production deployment with strict validation and tests
make deploy-prod

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

### Docker Build Issues

**Problem**: Docker build fails or container won't start

**Solutions**:
```bash
# Build with verbose output
docker build --no-cache -t goread2 .

# Check Go build within container
docker run -it goread2 go version

# Inspect container filesystem
docker run -it --entrypoint /bin/sh goread2

# Check container logs
docker logs container_name
```

**Common Causes**:
- Missing dependencies in Dockerfile
- Go build failures
- Missing static files
- Port binding issues

### SSL/HTTPS Issues

**Problem**: HTTPS not working or certificate errors

**Solutions**:
```bash
# Test SSL certificate
openssl s_client -connect your-domain.com:443

# Check certificate expiration
curl -I https://your-domain.com/

# Verify Let's Encrypt renewal
sudo certbot certificates

# Test with curl
curl -v https://your-domain.com/
```

**Common Causes**:
- Expired SSL certificates
- Incorrect nginx configuration
- DNS pointing to wrong server
- Firewall blocking HTTPS

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
make test          # Run test suite

# Manual troubleshooting
go mod tidy
go clean -modcache
go version
grep -r "os.Getenv" *.go
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

# Check application logs
tail -f logs/goread2.log

# Database query logging (development only)
export DB_DEBUG=true
```

### Common Log Locations

```bash
# Local development
./logs/goread2.log

# App Engine
gcloud app logs tail -s default

# Docker
docker logs container_name

# systemd service
sudo journalctl -u goread2 -f
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

### Check Documentation
- [Setup Guide](SETUP.md) - Installation and configuration
- [Deployment Guide](DEPLOYMENT.md) - Production deployment
- [Admin Guide](ADMIN.md) - User management
- [API Reference](API.md) - API documentation

### Collect Debug Information

When reporting issues, include:

```bash
# System information
go version
cat app.yaml  # or docker-compose.yml
env | grep -E "(GOOGLE|STRIPE|SUBSCRIPTION)"

# Application logs
tail -100 logs/goread2.log

# Database state
./admin.sh list
./admin.sh info your-email@example.com

# Test results
./test.sh 2>&1 | tail -50
```

### Report Issues

- **GitHub Issues**: [goread2/issues](https://github.com/jeffreyp/goread2/issues)
- **Include**: Error messages, logs, configuration, steps to reproduce
- **Environment**: Local development, Docker, App Engine, etc.

### Community Support

- **GitHub Discussions**: For questions and general help
- **Stack Overflow**: Tag questions with `goread2`
- **Documentation**: Check all docs before asking questions

This troubleshooting guide covers the most common issues. If you encounter problems not listed here, please check the logs carefully and report the issue with detailed information.