#!/bin/bash
# Verify API Metadata and Stats Logic

echo "Testing API Metadata..."
echo ""

# Start server in background
echo "Starting server..."
go run main.go &
PID=$!
sleep 2

# 1. Metadata Endpoint /api
# Should return app info JSON
echo "GET /api (Metadata)"
curlie http://localhost:8080/api

# 2. Stats Endpoint /api/
# Should return stats list
echo ""
echo "GET /api/ (Stats)"
curlie http://localhost:8080/api/

# Cleanup
echo ""
echo "Stopping server..."
kill $PID
wait $PID 2>/dev/null
echo "Test complete!"
