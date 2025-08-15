# Admin Quick Start Guide

This guide shows you exactly how to give yourself or others free access to GoRead2.

## TL;DR - Make Yourself Admin

```bash
# Replace with your actual email address
./admin.sh admin your-email@gmail.com on
```

That's it! You now have unlimited feeds forever.

## Option 1: Admin Access (Recommended)

**What it does**: Bypasses all subscription limits permanently
**Best for**: Yourself, co-admins, permanent free users

```bash
# Grant admin access
./admin.sh admin user@example.com on

# Revoke admin access  
./admin.sh admin user@example.com off
```

Admin users get:
- ✅ Unlimited feeds
- ✅ No time limits
- ✅ Purple "ADMIN" badge in UI
- ✅ Bypasses all payment checks

## Option 2: Free Months

**What it does**: Grants temporary unlimited access for X months
**Best for**: Beta testers, temporary promotions, trial extensions

```bash
# Grant 6 months free access
./admin.sh grant user@example.com 6

# Grant 1 year free access
./admin.sh grant user@example.com 12
```

Free month users get:
- ✅ Unlimited feeds (while months remain)
- ✅ Blue "FREE" badge in UI  
- ⏰ Time-limited access
- ⏰ Eventually need to subscribe or get more free months

## Quick Commands

```bash
# List all users
./admin.sh list

# Check specific user status
./admin.sh info user@example.com

# Make someone admin
./admin.sh admin user@example.com on

# Grant 3 free months
./admin.sh grant user@example.com 3
```

## Manual Database Method (Alternative)

If you don't want to use the script, you can update the database directly:

### Local SQLite
```sql
-- Make yourself admin (replace email)
UPDATE users SET is_admin = 1 WHERE email = 'your-email@gmail.com';

-- Grant 6 free months
UPDATE users SET free_months_remaining = 6 WHERE email = 'user@example.com';
```

### Production (Google Cloud Datastore)
Use the Google Cloud Console to edit user entities and set:
- `is_admin`: true
- `free_months_remaining`: 6

## User Status Hierarchy

1. **Admin** (`is_admin = true`) - Unlimited everything, forever
2. **Pro** (`subscription_status = 'active'`) - Paid Stripe subscriber  
3. **Free Months** (`free_months_remaining > 0`) - Temporary unlimited
4. **Trial** (`subscription_status = 'trial'`) - 20 feeds, 30 days
5. **Expired** - No access, must subscribe

## Troubleshooting

**"User not found" error**
- User must have logged in at least once
- Check email spelling and case

**Changes not showing**
- Restart GoRead2 application
- User should logout/login
- Clear browser cookies

**Script not executable**
```bash
chmod +x admin.sh
```

**Database locked**
- Stop GoRead2 before running admin commands
- Only one process can access SQLite at a time

## Security Notes

- **Admin users have unlimited access** - only grant to trusted people
- **Free months stack** - granting more adds to existing total
- **No audit trail** - consider logging admin actions in production
- **UI still shows payment options** - admin users can ignore these

## Examples

### Make yourself admin:
```bash
./admin.sh admin jeffreyp@gmail.com on
```

### Give beta testers 3 months:
```bash
./admin.sh grant beta1@example.com 3
./admin.sh grant beta2@example.com 3
./admin.sh grant beta3@example.com 3
```

### Check what you gave someone:
```bash
./admin.sh info beta1@example.com
```

### Remove admin access:
```bash
./admin.sh admin former-admin@example.com off
```

That's it! Your users now have unlimited access to GoRead2.