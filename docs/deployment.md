# Deployment Guide

Complete guide for deploying GoRead2 to production environments.

## Overview

GoRead2 supports multiple deployment options:
- **Google App Engine** (Recommended)
- **Docker/Containers**
- **Traditional VPS/Server**

All deployment methods require:
- Google OAuth 2.0 configuration
- Multi-user database setup
- Session management
- Production security considerations

## Prerequisites

### Google Cloud Setup

1. **Google Cloud Project**
   - Create a new Google Cloud Project or use existing one
   - Enable the following APIs:
     - App Engine Admin API (for GAE deployment)
     - Cloud Datastore API (for production database)
     - Cloud Build API (for deployment)

2. **Google OAuth 2.0 Setup**
   - Go to [Google Cloud Console](https://console.cloud.google.com/)
   - Navigate to APIs & Services → Credentials
   - Create OAuth 2.0 Client ID
   - Configure OAuth consent screen
   - Set authorized redirect URIs for your deployment

3. **Install Google Cloud SDK**
   ```bash
   # Download and install from: https://cloud.google.com/sdk/docs/install
   curl https://sdk.cloud.google.com | bash
   exec -l $SHELL
   gcloud init
   ```

4. **Authentication**
   ```bash
   gcloud auth login
   gcloud config set project YOUR_PROJECT_ID
   ```

## Google App Engine (Recommended)

### Environment Variables Setup

**Important**: For Google App Engine deployments, environment variables should be configured in Google Secret Manager for security, not hardcoded in `app.yaml`.

**Secret Reference Convention**: The application supports a `_secret:` prefix for environment variables to explicitly trigger Secret Manager lookups. For example, setting `GOOGLE_CLIENT_ID=_secret:my-client-id` will fetch the secret from Google Secret Manager. This convention is consistent across all credentials (OAuth and Stripe) and prevents accidental conflicts with actual secret values.

#### Setting up Google Secret Manager

1. **Enable the Secret Manager API:**
   ```bash
   gcloud services enable secretmanager.googleapis.com
   ```

2. **Create secrets for each environment variable:**
   ```bash
   # OAuth configuration
   echo -n "your-oauth-client-id" | gcloud secrets create google-client-id --data-file=-
   echo -n "your-oauth-client-secret" | gcloud secrets create google-client-secret --data-file=-

   # CSRF secret (REQUIRED for production)
   openssl rand -base64 32 | gcloud secrets create csrf-secret --data-file=-

   # Stripe configuration (if using subscriptions)
   echo -n "sk_live_your-secret-key" | gcloud secrets create stripe-secret-key --data-file=-
   echo -n "pk_live_your-publishable-key" | gcloud secrets create stripe-publishable-key --data-file=-
   echo -n "whsec_your-webhook-secret" | gcloud secrets create stripe-webhook-secret --data-file=-
   echo -n "price_your-price-id" | gcloud secrets create stripe-price-id --data-file=-
   ```

3. **Grant App Engine access to secrets:**
   ```bash
   PROJECT_ID=$(gcloud config get-value project)

   gcloud secrets add-iam-policy-binding google-client-id \
       --member="serviceAccount:${PROJECT_ID}@appspot.gserviceaccount.com" \
       --role="roles/secretmanager.secretAccessor"

   gcloud secrets add-iam-policy-binding google-client-secret \
       --member="serviceAccount:${PROJECT_ID}@appspot.gserviceaccount.com" \
       --role="roles/secretmanager.secretAccessor"

   gcloud secrets add-iam-policy-binding csrf-secret \
       --member="serviceAccount:${PROJECT_ID}@appspot.gserviceaccount.com" \
       --role="roles/secretmanager.secretAccessor"

   # Repeat for other secrets (Stripe, etc.)...
   ```

### app.yaml Configuration

**Option 1: Using Secret Manager References (Recommended)**
```yaml
runtime: go123

env_variables:
  GIN_MODE: release
  GOOGLE_CLIENT_ID: ${GOOGLE_CLIENT_ID}
  GOOGLE_CLIENT_SECRET: ${GOOGLE_CLIENT_SECRET}
  GOOGLE_REDIRECT_URL: "https://your-app.appspot.com/auth/callback"
  CSRF_SECRET: ${CSRF_SECRET}
  SUBSCRIPTION_ENABLED: "true"
  STRIPE_SECRET_KEY: ${STRIPE_SECRET_KEY}
  STRIPE_PUBLISHABLE_KEY: ${STRIPE_PUBLISHABLE_KEY}
  STRIPE_WEBHOOK_SECRET: ${STRIPE_WEBHOOK_SECRET}
  STRIPE_PRICE_ID: ${STRIPE_PRICE_ID}
```

**Option 2: Direct Secret Manager Integration**
```yaml
runtime: go123

env_variables:
  GIN_MODE: release
  GOOGLE_REDIRECT_URL: "https://your-app.appspot.com/auth/callback"
  SUBSCRIPTION_ENABLED: "true"

# Secrets are automatically injected by App Engine from Secret Manager
# when using the same names as environment variables
```

**Option 3: Manual Configuration (Less Secure)**
```yaml
# Only use this for development - not recommended for production
runtime: go123

env_variables:
  GIN_MODE: release
  GOOGLE_CLIENT_ID: "your-oauth-client-id"
  GOOGLE_CLIENT_SECRET: "your-oauth-client-secret"
  GOOGLE_REDIRECT_URL: "https://your-app.appspot.com/auth/callback"
  CSRF_SECRET: "your-base64-csrf-secret"
  SUBSCRIPTION_ENABLED: "true"
  STRIPE_SECRET_KEY: "sk_live_your-secret-key"
  STRIPE_PUBLISHABLE_KEY: "pk_live_your-publishable-key"
  STRIPE_WEBHOOK_SECRET: "whsec_your-webhook-secret"
  STRIPE_PRICE_ID: "price_your-price-id"

handlers:
- url: /static
  static_dir: web/static
  secure: always

- url: /.*
  script: auto
  secure: always

instance_class: F2  # 512MB RAM - good balance of cost and performance

automatic_scaling:
  min_instances: 0
  max_instances: 10
```

### cron.yaml Configuration

```yaml
cron:
- description: "Refresh RSS feeds for all users"
  url: /api/feeds/refresh
  schedule: every 1 hours
  target: default

- description: "Clean up expired sessions"
  url: /auth/cleanup
  schedule: every 1 hours
  target: default
```

### Deployment Steps

1. **Configure OAuth redirect URI:**
   Update OAuth configuration with production URL:
   `https://your-app.appspot.com/auth/callback`

2. **Set up environment variables:**
   ```bash
   # Option A: Export variables for substitution in app.yaml
   export GOOGLE_CLIENT_ID="your-oauth-client-id"
   export GOOGLE_CLIENT_SECRET="your-oauth-client-secret"
   export STRIPE_SECRET_KEY="sk_live_your-secret-key"
   export STRIPE_PUBLISHABLE_KEY="pk_live_your-publishable-key"
   export STRIPE_WEBHOOK_SECRET="whsec_your-webhook-secret"
   export STRIPE_PRICE_ID="price_your-price-id"

   # Option B: Use Secret Manager (recommended for production)
   # Secrets will be automatically accessed by App Engine if properly configured
   ```

3. **Initialize App Engine:**
   ```bash
   gcloud app create --region=us-central1
   ```

4. **Deploy application:**
   ```bash
   # Deploy to development environment (with validation)
   make deploy-dev

   # Deploy to production environment (with strict validation and tests)
   make deploy-prod

   # Deploy cron jobs (manual step)
   gcloud app deploy cron.yaml
   ```

5. **Verify deployment:**
   ```bash
   # Open application
   gcloud app browse

   # Check logs for any configuration issues
   gcloud app logs tail -s default
   ```

### Database Configuration (App Engine)

- **Production**: Google Cloud Datastore (automatically detected)
- **Multi-user entities**: Users, UserFeeds, UserArticles
- **User isolation**: All queries filtered by authenticated user ID
- **Scalability**: Handles multiple concurrent users efficiently

## Docker Deployment

### Dockerfile

```dockerfile
FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o goread2 .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/goread2 .
COPY --from=builder /app/web ./web

EXPOSE 8080
CMD ["./goread2"]
```

### docker-compose.yml

```yaml
version: '3.8'
services:
  goread2:
    build: .
    ports:
      - "8080:8080"
    environment:
      - GOOGLE_CLIENT_ID=your-oauth-client-id
      - GOOGLE_CLIENT_SECRET=your-oauth-client-secret
      - GOOGLE_REDIRECT_URL=https://your-domain.com/auth/callback
      - GIN_MODE=release
      - SUBSCRIPTION_ENABLED=true
      - STRIPE_SECRET_KEY=sk_live_your-secret-key
      - STRIPE_PUBLISHABLE_KEY=pk_live_your-publishable-key
    volumes:
      - ./data:/root/data
    restart: unless-stopped

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
      - ./ssl:/etc/nginx/ssl
    depends_on:
      - goread2
    restart: unless-stopped
```

### nginx.conf (SSL Termination)

```nginx
events {
    worker_connections 1024;
}

http {
    upstream goread2 {
        server goread2:8080;
    }
    
    server {
        listen 80;
        server_name your-domain.com;
        return 301 https://$server_name$request_uri;
    }
    
    server {
        listen 443 ssl http2;
        server_name your-domain.com;
        
        ssl_certificate /etc/nginx/ssl/cert.pem;
        ssl_certificate_key /etc/nginx/ssl/key.pem;
        
        location / {
            proxy_pass http://goread2;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }
    }
}
```

### Docker Deployment Steps

1. **Configure OAuth:**
   Set production redirect URI in Google Console:
   `https://your-domain.com/auth/callback`

2. **Create environment file:**
   ```bash
   cat > .env << EOF
   GOOGLE_CLIENT_ID=your-oauth-client-id
   GOOGLE_CLIENT_SECRET=your-oauth-client-secret
   GOOGLE_REDIRECT_URL=https://your-domain.com/auth/callback
   GIN_MODE=release
   SUBSCRIPTION_ENABLED=true
   EOF
   ```

3. **Deploy with Docker Compose:**
   ```bash
   docker-compose up -d
   ```

## Traditional VPS/Server

### Prerequisites

```bash
# Install Go 1.23+
wget https://golang.org/dl/go1.23.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.23.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Install dependencies
sudo apt update
sudo apt install -y nginx certbot python3-certbot-nginx sqlite3
```

### systemd Service

```ini
# /etc/systemd/system/goread2.service
[Unit]
Description=GoRead2 Multi-User RSS Reader
After=network.target

[Service]
Type=simple
User=goread2
WorkingDirectory=/opt/goread2
ExecStart=/opt/goread2/goread2
Restart=always
RestartSec=5

Environment=GOOGLE_CLIENT_ID=your-oauth-client-id
Environment=GOOGLE_CLIENT_SECRET=your-oauth-client-secret
Environment=GOOGLE_REDIRECT_URL=https://your-domain.com/auth/callback
Environment=GIN_MODE=release
Environment=SUBSCRIPTION_ENABLED=true

[Install]
WantedBy=multi-user.target
```

### Server Deployment Steps

1. **Create user and directories:**
   ```bash
   sudo useradd -r -s /bin/false goread2
   sudo mkdir -p /opt/goread2
   sudo chown goread2:goread2 /opt/goread2
   ```

2. **Build and deploy:**
   ```bash
   go build -o goread2 .
   sudo cp goread2 /opt/goread2/
   sudo cp -r web /opt/goread2/
   sudo chown -R goread2:goread2 /opt/goread2
   ```

3. **Configure service:**
   ```bash
   sudo systemctl enable goread2
   sudo systemctl start goread2
   ```

4. **Setup SSL with Let's Encrypt:**
   ```bash
   sudo certbot --nginx -d your-domain.com
   ```

## Environment Variables

### Security Note for Google App Engine

**Important**: When deploying to Google App Engine, sensitive environment variables (like API keys and secrets) should be stored in Google Secret Manager rather than hardcoded in `app.yaml` for security. See the [Google App Engine section](#google-app-engine-recommended) for detailed setup instructions.

### Required Variables

- `GOOGLE_CLIENT_ID` - OAuth 2.0 client ID from Google Console
- `GOOGLE_CLIENT_SECRET` - OAuth 2.0 client secret from Google Console ⚠️ **Store in Secret Manager for GAE**
- `GOOGLE_REDIRECT_URL` - OAuth callback URL (must match Google Console)
- `CSRF_SECRET` - Base64-encoded 32-byte secret for CSRF token generation ⚠️ **REQUIRED in production - app will fail to start if missing**

### Optional Variables

- `GOOGLE_CLOUD_PROJECT` - Use Google Cloud Datastore (for GAE)
- `GIN_MODE` - Set to "release" for production
- `PORT` - Server port (default: 8080)
- `SESSION_SECRET` - Custom session encryption key (auto-generated if not set)
- `SUBSCRIPTION_ENABLED` - Enable/disable subscription system (default: false)

### Stripe Variables (if using subscriptions)

⚠️ **All Stripe keys should be stored in Google Secret Manager for App Engine deployments**

- `STRIPE_SECRET_KEY` - Stripe secret key for API calls
- `STRIPE_PUBLISHABLE_KEY` - Stripe publishable key for frontend
- `STRIPE_WEBHOOK_SECRET` - Webhook endpoint secret for signature verification
- `STRIPE_PRICE_ID` - Stripe price ID for subscription product

## Security Considerations

### Authentication Security

- **OAuth 2.0**: Industry-standard Google OAuth integration
- **Session security**: HTTP-only cookies with secure flags
- **CSRF protection**: Built-in protection against cross-site requests
- **No password storage**: Leverages Google's authentication infrastructure

### Data Isolation

- **User separation**: Complete isolation of user data in database
- **Query filtering**: All database queries filtered by authenticated user ID
- **Session management**: Secure session creation, validation, and cleanup
- **API protection**: All endpoints require valid authentication

### Production Security

```yaml
# app.yaml security headers
handlers:
- url: /.*
  script: auto
  secure: always
  http_headers:
    Strict-Transport-Security: "max-age=31536000; includeSubDomains"
    X-Content-Type-Options: "nosniff"
    X-Frame-Options: "DENY"
    X-XSS-Protection: "1; mode=block"
```

## Monitoring and Maintenance

### Health Checks

```bash
# App Engine logs
gcloud app logs tail -s default

# Docker logs
docker-compose logs -f goread2

# System logs
sudo journalctl -u goread2 -f
```

### Performance Optimization

1. **Caching**: Session, feed, and static asset caching
2. **Database optimization**: Proper indexes and connection pooling
3. **Scaling**: Horizontal scaling with load balancers
4. **Cleanup**: Regular cleanup of expired sessions and old articles

## Testing in Production

### Build and Test Locally

```bash
# Run complete build and test suite
make all

# Run tests only
make test

# Validate configuration
make validate-config
```

### Deployment Testing

```bash
# Test OAuth flow
curl -I https://your-domain.com/auth/login

# Test API endpoints (requires authentication)
curl -H "Cookie: session=..." https://your-domain.com/api/feeds
```

### Load Testing

```bash
# Install and run load testing
npm install -g artillery
artillery quick --count 10 --num 50 https://your-domain.com
```

## Troubleshooting

### OAuth Issues

1. **Invalid redirect URI:**
   - Verify redirect URI in Google Console matches exactly
   - Check HTTPS vs HTTP protocol
   - Ensure no trailing slashes

2. **OAuth consent screen errors:**
   - Complete OAuth consent screen configuration
   - Add test users if in development mode
   - Verify app domain ownership

### Session Problems

1. **Users logged out frequently:**
   - Check session expiration settings
   - Verify session cleanup isn't too aggressive
   - Check cookie security settings

2. **Session not persisting:**
   - Verify HTTPS in production
   - Check cookie domain settings
   - Ensure session secret is consistent

### Database Issues

1. **User data isolation failures:**
   - Review database queries for user filtering
   - Run isolation tests: `go test ./test/integration/...`
   - Check middleware authentication

2. **Performance issues:**
   - Add database indexes on user_id columns
   - Monitor query performance
   - Consider database optimization

## Cost Optimization

### Google App Engine

- **Instance management**: Use automatic scaling with min 0 instances
- **Datastore usage**: Monitor read/write operations
- **Bandwidth**: Cache RSS feeds to reduce external requests
- **Free tier**: Leverage GAE free tier limits

### Alternative Hosting

- **VPS costs**: Consider resource requirements for expected user count
- **Database costs**: SQLite sufficient for moderate user bases
- **SSL certificates**: Use Let's Encrypt for free SSL
- **CDN**: CloudFlare free tier for static asset caching

## Next Steps

- Configure [Stripe payments](STRIPE.md) for subscription features
- Set up [monitoring and logging](monitoring.md) for production
- Review [security best practices](security.md)
- Plan [backup and recovery](backup.md) procedures