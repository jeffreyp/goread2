# Feature Flags

Configuration options for enabling/disabling GoRead2 features.

## Subscription System Toggle

### Overview

The subscription system can be enabled or disabled using the `SUBSCRIPTION_ENABLED` environment variable. This allows you to:

- **Safely deploy** subscription features without affecting existing users
- **Run GoRead2 as a free service** without any subscription limits
- **Test deployments** without enabling billing
- **Gradually roll out** subscription functionality

### Configuration

```bash
# Enable subscription system
export SUBSCRIPTION_ENABLED=true

# Disable subscription system (default)
export SUBSCRIPTION_ENABLED=false
```

**Accepted values:**
- `true`, `1`, `yes`, `on`, `enabled` → Enables subscriptions
- `false`, `0`, `no`, `off`, `disabled` → Disables subscriptions
- Empty or unset → Defaults to disabled (false)

### When Subscription System is ENABLED

✅ **Active Features:**
- 30-day free trial with 20 feed limit
- Stripe integration for payments
- Subscription upgrade prompts
- Feed limit enforcement
- Admin commands for subscription management
- Customer portal access
- Webhook processing

🔧 **Required Configuration:**
```bash
SUBSCRIPTION_ENABLED=true
STRIPE_SECRET_KEY=sk_test_...
STRIPE_PUBLISHABLE_KEY=pk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...
STRIPE_PRICE_ID=price_...
```

### When Subscription System is DISABLED

✅ **Default Behavior:**
- **Unlimited feeds** for all users
- **No billing** or payment processing
- **No subscription limits** or restrictions
- **Simplified UI** without upgrade prompts

❌ **Disabled Features:**
- Stripe payment processing
- Subscription upgrade flows
- Feed limit enforcement
- Trial period restrictions
- Customer portal
- Subscription webhooks

🔧 **Required Configuration:**
```bash
SUBSCRIPTION_ENABLED=false
# Stripe keys not required
```

## Implementation Details

### Backend Changes

**Service Layer:**
```go
func (ss *SubscriptionService) CanUserAddFeed(userID int) error {
    // If subscription system is disabled, allow unlimited feeds
    if !config.IsSubscriptionEnabled() {
        return nil
    }
    // ... normal subscription logic
}
```

**API Routes:**
- Subscription-related routes only registered when enabled
- Payment handlers conditionally initialized
- Webhook endpoints only active when enabled

### Frontend Changes

**Status Display:**
- Shows "UNLIMITED" badge when subscriptions disabled
- Hides upgrade buttons and subscription prompts
- Account page shows unlimited access message

**Error Handling:**
- No feed limit errors when disabled
- Subscription-related modals not shown
- Payment flows completely bypassed

### Admin Commands

**⚠️ Security:** All admin commands now require ADMIN_TOKEN authentication.

**Always Available:**
```bash
# SECURITY: Set admin tokens first
export ADMIN_TOKEN="$(openssl rand -hex 32)"
export ADMIN_TOKEN_VERIFY="$ADMIN_TOKEN"

go run cmd/admin/main.go list-users
go run cmd/admin/main.go set-admin user@example.com true
go run cmd/admin/main.go user-info user@example.com
```

**Only When Enabled:**
```bash
# SECURITY: Requires both tokens for sensitive operations
export ADMIN_TOKEN="$(openssl rand -hex 32)"
export ADMIN_TOKEN_VERIFY="$ADMIN_TOKEN"

go run cmd/admin/main.go grant-months user@example.com 3
```

## Migration Strategy

### Safe Deployment Process

1. **Deploy with subscriptions disabled:**
   ```bash
   SUBSCRIPTION_ENABLED=false
   ```

2. **Test deployment thoroughly:**
   - Verify unlimited feed access
   - Check UI shows no subscription prompts
   - Confirm admin commands work

3. **Enable gradually:**
   ```bash
   SUBSCRIPTION_ENABLED=true
   ```

4. **Full activation:**
   - Configure Stripe properly
   - Test payment flows
   - Enable in production

### Rollback Plan

If issues occur after enabling subscriptions:

```bash
# Immediate rollback - disable subscriptions
SUBSCRIPTION_ENABLED=false

# Restart application
# Users get unlimited access immediately
```

## Environment Variables Summary

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `SUBSCRIPTION_ENABLED` | No | `false` | Enable/disable subscription system |
| `STRIPE_SECRET_KEY` | When enabled | - | Stripe secret key for payments |
| `STRIPE_PUBLISHABLE_KEY` | When enabled | - | Stripe public key for frontend |
| `STRIPE_WEBHOOK_SECRET` | When enabled | - | Stripe webhook signature verification |
| `STRIPE_PRICE_ID` | When enabled | - | Stripe price ID for Pro subscription |

## Testing Different Modes

### Test Subscription Disabled

```bash
# Set environment
SUBSCRIPTION_ENABLED=false

# Verify behavior
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/subscription
# Should return: {"status": "unlimited", "feed_limit": -1, ...}
```

### Test Subscription Enabled

```bash
# Set environment  
SUBSCRIPTION_ENABLED=true
STRIPE_SECRET_KEY=sk_test_...

# Verify behavior
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/subscription
# Should return trial status with limits
```

## Troubleshooting

### Common Issues

1. **Subscription routes not found (404)**
   - Check `SUBSCRIPTION_ENABLED=true`
   - Verify Stripe configuration

2. **Unlimited access not working**
   - Confirm `SUBSCRIPTION_ENABLED=false`
   - Check application logs for config loading

3. **Admin commands failing**
   - Verify command exists for current configuration
   - Check environment variable spelling

### Debug Commands

```bash
# SECURITY: Set admin token first
export ADMIN_TOKEN="$(openssl rand -hex 32)"

# Check current configuration
go run cmd/admin/main.go user-info any@email.com

# Look for "System Configuration" section in output
```

This feature flag system provides flexible control over subscription functionality while maintaining a smooth user experience.