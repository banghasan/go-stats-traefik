#!/bin/bash

# Build script for go-stats-traefik with version support
# Usage: ./build.sh [version]
# Example: ./build.sh 1.1.0

# Set default version from VERSION file if not provided
VERSION=${1:-$(cat VERSION 2>/dev/null || echo "1.0.0")}

# Get git commit hash if available
if git rev-parse --git-dir > /dev/null 2>&1; then
    COMMIT_HASH=$(git rev-parse --short HEAD)
    BUILD_INFO="${VERSION}-${COMMIT_HASH}"
else
    BUILD_INFO="${VERSION}-local"
    COMMIT_HASH=""
fi

echo "Building go-stats-traefik binary version ${VERSION}..."
echo "Build info: ${BUILD_INFO}"

# Create bin directory if it doesn't exist
mkdir -p bin

# Build the binary with version and build information
if [ -n "$COMMIT_HASH" ]; then
    go build -ldflags="-X 'main.AppVersion=${VERSION}' -X 'main.BuildInfo=${BUILD_INFO}' -w -s" -o bin/go-stats-traefik main.go
else
    go build -ldflags="-X 'main.AppVersion=${VERSION}' -X 'main.BuildInfo=${BUILD_INFO}' -w -s" -o bin/go-stats-traefik main.go
fi

if [ $? -eq 0 ]; then
    echo "Build successful!"
    echo "Binary location: $(pwd)/bin/go-stats-traefik"
    echo "Version: ${VERSION}"
    echo "Build Info: ${BUILD_INFO}"
    ls -la bin/go-stats-traefik
else
    echo "Build failed!"
    exit 1
fi
