# Makefile for go-stats-traefik

# Get version from VERSION file, default to 1.0.0 if not found
VERSION := $(shell cat VERSION 2>/dev/null || echo "1.0.0")
BINARY_NAME := go-stats-traefik
BUILD_DIR := bin

# Go related variables
GO := go
GOFMT := gofmt

.PHONY: all build clean test help version bump-major bump-minor bump-patch

all: build

# Build the binary with version embedded
build:
	@echo "Building ${BINARY_NAME} version ${VERSION}..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build -ldflags="-X 'main.AppVersion=${VERSION}' -w -s" -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Build completed successfully!"

# Build with debug information
build-debug:
	@echo "Building ${BINARY_NAME} with debug info..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build -ldflags="-X 'main.AppVersion=${VERSION}'" -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Build with debug completed successfully!"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	@echo "Clean completed!"

# Run tests if available
test:
	@echo "Running tests..."
	$(GO) test ./...
	@echo "Tests completed!"

# Display current version
version:
	@echo "Current version: $(VERSION)"

# Bump version - Major
bump-major:
	@$(shell awk -F. '{print $$1+1"."0"."0}' VERSION > VERSION.tmp && mv VERSION.tmp VERSION)
	@echo "Version bumped to major: $(shell cat VERSION)"

# Bump version - Minor
bump-minor:
	@$(shell awk -F. '{print $$1"."$$2+1"."0}' VERSION > VERSION.tmp && mv VERSION.tmp VERSION)
	@echo "Version bumped to minor: $(shell cat VERSION)"

# Bump version - Patch
bump-patch:
	@$(shell awk -F. '{print $$1"."$$2"."$$3+1}' VERSION > VERSION.tmp && mv VERSION.tmp VERSION)
	@echo "Version bumped to patch: $(shell cat VERSION)"

# Help message
help:
	@echo "Go Stats Traefik Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make               - Build the application"
	@echo "  make build         - Build the application"
	@echo "  make build-debug   - Build with debug information"
	@echo "  make clean         - Remove build artifacts"
	@echo "  make test          - Run tests"
	@echo "  make version       - Show current version"
	@echo "  make bump-major    - Bump major version"
	@echo "  make bump-minor    - Bump minor version"
	@echo "  make bump-patch    - Bump patch version"
	@echo "  make help          - Show this help message"
