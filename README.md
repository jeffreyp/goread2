# GoRead2 - Multi-User RSS Reader

[![Tests](https://github.com/jeffreyp/goread2/actions/workflows/test.yml/badge.svg)](https://github.com/jeffreyp/goread2/actions/workflows/test.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/jeffreyp/goread2)](https://go.dev/)
[![License](https://img.shields.io/github/license/jeffreyp/goread2)](LICENSE)

A modern, multi-user RSS reader inspired by Google Reader and perhaps equally if not more so by [GoRead](https://github.com/madelynnblue/goread).

## Features

- Multi-user support with Google OAuth authentication
- Three-pane layout (feeds → articles → content) like Google Reader
- RSS/Atom feed support with OPML import and export
- Keyboard shortcuts for efficient navigation
- Subscription system with a 30-day free trial and Stripe integration

See the [Features Guide](docs/features.md) for the complete list and usage tips.

## Quick Start

### Prerequisites
- Go 1.25+
- Google Cloud Project (for OAuth)
- Node.js 16+ (required to build minified frontend assets; these are not checked into the repo)
- Stripe Account (optional, for subscriptions)

### Setup
```bash
git clone https://github.com/jeffreyp/goread2.git
cd goread2
go mod tidy
npm install
make build-frontend  # builds minified JS/CSS; the app won't render without this

export GOOGLE_CLIENT_ID="your-client-id"
export GOOGLE_CLIENT_SECRET="your-client-secret"
export GOOGLE_REDIRECT_URL="http://localhost:8080/auth/callback"

make dev  # validates config, then starts the dev server
```

Access at [http://localhost:8080](http://localhost:8080) and sign in with Google.

See the [Setup Guide](docs/setup.md) for Google OAuth configuration, frontend asset builds, and full environment variable reference.

## Build System

```bash
make help   # Show all available commands
make build  # Build the Go application binary
make test   # Run all tests
make lint   # Run golangci-lint code quality checks
```

Deployment is automated via GitHub Actions, not the Makefile. See [docs/deployment.md](docs/deployment.md).

## Documentation

| Guide | Purpose |
|-------|---------|
| [**Features Guide**](docs/features.md) | Complete feature overview and usage tips |
| [**Setup Guide**](docs/setup.md) | Complete installation and configuration |
| [**Authentication Guide**](docs/authentication.md) | OAuth flow, session management, and security |
| [**Deployment Guide**](docs/deployment.md) | Google App Engine deployment and CI/CD pipeline (includes Google Secret Manager setup) |
| [**Admin Guide**](docs/admin.md) | User management and admin commands |
| [**Stripe Setup**](docs/stripe.md) | Payment processing configuration |
| [**Testing Guide**](docs/testing.md) | Running and writing tests |
| [**API Reference**](docs/api.md) | API endpoints and usage |
| [**Feature Flags**](docs/feature-flags.md) | Configuration and feature toggles |
| [**Security Guide**](docs/security.md) | Security features and best practices |
| [**Performance & Cost**](docs/performance.md) | Optimization strategies and cost savings |
| [**Monitoring**](docs/monitoring.md) | Cloud Monitoring dashboards and cost tracking |
| [**Caching Strategy**](docs/caching.md) | HTTP and application-level caching |
| [**Troubleshooting**](docs/troubleshooting.md) | Common issues and solutions |
| [**Contributing**](CONTRIBUTING.md) | Development and contribution guide |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for the fork, branch, and pull request workflow.

## License

This project is licensed under the [MIT License](LICENSE).

---

**Need help?** Check the [Setup Guide](docs/setup.md) for detailed instructions or the [Troubleshooting section](docs/troubleshooting.md) for common issues.
