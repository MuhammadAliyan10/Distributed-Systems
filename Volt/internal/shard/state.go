package shard

import (
	"fmt"
	"os"
)

type Account struct {
	AvailableBalance int64
	PendingHolds     map[string]int64
	ExecutedTxns     map[string]int64
}

type Shard struct {
	state     map[string]*Account
	walFile   *os.File
	walQueue  chan *Transaction
	syncCount uint64
	txnStatus map[string]string
}

func InitializeShard(walPath string) (*Shard, error) {
	file, err := os.OpenFile(walPath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open WAL: %w", err)
	}

	s := &Shard{
		state: map[string]*Account{
			"Alice": {AvailableBalance: 1000000, PendingHolds: make(map[string]int64), ExecutedTxns: make(map[string]int64)},
			"Noah":  {AvailableBalance: 1000000, PendingHolds: make(map[string]int64), ExecutedTxns: make(map[string]int64)},
		},
		walFile:   file,
		walQueue:  make(chan *Transaction, 10000),
		txnStatus: make(map[string]string),
	}

	if err := s.recover(); err != nil {
		return nil, fmt.Errorf("recovery failed: %w", err)
	}

	go s.startGroupCommitLoop()

	return s, nil
}
