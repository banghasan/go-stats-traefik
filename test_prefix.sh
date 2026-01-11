#!/bin/bash
# Test path prefix extraction with Host support

echo "Testing path prefix extraction with Host support..."
echo ""

# Start server in background
echo "Starting server..."
go run main.go &
PID=$!
sleep 3

# Send test requests for Host 1
HOST1="api.test.com"
echo "Sending hits for $HOST1..."
curl -s -o /dev/null -H "X-Forwarded-Host: $HOST1" -H "X-Forwarded-Uri: /v3/cal/today" http://localhost:8080/verify
curl -s -o /dev/null -H "X-Forwarded-Host: $HOST1" -H "X-Forwarded-Uri: /v2/quran/ayat/acak" http://localhost:8080/verify

# Send test requests for Host 2
HOST2="stats.test.com"
echo "Sending hits for $HOST2..."
curl -s -o /dev/null -H "X-Forwarded-Host: $HOST2" -H "X-Forwarded-Uri: /v1/info" http://localhost:8080/verify
curl -s -o /dev/null -H "X-Forwarded-Host: $HOST2" -H "X-Forwarded-Uri: /v1/status" http://localhost:8080/verify

echo "Hits sent. Waiting for processing..."
sleep 2

# Check results
echo ""
echo "========================================"
echo "GET / (Root Stats - Grouped by Host):"
echo "========================================"
if command -v curlie &> /dev/null; then
    curlie http://localhost:8080/
else
    echo "curlie not found, using curl..."
    curl -s http://localhost:8080/
fi

# Cleanup
echo ""
echo "Stopping server..."
kill $PID
wait $PID 2>/dev/null
echo "Test complete!"
