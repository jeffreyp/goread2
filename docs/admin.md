# Admin Guide

Complete guide for managing users, permissions, and subscriptions in GoRead2.

## Overview

GoRead2 provides several ways to grant users unlimited access:

1. **Admin Users**: Complete bypass of all subscription limits (permanent)
2. **Free Months**: Temporary unlimited access for a specific duration
3. **Subscription Management**: Handle Stripe subscriptions and billing

## ⚠️ Security Requirements

**CRITICAL:** All admin operations now require authentication tokens to prevent unauthorized access.

### Required Environment Variables

```bash
# Generate a secure admin token
export ADMIN_TOKEN="$(openssl rand -hex 32)"

# For initial admin setup (users must sign in first)
export INITIAL_ADMIN_EMAILS="your-email@gmail.com"
```

## Quick Start

### Initial Admin Setup (Recommended)

The safest way to set up admin access:

1. **Set environment variables:**
   ```bash
   export INITIAL_ADMIN_EMAILS="your-email@gmail.com"
   export ADMIN_TOKEN="$(openssl rand -hex 32)"
   ```

2. **Start the application:**
   ```bash
   go run main.go
   ```

3. **Sign in with Google OAuth** - Admin privileges will be automatically granted

### Manual Admin Setup (Advanced)

If you need to grant admin access after deployment:

```bash
# Set secure token
export ADMIN_TOKEN="$(openssl rand -hex 32)"

# Grant admin access (user must exist - sign in first)
./admin.sh admin your-email@gmail.com on
```

## Admin Script Usage

The `admin.sh` script provides an easy interface for user management.

**⚠️ Security:** ADMIN_TOKEN environment variable is required for all operations.

```bash
# Set secure token first
export ADMIN_TOKEN="$(openssl rand -hex 32)"

# Make script executable (first time only)
chmod +x admin.sh

# List all users (read-only)
./admin.sh list

# Grant admin access (requires ADMIN_TOKEN)
./admin.sh admin your-email@gmail.com on

# Grant 6 free months to a user (requires ADMIN_TOKEN)
./admin.sh grant user@example.com 6

# View user information (read-only)
./admin.sh info user@example.com

# Revoke admin access (requires ADMIN_TOKEN)
./admin.sh admin user@example.com off
```

## Manual Commands

You can also run admin commands directly with proper security tokens:

```bash
# Set security tokens
export ADMIN_TOKEN="$(openssl rand -hex 32)"
export ADMIN_TOKEN_VERIFY="$ADMIN_TOKEN"  # Required for sensitive operations

# List all users (requires ADMIN_TOKEN)
go run cmd/admin/main.go list-users

# Set admin status (requires both tokens)
go run cmd/admin/main.go set-admin your-email@gmail.com true
go run cmd/admin/main.go set-admin user@example.com false

# Grant free months (requires both tokens)
go run cmd/admin/main.go grant-months user@example.com 6

# Show user information (requires ADMIN_TOKEN)
go run cmd/admin/main.go user-info user@example.com
```

## Web-Based Admin API

For programmatic access, use the authenticated admin endpoints:

```bash
# Get user info (requires admin session cookie)
curl -X GET "https://yourdomain.com/admin/users/user@example.com" \
  -H "Cookie: session=your-admin-session"

# Set admin status (requires admin session cookie)
curl -X POST "https://yourdomain.com/admin/users/user@example.com/admin" \
  -H "Content-Type: application/json" \
  -H "Cookie: session=your-admin-session" \
  -d '{"is_admin": true}'

# Grant free months (requires admin session cookie)
curl -X POST "https://yourdomain.com/admin/users/user@example.com/free-months" \
  -H "Content-Type: application/json" \
  -H "Cookie: session=your-admin-session" \
  -d '{"months": 6}'
```

## User Status Types

### Admin Users (`is_admin = true`)

- **Unlimited feeds**: No feed limits whatsoever
- **Bypass all restrictions**: All subscription checks are skipped
- **Status display**: Shows "Admin" or "Unlimited Access" in the UI
- **Permanent**: Remains until explicitly revoked
- **Use case**: Yourself, co-admins, permanent free users

**Benefits:**
- ✅ Unlimited feeds
- ✅ No time limits
- ✅ Purple "ADMIN" badge in UI
- ✅ Bypasses all payment checks

### Free Months (`free_months_remaining > 0`)

- **Temporary unlimited access**: Works like a Pro subscription
- **Automatic decrement**: Months decrease over time (when implemented)
- **Status display**: Shows "Free Months" in the UI
- **Stackable**: Additional months can be added
- **Use case**: Beta testers, temporary promotions, trial extensions

**Benefits:**
- ✅ Unlimited feeds (while months remain)
- ✅ Blue "FREE" badge in UI  
- ⏰ Time-limited access
- ⏰ Eventually need to subscribe or get more free months

### Trial Users (`subscription_status = 'trial'`)

- **Limited feeds**: Maximum 20 feeds
- **Time-limited**: 30 days from account creation
- **Standard behavior**: Default for new users

### Pro Users (`subscription_status = 'active'`)

- **Unlimited feeds**: Paid Stripe subscription
- **Managed by Stripe**: Webhooks handle status updates
- **Billing**: $2.99/month recurring

## Common Admin Tasks

### Setup Admin Access

```bash
# Make yourself admin
./admin.sh admin your-email@gmail.com on

# Make a co-admin
./admin.sh admin coworker@company.com on
```

### Grant Beta Access

```bash
# Grant 3 months free access to beta testers
./admin.sh grant beta-user1@example.com 3
./admin.sh grant beta-user2@example.com 3
./admin.sh grant beta-user3@example.com 3
```

### Check User Status

```bash
# View user details and subscription status
./admin.sh info problematic-user@example.com

# List all users
./admin.sh list

# Check admin users only
./admin.sh list | grep Admin
```

### Revoke Access

```bash
# Remove admin status
./admin.sh admin former-admin@example.com off

# Note: Cannot remove free months once granted
```

## Direct Database Access

For advanced users, you can modify the database directly:

### SQLite (Local Development)

```sql
-- Grant admin access
UPDATE users SET is_admin = 1 WHERE email = 'your-email@gmail.com';

-- Grant 6 free months
UPDATE users SET free_months_remaining = 6 WHERE email = 'user@example.com';

-- Check user status
SELECT email, name, subscription_status, is_admin, free_months_remaining 
FROM users WHERE email = 'user@example.com';

-- List all admin users
SELECT email, name FROM users WHERE is_admin = 1;

-- View subscription overview
SELECT 
  subscription_status,
  COUNT(*) as user_count
FROM users 
GROUP BY subscription_status;
```

### Google Cloud Datastore (Production)

Use the Google Cloud Console web interface for manual updates, or use the admin commands which work with both SQLite and Datastore:

```bash
# Use admin commands (works with both SQLite and Datastore)
export ADMIN_TOKEN="your-admin-token"
./admin.sh admin user@example.com on

# Or direct gcloud CLI for advanced cases
gcloud datastore entities update --kind=User --key=<user-key> --properties=is_admin=true
```

## User Status Hierarchy

The subscription check follows this priority order:

1. **Admin users** (`is_admin = true`) → Unlimited access
2. **Active subscribers** (`subscription_status = 'active'`) → Unlimited access  
3. **Free months remaining** (`free_months_remaining > 0`) → Unlimited access
4. **Trial users** → Limited to 20 feeds, expires after 30 days
5. **Expired users** → No access

## Status Display in UI

Users see different indicators based on their access level:

- **Admin**: "Admin - Unlimited Access" with purple badge
- **Pro**: "GoRead2 Pro - Unlimited feeds" with green badge
- **Free Months**: "Free Months - X months remaining" with blue badge
- **Trial**: "Trial - X/20 feeds, Y days left" with orange badge
- **Expired**: "Trial Expired - Subscribe to continue" with red warning

## Database Schema

The subscription-related fields in the users table:

```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    google_id TEXT UNIQUE NOT NULL,
    email TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    avatar TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    -- Subscription fields
    subscription_status TEXT DEFAULT 'trial',        -- 'trial', 'active', 'cancelled', 'expired'
    subscription_id TEXT,                            -- Stripe subscription ID
    trial_ends_at DATETIME,                         -- When free trial expires
    last_payment_date DATETIME,                     -- Last successful payment
    is_admin BOOLEAN DEFAULT 0,                     -- Admin users bypass limits
    free_months_remaining INTEGER DEFAULT 0         -- Free months granted
);
```

## Environment-Based Admin (Development)

Set admin users via environment variables during development:

```bash
# Set admin emails (comma-separated)
export ADMIN_EMAILS="admin@example.com,owner@example.com"
```

Then modify the auth service to check this list during user login.

## Subscription Management

### Stripe Integration

When subscriptions are enabled (`SUBSCRIPTION_ENABLED=true`):

- **Webhook handling**: Stripe webhooks update subscription status
- **Customer portal**: Users can manage billing through Stripe
- **Subscription lifecycle**: Automatic handling of payments, cancellations

### Manual Subscription Operations

```bash
# SECURITY: Set admin token first
export ADMIN_TOKEN="$(openssl rand -hex 32)"

# Check subscription system status
go run cmd/admin/main.go system-info

# View user subscription details
go run cmd/admin/main.go user-info user@example.com

# Note: Stripe subscriptions are managed via webhooks
# Manual subscription changes should be done through Stripe Dashboard
```

## Security Considerations

### Admin Access

- **Admin users have unlimited access** - only grant to trusted individuals
- **No billing bypass**: Admin users still see upgrade prompts (but can ignore limits)
- **Audit trail**: Consider logging admin actions in production
- **Revocation**: Admin status can be revoked at any time

### Free Months

- **Monitor usage**: Track free month consumption
- **Limited time**: Consider implementing automatic expiration
- **User experience**: Clearly communicate free month status
- **Stackable**: Multiple grants accumulate

### Best Practices

- **Principle of least privilege**: Only grant admin to those who need it
- **Regular audits**: Periodically review admin user list
- **Documentation**: Keep records of why admin access was granted
- **Monitoring**: Watch for unusual usage patterns

## Troubleshooting

### User Not Found

```bash
# Check email spelling and case sensitivity
./admin.sh info user@example.com

# Verify user has logged in at least once
./admin.sh list | grep user@example.com
```

**Solutions:**
- User must have logged in at least once (account must exist)
- Check email spelling and case sensitivity
- Verify Google OAuth is working

### Database Locked

```bash
# Stop GoRead2 before running admin commands
sudo systemctl stop goread2
./admin.sh admin user@example.com on
sudo systemctl start goread2
```

**Solutions:**
- Stop the GoRead2 application before running admin commands
- Check file permissions on SQLite database
- Ensure only one process accesses SQLite at a time

### Changes Not Reflected

```bash
# Restart the application
sudo systemctl restart goread2

# Or for development
# Stop and restart: go run main.go
```

**Solutions:**
- Restart the GoRead2 application
- Have user log out and log back in
- Clear browser cache/cookies
- Check server logs for errors

### Stripe Integration Conflicts

Admin and free month users still see Stripe UI elements:
- This is by design - they can ignore payment prompts
- Consider hiding payment UI for admin users in future updates
- Admin status overrides subscription requirements

## Monitoring and Analytics

### User Statistics

```bash
# SECURITY: Set admin token first
export ADMIN_TOKEN="$(openssl rand -hex 32)"

# View user breakdown
./admin.sh list

# Check system configuration
go run cmd/admin/main.go system-info
```

### Usage Tracking

Monitor admin user activity:
- Feed subscription patterns
- Article reading behavior
- System resource usage
- Support ticket trends

## Future Enhancements

### Planned Features

- **Automatic free month expiration**: Decrement monthly via cron job
- **Admin dashboard**: Web UI for user management
- **Usage analytics**: Track admin and free user activity
- **Bulk operations**: Import/export admin user lists
- **Audit logging**: Track all admin actions with timestamps

### Customization Ideas

- **Organization admin**: Company-wide admin access
- **Temporary admin**: Time-limited admin privileges
- **Feed quotas**: Custom feed limits per user
- **Feature flags**: Enable/disable features per user
- **Role-based access**: Different permission levels

## Migration and Backup

### Backup Admin Settings

```bash
# Export admin users
sqlite3 goread2.db "SELECT email, is_admin, free_months_remaining FROM users WHERE is_admin = 1 OR free_months_remaining > 0;" > admin_backup.csv
```

### Restore Admin Settings

```bash
# Import admin users (modify as needed)
# This is a manual process - review backup file and apply changes
```

## API Integration

### Programmatic User Management

For advanced integrations, you can manage users programmatically:

```go
// Example: Grant admin access via API
import "goread2/internal/database"

db, _ := database.InitDB()
err := db.SetUserAdmin(userID, true)
if err != nil {
    log.Printf("Failed to set admin: %v", err)
}
```

See [API Reference](api.md) for complete endpoint documentation.

This comprehensive admin system provides flexible user management while maintaining security and a good user experience.