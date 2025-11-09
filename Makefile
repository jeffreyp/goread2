.PHONY: all build lint test validate-config deploy-dev deploy-prod clean build-js build-css build-frontend deploy-monitoring deploy-monitoring-dashboard deploy-monitoring-alerts help

# Default target - build everything
all: build-frontend build test
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
	@echo "  test               Run all tests"
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
	go build -o goread2 .

# Run linter
lint:
	@echo "🔍 Running linter..."
	golangci-lint run

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

# Validate configuration
validate-config:
	@echo "🔍 Validating configuration..."
	go run cmd/validate-config/main.go

# Validate and build (recommended before deployment)
validate-build: validate-config build-frontend build
	@echo "✅ Validation and build completed"

# Deploy to development (with validation)
deploy-dev: validate-config build-frontend
	@echo "🚀 Deploying to development..."
	gcloud app deploy app.yaml --version="dev-$$(date +%Y%m%dt%H%M%S)" --no-promote --quiet

# Validate configuration in strict mode (for production)
validate-config-strict:
	@echo "🔍 Validating configuration (strict mode)..."
	VALIDATE_STRICT=true go run cmd/validate-config/main.go

# Deploy to production (with strict validation and tests)
deploy-prod: validate-config-strict test build-frontend
	@echo "🚀 Deploying to production..."
	gcloud app deploy app.yaml --version="prod-$$(date +%Y%m%dt%H%M%S)" --quiet
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