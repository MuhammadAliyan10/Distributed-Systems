package shard

import (
	"encoding/json"
	"net/http"
)

type TwoPhaseRequest struct {
	TxnID  string `json:"txn_id"`
	User   string `json:"user"`
	Amount int64  `json:"amount"`
}

func (s *Shard) HandlePrepare(w http.ResponseWriter, r *http.Request) {
	var req TwoPhaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	txn := &Transaction{
		User:       req.User,
		Amount:     req.Amount,
		TxnID:      req.TxnID,
		State:      "PENDING",
		ResultChan: make(chan error, 1),
	}

	select {
	case s.walQueue <- txn:
	case <-r.Context().Done():
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		return
	}

	if err := <-txn.ResultChan; err != nil {
		if err.Error() == "conflict" {
			http.Error(w, "Conflict: Resource Locked", http.StatusConflict)
		} else if err.Error() == "insufficient funds" {
			http.Error(w, "Payment Required: Insufficient Funds", http.StatusPaymentRequired)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Shard) HandleCommit(w http.ResponseWriter, r *http.Request) {
	var req TwoPhaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	txn := &Transaction{
		User:       req.User,
		Amount:     req.Amount,
		TxnID:      req.TxnID,
		State:      "COMMITTED",
		ResultChan: make(chan error, 1),
	}

	select {
	case s.walQueue <- txn:
	case <-r.Context().Done():
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		return
	}

	if err := <-txn.ResultChan; err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Shard) HandleRollback(w http.ResponseWriter, r *http.Request) {
	var req TwoPhaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	txn := &Transaction{
		User:       req.User,
		Amount:     req.Amount,
		TxnID:      req.TxnID,
		State:      "ABORTED",
		ResultChan: make(chan error, 1),
	}

	select {
	case s.walQueue <- txn:
	case <-r.Context().Done():
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		return
	}

	if err := <-txn.ResultChan; err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}
