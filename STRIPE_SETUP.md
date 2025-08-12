# Stripe Integration Setup Guide

This guide explains how to set up Stripe payment processing for GoRead2 subscriptions.

## Prerequisites

1. **Stripe Account**: Create a free account at [stripe.com](https://stripe.com)
2. **GoRead2 Application**: Have GoRead2 running with OAuth authentication configured

## Step 1: Get Stripe API Keys

1. **Log into Stripe Dashboard**: Go to [dashboard.stripe.com](https://dashboard.stripe.com)

2. **Get Test Keys** (for development):
   - Navigate to **Developers → API keys**
   - Copy the **Publishable key** (starts with `pk_test_`)
   - Copy the **Secret key** (starts with `sk_test_`)

3. **Get Live Keys** (for production):
   - Switch to **Live mode** in the dashboard
   - Navigate to **Developers → API keys**
   - Copy the **Publishable key** (starts with `pk_live_`)
   - Copy the **Secret key** (starts with `sk_live_`)

## Step 2: Configure Environment Variables

Add these environment variables to your application:

### Development (.env file or export)
```bash
# Stripe Configuration
STRIPE_SECRET_KEY=sk_test_your_secret_key_here
STRIPE_PUBLISHABLE_KEY=pk_test_your_publishable_key_here
STRIPE_WEBHOOK_SECRET=whsec_your_webhook_secret_here
STRIPE_PRICE_ID=price_your_price_id_here
```

### Production (App Engine app.yaml)
```yaml
env_variables:
  STRIPE_SECRET_KEY: "sk_live_your_secret_key_here"
  STRIPE_PUBLISHABLE_KEY: "pk_live_your_publishable_key_here"
  STRIPE_WEBHOOK_SECRET: "whsec_your_webhook_secret_here"
  STRIPE_PRICE_ID: "price_your_price_id_here"
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

This will output a Price ID like `price_1ABC123def456789`. Add this to your environment variables as `STRIPE_PRICE_ID`.

## Step 4: Set Up Webhooks

Webhooks allow Stripe to notify your application when subscription events occur.

### Development (using Stripe CLI)

1. **Install Stripe CLI**: Follow instructions at [stripe.com/docs/stripe-cli](https://stripe.com/docs/stripe-cli)

2. **Login to Stripe CLI**:
   ```bash
   stripe login
   ```

3. **Forward webhooks to your local server**:
   ```bash
   stripe listen --forward-to localhost:8080/webhooks/stripe
   ```

4. **Copy the webhook secret** from the CLI output and set it as `STRIPE_WEBHOOK_SECRET`

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

2. **Copy the webhook secret** and set it as `STRIPE_WEBHOOK_SECRET`

## Step 5: Test the Integration

1. **Start your application**:
   ```bash
   export STRIPE_SECRET_KEY=sk_test_...
   export STRIPE_PUBLISHABLE_KEY=pk_test_...
   export STRIPE_WEBHOOK_SECRET=whsec_...
   export STRIPE_PRICE_ID=price_...
   ./goread2
   ```

2. **Test subscription flow**:
   - Try adding more than 20 feeds to trigger the subscription prompt
   - Click "Upgrade to Pro" in the modal
   - Complete the test payment using Stripe's test card: `4242 4242 4242 4242`
   - Verify the subscription is activated

3. **Test webhook processing**:
   - Check your application logs for webhook events
   - Verify subscription status updates in the database

## Stripe Test Cards

Use these test card numbers for development:

- **Successful payment**: `4242 4242 4242 4242`
- **Declined payment**: `4000 0000 0000 0002`
- **Requires authentication**: `4000 0025 0000 3155`

Use any future expiry date, any 3-digit CVC, and any ZIP code.

## Subscription Management

### Customer Portal

GoRead2 includes a customer portal integration where users can:
- Update payment methods
- Download invoices
- Cancel subscriptions
- View billing history

Access via: `/api/subscription/portal` (creates a session)

### Manual Subscription Management

You can also manage subscriptions through the Stripe Dashboard:
- View all customers and subscriptions
- Issue refunds
- Cancel or modify subscriptions
- View analytics and revenue data

## Troubleshooting

### Common Issues

1. **"Stripe not configured" error**:
   - Ensure all environment variables are set
   - Run `go run cmd/setup-stripe/main.go validate`

2. **Webhook signature verification failed**:
   - Check that `STRIPE_WEBHOOK_SECRET` matches your webhook endpoint
   - Ensure webhook URL is publicly accessible

3. **Product/Price not found**:
   - Run `go run cmd/setup-stripe/main.go create-product`
   - Update `STRIPE_PRICE_ID` with the generated price ID

4. **Payment fails in production**:
   - Switch to live API keys
   - Ensure webhook endpoint uses HTTPS
   - Check Stripe Dashboard for error details

### Logs and Monitoring

- **Application logs**: Check for Stripe webhook processing logs
- **Stripe Dashboard**: View webhook delivery attempts and errors
- **Database**: Verify subscription status updates in users table

## Security Considerations

1. **API Keys**: Never commit secret keys to version control
2. **Webhooks**: Always verify webhook signatures
3. **HTTPS**: Use HTTPS in production for webhook endpoints
4. **Scope**: Use restricted API keys when possible

## Cost Structure

- **Stripe fees**: 2.9% + 30¢ per successful charge
- **GoRead2 Pro**: $2.99/month per user
- **Free trial**: 30 days, up to 20 feeds

## Going Live

When ready for production:

1. **Switch to live API keys** in your environment variables
2. **Update webhook endpoints** to production URLs
3. **Test the complete flow** with real payment methods
4. **Monitor** the Stripe Dashboard for any issues

## Support

- **Stripe Documentation**: [stripe.com/docs](https://stripe.com/docs)
- **Stripe Support**: Available through the dashboard
- **GoRead2 Issues**: Report integration issues in the GitHub repository