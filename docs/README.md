# GoRead2 Documentation

The guides in this directory, grouped by who they are written for. Each guide describes the product as it currently is.

## Using GoRead2

| Guide | Covers |
|-------|--------|
| [Features](features.md) | The web interface, keyboard shortcuts, and account settings |

## Building on the API

| Guide | Covers |
|-------|--------|
| [API Reference](api.md) | Every endpoint, with authentication, CSRF, rate limits, and errors |

## Operating GoRead2

| Guide | Covers |
|-------|--------|
| [Deployment](deployment.md) | App Engine, the staging and production pipeline, rollback, Secret Manager, and Cloud Tasks |
| [Configuration Reference](configuration.md) | Every environment variable, with its default and Secret Manager name |
| [Monitoring](monitoring.md) | Dashboards, alert policies, usage metrics, and the budget |
| [Admin Guide](admin.md) | Admin access, admin tokens, the admin CLI and API, the audit log, and usage statistics |
| [Stripe Setup](stripe.md) | The subscription flag, Stripe keys, products, webhooks, and the customer portal |
| [Apple Release Guide](apple-releases.md) | Releasing the native app through TestFlight and as a notarized macOS DMG |
| [Troubleshooting](troubleshooting.md) | Errors, their causes, and fixes |

## Contributing

| Guide | Covers |
|-------|--------|
| [Contributing](../CONTRIBUTING.md) | The fork, branch, and pull request workflow |
| [Setup](setup.md) | Running GoRead2 locally |
| [Architecture](architecture.md) | How the components fit together, and the data model |
| [Testing](testing.md) | Running and writing tests, the Datastore emulator, and CI |
| [Apple Apps](apple-apps.md) | Developing and testing the iOS, iPadOS, and macOS app |

## Design Reference

| Guide | Covers |
|-------|--------|
| [Authentication](authentication.md) | Google OAuth, sessions, cookies, and the native sign-in handoff |
| [Security](security.md) | Security controls and where each is implemented |
| [Privacy and Personal Data](privacy.md) | The personal data stored, cookies, export, deletion, and retention |
| [Performance and Caching](performance.md) | Feed refresh, query patterns, and HTTP and in-memory caching |

## Conventions

Each guide opens with a one-paragraph summary and a table of contents, and ends with Related Documentation. A guide has a Troubleshooting section only when it has troubleshooting content of its own, or when [troubleshooting.md](troubleshooting.md) has a section on its topic. `CLAUDE.md`, at the repository root, gives the writing rules and maps each area of the code to the guide that describes it.
