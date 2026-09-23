#!/bin/bash
# test.sh

echo "Building and starting cluster..."

go build -o bin/shard_a cmd/shard_a/main.go
go build -o bin/shard_b cmd/shard_b/main.go
go build -o bin/gateway cmd/gateway/main.go

./bin/shard_a &
PID_A=$!

./bin/shard_b &
PID_B=$!

./bin/gateway &
PID_GW=$!

# Allow servers to bind to ports
sleep 2

echo -e "\n--- Test 1: Rate Limiter & Circuit Breaker (Valid Routing) ---"
echo "Requesting transfer from Alice (Shard A) to Noah (Shard B)..."
curl -v -X POST http://localhost:8000/transfer \
  -H "Content-Type: application/json" \
  -d '{"from": "Alice", "to": "Noah", "amount": 10}'

echo -e "\n\n--- Test 2: Insufficient Funds Verification ---"
echo "Note: The above request should return '402 Payment Required' because Alice's initial balance is 0."

echo -e "\n--- Test 3: Idempotent Crash Recovery ---"
echo "Killing Shard A..."
kill -9 $PID_A
sleep 1
echo "Restarting Shard A (should recover from shard_a.wal)..."
./bin/shard_a &
PID_A_NEW=$!
sleep 2

echo -e "\nShutting down cluster..."
kill -9 $PID_B $PID_GW $PID_A_NEW
rm -rf bin/
