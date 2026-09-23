// cmd/shard_a/main.go
package main

import (
	"log"
	"net/http"
	"time"

	"volt/internal/shard"
)

func main() {
	shardEngine, err := shard.InitializeShard("shard_a.wal")
	if err != nil {
		log.Fatalf("Failed to initialize Shard A: %v", err)
	}

	serverMux := http.NewServeMux()
	serverMux.HandleFunc("/prepare", shardEngine.HandlePrepare)
	serverMux.HandleFunc("/commit", shardEngine.HandleCommit)
	serverMux.HandleFunc("/rollback", shardEngine.HandleRollback)

	server := &http.Server{
		Addr:         ":8001",
		Handler:      serverMux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Println("Shard A (A-M) listening on port 8001")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Shard A server failed: %v", err)
	}
}
