// cmd/shard_b/main.go
package main

import (
	"log"
	"net/http"
	"time"

	"volt/internal/shard"
)

func main() {
	// Initialize Shard B and its WAL
	shardEngine, err := shard.InitializeShard("shard_b.wal")
	if err != nil {
		log.Fatalf("Failed to initialize Shard B: %v", err)
	}

	serverMux := http.NewServeMux()

	// Wire the Two-Phase Commit endpoints
	serverMux.HandleFunc("/prepare", shardEngine.HandlePrepare)
	serverMux.HandleFunc("/commit", shardEngine.HandleCommit)
	serverMux.HandleFunc("/rollback", shardEngine.HandleRollback)

	server := &http.Server{
		Addr:         ":8002",
		Handler:      serverMux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Println("Shard B (N-Z) listening on port 8002")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Shard B server failed: %v", err)
	}
}
