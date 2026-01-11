#!/bin/bash
# Test New Route Structure: / (Middleware) and /api/ (Stats)

echo "Testing New Route Structure..."
echo ""

# Start server in background
echo "Starting server..."
go run main.go &
PID=$!
sleep 2

# 1. Test Middleware (Path /)
# Should return 200 OK
echo "Testing Middleware (HEAD /)..."
curl -I -H "X-Forwarded-Host: new.host.com" -H "X-Forwarded-Uri: /v1/test" http://localhost:8080/

# 2. Test Stats API (Path /api/)
echo ""
echo "Testing Stats API (GET /api/)..."
curlie "http://localhost:8080/api/"

# 3. Test Host Stats API (Path /api/:host)
echo ""
echo "Testing Host Stats API (GET /api/new.host.com)..."
curlie "http://localhost:8080/api/new.host.com"

# Cleanup
echo ""
echo "Stopping server..."
kill $PID
wait $PID 2>/dev/null
echo "Test complete!"
