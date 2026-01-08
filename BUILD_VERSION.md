# Build and Version Management

This document explains how to manage versions and build the application properly.

## Version Convention

This project uses Semantic Versioning (SemVer) with format `MAJOR.MINOR.PATCH`:

- `MAJOR` version: Incremented for incompatible API changes
- `MINOR` version: Incremented for functionality added in backward-compatible manner  
- `PATCH` version: Incremented for backward-compatible bug fixes

## Version Storage

The application version is maintained in multiple places:

1. **VERSION file**: Contains the current version string (e.g., `1.0.0`)
2. **main.go**: Contains the `AppVersion` constant, updated during build
3. **Documentation**: README.md and API.md show version in examples

## Managing Versions

### Using the version.sh script

The project includes a `version.sh` script for easy version management:

```bash
# Get current version
./version.sh get

# Set specific version
./version.sh set 1.2.0

# Bump version (major, minor, or patch)
./version.sh bump major  # 1.0.0 -> 2.0.0
./version.sh bump minor  # 1.0.0 -> 1.1.0
./version.sh bump patch  # 1.0.0 -> 1.0.1
```

### Using the Makefile

The project also includes a Makefile with version management targets:

```bash
# Show current version
make version

# Bump versions
make bump-major  # Increase major version
make bump-minor  # Increase minor version  
make bump-patch  # Increase patch version
```

## Building with Versions

### Quick Build

```bash
# Build with default version from VERSION file
go build -o bin/go-stats-traefik .

# Build with specific version
go build -ldflags="-X 'main.AppVersion=1.2.0'" -o bin/go-stats-traefik .
```

### Using Build Script

```bash
# Build with version from VERSION file
./build.sh

# Build with specific version
./build.sh 1.2.0
```

The build script automatically embeds version information into the binary.

### Using Make

```bash
# Build with current version
make

# Clean and rebuild
make clean && make
```

## Build Information

The build system can also embed additional build information like git commit hash:

```bash
# Build with version and git commit info
go build -ldflags="-X 'main.AppVersion=1.2.0' -X 'main.BuildInfo=1.2.0-gabc1234'" -o bin/go-stats-traefik .
```

## Checking Version

After building, you can check the application version:

```bash
./bin/go-stats-traefik --version
```

This will show the version embedded in the binary.

## Health Check Version

The application also exposes version information through the health check endpoint:

```bash
curl http://localhost:8080/health
```

Response example:
```json
{
    "status": "healthy",
    "version": "1.0.0",
    "build": "1.0.0-gabc1234",
    "timestamp": 1704672000
}
```

## Docker Builds

When building Docker images, you can specify the version:

```bash
docker build --build-arg BUILD_VERSION=1.2.0 -t my-image .
```

## Best Practices

1. Always update the version before releasing
2. Use semantic versioning consistently
3. Document breaking changes in release notes
4. Test version-specific behaviors
5. Keep version numbers synchronized across files