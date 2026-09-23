// internal/gateway/two_phase_commit.go
package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sort"
	"time"

	"github.com/google/uuid"
)

type TransferRequest struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Amount int64  `json:"amount"`
}

type ShardPrepareRequest struct {
	TxnID  string `json:"txn_id"`
	User   string `json:"user"`
	Amount int64  `json:"amount"`
}

type Coordinator struct {
	router     *Router
	httpClient *http.Client
}

func NewCoordinator(router *Router) *Coordinator {
	return &Coordinator{
		router: router,
		httpClient: &http.Client{
			Timeout: SLOW_RESPONSE_TIMEOUT,
		},
	}
}

func (c *Coordinator) Execute(req TransferRequest) error {
	const MAX_RETRIES = 150
	var lastErr error
	baseDelay := 10 * time.Millisecond
	maxDelay := 500 * time.Millisecond

	for attempt := 0; attempt < MAX_RETRIES; attempt++ {
		txnID := uuid.New().String()
		lastErr = c.tryExecute(req, txnID)
		if lastErr == nil {
			return nil
		}

		sleep := baseDelay * time.Duration(1<<attempt)
		if sleep > maxDelay || sleep <= 0 {
			sleep = maxDelay
		}
		// Add randomized jitter to prevent Thundering Herd
		jitter := time.Duration(rand.Int63n(int64(sleep)))
		time.Sleep(baseDelay + jitter)
	}
	return fmt.Errorf("transaction failed after %d attempts: %w", MAX_RETRIES, lastErr)
}

func (c *Coordinator) tryExecute(req TransferRequest, txnID string) error {

	// Lexicographical Ordering to prevent Deadlocks
	participants := []struct {
		user   string
		amount int64
	}{
		{req.From, -req.Amount},
		{req.To, req.Amount},
	}
	sort.Slice(participants, func(i, j int) bool {
		return participants[i].user < participants[j].user
	})

	// --- Phase 1: Prepare ---
	preparedShards := make([]*ShardRoute, 0, len(participants))
	for _, participant := range participants {
		shard, err := c.router.GetShard(participant.user)
		if err != nil {
			c.rollbackAll(txnID, participants[0].user, participants[1].user)
			return err
		}

		if shard.CircuitBreaker.IsOpen() {
			c.rollbackAll(txnID, participants[0].user, participants[1].user)
			return fmt.Errorf("circuit open for user %s", participant.user)
		}

		payload := ShardPrepareRequest{
			TxnID:  txnID,
			User:   participant.user,
			Amount: participant.amount,
		}

		start := time.Now()
		statusCode, err := c.sendToShard(shard.Address+"/prepare", payload)
		if err != nil {
			if statusCode == 0 || statusCode >= 500 {
				shard.CircuitBreaker.RecordFailure()
			} else {
				shard.CircuitBreaker.RecordSuccess()
			}
			c.rollbackAll(txnID, participants[0].user, participants[1].user)
			return err
		}

		if time.Since(start) > SLOW_RESPONSE_TIMEOUT {
			shard.CircuitBreaker.RecordFailure()
		} else {
			shard.CircuitBreaker.RecordSuccess()
		}

		preparedShards = append(preparedShards, shard)
	}

	// --- Phase 2: Commit ---
	// Errors in commit are ignored here because 2PC guarantees eventual consistency via WAL recovery/gossip
	for i, participant := range participants {
		payload := ShardPrepareRequest{
			TxnID:  txnID,
			User:   participant.user,
			Amount: participant.amount,
		}
		_, _ = c.sendToShard(preparedShards[i].Address+"/commit", payload)
	}

	return nil
}

func (c *Coordinator) rollbackAll(txnID, userA, userB string) {
	for _, user := range []string{userA, userB} {
		shard, err := c.router.GetShard(user)
		if err != nil {
			continue
		}
		payload := ShardPrepareRequest{TxnID: txnID, User: user}
		_, _ = c.sendToShard(shard.Address+"/rollback", payload)
	}
}

func (c *Coordinator) sendToShard(url string, payload interface{}) (int, error) {
	body, _ := json.Marshal(payload)
	ctx, cancel := context.WithTimeout(context.Background(), SLOW_RESPONSE_TIMEOUT)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, fmt.Errorf("non-200 status: %d", resp.StatusCode)
	}
	return http.StatusOK, nil
}
