#!/bin/bash
# Linting script for API Stats project

echo "Running golangci-lint..."
if ! command -v golangci-lint &> /dev/null; then
    echo "golangci-lint not found. Installing..."
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
fi

golangci-lint run ./...
