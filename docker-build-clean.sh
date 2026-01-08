#!/bin/bash
# Script to clean build and push fresh Docker image

echo "Cleaning buildx cache..."
docker buildx prune -af

echo "Building fresh image for linux/amd64..."
docker buildx build \
  --platform linux/amd64 \
  -t ghcr.io/banghasan/go-stats-traefik:latest \
  --push \
  .

echo "Done! Check image manifest:"
echo "docker buildx imagetools inspect ghcr.io/banghasan/go-stats-traefik:latest"
