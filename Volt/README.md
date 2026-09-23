# Volt - Distributed Ledger System

Volt is a high-performance, fault-tolerant distributed ledger (banking) system designed to handle cross-shard transactions reliably. It implements advanced distributed systems concepts to ensure ACID guarantees, strict consistency, and high availability.

## Architecture Overview

The system is separated into a stateless **Gateway** and stateful **Shards**.

1. **Gateway (`cmd/gateway`)**: Acts as the coordinator. It authenticates, rate-limits, routes requests, and orchestrates the Two-Phase Commit (2PC) protocol across multiple storage nodes.
2. **Shards (`cmd/shard_a`, `cmd/shard_b`)**: The durable storage nodes. They handle locking, in-memory state management, and Write-Ahead Logging (WAL) for fault tolerance.

---

## Directory Structure

### `/cmd` - Application Entrypoints

*   **`cmd/gateway/main.go`**: The entry point for the API Gateway (Port 8000). It initializes the rate limiter, circuit breakers, routing logic, and HTTP server.
*   **`cmd/shard_a/main.go`**: The entry point for Shard A (Port 8001), responsible for users `A-M`.
*   **`cmd/shard_b/main.go`**: The entry point for Shard B (Port 8002), responsible for users `N-Z`.

### `/internal/gateway` - Gateway Logic

*   **`internal/gateway/two_phase_commit.go`**: The Distributed Coordinator. Orchestrates transactions across shards. Enforces **Lexicographical Key Ordering** to mathematically eliminate circular wait deadlocks.
*   **`internal/gateway/rate_limiter.go`**: Implements a per-user **Token Bucket Rate Limiter** to prevent DDoS and spam. Uses Double-Checked Locking for high-throughput concurrency.
*   **`internal/gateway/circuit_breaker.go`**: Protects backend shards from cascading failures. Trips the circuit (rejects requests instantly) if a shard starts timing out or failing.
*   **`internal/gateway/router.go`**: Static routing map. Maps user identities (by first character) to the correct backend shard network address.

### `/internal/shard` - Storage Node Logic

*   **`internal/shard/handler.go`**: The HTTP handlers for the 2PC protocol (`/prepare`, `/commit`, `/rollback`). Enforces strict state-machine transitions and verifies sufficient funds before locking resources.
*   **`internal/shard/lock_table.go`**: A **Striped Lock Table** (256 stripes) mapped via FNV hashing. Enables fine-grained row-level locking, preventing global mutex contention during high load.
*   **`internal/shard/wal.go`**: The Write-Ahead Log implementation. Features a non-blocking **Group Commit Loop** that batches up to 1,000 transactions before issuing an `fsync()` to disk, ensuring maximum disk throughput. Appends CRC32 checksums for corruption detection.
*   **`internal/shard/state.go`**: Manages the in-memory map of user balances and initializes background routines (like the Group Commit Loop).
*   **`internal/shard/recovery.go`**: Boot-time recovery. Sequentially replays the WAL from disk into memory, verifying CRC32 checksums. If a torn write (partial flush) is detected, it automatically truncates the corrupted tail to restore integrity.

### Root Files
*   **`test.sh`**: An automated end-to-end integration test script. Builds the binaries, spins up the cluster, executes a distributed transaction, and tests idempotency by crashing and restarting a shard.

---

## Technical Highlights

*   **Durability**: Uses a Write-Ahead Log (WAL) with `fsync()`.
*   **Idempotency**: All shard handlers are idempotent to survive network retries.
*   **Deadlock Prevention**: Coordinator sorts keys lexicographically before acquiring locks.
*   **Lock Striping**: Row-level locking without massive memory overhead.
*   **Group Commit**: Converts thousands of small disk IOPS into large sequential writes.
