#!/bin/bash

echo "=========================================="
echo "Starting Load Balancer on port 8080"
echo "=========================================="

mkdir -p logs

go run ./cmd/load-balancer/main.go > logs/loadbalancer.log 2>&1 &

sleep 2

echo "Load Balancer started!"
echo "URL: http://localhost:8080"
echo ""
echo "To view logs:"
echo "  tail -f logs/loadbalancer.log"
echo ""
echo "To test:"
echo "  curl http://localhost:8080/"
echo "=========================================="