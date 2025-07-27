# GoRead2 - Google App Engine Deployment Guide

This guide explains how to deploy GoRead2 RSS Reader to Google App Engine.

## Prerequisites

1. **Google Cloud Project**
   - Create a new Google Cloud Project or use existing one
   - Enable the following APIs:
     - App Engine Admin API
     - Cloud Datastore API
     - Cloud Build API (for deployment)

2. **Install Google Cloud SDK**
   ```bash
   # Download and install from: https://cloud.google.com/sdk/docs/install
   curl https://sdk.cloud.google.com | bash
   exec -l $SHELL
   gcloud init
   ```

3. **Authentication**
   ```bash
   gcloud auth login
   gcloud config set project YOUR_PROJECT_ID
   ```

## Project Structure for App Engine

The project has been modified for App Engine compatibility:

```
goread2/
├── app.yaml                     # App Engine configuration
├── cron.yaml                    # Cron jobs for feed refresh
├── main.go                      # Modified for GAE
├── go.mod                       # Dependencies
├── internal/
│   └── database/
│       ├── schema.go            # Database interface
│       └── datastore.go         # Google Datastore implementation
└── web/                         # Static files served by GAE
    ├── templates/
    └── static/
```

## Configuration Files

### app.yaml
The main App Engine configuration file:

```yaml
runtime: go121

env_variables:
  GIN_MODE: release

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
```

### cron.yaml
Scheduled tasks for automatic feed refresh:

```yaml
cron:
- description: "Refresh RSS feeds"
  url: /api/feeds/refresh
  schedule: every 30 minutes
  target: default
```

## Database Configuration

The application automatically detects the App Engine environment and switches to Google Cloud Datastore:

- **Local Development**: Uses SQLite database (`goread2.db`)
- **App Engine**: Uses Google Cloud Datastore
- **Detection**: Based on `GOOGLE_CLOUD_PROJECT` environment variable

### Datastore Entities

The following entities are created in Datastore:

1. **Feed**
   - Properties: title, url, description, created_at, updated_at, last_fetch
   - Kind: "Feed"

2. **Article**
   - Properties: feed_id, title, url, content, description, author, published_at, created_at, is_read, is_starred
   - Kind: "Article"

## Deployment Steps

### 1. Prepare the Application

Ensure all files are ready and dependencies are properly configured:

```bash
cd goread2
go mod tidy
```

### 2. Initialize App Engine

If this is your first App Engine app in the project:

```bash
gcloud app create --region=us-central1
```

### 3. Deploy the Application

```bash
# Deploy the main application
gcloud app deploy app.yaml

# Deploy cron jobs
gcloud app deploy cron.yaml
```

### 4. Open the Application

```bash
gcloud app browse
```

## Environment Variables

The application uses these environment variables in App Engine:

- `GAE_ENV=standard` - Indicates App Engine standard environment
- `GOOGLE_CLOUD_PROJECT` - Your Google Cloud project ID (automatically set)
- `PORT` - Port number (automatically set by App Engine)
- `GIN_MODE=release` - Sets Gin framework to release mode

## Local Development vs Production

### Local Development
```bash
# Uses SQLite database
go run .
# Access at http://localhost:8080
```

### App Engine Production
- Uses Google Cloud Datastore
- Static files served by App Engine infrastructure
- Automatic scaling and load balancing
- HTTPS enforced
- Cron jobs handle feed refresh

## Monitoring and Logs

### View Logs
```bash
gcloud app logs tail -s default
```

### Monitor Performance
- Go to Google Cloud Console → App Engine → Services
- View metrics, requests, and errors

### Datastore Management
- Go to Google Cloud Console → Datastore
- View entities, run queries, manage data

## Cost Considerations

App Engine pricing is based on:

1. **Instance Hours**
   - F1 instances (automatically chosen for this app)
   - Free tier: 28 hours per day

2. **Datastore Operations**
   - Read/Write operations
   - Storage costs
   - Free tier: 50,000 reads, 20,000 writes, 1GB storage per day

3. **Outbound Traffic**
   - RSS feed fetching
   - Free tier: 1GB per day

## Scaling Configuration

The app.yaml includes scaling settings:

- **Min instances**: 0 (scales to zero when not in use)
- **Max instances**: 10 (prevents runaway costs)
- **Target CPU**: 60% (scales up when CPU usage exceeds this)

## Security Features

- **HTTPS Only**: All traffic forced to HTTPS
- **Secure Cookies**: Session cookies marked secure
- **CSRF Protection**: Built into the application
- **Input Validation**: All user inputs validated

## Troubleshooting

### Common Issues

1. **Deployment Fails**
   ```bash
   # Check for syntax errors
   gcloud app validate app.yaml
   
   # View detailed deployment logs
   gcloud app logs tail -s default
   ```

2. **Database Connection Issues**
   ```bash
   # Verify Datastore API is enabled
   gcloud services enable datastore.googleapis.com
   ```

3. **Static Files Not Loading**
   - Ensure files are in `web/static/` directory
   - Check `app.yaml` static file handlers

4. **Cron Jobs Not Running**
   ```bash
   # Check cron status
   gcloud app describe
   
   # View cron logs
   gcloud app logs tail -s default --filter="cron"
   ```

### Debug Mode

For debugging in production:

```bash
# Set debug environment variable in app.yaml
env_variables:
  GIN_MODE: debug
  DEBUG: true
```

## Updates and Maintenance

### Deploying Updates
```bash
# Deploy new version
gcloud app deploy

# Route traffic to new version
gcloud app services set-traffic default --splits=v2=1
```

### Database Migration
- Datastore is schemaless, migrations are typically not needed
- For major changes, consider versioning your entity kinds

### Backup Strategy
- App Engine automatically backs up Datastore
- For additional backups, use Cloud Datastore export

## Performance Optimization

1. **Caching**
   - Consider adding memcache for frequently accessed data
   - Cache RSS feed responses

2. **Indexing**
   - Datastore automatically creates indexes
   - Monitor index usage in Cloud Console

3. **Static Assets**
   - App Engine automatically handles static file caching
   - Consider CDN for heavy traffic

## Support and Resources

- [App Engine Go Documentation](https://cloud.google.com/appengine/docs/standard/go)
- [Cloud Datastore Documentation](https://cloud.google.com/datastore/docs)
- [App Engine Pricing](https://cloud.google.com/appengine/pricing)
- [Google Cloud Status](https://status.cloud.google.com/)

## Example Commands Summary

```bash
# Initial setup
gcloud init
gcloud app create --region=us-central1

# Deploy application
gcloud app deploy app.yaml
gcloud app deploy cron.yaml

# Monitor
gcloud app logs tail -s default
gcloud app browse

# Update
gcloud app deploy
```

This deployment guide provides everything needed to successfully deploy GoRead2 to Google App Engine with automatic scaling, HTTPS, and background feed refresh capabilities.