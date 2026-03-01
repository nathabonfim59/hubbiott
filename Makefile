.PHONY: all build clean test lint fmt vet run dev air release snapshot install

BINARY_NAME=hubbiott
BUILD_DIR=bin
MAIN_PACKAGE=./cmd/server

# Default target
all: build

# Build binary for current platform
build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)

# Run the server
run:
	go run $(MAIN_PACKAGE)

# Development with hot reload (requires air)
dev:
	air

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -cover ./...

# Format code
fmt:
	go fmt ./...

# Run go vet
vet:
	go vet ./...

# Run linter (requires golangci-lint)
lint: fmt vet
	golangci-lint run

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)/ dist/

# Install dependencies
deps:
	go mod tidy
	go mod download

# Build for all platforms using goreleaser (local snapshot)
snapshot:
	goreleaser release --snapshot --clean

# Build for all platforms using goreleaser (release)
release:
	goreleaser release --clean

# Build nfpm packages only (deb, rpm)
packages:
	goreleaser release --snapshot --clean --id packages

# Install locally
install:
	go install $(MAIN_PACKAGE)

# Generate code (if needed)
generate:
	go generate ./...

# Check for vulnerabilities (requires govulncheck)
vulncheck:
	govulncheck ./...

# Full CI check
ci: deps fmt vet lint test

# Help
help:
	@echo "Available targets:"
	@echo "  build         - Build binary for current platform"
	@echo "  run           - Run the server"
	@echo "  dev           - Run with hot reload (requires air)"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  fmt           - Format code"
	@echo "  vet           - Run go vet"
	@echo "  lint          - Run linter"
	@echo "  clean         - Clean build artifacts"
	@echo "  deps          - Install dependencies"
	@echo "  snapshot      - Build all platforms (local snapshot)"
	@echo "  release       - Build and release (requires GITHUB_TOKEN)"
	@echo "  packages      - Build deb/rpm packages only"
	@echo "  install       - Install binary locally"
	@echo "  generate      - Run go generate"
	@echo "  vulncheck     - Check for vulnerabilities"
	@echo "  ci            - Full CI check (deps, fmt, vet, lint, test)"
