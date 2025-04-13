.PHONY: build clean test run generate-urls schedule serve test-api help

# Application name
APP_NAME := s3perftest
BUILD_DIR := build

# Default build target
all: build

# Build the application
build:
	@echo "Building ${APP_NAME}..."
	@mkdir -p ${BUILD_DIR}
	@go build -o ${BUILD_DIR}/${APP_NAME} ./cmd/s3perftest

# Install the application to $GOPATH/bin
install: build
	@echo "Installing ${APP_NAME}..."
	@cp ${BUILD_DIR}/${APP_NAME} ${GOPATH}/bin/

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf ${BUILD_DIR}
	@go clean

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run the application
run: build
	@echo "Running ${APP_NAME}..."
	@${BUILD_DIR}/${APP_NAME} run

# Generate presigned URLs
generate-urls: build
	@echo "Generating presigned URLs..."
	@${BUILD_DIR}/${APP_NAME} generate-urls

# Schedule tests
schedule: build
	@echo "Starting scheduled tests..."
	@${BUILD_DIR}/${APP_NAME} schedule

# Start the API server
serve: build
	@echo "Starting API server..."
	@${BUILD_DIR}/${APP_NAME} serve

# Run tests against the API server
test-api: build
	@echo "Running tests against the API server..."
	@${BUILD_DIR}/${APP_NAME} test-api

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

# Show help
help:
	@echo "Available targets:"
	@echo "  build         - Build the application"
	@echo "  install       - Install the application to GOPATH/bin"
	@echo "  clean         - Clean build artifacts"
	@echo "  test          - Run tests"
	@echo "  run           - Run the application (equivalent to '${APP_NAME} run')"
	@echo "  generate-urls - Generate presigned URLs (equivalent to '${APP_NAME} generate-urls')"
	@echo "  schedule      - Schedule tests (equivalent to '${APP_NAME} schedule')"
	@echo "  serve         - Start the API server (equivalent to '${APP_NAME} serve')"
	@echo "  test-api      - Run tests against the API server (equivalent to '${APP_NAME} test-api')"
	@echo "  fmt           - Format code"
	@echo "  lint          - Lint code"
	@echo "  help          - Show this help message"

# Default target
.DEFAULT_GOAL := help 