#!/bin/bash
# Test Host Total Logic

echo "Testing Host Total Logic..."
echo ""

# Start server in background
echo "Starting server..."
go run main.go &
PID=$!
sleep 2

# Verify Logic:
# 1. Total hits for api.test.com should be X (sum of previous hits from other runs if DB persisted, plus new ones)
# 2. Filter by prefix should show smaller total in 'data', but same grand 'total' in host object.

echo "Sending new hits..."
curl -s -o /dev/null -H "X-Forwarded-Host: api.test.com" -H "X-Forwarded-Uri: /v3/cal/today?debug=1" http://localhost:8080/verify
curl -s -o /dev/null -H "X-Forwarded-Host: api.test.com" -H "X-Forwarded-Uri: /v3/cal/tomorrow" http://localhost:8080/verify
curl -s -o /dev/null -H "X-Forwarded-Host: api.test.com" -H "X-Forwarded-Uri: /v2/quran/ayat" http://localhost:8080/verify

sleep 1

echo "========================================"
echo "GET /api.test.com?prefix=/v3"
echo "Expect: Host Total > Data Total"
curlie "http://localhost:8080/api.test.com?prefix=/v3"

# Cleanup
echo ""
echo "Stopping server..."
kill $PID
wait $PID 2>/dev/null
echo "Test complete!"
