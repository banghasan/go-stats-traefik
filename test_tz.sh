#!/bin/bash
# Verify Timezone Flag and Logging

echo "Testing Timezone Support..."
echo ""

# Build first to ensure code valid
go build -o bin/go-stats-traefik

# Start server with UTC
echo "Starting server with -tz UTC..."
./bin/go-stats-traefik -tz "UTC" &
PID_UTC=$!
sleep 2

# Hit API
echo "Requesting /api (UTC)..."
curlie http://localhost:8080/api

# Stop
kill $PID_UTC
wait $PID_UTC 2>/dev/null

echo ""
echo "Starting server with -tz Asia/Jakarta..."
./bin/go-stats-traefik -tz "Asia/Jakarta" &
PID_JKT=$!
sleep 2

# Hit API
echo "Requesting /api (Jakarta)..."
curlie http://localhost:8080/api

# Stop
kill $PID_JKT
wait $PID_JKT 2>/dev/null

echo ""
echo "Test complete! Check console output above for timestamps."
