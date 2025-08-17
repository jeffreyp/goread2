# Stripe Setup Guide

Complete guide for integrating Stripe payment processing with GoRead2 subscriptions.

## Overview

GoRead2 uses Stripe for subscription billing with the following features:
- **30-day free trial** with 20 feed limit
- **$2.99/month Pro subscription** for unlimited feeds
- **Customer portal** for billing management
- **Webhook integration** for real-time subscription updates

## Prerequisites

1. **Stripe Account**: Create a free account at [stripe.com](https://stripe.com)
2. **GoRead2 Running**: Have GoRead2 configured with OAuth authentication
3. **HTTPS in Production**: Required for webhook endpoints

## Step 1: Get Stripe API Keys

### Development (Test Keys)

1. Log into [Stripe Dashboard](https://dashboard.stripe.com)
2. Navigate to **Developers → API keys**
3. Copy the **Publishable key** (starts with `pk_test_`)
4. Copy the **Secret key** (starts with `sk_test_`)

### Production (Live Keys)

1. Switch to **Live mode** in the dashboard
2. Navigate to **Developers → API keys**
3. Copy the **Publishable key** (starts with `pk_live_`)
4. Copy the **Secret key** (starts with `sk_live_`)

## Step 2: Configure Environment Variables

### Development

```bash
# Stripe Configuration
export STRIPE_SECRET_KEY=sk_test_your_secret_key_here
export STRIPE_PUBLISHABLE_KEY=pk_test_your_publishable_key_here
export STRIPE_WEBHOOK_SECRET=whsec_your_webhook_secret_here
export STRIPE_PRICE_ID=price_your_price_id_here

# Enable subscription system
export SUBSCRIPTION_ENABLED=true
```

### Production (App Engine)

```yaml
# app.yaml
env_variables:
  SUBSCRIPTION_ENABLED: "true"
  STRIPE_SECRET_KEY: "sk_live_your_secret_key_here"
  STRIPE_PUBLISHABLE_KEY: "pk_live_your_publishable_key_here"
  STRIPE_WEBHOOK_SECRET: "whsec_your_webhook_secret_here"
  STRIPE_PRICE_ID: "price_your_price_id_here"
```

### Production (Docker)

```bash
# .env file
SUBSCRIPTION_ENABLED=true
STRIPE_SECRET_KEY=sk_live_your_secret_key_here
STRIPE_PUBLISHABLE_KEY=pk_live_your_publishable_key_here
STRIPE_WEBHOOK_SECRET=whsec_your_webhook_secret_here
STRIPE_PRICE_ID=price_your_price_id_here
```

## Step 3: Create Product and Price

Use the setup tool to create the GoRead2 Pro product:

```bash
# Set your environment variables first
export STRIPE_SECRET_KEY=sk_test_your_secret_key_here
export STRIPE_PUBLISHABLE_KEY=pk_test_your_publishable_key_here

# Validate configuration
go run cmd/setup-stripe/main.go validate

# Create product and price
go run cmd/setup-stripe/main.go create-product
```

**Output example:**
```
✅ Product created: GoRead2 Pro
✅ Price created: price_1ABC123def456789
📝 Add this to your environment: STRIPE_PRICE_ID=price_1ABC123def456789
```

Update your environment variables with the returned Price ID.

## Step 4: Set Up Webhooks

Webhooks notify your application when subscription events occur.

### Development (using Stripe CLI)

1. **Install Stripe CLI**: Follow instructions at [stripe.com/docs/stripe-cli](https://stripe.com/docs/stripe-cli)

2. **Login to Stripe CLI**:
   ```bash
   stripe login
   ```

3. **Forward webhooks to local server**:
   ```bash
   stripe listen --forward-to localhost:8080/webhooks/stripe
   ```

4. **Copy the webhook secret** from CLI output and set as `STRIPE_WEBHOOK_SECRET`

**Example CLI output:**
```
> Ready! Your webhook signing secret is whsec_1234567890abcdef...
```

### Production

1. **Create webhook endpoint** in Stripe Dashboard:
   - Go to **Developers → Webhooks**
   - Click **Add endpoint**
   - URL: `https://your-domain.com/webhooks/stripe`
   - Events to send:
     - `checkout.session.completed`
     - `customer.subscription.created`
     - `customer.subscription.updated`
     - `customer.subscription.deleted`
     - `invoice.payment_succeeded`
     - `invoice.payment_failed`

2. **Copy the webhook secret** and set as `STRIPE_WEBHOOK_SECRET`

## Step 5: Test the Integration

### Start Application

```bash
# Set all environment variables
export SUBSCRIPTION_ENABLED=true
export STRIPE_SECRET_KEY=sk_test_...
export STRIPE_PUBLISHABLE_KEY=pk_test_...
export STRIPE_WEBHOOK_SECRET=whsec_...
export STRIPE_PRICE_ID=price_...

# Start GoRead2
./goread2
```

### Test Subscription Flow

1. **Trigger subscription prompt**:
   - Add more than 20 feeds to hit the limit
   - Should see "Upgrade to Pro" modal

2. **Test payment process**:
   - Click "Upgrade to Pro" 
   - Complete checkout using test card: `4242 4242 4242 4242`
   - Use any future expiry date, any 3-digit CVC, any ZIP code

3. **Verify subscription activation**:
   - Should redirect back with unlimited access
   - Check subscription status in UI
   - Verify in Stripe Dashboard

### Test Webhook Processing

```bash
# Monitor application logs for webhook events
tail -f logs/goread2.log

# Look for entries like:
# "Stripe webhook received: checkout.session.completed"
# "Subscription updated for user: user@example.com"
```

## Stripe Test Cards

Use these test card numbers for development:

| Card Number | Description |
|-------------|-------------|
| `4242 4242 4242 4242` | Successful payment |
| `4000 0000 0000 0002` | Declined payment |
| `4000 0025 0000 3155` | Requires authentication (3D Secure) |
| `4000 0000 0000 9995` | Insufficient funds |
| `4000 0000 0000 9987` | Lost card |

**For all test cards:**
- Use any future expiry date (e.g., 12/34)
- Use any 3-digit CVC (e.g., 123)
- Use any ZIP code (e.g., 12345)

## Subscription Management

### Customer Portal

GoRead2 includes Stripe's Customer Portal for self-service billing:

**Features:**
- Update payment methods
- Download invoices
- Cancel subscriptions
- View billing history
- Update billing address

**Access:**
- Users click "Account" button in GoRead2
- Creates temporary portal session
- Redirects to Stripe-hosted portal
- Returns to GoRead2 after changes

### Manual Management

Manage subscriptions through the Stripe Dashboard:

1. **View customers**: Customers → All customers
2. **Subscription details**: Click customer → Subscriptions tab
3. **Actions available**:
   - Pause/resume subscriptions
   - Issue refunds
   - Update pricing
   - Cancel subscriptions
   - View payment history

## Configuration Options

### Subscription Settings

Customize subscription behavior in your environment:

```bash
# Required - Enable subscription system
SUBSCRIPTION_ENABLED=true

# Required - Stripe integration
STRIPE_SECRET_KEY=sk_...
STRIPE_PUBLISHABLE_KEY=pk_...
STRIPE_WEBHOOK_SECRET=whsec_...
STRIPE_PRICE_ID=price_...

# Optional - Subscription behavior
FREE_TRIAL_DAYS=30          # Default: 30 days
FREE_TRIAL_FEED_LIMIT=20    # Default: 20 feeds
```

### Product Configuration

The default GoRead2 Pro product includes:

- **Price**: $2.99/month USD
- **Billing**: Monthly recurring
- **Trial**: 30-day free trial
- **Features**: Unlimited feeds, all features

To modify pricing:

1. Create new price in Stripe Dashboard
2. Update `STRIPE_PRICE_ID` environment variable
3. Deploy updated configuration

## Monitoring and Analytics

### Stripe Dashboard

Monitor key metrics:

- **Revenue**: Monthly recurring revenue (MRR)
- **Subscriptions**: Active, churned, past due
- **Customers**: New signups, payment failures
- **Billing**: Successful payments, refunds

### Application Logs

Monitor webhook processing:

```bash
# Successful webhook processing
grep "Stripe webhook" logs/goread2.log

# Failed webhook processing  
grep "ERROR.*stripe" logs/goread2.log

# Subscription status changes
grep "Subscription.*updated" logs/goread2.log
```

### Health Checks

Verify Stripe integration:

```bash
# Test webhook endpoint
curl -X POST https://your-domain.com/webhooks/stripe \
  -H "Content-Type: application/json" \
  -d '{"type": "ping"}'

# Should return 200 OK (may show signature error, but endpoint is reachable)
```

## Troubleshooting

### Common Issues

#### "Stripe not configured" Error

**Symptoms:**
- Subscription features not available
- Error in application logs

**Solutions:**
```bash
# Verify all environment variables are set
go run cmd/setup-stripe/main.go validate

# Check required variables
echo $SUBSCRIPTION_ENABLED
echo $STRIPE_SECRET_KEY
echo $STRIPE_PUBLISHABLE_KEY
```

#### Webhook Signature Verification Failed

**Symptoms:**
- Webhooks returning 400 errors
- "Invalid signature" in logs

**Solutions:**
- Verify `STRIPE_WEBHOOK_SECRET` matches webhook endpoint
- Ensure webhook URL is publicly accessible
- Check webhook endpoint in Stripe Dashboard

#### Product/Price Not Found

**Symptoms:**
- Checkout fails to load
- "Price not found" errors

**Solutions:**
```bash
# Recreate product and price
go run cmd/setup-stripe/main.go create-product

# Update STRIPE_PRICE_ID with new price ID
export STRIPE_PRICE_ID=price_new_id_here
```

#### Payment Fails in Production

**Symptoms:**
- Test cards work, real cards don't
- Production checkout errors

**Solutions:**
- Switch to live API keys in production
- Ensure webhook endpoint uses HTTPS
- Check Stripe Dashboard for detailed error messages
- Verify business information is complete

### Debug Commands

```bash
# Validate Stripe configuration
go run cmd/setup-stripe/main.go validate

# Test webhook endpoint
go run cmd/setup-stripe/main.go test-webhook

# List products and prices
go run cmd/setup-stripe/main.go list-products
```

### Logs and Monitoring

#### Application Logs

```bash
# Stripe-related logs
grep -i stripe logs/goread2.log

# Webhook processing
grep "webhook" logs/goread2.log

# Subscription changes
grep "subscription" logs/goread2.log
```

#### Stripe Dashboard

- **Events**: Developers → Events (shows all webhook deliveries)
- **Logs**: Developers → Logs (API request logs)
- **Webhooks**: Developers → Webhooks (delivery attempts)

## Security Best Practices

### API Key Management

- **Never commit keys** to version control
- **Use environment variables** for all deployments
- **Rotate keys regularly** if compromised
- **Use restricted keys** when possible

### Webhook Security

- **Always verify signatures** (automatically handled)
- **Use HTTPS** for webhook endpoints in production
- **Implement idempotency** for webhook processing
- **Log security events** for monitoring

### Customer Data

- **Follow PCI compliance** guidelines
- **Never store card details** (Stripe handles this)
- **Secure customer information** with encryption
- **Implement proper access controls**

## Going Live

### Pre-Launch Checklist

- [ ] Switch to live Stripe API keys
- [ ] Update webhook endpoints to production URLs
- [ ] Test complete subscription flow with real payment
- [ ] Verify customer portal functionality
- [ ] Monitor webhook delivery for 24 hours
- [ ] Set up monitoring and alerting

### Launch Process

1. **Update environment variables** to live keys
2. **Deploy application** with new configuration
3. **Test subscription flow** end-to-end
4. **Monitor Stripe Dashboard** for any issues
5. **Notify users** of new Pro features (optional)

### Post-Launch Monitoring

- **Daily**: Check failed payments and webhook deliveries
- **Weekly**: Review subscription metrics and churn
- **Monthly**: Analyze revenue and customer growth
- **Quarterly**: Review pricing and feature usage

## Support and Resources

- **Stripe Documentation**: [stripe.com/docs](https://stripe.com/docs)
- **Stripe Support**: Available through dashboard chat
- **GoRead2 Issues**: Report integration problems on GitHub
- **Community**: Stripe Discord and forums for developer help

## Advanced Configuration

### Custom Pricing

Create multiple price tiers:

```bash
# Create additional prices
go run cmd/setup-stripe/main.go create-price --amount 499 --interval year
go run cmd/setup-stripe/main.go create-price --amount 999 --interval month --name "Premium"
```

### Multiple Products

Support different subscription tiers:

1. Create products in Stripe Dashboard
2. Configure environment variables for each tier
3. Modify checkout logic to select appropriate price
4. Update UI to show different options

### Usage-Based Billing

Implement metered billing for heavy users:

1. Create metered pricing in Stripe
2. Track usage in GoRead2 (feed count, API calls)
3. Report usage to Stripe via API
4. Handle usage-based invoicing

This comprehensive Stripe integration provides secure, scalable subscription management for GoRead2.