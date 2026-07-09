# Stripe Setup Guide

Guide for integrating Stripe payment processing with GoRead2's subscription system, for developers configuring billing in a new environment.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Step 1: Get Stripe API Keys](#step-1-get-stripe-api-keys)
- [Step 2: Configure Environment Variables](#step-2-configure-environment-variables)
- [Step 3: Create Product and Price](#step-3-create-product-and-price)
- [Step 4: Set Up Webhooks](#step-4-set-up-webhooks)
- [Step 5: Test the Integration](#step-5-test-the-integration)
- [Stripe Test Cards](#stripe-test-cards)
- [Subscription Management](#subscription-management)
- [Configuration Options](#configuration-options)
- [Monitoring and Analytics](#monitoring-and-analytics)
- [Security Best Practices](#security-best-practices)
- [Going Live](#going-live)
- [Troubleshooting](#troubleshooting)
- [Related Documentation](#related-documentation)

## Overview

GoRead2 uses Stripe for subscription billing with the following features:
- **30-day free trial** with 20 feed limit
- **$9.99/month Pro subscription** for unlimited feeds
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

In practice these are stored in Google Secret Manager and referenced from `app.yaml`; see [deployment.md](deployment.md#setting-up-google-secret-manager).

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

   These are the only events `internal/handlers/payment_handler.go` handles; sending others is harmless but they won't trigger any action.

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
make dev
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

Watch the application's stdout (see [troubleshooting.md](troubleshooting.md#logging-and-debugging)) for entries like:
```
Stripe webhook received: checkout.session.completed
Subscription updated for user: user@example.com
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
```

The 30-day trial length and 20-feed free-tier limit are not configurable via environment variables. They're compile-time constants (`internal/services/subscription_service.go`'s `FreeTrialFeedLimit`, and a hardcoded 30-day offset in the database layer), so changing them requires a code change.

### Product Configuration

The default GoRead2 Pro product includes:

- **Price**: $9.99/month USD
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

- **Events**: Developers → Events (shows all webhook deliveries)
- **Logs**: Developers → Logs (API request logs)
- **Webhooks**: Developers → Webhooks (delivery attempts)

### Health Check

Verify the webhook endpoint is reachable:

```bash
curl -X POST https://your-domain.com/webhooks/stripe \
  -H "Content-Type: application/json" \
  -d '{"type": "ping"}'

# Should return 200 OK (may show signature error, but endpoint is reachable)
```

## Security Best Practices

### API Key Management

- Never commit keys to version control
- Use environment variables (or Secret Manager, in production) for all deployments
- Rotate keys if compromised
- Use restricted keys when possible

### Webhook Security

- Signature verification is handled automatically by the Stripe SDK
- Use HTTPS for webhook endpoints in production

### Customer Data

- Card details are never stored by GoRead2; Stripe handles PCI compliance
- Customer information is stored only as needed to link a Stripe customer ID to a GoRead2 user

## Going Live

### Pre-Launch Checklist

- [ ] Switch to live Stripe API keys
- [ ] Update webhook endpoints to production URLs
- [ ] Test complete subscription flow with real payment
- [ ] Verify customer portal functionality
- [ ] Monitor webhook delivery for 24 hours

### Launch Process

1. Update environment variables to live keys
2. Deploy application with new configuration
3. Test subscription flow end-to-end
4. Monitor Stripe Dashboard for issues

## Troubleshooting

See [Troubleshooting Guide](troubleshooting.md#subscription-issues) for Stripe configuration errors, webhook failures, missing products/prices, and production payment issues.

## Related Documentation

- [Setup Guide](setup.md) - Environment variables and installation
- [Feature Flags](feature-flags.md) - Enabling/disabling the subscription system
- [Deployment Guide](deployment.md) - Storing Stripe secrets in Secret Manager
- [Troubleshooting Guide](troubleshooting.md) - Stripe-related issues
