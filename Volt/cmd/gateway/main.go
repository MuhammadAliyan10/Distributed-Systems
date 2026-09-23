// cmd/gateway/main.go
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"volt/internal/gateway"
)

var (
	rateLimiter *gateway.RateLimiter
	coordinator *gateway.Coordinator
	router      *gateway.Router
)

func handleTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req gateway.TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// 1. Rate Limit on the sender's identity
	if !rateLimiter.Allow(req.From) {
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		return
	}

	// 2. Circuit Breaker pre-check on the sender's shard before opening TCP connections
	fromShard, err := router.GetShard(req.From)
	if err != nil {
		http.Error(w, "Bad Request: Invalid user", http.StatusBadRequest)
		return
	}
	if fromShard.CircuitBreaker.IsOpen() {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}

	// 3. Execute the Two-Phase Commit Protocol
	if err := coordinator.Execute(req); err != nil {
		log.Printf("Transfer failed: %v", err)
		http.Error(w, "Transfer Failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func main() {
	router = gateway.NewRouter()
	rateLimiter = gateway.NewRateLimiter()
	coordinator = gateway.NewCoordinator(router)

	serverMux := http.NewServeMux()
	serverMux.HandleFunc("/transfer", handleTransfer)

	server := &http.Server{
		Addr:         ":8000",
		Handler:      serverMux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Println("Gateway listening on port 8000")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Gateway server failed: %v", err)
	}
}
