#!/bin/bash

# Build script for go-stats-traefik

echo "Building go-stats-traefik binary..."

# Create bin directory if it doesn't exist
mkdir -p bin

# Build the binary
go build -o bin/go-stats-traefik main.go

if [ $? -eq 0 ]; then
    echo "Build successful!"
    echo "Binary location: $(pwd)/bin/go-stats-traefik"
    ls -la bin/go-stats-traefik
else
    echo "Build failed!"
    exit 1
fi
