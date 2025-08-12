# Admin Management Guide

This guide explains how to manage users, grant admin privileges, and provide free months for GoRead2.

## Overview

GoRead2 supports multiple ways to bypass subscription restrictions:

1. **Admin Users**: Complete bypass of all subscription limits
2. **Free Months**: Grant temporary unlimited access for a specific number of months
3. **Direct Database Modification**: Manual database updates for advanced users

## Admin Script Usage

The `admin.sh` script provides an easy way to manage users:

```bash
# Make script executable (first time only)
chmod +x admin.sh

# List all users
./admin.sh list

# Grant admin access to yourself
./admin.sh admin your-email@gmail.com on

# Grant 6 free months to a user
./admin.sh grant user@example.com 6

# View user information
./admin.sh info user@example.com

# Revoke admin access
./admin.sh admin user@example.com off
```

## Manual Command Usage

You can also run the admin commands directly:

```bash
# List all users
go run cmd/admin/main.go list-users

# Set admin status
go run cmd/admin/main.go set-admin your-email@gmail.com true
go run cmd/admin/main.go set-admin user@example.com false

# Grant free months
go run cmd/admin/main.go grant-months user@example.com 6

# Show user information
go run cmd/admin/main.go user-info user@example.com
```

## User Status Types

### Admin Users (`is_admin = true`)
- **Unlimited feeds**: No feed limits whatsoever
- **Bypass all restrictions**: All subscription checks are skipped
- **Status display**: Shows "Admin" in the UI
- **Permanent**: Remains until explicitly revoked

### Free Months (`free_months_remaining > 0`)
- **Temporary unlimited access**: Works like a Pro subscription
- **Automatic decrement**: Months decrease over time (when implemented)
- **Status display**: Shows "Free Months" in the UI
- **Stackable**: Additional months can be added

### Trial Users (`subscription_status = 'trial'`)
- **Limited feeds**: Maximum 20 feeds
- **Time-limited**: 30 days from account creation
- **Standard behavior**: Default for new users

### Pro Users (`subscription_status = 'active'`)
- **Unlimited feeds**: Paid Stripe subscription
- **Managed by Stripe**: Webhooks handle status updates
- **Billing**: $2.99/month recurring

## Common Admin Tasks

### Make Yourself Admin

```bash
# Replace with your actual email
./admin.sh admin your-email@gmail.com on
```

### Grant Free Access to Beta Users

```bash
# Grant 3 months free access
./admin.sh grant beta-user@example.com 3
```

### Check User Status

```bash
./admin.sh info problematic-user@example.com
```

### List All Users

```bash
./admin.sh list
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
```

### Google Cloud Datastore (Production)

Use the Google Cloud Console or `gcloud` CLI to update entities:

```bash
# Using gcloud datastore (example)
gcloud datastore entities update --kind=User --key=<user-key> --properties=is_admin=true
```

## Environment-Based Admin

You can also set admin users via environment variables during development:

```bash
# Set admin emails (comma-separated)
export ADMIN_EMAILS="admin@example.com,owner@example.com"

# Then modify the auth service to check this list
```

## Security Considerations

### Admin Access
- **Admin users have unlimited access** - only grant to trusted individuals
- **No billing bypass**: Admin users still see upgrade prompts (but can ignore limits)
- **Audit trail**: Consider logging admin actions

### Free Months
- **Monitor usage**: Track free month consumption
- **Limited time**: Consider implementing automatic expiration
- **User experience**: Clearly communicate free month status

## Implementation Details

### Database Schema

The user table includes these subscription-related fields:

```sql
CREATE TABLE users (
    -- ... other fields
    subscription_status TEXT DEFAULT 'trial',        -- 'trial', 'active', 'cancelled', 'expired', 'admin'
    subscription_id TEXT,                            -- Stripe subscription ID
    trial_ends_at DATETIME,                         -- When free trial expires
    last_payment_date DATETIME,                     -- Last successful payment
    is_admin BOOLEAN DEFAULT 0,                     -- Admin users bypass limits
    free_months_remaining INTEGER DEFAULT 0         -- Free months granted
);
```

### Subscription Check Logic

The feed limit check follows this priority:

1. **Admin users** (`is_admin = true`) → Unlimited access
2. **Active subscribers** (`subscription_status = 'active'`) → Unlimited access  
3. **Free months remaining** (`free_months_remaining > 0`) → Unlimited access
4. **Trial users** → Limited to 20 feeds, expires after 30 days
5. **Expired users** → No access

### Status Display

Users see different status indicators based on their access level:

- **Admin**: "Admin - Unlimited Access"
- **Pro**: "GoRead2 Pro - Unlimited feeds" 
- **Free Months**: "Free Months - X months remaining"
- **Trial**: "Trial - X/20 feeds, Y days left"
- **Expired**: "Trial Expired - Subscribe to continue"

## Troubleshooting

### User Not Found
- Check email spelling and case sensitivity
- Verify user has logged in at least once (account must exist)

### Database Locked
- Stop the GoRead2 application before running admin commands
- Check file permissions on SQLite database

### Changes Not Reflected
- Restart the GoRead2 application
- Have user log out and log back in
- Check browser cache/cookies

### Stripe Integration Conflicts
- Admin and free month users still see Stripe UI elements
- This is by design - they can ignore payment prompts
- Consider hiding payment UI for admin users in future updates

## Future Enhancements

### Planned Features
- **Automatic free month expiration**: Decrement monthly via cron job
- **Admin dashboard**: Web UI for user management
- **Usage analytics**: Track admin and free user activity
- **Bulk operations**: Import/export admin user lists
- **Audit logging**: Track all admin actions

### Customization Ideas
- **Organization admin**: Company-wide admin access
- **Temporary admin**: Time-limited admin privileges
- **Feed quotas**: Custom feed limits per user
- **Feature flags**: Enable/disable features per user

This system provides flexible user management while maintaining security and a good user experience.