# Stripe Setup Guide

Guide for integrating Stripe payment processing with GoRead2's subscription system, for developers configuring billing in a new environment.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Subscription Flag](#subscription-flag)
- [Step 1: Get Stripe API Keys](#step-1-get-stripe-api-keys)
- [Step 2: Configure Environment Variables](#step-2-configure-environment-variables)
- [Step 3: Create Product and Price](#step-3-create-product-and-price)
- [Step 4: Set Up Webhooks](#step-4-set-up-webhooks)
- [Step 5: Test the Integration](#step-5-test-the-integration)
- [Subscription Management](#subscription-management)
- [Trial and Price](#trial-and-price)
- [Checking the Webhook Endpoint](#checking-the-webhook-endpoint)
- [Security](#security)
- [Troubleshooting](#troubleshooting)
- [Related Documentation](#related-documentation)

## Overview

GoRead2 uses Stripe for subscription billing:

- A new account gets a 30-day free trial limited to 20 feeds.
- GoRead2 Pro, a $9.99 monthly subscription, removes the feed limit.
- The Stripe customer portal handles payment methods, invoices, and cancellation.
- Stripe webhooks keep each account's subscription status current.

Billing is active only when `SUBSCRIPTION_ENABLED=true`.

## Subscription Flag

`SUBSCRIPTION_ENABLED` turns the subscription system on or off. Production sets it to `true` in `app.yaml`, and the server logs `Subscription system enabled: true` or `false` at startup. [configuration.md](configuration.md#subscription-flag) lists the accepted values.

**When enabled:**

- A new account starts a 30-day free trial limited to 20 feeds (see [Trial and Price](#trial-and-price)).
- The server registers the Stripe routes: `GET /api/stripe/config`, `POST /api/subscription/checkout`, `POST /api/subscription/portal`, `POST /webhooks/stripe`, and the `/subscription/success` and `/subscription/cancel` pages.
- The web app shows upgrade prompts when the feed limit is reached, and the account page offers checkout and the Stripe customer portal.
- The admin CLI offers `grant-months`.

**When disabled:**

- `GET /api/subscription` reports every account with `status` `unlimited` and `feed_limit` `-1`, and no feed limit or trial is enforced.
- The Stripe routes listed above are not registered and return 404.
- The account page shows "Unlimited Access" and no upgrade or billing controls.
- The admin CLI refuses `grant-months`.

To check the mode from a signed-in session, call `GET /api/subscription` with the session cookie, which is named `session_id_local` on a local server and `session_id` in production:

```bash
curl -b "session_id_local=<session-id>" http://localhost:8080/api/subscription
```

A new account reports `"status":"trial"` and `"feed_limit":20` when subscriptions are enabled, and `"status":"unlimited"` and `"feed_limit":-1` when they are disabled.

**Disabling after users have subscribed:** disabling the flag does not cancel existing Stripe subscriptions, which keep billing. Cancel them in the Stripe Dashboard if users should no longer be charged. Keep the Stripe credentials configured while any user has a subscription status of `active`, `past_due`, or `unpaid`: with the flag disabled, the server still loads them when they are available and uses them only for [account deletion](#account-deletion).

## Prerequisites

1. **Stripe account**: create one at [stripe.com](https://stripe.com).
2. **GoRead2 running locally** with OAuth configured (see [setup.md](setup.md)).
3. **HTTPS in production**, which Stripe requires for webhook endpoints.

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

`app.yaml` carries only the feature flag:

```yaml
# app.yaml
env_variables:
  SUBSCRIPTION_ENABLED: "true"
```

The four Stripe values are stored in Google Secret Manager under the fixed names `stripe-secret-key`, `stripe-publishable-key`, `stripe-webhook-secret`, and `stripe-price-id`. The server fetches them at runtime, and they must not appear in `app.yaml`. See [deployment.md](deployment.md#setting-up-google-secret-manager) for the commands that create them, and [configuration.md](configuration.md#subscriptions-and-stripe) for when each is required.

## Step 3: Create Product and Price

Use the setup tool to create the GoRead2 Pro product:

With the variables from Step 2 exported:

```bash
go run cmd/setup-stripe/main.go validate        # checks the Stripe configuration
go run cmd/setup-stripe/main.go create-product  # creates the product and price
```

`create-product` creates the GoRead2 Pro product with a $9.99 monthly USD price and prints its ID:

```
✅ Successfully created product and price!
Price ID: price_1ABC123def456789
Amount: $9.99/month
```

Set `STRIPE_PRICE_ID` to the printed ID.

## Step 4: Set Up Webhooks

Webhooks notify the application when subscription events occur.

### Development (using Stripe CLI)

1. Install the [Stripe CLI](https://docs.stripe.com/stripe-cli) and run `stripe login`.
2. Forward webhooks to the local server:
   ```bash
   stripe listen --forward-to localhost:8080/webhooks/stripe
   ```
3. Set `STRIPE_WEBHOOK_SECRET` to the `whsec_` signing secret the command prints, then restart the server.

### Production

1. In the Stripe Dashboard, under **Developers → Webhooks**, add an endpoint:
   - URL: `https://your-domain.com/webhooks/stripe`
   - Events to send:
     - `checkout.session.completed`
     - `customer.subscription.created`
     - `customer.subscription.updated`
     - `customer.subscription.deleted`

   These are the only events `internal/handlers/payment_handler.go` handles. Other events are acknowledged and ignored.

2. Store the endpoint's signing secret as the `stripe-webhook-secret` secret (see [Step 2](#production-app-engine)).

## Step 5: Test the Integration

With the Step 2 variables exported and `stripe listen` running, start the server with `make dev`, then:

1. Add feeds until the 20-feed limit is reached. The web app shows the Upgrade to Pro prompt.
2. Choose Upgrade to Pro and complete checkout with the test card `4242 4242 4242 4242`, any future expiry date, and any CVC. Stripe's [testing guide](https://docs.stripe.com/testing) lists cards for declines, 3D Secure, and other cases.
3. Checkout returns to `/subscription/success`, and the account page shows the Pro subscription with unlimited feeds.

Watch the application's stdout (see [troubleshooting.md](troubleshooting.md#logging-and-debugging)) for entries like the following. Stripe object IDs are truncated to their first 8 characters, and user IDs, customer IDs, and subscription metadata are not logged.
```
INFO: Webhook - Received event: checkout.session.completed (ID: evt_...)
INFO: Webhook - Checkout session cs_test_*** completed
INFO: Webhook - Received event: customer.subscription.created (ID: evt_...)
INFO: Webhook - Updated subscription sub_1Abc*** (status: active)
```

## Subscription Management

### Customer Portal

The Manage Subscription button on the account page calls `POST /api/subscription/portal`, which creates a Stripe customer portal session, and redirects the browser to it. The portal's features are configured in the Stripe Dashboard, not in GoRead2.

### Account Deletion

Deleting a GoRead2 account (`DELETE /api/account`) deletes the user's Stripe customers, which cancels any subscription immediately, with no proration or refund. GoRead2 does not store customer IDs, so it deletes the customer on the user's subscription plus any customer found by email search whose `user_id` metadata names this user. Customers without that metadata are never deleted by email match alone, since they may have been created outside GoRead2. Stripe keeps the payment records it is legally required to retain.

A second deletion pass runs 15 minutes later, after the `User` record is gone, and deletes any customer whose `user_id` metadata names the deleted user. This removes a customer created by a checkout or portal request on another instance's stale session cache after the first pass searched Stripe.

Account deletion uses the Stripe credentials even when `SUBSCRIPTION_ENABLED=false`, so that subscriptions started while the flag was on are still cancelled. Without credentials, deleting an account with a live subscription (status `active`, `past_due`, or `unpaid`) fails and leaves the account in place, and the server logs a warning at startup.

Stripe then sends `customer.subscription.deleted`, often while the deletion is still in progress. The subscription update reads and writes the user record in one transaction, so it never recreates a user that the deletion has removed. The webhook handler acknowledges events for missing users instead of failing, so Stripe does not retry them.

## Trial and Price

The trial length and the free-tier feed limit are fixed in code: `FreeTrialFeedLimit` in `internal/services/subscription_service.go` sets the 20-feed limit, and the database layer sets each account's trial end to 30 days after creation. Changing either requires a code change.

The trial is GoRead2's own; the Stripe price carries no trial period, so checkout bills immediately. To change the price, create a new price on the GoRead2 Pro product in the Stripe Dashboard and update the `stripe-price-id` secret. Existing subscriptions keep their original price. The web app shows the price as text in `web/static/js/account.js`, which must be updated to match.

## Checking the Webhook Endpoint

A request without a valid Stripe signature confirms the route is registered and verifying signatures:

```bash
curl -i -X POST https://your-domain.com/webhooks/stripe -d '{}'
```

The expected response is `400` with `Webhook signature verification failed.` A `404` means the route is not registered, because `SUBSCRIPTION_ENABLED` is not `true`. Stripe's own delivery attempts are listed under **Developers → Webhooks** in the Stripe Dashboard.

## Security

- Stripe keys are stored in Secret Manager in production and never committed. Rotate a key in the Stripe Dashboard if it is exposed, then update its secret.
- The webhook handler verifies each event's signature against `STRIPE_WEBHOOK_SECRET` before acting on it.

- GoRead2 never handles card details, which Stripe collects on its hosted checkout page.
- GoRead2 stores the Stripe subscription ID on the user record, not the customer ID. Customers are linked to GoRead2 users through `user_id` metadata on the Stripe customer.
- Account deletion deletes the Stripe customer (see [Account Deletion](#account-deletion)).

## Troubleshooting

See [Troubleshooting Guide](troubleshooting.md#subscription-issues) for Stripe configuration errors, webhook failures, missing products/prices, and production payment issues.

## Related Documentation

- [Setup Guide](setup.md) - Environment variables and installation
- [Configuration Reference](configuration.md) - Every Stripe variable and its Secret Manager name
- [Deployment Guide](deployment.md) - Storing Stripe secrets in Secret Manager
- [Troubleshooting Guide](troubleshooting.md) - Stripe-related issues
