#!/bin/bash

mkdir -p .pid logs

./scripts/stop_network.sh

echo "Starting AetherWave network with mDNS discovery..."

KEY=$(go run -mod=mod scripts/generate_key.go)
if [ $? -ne 0 ] || [ -z "$KEY" ]; then
    echo "Error generating encryption key"
    exit 1
fi
echo "Generated encryption key: $KEY"

go run cmd/aetherwave/main.go -address :3000 -name Node1 -key "$KEY" -discovery > logs/node1.log 2>&1 &
PID1=$!
echo $PID1 > .pid/node1.pid
echo "Node 1 started with PID $PID1"
sleep 2

go run cmd/aetherwave/main.go -address :3001 -name Node2 -key "$KEY" -discovery > logs/node2.log 2>&1 &
PID2=$!
echo $PID2 > .pid/node2.pid
echo "Node 2 started with PID $PID2"
sleep 1

go run cmd/aetherwave/main.go -address :3002 -name Node3 -key "$KEY" -discovery > logs/node3.log 2>&1 &
PID3=$!
echo $PID3 > .pid/node3.pid
echo "Node 3 started with PID $PID3"

echo "AetherWave network started with automatic node discovery"
echo "To stop the network: ./scripts/stop_network.sh"
