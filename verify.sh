#!/bin/bash

# Default values
DEFAULT_HOST="localhost"
DEFAULT_PORT="8080"
HOST=${1:-$DEFAULT_HOST}
PORT=${2:-$DEFAULT_PORT}

echo "Testing server at: http://$HOST:$PORT"

# Start server in background
echo "Starting server..."
go run main.go -host $HOST -port $PORT &
PID=$!
echo "Server PID: $PID"

# Wait for server to be ready
sleep 5

# Test Middleware Hits
echo "---------------------------------------------------"
echo "Sending Hits..."
curl -s -o /dev/null -H "X-Forwarded-Uri: /api/v1/test" http://$HOST:$PORT/
curl -s -o /dev/null -H "X-Forwarded-Uri: /api/v1/test" http://$HOST:$PORT/
curl -s -o /dev/null -H "X-Forwarded-Uri: /api/v2/foo" http://$HOST:$PORT/
echo "Hits sent."

# Allow worker time to process
sleep 2

# Test API Stats
echo "---------------------------------------------------"
echo "GET /api/stats (Root Stats):"
curl -s http://$HOST:$PORT/api/stats
echo ""

echo "---------------------------------------------------"
YEAR=$(date +%Y)
echo "GET /api/stats/$YEAR (Year Stats):"
curl -s "http://$HOST:$PORT/api/stats/$YEAR"
echo ""

# Cleanup
echo "---------------------------------------------------"
echo "Stopping server..."
kill $PID
wait $PID 2>/dev/null
echo "Server stopped."
