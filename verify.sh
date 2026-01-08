#!/bin/bash
# Start server in background
echo "Starting server..."
go run main.go &
PID=$!
echo "Server PID: $PID"

# Wait for server to be ready
sleep 5

# Test Middleware Hits
echo "---------------------------------------------------"
echo "Sending Hits..."
curl -s -o /dev/null -H "X-Forwarded-Uri: /api/v1/test" http://localhost:8080/
curl -s -o /dev/null -H "X-Forwarded-Uri: /api/v1/test" http://localhost:8080/
curl -s -o /dev/null -H "X-Forwarded-Uri: /api/v2/foo" http://localhost:8080/
echo "Hits sent."

# Allow worker time to process
sleep 2

# Test API Stats
echo "---------------------------------------------------"
echo "GET /api/stats (Root Stats):"
curl -s http://localhost:8080/api/stats
echo ""

echo "---------------------------------------------------"
YEAR=$(date +%Y)
echo "GET /api/stats/$YEAR (Year Stats):"
curl -s "http://localhost:8080/api/stats/$YEAR"
echo ""

# Cleanup
echo "---------------------------------------------------"
echo "Stopping server..."
kill $PID
wait $PID 2>/dev/null
echo "Server stopped."
