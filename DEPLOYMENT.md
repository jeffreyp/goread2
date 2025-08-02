# GoRead2 - Multi-User Deployment Guide

This guide explains how to deploy the multi-user GoRead2 RSS Reader with Google OAuth authentication to various platforms.

## Overview

GoRead2 now supports multiple users with Google OAuth authentication, requiring additional setup for:
- Google OAuth 2.0 configuration
- Multi-user database schema
- Session management
- User data isolation
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

## Deployment Options

### Option 1: Google App Engine (Recommended)

#### Project Structure for App Engine

```
goread2/
├── app.yaml                     # App Engine configuration
├── cron.yaml                    # Cron jobs for feed refresh
├── main.go                      # Application entry point
├── go.mod                       # Dependencies with OAuth libraries
├── internal/
│   ├── auth/                    # Authentication system
│   │   ├── auth.go             # Google OAuth integration
│   │   ├── middleware.go       # Auth middleware
│   │   └── session.go          # Session management
│   ├── database/
│   │   ├── schema.go           # Multi-user database schema
│   │   └── datastore.go        # Google Datastore implementation
│   ├── handlers/
│   │   ├── feed_handler.go     # User-aware feed handlers
│   │   └── auth_handler.go     # Authentication handlers
│   └── services/
│       └── feed_service.go     # Multi-user business logic
└── web/                        # Static files served by GAE
    ├── templates/
    │   └── index.html          # Updated with auth UI
    └── static/
        ├── css/
        └── js/
            └── app.js          # Frontend with auth integration
```

#### app.yaml Configuration

```yaml
runtime: go121

env_variables:
  GIN_MODE: release
  GOOGLE_CLIENT_ID: "your-oauth-client-id"
  GOOGLE_CLIENT_SECRET: "your-oauth-client-secret"
  GOOGLE_REDIRECT_URL: "https://your-app.appspot.com/auth/callback"

handlers:
- url: /static
  static_dir: web/static
  secure: always

- url: /.*
  script: auto
  secure: always

automatic_scaling:
  min_instances: 0
  max_instances: 10
  target_cpu_utilization: 0.6

resources:
  cpu: 1
  memory_gb: 0.5
  disk_size_gb: 10

# Session configuration for security
vpc_access_connector:
  name: projects/PROJECT_ID/locations/REGION/connectors/CONNECTOR_NAME
```

#### cron.yaml Configuration

```yaml
cron:
- description: "Refresh RSS feeds for all users"
  url: /api/feeds/refresh
  schedule: every 30 minutes
  target: default
  
- description: "Clean up expired sessions"
  url: /auth/cleanup
  schedule: every 1 hours
  target: default
```

#### Deployment Steps

1. **Configure OAuth redirect URI:**
   ```bash
   # Update OAuth configuration with production URL
   # https://your-app.appspot.com/auth/callback
   ```

2. **Set environment variables in app.yaml:**
   ```yaml
   env_variables:
     GOOGLE_CLIENT_ID: "your-production-client-id"
     GOOGLE_CLIENT_SECRET: "your-production-client-secret"
     GOOGLE_REDIRECT_URL: "https://your-app.appspot.com/auth/callback"
   ```

3. **Initialize App Engine:**
   ```bash
   gcloud app create --region=us-central1
   ```

4. **Deploy application:**
   ```bash
   # Deploy main application
   gcloud app deploy app.yaml
   
   # Deploy cron jobs
   gcloud app deploy cron.yaml
   ```

5. **Open application:**
   ```bash
   gcloud app browse
   ```

#### Database Configuration (App Engine)

- **Production**: Google Cloud Datastore (automatically detected)
- **Multi-user entities**: Users, UserFeeds, UserArticles
- **User isolation**: All queries filtered by authenticated user ID
- **Scalability**: Handles multiple concurrent users efficiently

### Option 2: Docker Deployment

#### Dockerfile

```dockerfile
FROM golang:1.21-alpine AS builder

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

#### docker-compose.yml

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

#### nginx.conf (SSL Termination)

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

#### Docker Deployment Steps

1. **Configure OAuth:**
   ```bash
   # Set production redirect URI in Google Console
   # https://your-domain.com/auth/callback
   ```

2. **Create environment file:**
   ```bash
   cat > .env << EOF
   GOOGLE_CLIENT_ID=your-oauth-client-id
   GOOGLE_CLIENT_SECRET=your-oauth-client-secret
   GOOGLE_REDIRECT_URL=https://your-domain.com/auth/callback
   GIN_MODE=release
   EOF
   ```

3. **Deploy with Docker Compose:**
   ```bash
   docker-compose up -d
   ```

### Option 3: Traditional VPS/Server

#### Prerequisites

```bash
# Install Go 1.21+
wget https://golang.org/dl/go1.21.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Install dependencies
sudo apt update
sudo apt install -y nginx certbot python3-certbot-nginx sqlite3
```

#### systemd Service

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

[Install]
WantedBy=multi-user.target
```

#### Server Deployment Steps

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

### Required Variables

- `GOOGLE_CLIENT_ID` - OAuth 2.0 client ID from Google Console
- `GOOGLE_CLIENT_SECRET` - OAuth 2.0 client secret from Google Console
- `GOOGLE_REDIRECT_URL` - OAuth callback URL (must match Google Console)

### Optional Variables

- `GOOGLE_CLOUD_PROJECT` - Use Google Cloud Datastore (for GAE)
- `GIN_MODE` - Set to "release" for production
- `PORT` - Server port (default: 8080)
- `SESSION_SECRET` - Custom session encryption key (auto-generated if not set)

## Database Configuration

### Multi-User Schema

The application now includes comprehensive multi-user support:

```sql
-- Users table
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    google_id TEXT UNIQUE NOT NULL,
    email TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    avatar TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- User-specific feed subscriptions
CREATE TABLE user_feeds (
    user_id INTEGER NOT NULL,
    feed_id INTEGER NOT NULL,
    PRIMARY KEY (user_id, feed_id),
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (feed_id) REFERENCES feeds (id) ON DELETE CASCADE
);

-- User-specific article status
CREATE TABLE user_articles (
    user_id INTEGER NOT NULL,
    article_id INTEGER NOT NULL,
    is_read BOOLEAN DEFAULT FALSE,
    is_starred BOOLEAN DEFAULT FALSE,
    PRIMARY KEY (user_id, article_id),
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (article_id) REFERENCES articles (id) ON DELETE CASCADE
);
```

### Database Backends

- **Local Development**: SQLite with multi-user schema
- **Google App Engine**: Cloud Datastore with user entity isolation
- **Docker/VPS**: SQLite with persistent volume mounting

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

```go
// Health check endpoint for monitoring
func healthCheck(c *gin.Context) {
    c.JSON(200, gin.H{
        "status": "healthy",
        "timestamp": time.Now(),
        "users": getUserCount(),
        "feeds": getFeedCount(),
    })
}
```

### Logging

```bash
# App Engine logs
gcloud app logs tail -s default

# Docker logs
docker-compose logs -f goread2

# System logs
sudo journalctl -u goread2 -f
```

### User Management

```bash
# View active users (via database query)
sqlite3 goread2.db "SELECT COUNT(*) FROM users;"

# View user sessions
sqlite3 goread2.db "SELECT user_id, created_at FROM sessions WHERE expires_at > datetime('now');"
```

## Performance Optimization

### Caching Strategies

1. **Session caching**: In-memory session store for frequently accessed sessions
2. **Feed caching**: Cache RSS feed responses to reduce external requests
3. **User data caching**: Cache user preferences and feed subscriptions
4. **Static asset caching**: Leverage CDN for static files

### Database Optimization

1. **Indexing**: Proper indexes on user_id and feed_id columns
2. **Connection pooling**: For high-traffic deployments
3. **Query optimization**: Efficient user-filtered queries
4. **Cleanup jobs**: Regular cleanup of expired sessions and old articles

### Scaling Considerations

1. **Horizontal scaling**: Multiple app instances behind load balancer
2. **Database scaling**: Consider PostgreSQL for very high user counts
3. **Feed fetching**: Queue-based system for many concurrent users
4. **Session storage**: Redis for distributed session management

## Testing in Production

### Deployment Testing

```bash
# Run comprehensive test suite
./test.sh

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

### Feed Fetching

1. **Feeds not updating for users:**
   - Check cron job/background task logs
   - Verify user-specific feed refresh logic
   - Monitor external RSS feed accessibility

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

## Migration Guide

### From Single-User to Multi-User

If upgrading from a single-user installation:

1. **Backup existing data:**
   ```bash
   cp goread2.db goread2_backup.db
   ```

2. **Run migration script:**
   ```bash
   # Create default user and migrate data
   go run scripts/migrate_to_multiuser.go
   ```

3. **Update configuration:**
   - Add OAuth environment variables
   - Update frontend to include authentication UI
   - Test multi-user functionality

## Support and Resources

- [Google OAuth 2.0 Documentation](https://developers.google.com/identity/protocols/oauth2)
- [App Engine Go Documentation](https://cloud.google.com/appengine/docs/standard/go)
- [Cloud Datastore Documentation](https://cloud.google.com/datastore/docs)
- [GoRead2 Testing Guide](./README_TESTING.md)
- [Multi-User Security Best Practices](https://owasp.org/www-community/Multi-User_Security)

## Example Deployment Commands

```bash
# Google App Engine deployment
gcloud app deploy app.yaml
gcloud app deploy cron.yaml

# Docker deployment
docker-compose up -d

# Traditional server deployment
sudo systemctl start goread2
sudo nginx -t && sudo systemctl reload nginx

# Health check
curl -f https://your-domain.com/api/health || exit 1
```

This comprehensive deployment guide covers all aspects of deploying the multi-user GoRead2 application with proper authentication, security, and scalability considerations.