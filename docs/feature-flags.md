# Feature Flags

GoRead2 has one feature flag, `SUBSCRIPTION_ENABLED`, which turns the subscription system on or off. Production sets it to `true` in `app.yaml`.

## Table of Contents

- [Configuration](#configuration)
- [When Subscriptions Are Enabled](#when-subscriptions-are-enabled)
- [When Subscriptions Are Disabled](#when-subscriptions-are-disabled)
- [Environment Variables Summary](#environment-variables-summary)
- [Checking the Current Mode](#checking-the-current-mode)
- [Troubleshooting](#troubleshooting)
- [Related Documentation](#related-documentation)

## Configuration

```bash
export SUBSCRIPTION_ENABLED=true   # enable
export SUBSCRIPTION_ENABLED=false  # disable (the default)
```

The server reads the flag once at startup, so a change takes effect on the next restart or deployment. Accepted values, compared without regard to case:

- `true`, `1`, `yes`, `on`, `enabled` enable subscriptions.
- `false`, `0`, `no`, `off`, `disabled` disable them.
- An empty or unset value disables them.

## When Subscriptions Are Enabled

- A new account starts a 30-day free trial limited to 20 feeds. Both values are fixed in code (see [stripe.md](stripe.md#trial-and-price)).
- The server registers the Stripe routes: `GET /api/stripe/config`, `POST /api/subscription/checkout`, `POST /api/subscription/portal`, `POST /webhooks/stripe`, and the `/subscription/success` and `/subscription/cancel` pages.
- The web app shows upgrade prompts when the feed limit is reached, and the account page offers checkout and the Stripe customer portal.
- The admin CLI offers `grant-months`.

The four Stripe variables must be configured (see [stripe.md](stripe.md#step-2-configure-environment-variables)).

## When Subscriptions Are Disabled

- `GET /api/subscription` reports every account with `status` `unlimited` and `feed_limit` `-1`, and no feed limit or trial is enforced.
- The Stripe routes listed above are not registered and return 404.
- The account page shows "Unlimited Access" and no upgrade or billing controls.
- The admin CLI refuses `grant-months`.

**Disabling after users have subscribed:** disabling the flag does not cancel existing Stripe subscriptions, which keep billing. Cancel them in the Stripe Dashboard if users should no longer be charged. Keep the Stripe credentials configured while any user has a subscription status of `active`, `past_due`, or `unpaid`. With the flag disabled, the server still loads the credentials when they are available and uses them only for account deletion, which deletes the user's Stripe customers and so cancels the subscription (see [stripe.md](stripe.md#account-deletion)). Without the credentials, deleting an account with a live subscription fails with an error and leaves the account in place, and the server logs a warning at startup.

## Environment Variables Summary

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `SUBSCRIPTION_ENABLED` | No | `false` | Enables the subscription system |
| `STRIPE_SECRET_KEY` | When enabled, or while any subscription is live | None | Stripe secret key for API calls |
| `STRIPE_PUBLISHABLE_KEY` | When enabled | None | Stripe publishable key for the frontend |
| `STRIPE_WEBHOOK_SECRET` | When enabled | None | Webhook signing secret |
| `STRIPE_PRICE_ID` | When enabled | None | Stripe price ID for the Pro subscription |

[deployment.md](deployment.md#environment-variables) lists every variable the server reads.

## Checking the Current Mode

The server logs `Subscription system enabled: true` or `false` at startup. To check from a signed-in session, call `GET /api/subscription` with the session cookie, which is named `session_id_local` on a local server and `session_id` in production:

```bash
curl -b "session_id_local=<session-id>" http://localhost:8080/api/subscription
```

With subscriptions disabled, the response contains `"status":"unlimited"` and `"feed_limit":-1`. With them enabled, a new account reports `"status":"trial"` with `"feed_limit":20`.

## Troubleshooting

See [Troubleshooting Guide](troubleshooting.md#subscription-issues) for routing, configuration, and admin command issues related to the subscription flag.

## Related Documentation

- [Stripe Setup](stripe.md) - Configuring the Stripe integration this flag gates
- [Admin Guide](admin.md) - The `grant-months` command
- [Troubleshooting Guide](troubleshooting.md) - Common issues
