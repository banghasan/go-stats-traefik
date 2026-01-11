#!/bin/bash
#./bin/go-stats-traefik

PORT=8080
echo "GET /"
curlie -s http://localhost:$PORT/

#echo ""
#echo "HEAD /verify"
#curl -s -I http://localhost:$PORT/verify
#echo ""
#echo "GET /total"
#curl -s -v http://localhost:$PORT/total
#echo ""
#echo "GET /year/2025"
#curl -s -v http://localhost:$PORT/year/2025
#echo ""
#echo "GET /data/v3"
#curl -s -v http://localhost:$PORT/data/v3