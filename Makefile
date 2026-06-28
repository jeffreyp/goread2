.PHONY: all build lint test test-quick test-race validate-config deploy-dev deploy-prod clean build-js build-css build-frontend deploy-monitoring deploy-monitoring-dashboard deploy-monitoring-alerts help

# Default target - build everything
all: build-frontend build test-quick
	@echo "✅ Complete build finished successfully!"

# Default target when just typing 'make'
.DEFAULT_GOAL := all

# Show help information
help:
	@echo "🛠️  GoRead2 Build System"
	@echo ""
	@echo "Available targets:"
	@echo "  all                Build frontend, service, and run tests (default)"
	@echo "  build              Build the Go application binary"
	@echo "  lint               Run golangci-lint to check code quality"
	@echo "  build-js           Build minified JavaScript files"
	@echo "  build-css          Build minified CSS files"
	@echo "  build-frontend     Build all frontend assets (JS + CSS)"
	@echo "  test               Run all tests with coverage (CI/pre-deploy)"
	@echo "  test-quick         Run tests using Go cache — fast for dev iteration"
	@echo "  test-race          Run Go tests with race detector — use before merging concurrent code changes"
	@echo "  validate-config    Validate application configuration"
	@echo "  validate-build     Validate config + build frontend + build app"
	@echo "  deploy-dev         Deploy to development environment"
	@echo "  deploy-prod        Deploy to production environment"
	@echo "  deploy-monitoring  Deploy monitoring dashboard and alerts"
	@echo "  clean              Remove all build artifacts"
	@echo "  dev                Start development server"
	@echo "  help               Show this help message"
	@echo ""
	@echo "Frontend build requirements:"
	@echo "  - Node.js and npm must be installed"
	@echo "  - Dependencies are installed automatically via 'npm install'"

# Build the application
build:
	@echo "🔨 Building GoRead2..."
	go build -ldflags "-X main.version=$(shell date +%Y.%m.%d)" -o goread2 .

# Run linter
lint:
	@echo "🔍 Running linter..."
	golangci-lint run --timeout=3m

# Install npm dependencies if needed
node_modules: package.json package-lock.json
	@echo "📦 Installing npm dependencies..."
	@command -v npm >/dev/null 2>&1 || (echo "❌ npm not found. Please install Node.js" && exit 1)
	npm install
	@touch node_modules

# Build minified JavaScript files
build-js: node_modules
	@echo "📦 Building minified JavaScript..."
	@command -v npm >/dev/null 2>&1 || (echo "❌ npm not found. Please install Node.js" && exit 1)
	npx terser web/static/js/app.js -o web/static/js/app.min.js --compress --mangle
	npx terser web/static/js/modals.js -o web/static/js/modals.min.js --compress --mangle
	npx terser web/static/js/account.js -o web/static/js/account.min.js --compress --mangle
	npx terser web/static/js/animations.js -o web/static/js/animations.min.js --compress --mangle
	npx terser web/static/js/animations-integration.js -o web/static/js/animations-integration.min.js --compress --mangle
	@echo "✅ JavaScript minification completed"

# Build minified CSS files
build-css: node_modules
	@echo "🎨 Building minified CSS..."
	@command -v npm >/dev/null 2>&1 || (echo "❌ npm not found. Please install Node.js" && exit 1)
	npx csso web/static/css/styles.css --output web/static/css/styles.min.css
	@echo "✅ CSS minification completed"

# Build all frontend assets
build-frontend: build-js build-css
	@echo "✅ Frontend build completed"

# Run tests
test:
	@echo "🧪 Running tests..."
	./test.sh

# Run tests quickly using Go's test cache (no coverage output)
# Uses same env vars as test.sh; sub-second when nothing has changed.
test-quick:
	@echo "⚡ Running tests (cached)..."
	@GOOGLE_CLOUD_PROJECT="" \
	GOOGLE_CLIENT_ID="test_client_id" \
	GOOGLE_CLIENT_SECRET="test_client_secret" \
	GOOGLE_REDIRECT_URL="http://localhost:8080/auth/callback" \
	go test ./...

# Run tests with the Go race detector enabled.
# Slower than test-quick (~2x), but catches data races in concurrent code.
# CI already runs with -race; use this locally before merging changes to
# FeedScheduler, DomainRateLimiter, RequestCache, or any other shared state.
test-race:
	@echo "🏁 Running tests with race detector..."
	@GOOGLE_CLOUD_PROJECT="" \
	GOOGLE_CLIENT_ID="test_client_id" \
	GOOGLE_CLIENT_SECRET="test_client_secret" \
	GOOGLE_REDIRECT_URL="http://localhost:8080/auth/callback" \
	go test -race ./...

# Validate configuration
validate-config:
	@echo "🔍 Validating configuration..."
	go run cmd/validate-config/main.go

# Validate and build (recommended before deployment)
validate-build: validate-config build-frontend build
	@echo "✅ Validation and build completed"

# Substitute secrets from Secret Manager into app.yaml
substitute-secrets:
	@echo "🔐 Fetching secrets from Secret Manager..."
	@CSRF_SECRET=$$(gcloud secrets versions access latest --secret="csrf-secret") \
	ADMIN_TOKEN=$$(gcloud secrets versions access latest --secret="admin-token") \
	INITIAL_ADMIN_EMAILS=$$(gcloud secrets versions access latest --secret="initial-admin-emails") \
	STRIPE_SECRET_KEY=$$(gcloud secrets versions access latest --secret="stripe-secret-key") \
	STRIPE_PUBLISHABLE_KEY=$$(gcloud secrets versions access latest --secret="stripe-publishable-key") \
	STRIPE_WEBHOOK_SECRET=$$(gcloud secrets versions access latest --secret="stripe-webhook-secret") \
	STRIPE_PRICE_ID=$$(gcloud secrets versions access latest --secret="stripe-price-id") \
	envsubst < app.yaml > app-deploy.yaml
	@echo "✓ Secrets substituted into app-deploy.yaml"

# Deploy to development (with validation)
deploy-dev: validate-config build-frontend substitute-secrets
	@echo "🚀 Deploying to development..."
	@echo "🧹 Stopping beads daemon and removing socket file..."
	@-bd daemon --stop 2>/dev/null || true
	@-rm -f .beads/bd.sock 2>/dev/null || true
	@sleep 1
	@gcloud app deploy app-deploy.yaml --version="dev-$$(date +%Y%m%dt%H%M%S)" --no-promote --quiet; \
	EXIT_CODE=$$?; \
	bd daemon --start 2>/dev/null || true; \
	rm -f app-deploy.yaml; \
	exit $$EXIT_CODE

# Validate configuration in strict mode (for production)
validate-config-strict:
	@echo "🔍 Validating configuration (strict mode)..."
	VALIDATE_STRICT=true go run cmd/validate-config/main.go

# Deploy to production (with strict validation and tests)
deploy-prod: validate-config-strict test build-frontend substitute-secrets
	@echo "🚀 Deploying to production..."
	@echo "🧹 Stopping beads daemon and removing socket file..."
	@-bd daemon --stop 2>/dev/null || true
	@-rm -f .beads/bd.sock 2>/dev/null || true
	@sleep 1
	@gcloud app deploy app-deploy.yaml --version="prod-$$(date +%Y%m%dt%H%M%S)" --quiet; \
	EXIT_CODE=$$?; \
	bd daemon --start 2>/dev/null || true; \
	rm -f app-deploy.yaml; \
	exit $$EXIT_CODE
	@echo "🧹 Cleaning up old versions..."
	@gcloud app versions list --sort-by=LAST_DEPLOYED --format="value(id)" --filter="TRAFFIC_SPLIT=0" | head -1 | xargs -r gcloud app versions delete --quiet

# Clean build artifacts
clean:
	@echo "🧹 Cleaning up..."
	rm -f goread2
	rm -f web/static/js/*.min.js
	rm -f web/static/css/*.min.css

# Development server with validation
dev: validate-config
	@echo "🔧 Starting development server..."
	go run main.go

# Deploy Cloud Monitoring dashboard
deploy-monitoring-dashboard:
	@echo "📊 Deploying monitoring dashboard..."
	./monitoring/deploy-dashboard.sh

# Deploy Cloud Monitoring alerting policies
deploy-monitoring-alerts:
	@echo "🚨 Deploying alerting policies..."
	./monitoring/deploy-alerts.sh

# Deploy all monitoring (dashboard + alerts)
deploy-monitoring: deploy-monitoring-dashboard deploy-monitoring-alerts
	@echo "✅ All monitoring resources deployed"