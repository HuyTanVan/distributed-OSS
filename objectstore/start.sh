#!/bin/bash

# Kill any existing instances
pkill -f "go run.*main.go" 2>/dev/null

# Create data directories
mkdir -p data/node1 data/node2 data/node3

echo "=========================================="
echo "Starting 3 Object Storage Nodes Locally"
echo "=========================================="

# Start Node 1
echo "Starting Node 1 on port 8081..."
DATA_DIR=./data/node1 PORT=8081 NODE_ID=node-1 go run ./cmd/server/main.go > logs/node1.log 2>&1 &
sleep 2

# Start Node 2
echo "Starting Node 2 on port 8082..."
DATA_DIR=./data/node2 PORT=8082 NODE_ID=node-2 go run ./cmd/server/main.go > logs/node2.log 2>&1 &
sleep 2

# Start Node 3
echo "Starting Node 3 on port 8083..."
DATA_DIR=./data/node3 PORT=8083 NODE_ID=node-3 go run ./cmd/server/main.go > logs/node3.log 2>&1 &
sleep 2

echo "=========================================="
echo "All nodes started!"
echo "=========================================="
echo "Node 1: http://localhost:8081"
echo "Node 2: http://localhost:8082"
echo "Node 3: http://localhost:8083"
echo "=========================================="
echo ""
echo "To view logs:"
echo "  tail -f logs/node1.log"
echo "  tail -f logs/node2.log"
echo "  tail -f logs/node3.log"
echo ""
echo "To stop all nodes:"
echo "  ./stop.sh"
echo "=========================================="