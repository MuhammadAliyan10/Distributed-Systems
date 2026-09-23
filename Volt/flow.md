# System Flow & Architecture Diagram

This document illustrates the end-to-end execution of a distributed transaction in the Escrow Architecture using the Actor Model.

## System Graph (Nodes & Edges)

```mermaid
graph TD
    %% External
    Client((Client API Call))
    
    %% Gateway Nodes
    subgraph Gateway [API Gateway Process]
        Main[cmd/gateway/main.go]
        Router[internal/gateway/router.go]
        Coordinator[internal/gateway/two_phase_commit.go]
        LockManager[internal/gateway/lock_manager.go]
    end
    
    %% Shard A Nodes
    subgraph ShardA [Shard A Process - Port 8001]
        HandlerA[internal/shard/handler.go]
        WALQueueA((walQueue Channel))
        WALA[internal/shard/wal.go]
        DiskA[(shard_a.wal)]
    end
    
    %% Shard B Nodes
    subgraph ShardB [Shard B Process - Port 8002]
        HandlerB[internal/shard/handler.go]
        WALQueueB((walQueue Channel))
        WALB[internal/shard/wal.go]
        DiskB[(shard_b.wal)]
    end

    %% Flow Edges
    Client -->|1. POST /transfer| Main
    Main -->|2. Rate Limit & Context| Router
    Router -->|3. Route & SLA Check| LockManager
    LockManager -->|4. Acquire SLA & Start| Coordinator
    
    Coordinator -->|5a. PREPARE Sender| HandlerA
    Coordinator -->|5b. PREPARE Receiver| HandlerB
    
    HandlerA -->|6. Push Intent| WALQueueA
    HandlerB -->|6. Push Intent| WALQueueB
    
    WALQueueA -->|7. Pull Batch| WALA
    WALQueueB -->|7. Pull Batch| WALB
    
    WALA -->|8. Memory Validation| WALA
    WALB -->|8. Memory Validation| WALB
    
    WALA -->|9. fsync() Batch| DiskA
    WALB -->|9. fsync() Batch| DiskB
    
    WALA -.->|10. 200 OK Callback| HandlerA
    WALB -.->|10. 200 OK Callback| HandlerB
    
    HandlerA -.->|11. Response| Coordinator
    HandlerB -.->|11. Response| Coordinator
    
    Coordinator -->|12. COMMIT/ABORT| HandlerA
    Coordinator -->|12. COMMIT/ABORT| HandlerB
```

---

## Dry Run: Transfer $100 from Alice to Bob

**Context:** Alice is mapped to `Shard A`. Bob is mapped to `Shard B`. Alice has $1,000. Bob has $0.

### Step 1: The Request Entry (`cmd/gateway/main.go`)
1. The user sends a POST request with payload `{"from": "Alice", "to": "Bob", "amount": 100}`.
2. `handleTransfer` intercepts the request. It checks the token bucket (`rateLimiter`). Assuming the bucket allows traffic, it passes the request forward.

### Step 2: SLA & Load Shedding (`internal/gateway/two_phase_commit.go`)
1. The Coordinator creates a 2-second timeout `context`.
2. It attempts to acquire the `ContextAwareLockManager` semaphore for "Alice" and "Bob" at the Gateway layer to prevent network saturation.
3. It generates a unique `TxnID`, e.g., `Txn_123`.

### Step 3: Phase 1 - PREPARE
1. **Network Dispatch**: The Coordinator executes two concurrent `http.Post` requests (Goroutines):
   - To Shard A: `{"txn_id": "Txn_123", "user": "Alice", "amount": -100}` (Deduction)
   - To Shard B: `{"txn_id": "Txn_123", "user": "Bob", "amount": 100}` (Deposit)

2. **Lock-Free Handlers** (`internal/shard/handler.go`):
   - `HandlePrepare` at Shard A receives the request. It does **not** acquire a mutex.
   - It wraps the payload in a `Transaction` struct with a `ResultChan`.
   - It pushes the struct to `s.walQueue`. It blocks on `<-txn.ResultChan` until the background thread replies.

3. **Escrow Memory Validation** (`internal/shard/wal.go`):
   - The single `startGroupCommitLoop` thread wakes up, drains `walQueue`, and grabs a batch of transactions (e.g. 500 requests at once).
   - **At Shard A (Alice)**: The thread checks Alice's `AvailableBalance` ($1,000). Since $1,000 - $100 = $900 >= 0, the check passes. It deducts $100 from `AvailableBalance` and sets `PendingHolds["Txn_123"] = -100`.
   - **At Shard B (Bob)**: The thread checks `Amount > 0`. It does nothing to `AvailableBalance` (still $0) but sets `PendingHolds["Txn_123"] = 100`.

4. **Disk Serialization** (`internal/shard/wal.go`):
   - The thread compiles all valid transactions in the batch into a single byte buffer.
   - It calls `walFile.Write()` and `walFile.Sync()` (**fsync**), physically writing `PREPARE Txn_123` to disk.
   - It broadcasts `nil` (Success) to the `ResultChan` of all valid transactions.
   
5. **Phase 1 Return**:
   - `HandlePrepare` unblocks and returns `200 OK` to the Gateway Coordinator.

### Step 4: Phase 2 - COMMIT
1. **Decision**: The Gateway Coordinator receives `200 OK` from both Shard A and Shard B. It decides to COMMIT.
2. **Network Dispatch**: The Coordinator sends `COMMIT Txn_123` concurrently to Shard A and Shard B.
3. **Escrow Resolution** (`internal/shard/wal.go`):
   - Shard A receives the COMMIT in the `walQueue`. The WAL thread deletes `PendingHolds["Txn_123"]`. Alice's balance remains $900. It adds `Txn_123` to the `ExecutedTxns` tombstone cache.
   - Shard B receives the COMMIT in the `walQueue`. The WAL thread adds the $100 from `PendingHolds` into Bob's `AvailableBalance` (now $100). It deletes the hold and adds `Txn_123` to `ExecutedTxns`.
4. **Disk Serialization & Fan-out**: The WAL threads write the COMMIT records to disk via batch `fsync()` and return success to the handlers.
5. The Gateway receives the final successes and returns `200 OK` to the Client API Call.

### Step 5: Background Pruning (Garbage Collection)
1. At a scheduled interval, the system generates a `SYS_SWEEP` transaction into the `walQueue`.
2. The WAL thread processes it in memory, iterating over the `ExecutedTxns` map and deleting tombstones (e.g. `Txn_123`) older than 1 hour to prevent OOM memory leaks. No disk write occurs.
