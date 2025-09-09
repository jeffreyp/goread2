.PHONY: build test validate-config deploy-dev deploy-prod clean

# Build the application
build:
	@echo "🔨 Building GoRead2..."
	go build -o goread2 .

# Run tests
test:
	@echo "🧪 Running tests..."
	./test.sh

# Validate configuration
validate-config:
	@echo "🔍 Validating configuration..."
	cd scripts && go run validate-config.go

# Validate and build (recommended before deployment)
validate-build: validate-config build
	@echo "✅ Validation and build completed"

# Deploy to development (with validation)
deploy-dev: validate-config
	@echo "🚀 Deploying to development..."
	gcloud app deploy app.yaml --version="dev-$$(date +%Y%m%dt%H%M%S)" --no-promote --quiet

# Validate configuration in strict mode (for production)
validate-config-strict:
	@echo "🔍 Validating configuration (strict mode)..."
	cd scripts && VALIDATE_STRICT=true go run validate-config.go

# Deploy to production (with strict validation and tests)
deploy-prod: validate-config-strict test
	@echo "🚀 Deploying to production..."
	gcloud app deploy app.yaml --version="prod-$$(date +%Y%m%dt%H%M%S)" --quiet

# Clean build artifacts
clean:
	@echo "🧹 Cleaning up..."
	rm -f goread2

# Development server with validation
dev: validate-config
	@echo "🔧 Starting development server..."
	go run main.go