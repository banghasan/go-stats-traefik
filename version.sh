#!/bin/bash

# Version management script for go-stats-traefik

VERSION_FILE="VERSION"
DEFAULT_VERSION="1.0.0"

# Function to get current version
get_version() {
    if [ -f "$VERSION_FILE" ]; then
        cat "$VERSION_FILE"
    else
        echo "$DEFAULT_VERSION"
    fi
}

# Function to set version
set_version() {
    local new_version=$1
    if [ -z "$new_version" ]; then
        echo "Error: Version not provided"
        echo "Usage: $0 set <version>"
        exit 1
    fi

    echo "$new_version" > "$VERSION_FILE"
    echo "Version updated to: $new_version"

    # Update README.md if it contains version
    if [ -f "README.md" ]; then
        sed -i "s/Version [0-9]\+\.[0-9]\+\(\.[0-9]\+\)\{0,1\}/Version $new_version/g" README.md
        sed -i "s/\"version\": \"[0-9]\+\.[0-9]\+\(\.[0-9]\+\)\{0,1\}\"/\"version\": \"$new_version\"/g" README.md
    fi

    # Update API.md if it contains version
    if [ -f "API.md" ]; then
        sed -i "s/\"version\": \"[0-9]\+\.[0-9]\+\(\.[0-9]\+\)\{0,1\}\"/\"version\": \"$new_version\"/g" API.md
    fi
}

# Function to bump version
bump_version() {
    local bump_type=$1
    local current_version=$(get_version)

    if [ -z "$bump_type" ]; then
        echo "Error: Bump type not provided (major, minor, patch)"
        echo "Usage: $0 bump <major|minor|patch>"
        exit 1
    fi

    IFS='.' read -ra VERSION_PARTS <<< "$current_version"
    local major=${VERSION_PARTS[0]}
    local minor=${VERSION_PARTS[1]}
    local patch=${VERSION_PARTS[2]:-0}

    case $bump_type in
        major)
            major=$((major + 1))
            minor=0
            patch=0
            ;;
        minor)
            minor=$((minor + 1))
            patch=0
            ;;
        patch)
            patch=$((patch + 1))
            ;;
        *)
            echo "Error: Invalid bump type. Use major, minor, or patch"
            exit 1
            ;;
    esac

    local new_version="$major.$minor.$patch"
    set_version "$new_version"
}

# Main script logic
case "$1" in
    get)
        get_version
        ;;
    set)
        set_version "$2"
        ;;
    bump)
        bump_version "$2"
        ;;
    *)
        echo "Usage: $0 {get|set|bump}"
        echo "  get           - Get current version"
        echo "  set <version> - Set specific version"
        echo "  bump <type>   - Bump version (major, minor, patch)"
        echo ""
        echo "Current version: $(get_version)"
        exit 1
        ;;
esac
