# Setup Guide

Installation and configuration guide for running GoRead2 locally.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Google OAuth Setup](#google-oauth-setup)
- [Installation](#installation)
- [Configuration](#configuration)
- [Subscriptions](#subscriptions)
- [First Run](#first-run)
- [Troubleshooting](#troubleshooting)
- [Related Documentation](#related-documentation)

## Prerequisites

- **Go**, the version `go.mod`'s `go` directive names
- **Node.js**, the version `package.json`'s `engines` field names (needed to build minified frontend assets; these are not checked into the repo)
- **Google Cloud Project**, for OAuth authentication; not needed for `make dev-login` (see [First Run](#first-run) below)
- **SQLite3**, automatically included with `go-sqlite3`
- **Stripe account**, optional, for subscription features (see [stripe.md](stripe.md))

## Google OAuth Setup

Skip this section entirely to use `make dev-login` instead (see [First Run](#first-run)); it needs no Google credentials.

### 1. Create a Google Cloud Project

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select an existing one

### 2. Configure the OAuth Consent Screen

1. Navigate to APIs & Services → OAuth consent screen
2. Configure the consent screen with the application's details
3. Add test users if in development mode

### 3. Create OAuth Credentials

1. Go to APIs & Services → Credentials
2. Create OAuth 2.0 Client ID
3. Choose "Web application"
4. Set authorized redirect URIs:
   - Development: `http://localhost:8080/auth/callback`
   - Production: `https://your-domain.com/auth/callback`

The resulting client ID and secret go into `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET` in [Installation](#installation) below.

## Installation

### Local Development

1. **Clone the project:**
   ```bash
   git clone https://github.com/jeffreyp/goread2.git
   cd goread2
   ```

2. **Install dependencies:**
   ```bash
   go mod tidy
   ```

3. **Set up environment:**

   Export the required values in the shell; the application does not load `.env` files. `.env.example` describes the configuration variables, as a reference only. [configuration.md](configuration.md) lists every variable.

   ```bash
   export GOOGLE_CLIENT_ID="your-client-id"
   export GOOGLE_CLIENT_SECRET="your-client-secret"
   export GOOGLE_REDIRECT_URL="http://localhost:8080/auth/callback"

   # Optional: for subscription features, see stripe.md
   export STRIPE_SECRET_KEY="sk_test_your-secret-key"
   export STRIPE_PUBLISHABLE_KEY="pk_test_your-publishable-key"
   export STRIPE_WEBHOOK_SECRET="whsec_your-webhook-secret"
   export STRIPE_PRICE_ID="price_your-price-id"
   ```

4. **Build frontend assets:**

   This step is required before first run and after any change to frontend source: minified JS and CSS are not checked into `web/static/`, and the app will not render without them.

   ```bash
   npm install
   make build-frontend
   ```

5. **Run the application:**
   ```bash
   make dev
   ```

   `make dev` validates configuration before starting the server. Alternatively, build and run the binary directly:

   ```bash
   make build
   ./goread2
   ```

   Or skip Google OAuth entirely with `make dev-login` (see [First Run](#first-run) below).

6. **Access the application:**
   Navigate to `http://localhost:8080` and sign in.

## Configuration

### Environment Variables

A local server needs only `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, and `GOOGLE_REDIRECT_URL`, exported as in step 3 above (or none at all, with `make dev-login`). [configuration.md](configuration.md) lists every variable, with defaults and the values production requires.

### Database Configuration

- **Local Development**: SQLite database (`goread2.db`)
- **Production**: Google Cloud Datastore (when `GOOGLE_CLOUD_PROJECT` is set)

### Session Configuration

- **Security**: HTTP-only cookies with secure flags in production
- **Expiration**: 7 days, extended on each request (a sliding window), not a fixed 24-hour lifetime
- **Cleanup**: Automatic cleanup of expired sessions

## Subscriptions

The subscription system is off unless `SUBSCRIPTION_ENABLED=true`, so every local account has unlimited feeds and no billing. To work on billing locally, follow [stripe.md](stripe.md), which also describes what the flag changes.

## First Run

### 1. Authentication

The simplest way to sign in locally is `make dev-login`, which needs no Google credentials and opens a local sign-in form instead of Google's consent screen; see [testing.md](testing.md#local-dev-login-make-dev-login) for how it works. With real Google OAuth configured instead:

1. Navigate to the application URL
2. Click "Sign in with Google" to authenticate
3. Grant the requested permissions
4. The login redirects to the user's personal dashboard

### 2. Managing Feeds

1. **Adding feeds**: Click "Add Feed" and enter an RSS/Atom URL
2. **OPML import**: Click "Import OPML" to import feeds from other RSS readers
3. **Feed discovery**: Supports both RSS and Atom formats

### 3. Admin Setup (Optional)

Granting the first admin account needs `ADMIN_TOKEN` or `INITIAL_ADMIN_EMAILS` to be set up first; see [Granting the First Admin](admin.md#granting-the-first-admin) for the commands.

## Troubleshooting

See [Troubleshooting Guide](troubleshooting.md) for authentication, feed, database, and performance issues.

## Related Documentation

- [Deployment Guide](deployment.md) - Production deployment
- [Stripe Setup](stripe.md) - Configuring subscription payments
- [Admin Guide](admin.md) - User management
- [Testing Guide](testing.md) - Running the server with `make dev-login`
- [API Reference](api.md) - Integration reference
- [Troubleshooting Guide](troubleshooting.md) - Common issues
