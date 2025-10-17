# Admin Guide

Complete guide for managing users, permissions, and subscriptions in GoRead2.

## Table of Contents

- [Overview](#overview)
- [Security Requirements](#security-requirements)
- [Quick Start](#quick-start)
- [Admin Token System](#admin-token-system)
- [Admin Commands](#admin-commands)
- [Audit Logging](#audit-logging)
- [Web-Based Admin API](#web-based-admin-api)
- [User Status Types](#user-status-types)
- [Common Admin Tasks](#common-admin-tasks)
- [Direct Database Access](#direct-database-access)
- [Subscription Management](#subscription-management)
- [Security Considerations](#security-considerations)
- [Troubleshooting](#troubleshooting)
- [Database Schema](#database-schema)

## Overview

GoRead2 provides several ways to grant users unlimited access:

1. **Admin Users**: Complete bypass of all subscription limits (permanent)
2. **Free Months**: Temporary unlimited access for a specific duration
3. **Subscription Management**: Handle Stripe subscriptions and billing

## Security Requirements

**CRITICAL:** All admin operations now require database-based token authentication to prevent unauthorized access.

### Admin Token Security System

The GoRead2 admin system uses a secure database-based token authentication system:

**Security Features:**
- Cryptographically secure 64-character (32-byte) random tokens
- SHA-256 hashed storage in database
- Token validation against database records
- Token revocation and usage tracking
- Bootstrap protection with security warnings

**Previous Vulnerability Fixed:** The original system only required environment variables, allowing anyone with server access to create admin tokens. The new system uses database-validated tokens with cryptographic hashing.

### Database Compatibility

The admin token system works with both database backends:

**SQLite (Local Development)**:
- Admin tokens stored in `admin_tokens` table
- Direct SQL queries for validation and management
- File-based database persistence

**Google Datastore (Production/GAE)**:
- Admin tokens stored as `AdminToken` entities
- Datastore queries with filters for validation
- Cloud-native scaling and persistence
- Automatic detection via `GOOGLE_CLOUD_PROJECT` environment variable

## Quick Start

### Initial Setup (Bootstrap)

#### Local Development (SQLite)

**Step 1: Create First User and Admin**
1. Start the web application and create your first user account through the normal signup process
2. Set the user as admin by directly updating the database:
   ```bash
   sqlite3 goread2.db "UPDATE users SET is_admin = 1 WHERE email = 'your@email.com';"
   ```

**Step 2: Create First Admin Token**
```bash
# Bootstrap requires an existing admin user in database
ADMIN_TOKEN="bootstrap" go run cmd/admin/main.go create-token "Initial setup"
```

This will output a secure token like:
```
Token: e2de0bb29b84ea94d2785c1ebfc65b450c767078f52e0e6df1d68d9e0fd13b08
```

**Step 3: Set Environment Variable**
```bash
export ADMIN_TOKEN="e2de0bb29b84ea94d2785c1ebfc65b450c767078f52e0e6df1d68d9e0fd13b08"
```

**Step 4: Verify Setup**
```bash
go run cmd/admin/main.go list-users
```

#### Google App Engine Deployment (Datastore)

**Step 1: Create First User and Admin**
1. Deploy the application to Google App Engine
2. Create your first user account through the web interface
3. Set the user as admin using Google Cloud Console web interface to manually set `is_admin = true`, or using gcloud CLI:
   ```bash
   gcloud datastore entities update --kind=User --key-id=USER_ID --set-property=is_admin:boolean=true
   ```

**Step 2: Create First Admin Token**
```bash
# Set environment for GAE deployment
export GOOGLE_CLOUD_PROJECT="your-project-id"
ADMIN_TOKEN="bootstrap" go run cmd/admin/main.go create-token "GAE Initial setup"
```

**Step 3: Set Environment Variable and Verify**
```bash
export ADMIN_TOKEN="your_generated_token"
go run cmd/admin/main.go list-users
```

### Alternative: Initial Admin Setup via Environment Variable

The safest way to set up admin access for new deployments:

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

## Admin Token System

### Token Management

#### Admin Token Properties
- **Format**: 64-character hexadecimal string (32 random bytes)
- **Storage**: SHA-256 hash stored in `admin_tokens` table
- **Security**: Tokens are never stored in plain text
- **Tracking**: Creation time, last used time, and active/revoked status
- **Descriptions**: Human-readable descriptions for token identification

#### ADMIN_TOKEN Environment Variable
- **Purpose**: Contains the actual admin token for authentication
- **Format**: Exactly 64 hexadecimal characters
- **Security**: Validated against database on every use
- **Usage**: Required for all admin commands except initial bootstrap

### Token Lifecycle Management

#### Creating Additional Tokens
```bash
# With existing valid token
ADMIN_TOKEN="your_existing_token" go run cmd/admin/main.go create-token "CI/CD automation"
```

**Security Warning**: Creating additional tokens when others exist requires confirmation to prevent unauthorized token creation.

#### Listing Active Tokens
```bash
ADMIN_TOKEN="your_token" go run cmd/admin/main.go list-tokens
```

#### Revoking Compromised Tokens
```bash
# Revoke token by ID (from list-tokens output)
ADMIN_TOKEN="your_token" go run cmd/admin/main.go revoke-token 2
```

### Migration from Previous System

If you were using the old environment variable system:

1. **Remove old variables**:
   ```bash
   unset ADMIN_TOKEN_VERIFY
   unset SERVER_SECRET_KEY
   ```

2. **Bootstrap new system**:
   ```bash
   ADMIN_TOKEN="bootstrap" go run cmd/admin/main.go create-token "Migration to secure system"
   ```

3. **Set new secure token**:
   ```bash
   export ADMIN_TOKEN="new_generated_64_char_token_from_step_2"
   ```

## Admin Commands

### Admin Script Usage

The `admin.sh` script provides an easy interface for user management.

**Security:** ADMIN_TOKEN environment variable is required for all operations.

```bash
# Set secure token first
export ADMIN_TOKEN="your_64_char_token"

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

### Manual Commands

You can also run admin commands directly:

```bash
# Set security token
export ADMIN_TOKEN="your_64_char_token"

# Token Management
go run cmd/admin/main.go create-token "description"
go run cmd/admin/main.go list-tokens
go run cmd/admin/main.go revoke-token <token-id>

# User Management
go run cmd/admin/main.go list-users
go run cmd/admin/main.go user-info user@example.com
go run cmd/admin/main.go set-admin your-email@gmail.com true
go run cmd/admin/main.go set-admin user@example.com false
go run cmd/admin/main.go grant-months user@example.com 6
```

**Security**: All commands except `create-token` (bootstrap mode) require valid database token authentication.

## Audit Logging

GoRead2 automatically logs all admin operations to provide a comprehensive audit trail for security and compliance.

### What Gets Logged

All admin operations are automatically logged with the following details:
- **Timestamp**: When the operation occurred
- **Admin User**: Who performed the action (ID and email)
- **Operation Type**: What action was performed
- **Target User**: Which user was affected (ID and email)
- **Operation Details**: Additional context (JSON format)
- **IP Address**: Where the request came from (for web operations)
- **Result**: Success or failure
- **Error Message**: Details if the operation failed

### Logged Operations

The following admin operations are automatically logged:

**User Management:**
- `grant_admin` - Admin privileges granted to a user
- `revoke_admin` - Admin privileges revoked from a user
- `grant_free_months` - Free months granted to a user
- `view_user_info` - Admin viewed user details

**CLI Operations:**
- All CLI admin commands are logged with admin_email="CLI_ADMIN" and IP address="CLI"

### Viewing Audit Logs

#### CLI Command

```bash
# View recent audit logs (default: last 50)
ADMIN_TOKEN="your_token" go run cmd/admin/main.go audit-logs

# Limit number of results
ADMIN_TOKEN="your_token" go run cmd/admin/main.go audit-logs --limit 100

# Filter by operation type
ADMIN_TOKEN="your_token" go run cmd/admin/main.go audit-logs --operation grant_admin
```

**Output Format:**
```
Recent Audit Logs (Last 50):
================================================================================

[2025-10-16 10:30:45] SUCCESS
  Admin: admin@example.com (ID: 1)
  Operation: grant_admin
  Target: user@example.com (ID: 123)
  IP: 192.168.1.100
  Details:
    - is_admin: true
    - user_name: John Doe

[2025-10-16 09:15:22] FAILURE
  Admin: CLI_ADMIN (ID: 0)
  Operation: grant_free_months
  Target: unknown@example.com (ID: 0)
  IP: CLI
  Error: user not found
  Details:
    - months_granted: 6
```

#### Web API

```bash
# Get audit logs via API (requires admin session)
curl -X GET "https://yourdomain.com/admin/audit-logs?limit=50&offset=0" \
  -H "Cookie: session=your-admin-session"

# Filter by admin user
curl -X GET "https://yourdomain.com/admin/audit-logs?admin_user_id=1" \
  -H "Cookie: session=your-admin-session"

# Filter by target user
curl -X GET "https://yourdomain.com/admin/audit-logs?target_user_id=123" \
  -H "Cookie: session=your-admin-session"

# Filter by operation type
curl -X GET "https://yourdomain.com/admin/audit-logs?operation_type=grant_admin" \
  -H "Cookie: session=your-admin-session"
```

**Response Format:**
```json
{
  "logs": [
    {
      "id": 1,
      "timestamp": "2025-10-16T10:30:45Z",
      "admin_user_id": 1,
      "admin_email": "admin@example.com",
      "operation_type": "grant_admin",
      "target_user_id": 123,
      "target_user_email": "user@example.com",
      "operation_details": "{\"is_admin\":true,\"user_name\":\"John Doe\"}",
      "ip_address": "192.168.1.100",
      "result": "success",
      "error_message": ""
    }
  ],
  "limit": 50,
  "offset": 0
}
```

### Audit Log Retention

**SQLite (Local Development):**
- Audit logs are stored indefinitely in the `audit_logs` table
- Consider implementing periodic cleanup for very old logs

**Google Datastore (Production):**
- Audit logs are stored as `AuditLog` entities
- Consider implementing automatic expiration policies
- Monitor Datastore costs for high-volume audit logging

### Security and Compliance

**Audit Trail Benefits:**
- Track all administrative actions
- Investigate security incidents
- Demonstrate compliance with security policies
- Monitor for unauthorized access attempts
- Identify patterns of admin activity

**Best Practices:**
- Regularly review audit logs for suspicious activity
- Export and archive logs for long-term compliance
- Monitor failed operations for security issues
- Use audit logs to track admin privilege grants/revocations

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

### User Status Hierarchy

The subscription check follows this priority order:

1. **Admin users** (`is_admin = true`) → Unlimited access
2. **Active subscribers** (`subscription_status = 'active'`) → Unlimited access
3. **Free months remaining** (`free_months_remaining > 0`) → Unlimited access
4. **Trial users** → Limited to 20 feeds, expires after 30 days
5. **Expired users** → No access

### Status Display in UI

Users see different indicators based on their access level:

- **Admin**: "Admin - Unlimited Access" with purple badge
- **Pro**: "GoRead2 Pro - Unlimited feeds" with green badge
- **Free Months**: "Free Months - X months remaining" with blue badge
- **Trial**: "Trial - X/20 feeds, Y days left" with orange badge
- **Expired**: "Trial Expired - Subscribe to continue" with red warning

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

## Subscription Management

### Stripe Integration

When subscriptions are enabled (`SUBSCRIPTION_ENABLED=true`):

- **Webhook handling**: Stripe webhooks update subscription status
- **Customer portal**: Users can manage billing through Stripe
- **Subscription lifecycle**: Automatic handling of payments, cancellations

### Manual Subscription Operations

```bash
# SECURITY: Set admin token first
export ADMIN_TOKEN="your_64_char_token"

# Check subscription system status
go run cmd/admin/main.go system-info

# View user subscription details
go run cmd/admin/main.go user-info user@example.com

# Note: Stripe subscriptions are managed via webhooks
# Manual subscription changes should be done through Stripe Dashboard
```

## Security Considerations

### Token Security Best Practices

- **Store tokens in secure password managers** or encrypted configuration
- **Never commit tokens** to version control or logs
- **Use descriptive names** when creating tokens for audit purposes
- **Regularly rotate tokens** (create new, revoke old)
- **Immediately revoke tokens** if compromised or when team members leave

### Environment Management

- **Use separate tokens per environment** (dev, staging, production)
- **Restrict server access** to authorized personnel only
- **Monitor token usage** through `list-tokens` command
- **Keep audit logs** of admin operations

### Operational Security

- **Principle of least privilege**: Create tokens only when needed
- **Regularly review active tokens** and revoke unused ones
- **Set up alerting** for admin command usage in production
- **Document token ownership** and purpose

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

### Error Messages Guide

#### "ADMIN_TOKEN must be exactly 64 characters (32 bytes as hex)"
- **Cause**: Token format is invalid (not 64 hexadecimal characters)
- **Solution**: Use `create-token` command to generate valid token

#### "Invalid ADMIN_TOKEN - token not found in database or inactive"
- **Cause**: Token doesn't exist in database or has been revoked
- **Solution**: Generate new token with `create-token` or contact administrator

#### "ADMIN_TOKEN environment variable must be set"
- **Cause**: Missing required environment variable
- **Solution**: Set ADMIN_TOKEN with valid 64-character token

#### Bootstrap Security Errors
- **"No admin users found in database"**: Initial token creation blocked
- **Solution**: Create user account and set as admin in database first

#### Token Creation Security Warnings
- **"Admin tokens already exist"**: Shown when creating additional tokens
- **Solution**: Confirm with 'y' if you're authorized to create tokens

## Database Schema

### Users Table

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

### Admin Tokens Table

The secure admin token system:

```sql
CREATE TABLE admin_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    token_hash TEXT UNIQUE NOT NULL,    -- SHA-256 hash of token
    description TEXT NOT NULL,          -- Human-readable description
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_used_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT 1         -- 0 = revoked, 1 = active
);
```

### Audit Logs Table

The audit logging system for tracking admin operations:

```sql
CREATE TABLE audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    admin_user_id INTEGER,              -- ID of admin who performed action (NULL for CLI)
    admin_email TEXT NOT NULL,          -- Email of admin (or "CLI_ADMIN")
    operation_type TEXT NOT NULL,       -- Type of operation performed
    target_user_id INTEGER,             -- ID of user affected (NULL if not applicable)
    target_user_email TEXT,             -- Email of user affected
    operation_details TEXT,             -- JSON with additional context
    ip_address TEXT,                    -- IP address of request (or "CLI")
    result TEXT NOT NULL,               -- "success" or "failure"
    error_message TEXT                  -- Error details if result = "failure"
);

-- Indexes for efficient querying
CREATE INDEX idx_audit_timestamp ON audit_logs(timestamp);
CREATE INDEX idx_audit_admin_user ON audit_logs(admin_user_id);
CREATE INDEX idx_audit_target_user ON audit_logs(target_user_id);
CREATE INDEX idx_audit_operation_type ON audit_logs(operation_type);
```

**Google Datastore Equivalent:**
- Kind: `AuditLog`
- Indexes on: `Timestamp`, `AdminUserID`, `TargetUserID`, `OperationType`

## Testing and Validation

The admin token security system includes comprehensive test coverage:

### Unit Tests (`internal/services/`)
- **SQLite Backend**: 6 test suites with 20+ individual test cases covering token generation, validation, listing, revocation, and lifecycle management
- **Datastore Backend**: 6 test suites for Google App Engine deployment compatibility (requires emulator)
- **Security Features**: Cryptographic token generation, database validation, bootstrap protection
- **Edge Cases**: Invalid formats, non-existent tokens, already-revoked tokens, concurrent operations

### Integration Tests (`test/integration/`)
- **Admin Command Testing**: Full CLI command validation with proper environment setup
- **Bootstrap Security**: Tests preventing unauthorized token creation without existing admin users
- **Token Lifecycle**: End-to-end testing of create → validate → list → revoke operations
- **Security Warnings**: Verification of prompts when creating additional tokens

### Running Admin Security Tests

```bash
# Run all admin token unit tests
go test ./internal/services -run "TestGenerateAdminToken|TestValidateAdminToken|TestListAdminTokens|TestRevokeAdminToken|TestHasAdminTokens|TestAdminTokenUniqueGeneration" -v

# Run Datastore tests (requires emulator)
DATASTORE_EMULATOR_HOST=localhost:8081 go test ./internal/services -run "TestDatastore" -v

# Run security integration tests
go test ./test/integration -run "TestAdminSecurity" -v

# Run all tests with coverage
go test ./internal/services -cover
```

## Monitoring and Analytics

### User Statistics

```bash
# SECURITY: Set admin token first
export ADMIN_TOKEN="your_64_char_token"

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
- **Audit log export**: Export audit logs to CSV/JSON for archival
- **Audit log cleanup**: Automatic expiration of old audit logs
- **Token expiration dates**: Automatic cleanup of expired tokens
- **Multi-factor authentication**: For token creation
- **Role-based permissions**: Read-only vs full admin
- **External identity providers**: Integration support

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

See [API Reference](API.md) for complete endpoint documentation.

---

This comprehensive admin system provides flexible user management while maintaining security and a good user experience.
