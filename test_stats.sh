#!/bin/bash
#./bin/go-stats-traefik

PORT=8080
echo "GET /"
curlie -s http://localhost:$PORT/

#echo ""
#echo "HEAD /verify"
#curl -s -I http://localhost:$PORT/verify
#echo ""

## PATH /:host?year=:year&prefix=:prefix&all=1 
