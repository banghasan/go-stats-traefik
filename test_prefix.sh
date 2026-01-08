#!/bin/bash
# Test path prefix extraction

echo "Testing path prefix extraction..."
echo ""

# Start server in background
echo "Starting server..."
go run main.go &
PID=$!
sleep 3

# Send test requests
echo "Sending test hits with different paths..."
curl -s -o /dev/null -H "X-Forwarded-Uri: /v3/cal/today" http://localhost:8080/
curl -s -o /dev/null -H "X-Forwarded-Uri: /v3/tools/ip" http://localhost:8080/
curl -s -o /dev/null -H "X-Forwarded-Uri: /v2/quran/ayat/acak" http://localhost:8080/
curl -s -o /dev/null -H "X-Forwarded-Uri: /v2/quran/surah/1" http://localhost:8080/
curl -s -o /dev/null -H "X-Forwarded-Uri: /" http://localhost:8080/

echo "Hits sent. Waiting for processing..."
sleep 2

# Check results
echo ""
echo "========================================"
echo "GET /api/stats (Should show /v2 and /v3 only):"
echo "========================================"
curl -s http://localhost:8080/api/stats | python3 -m json.tool

# Cleanup
echo ""
echo "Stopping server..."
kill $PID
wait $PID 2>/dev/null
echo "Test complete!"
