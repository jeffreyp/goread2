#!/bin/bash

# GoRead2 Test Runner
# This script runs the complete test suite with coverage reporting

set -e

echo "🧪 Running GoRead2 Test Suite"
echo "================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

# Set test environment variables
export GOOGLE_CLIENT_ID="test_client_id"
export GOOGLE_CLIENT_SECRET="test_client_secret"
export GOOGLE_REDIRECT_URL="http://localhost:8080/auth/callback"

print_status "Environment variables set for testing"

# Run unit tests
echo ""
echo "📋 Running Unit Tests..."
echo "------------------------"
if go test ./test/unit/... -v -coverprofile=unit_coverage.out; then
    print_status "Unit tests passed"
else
    print_error "Unit tests failed"
    exit 1
fi

# Run integration tests
echo ""
echo "🔗 Running Integration Tests..."
echo "-------------------------------"
if go test ./test/integration/... -v -coverprofile=integration_coverage.out; then
    print_status "Integration tests passed"
else
    print_error "Integration tests failed"
    exit 1
fi

# Combine coverage files
echo ""
echo "📊 Generating Coverage Report..."
echo "--------------------------------"
if command -v gocovmerge &> /dev/null; then
    gocovmerge unit_coverage.out integration_coverage.out > coverage.out
    print_status "Coverage files merged"
else
    print_warning "gocovmerge not found, using unit test coverage only"
    cp unit_coverage.out coverage.out
fi

# Generate HTML coverage report
if go tool cover -html=coverage.out -o coverage.html; then
    print_status "Coverage report generated: coverage.html"
fi

# Show coverage summary
if go tool cover -func=coverage.out | tail -1; then
    print_status "Coverage summary displayed"
fi

# Run linting if available
echo ""
echo "🔍 Running Code Quality Checks..."
echo "---------------------------------"
if command -v golangci-lint &> /dev/null; then
    if golangci-lint run; then
        print_status "Linting passed"
    else
        print_warning "Linting issues found"
    fi
else
    print_warning "golangci-lint not found, running go vet instead"
    if go vet ./...; then
        print_status "go vet passed"
    else
        print_error "go vet failed"
    fi
fi

# Build test
echo ""
echo "🏗️  Testing Build..."
echo "-------------------"
if go build .; then
    print_status "Build successful"
else
    print_error "Build failed"
    exit 1
fi

# Clean up
rm -f unit_coverage.out integration_coverage.out

echo ""
echo "🎉 All tests completed successfully!"
echo "Coverage report: coverage.html"
echo "================================"