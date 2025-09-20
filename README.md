# GoRead2 - Multi-User RSS Reader

[![Tests](https://github.com/jeffreyp/goread2/actions/workflows/test.yml/badge.svg)](https://github.com/jeffreyp/goread2/actions/workflows/test.yml)

A modern, multi-user RSS reader inspired by Google Reader, built with Go and featuring Google OAuth authentication, a clean three-pane interface, and comprehensive user data isolation.

## ✨ Features

- **Multi-user support** with Google OAuth authentication
- **Three-pane layout** (feeds → articles → content) like Google Reader
- **RSS/Atom feed support** with OPML import capability
- **Real-time updates** with background polling every 30 minutes
- **Per-user article management** (read/unread, starred status)
- **Subscription system** with 30-day free trial and Stripe integration
- **Keyboard shortcuts** for efficient navigation
- **Reading optimization** with sans-serif/serif font choice
- **Self-hosted** - runs locally or deploys to cloud platforms

## 🚀 Quick Start

### Prerequisites
- Go 1.23+
- Google Cloud Project (for OAuth)
- Stripe Account (optional, for subscriptions)

### 1. Clone and Setup
```bash
git clone https://github.com/jeffreyp/goread2.git
cd goread2
go mod tidy
```

### 2. Google OAuth Setup
1. Create a Google Cloud Project at [console.cloud.google.com](https://console.cloud.google.com/)
2. Enable APIs & Services → Credentials → Create OAuth 2.0 Client ID
3. Set authorized redirect URI: `http://localhost:8080/auth/callback`
4. Export your credentials:
```bash
export GOOGLE_CLIENT_ID="your-client-id"
export GOOGLE_CLIENT_SECRET="your-client-secret"
export GOOGLE_REDIRECT_URL="http://localhost:8080/auth/callback"
```

### 3. Build Frontend Assets (Optional)
For production deployments, build minified frontend assets:
```bash
# Install npm dependencies
npm install

# Build all frontend assets (JS + CSS)
make build-frontend

# Or build individually
make build-js    # Build minified JavaScript
make build-css   # Build minified CSS
```

### 4. Run the Application
```bash
go run main.go
# Or use make for development with config validation
make dev
```

Access at [http://localhost:8080](http://localhost:8080) and sign in with Google!

## 🔧 Build System

The project includes a comprehensive Makefile with the following targets:

```bash
make help              # Show all available commands
make build             # Build the Go application binary
make build-frontend    # Build minified JS and CSS assets
make test              # Run all tests
make validate-build    # Validate config + build frontend + build app
make deploy-dev        # Deploy to development environment
make deploy-prod       # Deploy to production environment
make clean             # Remove all build artifacts
```

## 📚 Documentation

| Guide | Purpose |
|-------|---------|
| [**Features Guide**](docs/features.md) | Complete feature overview and usage tips |
| [**Setup Guide**](docs/setup.md) | Complete installation and configuration |
| [**Deployment Guide**](docs/deployment.md) | Production deployment options (includes Google Secret Manager setup) |
| [**Admin Guide**](docs/admin.md) | User management and admin commands |
| [**Stripe Setup**](docs/stripe.md) | Payment processing configuration |
| [**Testing Guide**](docs/testing.md) | Running and writing tests |
| [**API Reference**](docs/api.md) | API endpoints and usage |
| [**Feature Flags**](docs/feature-flags.md) | Configuration and feature toggles |
| [**Troubleshooting**](docs/troubleshooting.md) | Common issues and solutions |
| [**Contributing**](docs/contributing.md) | Development and contribution guide |

## 🏗️ Architecture

- **Authentication**: Google OAuth 2.0 with secure session management
- **Database**: SQLite (local) or Google Cloud Datastore (production)
- **Frontend**: Vanilla JavaScript with clean three-pane interface
- **Backend**: Go with Gin framework
- **Payments**: Stripe integration for subscriptions

## 🖼️ Interface

The interface features three main sections:
- **Left pane**: Personal feed subscriptions with unread counts
- **Center pane**: Article list with personal read/unread status  
- **Right pane**: Full article content with original formatting

## ⌨️ Keyboard Shortcuts

- `j/k` - Navigate articles up/down
- `o/Enter` - Open article in new tab
- `m` - Toggle read/unread status
- `s` - Star/unstar article
- `r` - Refresh all feeds
- `f` - Toggle font style (sans-serif ↔ serif)

## 🔒 Security & Privacy

- **Complete user data isolation** - users only see their own feeds and article status
- **Secure authentication** via Google OAuth (no password storage)
- **Session security** with HTTP-only cookies and CSRF protection
- **Input validation** and XSS protection throughout

## 🧪 Testing

Run the comprehensive test suite:
```bash
./test.sh  # Runs both backend and frontend tests
```

- **Comprehensive test coverage** with significant improvements in critical areas
- **Admin security testing** for the new database-based token authentication system
- **Multi-user isolation testing** to ensure data security
- **Integration tests** for API endpoints, authentication, and admin functionality
- **Dual database support testing** (SQLite + Google Datastore) 
- **Frontend tests** with Jest and jsdom

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure `./test.sh` passes
5. Submit a pull request

See [docs/contributing.md](docs/contributing.md) for detailed guidelines.

## 📄 License

This project is licensed under the [MIT License](LICENSE).

---

**Need help?** Check the [Setup Guide](docs/setup.md) for detailed instructions or the [Troubleshooting section](docs/troubleshooting.md) for common issues.