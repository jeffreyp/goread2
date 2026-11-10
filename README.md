# GoRead2 - Multi-User RSS Reader

[![Tests](https://github.com/jeffreyp/goread2/actions/workflows/test.yml/badge.svg)](https://github.com/jeffreyp/goread2/actions/workflows/test.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/jeffreyp/goread2)](https://go.dev/)
[![License](https://img.shields.io/github/license/jeffreyp/goread2)](LICENSE)

A modern, multi-user RSS reader inspired by Google Reader and [GoRead](https://github.com/madelynnblue/goread).

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

## Build

```bash
make help   # Show all available commands
make build  # Build the Go application binary
make test   # Run all tests
make lint   # Run golangci-lint code quality checks
```

Deployment is automated via GitHub Actions, not the Makefile. See
[Deployment Guide](docs/deployment.md).

## Documentation

The [documentation index](docs/README.md) lists every guide, grouped for users, API clients, operators, and contributors. Start with the [Setup Guide](docs/setup.md) to run GoRead2 locally, or the [Architecture](docs/architecture.md) guide for an overview of how it works.

## Contributing

See [CONTRIBUTING](CONTRIBUTING.md) for the fork, branch, and pull request workflow.

## License

This project is licensed under the [MIT License](LICENSE).

---

**Need help?** Check the [Setup Guide](docs/setup.md) for detailed instructions or the [Troubleshooting section](docs/troubleshooting.md) for common issues.
