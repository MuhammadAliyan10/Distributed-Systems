package main

import (
	"bytes"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	const NUM_ROUTINES = 100
	const TRANSFERS_PER_ROUTINE = 5

	var successCount int64
	var failureCount int64

	var wg sync.WaitGroup

	start := time.Now()

	for i := 0; i < NUM_ROUTINES; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			client := &http.Client{Timeout: 5 * time.Second}
			payload := []byte(`{"from": "Alice", "to": "Noah", "amount": 10}`)

			for j := 0; j < TRANSFERS_PER_ROUTINE; j++ {
				req, _ := http.NewRequest(http.MethodPost, "http://localhost:8000/transfer", bytes.NewReader(payload))
				req.Header.Set("Content-Type", "application/json")
				
				resp, err := client.Do(req)
				if err == nil {
					if resp.StatusCode == http.StatusOK {
						atomic.AddInt64(&successCount, 1)
					} else {
						atomic.AddInt64(&failureCount, 1)
					}
					resp.Body.Close()
				} else {
					atomic.AddInt64(&failureCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	fmt.Printf("--- Load Test Results ---\n")
	fmt.Printf("Total Time: %v\n", elapsed)
	fmt.Printf("Successful Transfers: %d\n", successCount)
	fmt.Printf("Failed Transfers: %d\n", failureCount)
	fmt.Printf("Throughput: %.2f req/sec\n", float64(NUM_ROUTINES*TRANSFERS_PER_ROUTINE)/elapsed.Seconds())
}
