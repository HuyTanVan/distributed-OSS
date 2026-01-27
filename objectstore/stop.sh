#!/bin/bash

echo "Stopping all nodes..."
pkill -f "go run.*main.go"

echo "All nodes stopped!"