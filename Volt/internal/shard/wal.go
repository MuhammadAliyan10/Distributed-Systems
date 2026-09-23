package shard

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"sync/atomic"
	"time"
)

type Transaction struct {
	User       string
	Amount     int64
	TxnID      string
	State      string
	ResultChan chan error
}

func (s *Shard) startGroupCommitLoop() {
	const MAX_BATCH_SIZE = 10000
	crcTable := crc32.MakeTable(crc32.Castagnoli)

	for {
		func() {
			var batch []*Transaction
			
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("WAL Thread Panic: %v\n", r)
					for _, txn := range batch {
						select {
						case txn.ResultChan <- fmt.Errorf("internal server error: WAL panic"):
						default:
						}
					}
				}
			}()

			firstTxn := <-s.walQueue
			batch = append(batch, firstTxn)

		DrainLoop:
			for len(batch) < MAX_BATCH_SIZE {
				select {
				case next := <-s.walQueue:
					batch = append(batch, next)
				default:
					break DrainLoop
				}
			}

			// Phase 1: Memory Validation (Commutative Reservations & Idempotency)
			validTxns := make([]*Transaction, 0, len(batch))
			now := time.Now().Unix()

			for _, txn := range batch {
				account := s.state[txn.User]
				if account == nil {
					account = &Account{
						AvailableBalance: 0,
						PendingHolds:     make(map[string]int64),
						ExecutedTxns:     make(map[string]int64),
					}
					s.state[txn.User] = account
				}

				if txn.TxnID == "SYS_SWEEP" {
					// Garbage Collect old tombstones
					for id, timestamp := range account.ExecutedTxns {
						if now-timestamp > 3600 { // 1 hour TTL
							delete(account.ExecutedTxns, id)
						}
					}
					// Garbage Collect old holds
					// Note: Real implementation needs timestamps on holds, omitting for brevity
					continue
				}

				if txn.State == "READ" {
					txn.ResultChan <- fmt.Errorf("balance:%d", account.AvailableBalance)
					continue
				}

				var validationErr error
				_, executed := account.ExecutedTxns[txn.TxnID]
				_, pending := account.PendingHolds[txn.TxnID]

				switch txn.State {
				case "PENDING":
					if executed || pending {
						// Idempotent success: already executed or already pending
						txn.ResultChan <- nil
						continue
					}
					if txn.Amount < 0 {
						if account.AvailableBalance+txn.Amount < 0 {
							validationErr = errors.New("insufficient funds")
						} else {
							account.AvailableBalance += txn.Amount
							account.PendingHolds[txn.TxnID] = txn.Amount
						}
					} else {
						account.PendingHolds[txn.TxnID] = txn.Amount
					}

				case "COMMITTED":
					if executed {
						txn.ResultChan <- nil
						continue
					}
					if !pending {
						validationErr = errors.New("invalid state transition: no pending hold found")
					} else {
						if txn.Amount > 0 {
							account.AvailableBalance += account.PendingHolds[txn.TxnID]
						}
						delete(account.PendingHolds, txn.TxnID)
						account.ExecutedTxns[txn.TxnID] = now
					}

				case "ABORTED":
					if executed {
						txn.ResultChan <- nil
						continue
					}
					if !pending {
						// Idempotent abort: if it's not pending and not executed, it either never arrived or was already swept.
						txn.ResultChan <- nil
						continue
					}
					if txn.Amount < 0 {
						account.AvailableBalance -= account.PendingHolds[txn.TxnID]
					}
					delete(account.PendingHolds, txn.TxnID)
					account.ExecutedTxns[txn.TxnID] = now
				}

				if validationErr != nil {
					txn.ResultChan <- validationErr
				} else {
					validTxns = append(validTxns, txn)
				}
			}

			if len(validTxns) == 0 {
				return
			}

			// Phase 2: Serialization
			var buffer bytes.Buffer
			for _, txn := range validTxns {
				payload := []byte(fmt.Sprintf("%s,%s,%d,%s\n", txn.TxnID, txn.User, txn.Amount, txn.State))

				lengthBytes := make([]byte, 4)
				binary.LittleEndian.PutUint32(lengthBytes, uint32(len(payload)))
				buffer.Write(lengthBytes)

				checksumBytes := make([]byte, 4)
				binary.LittleEndian.PutUint32(checksumBytes, crc32.Checksum(payload, crcTable))
				buffer.Write(checksumBytes)

				buffer.Write(payload)
			}

			// Phase 3: Disk I/O
			var syncErr error
			if _, err := s.walFile.Write(buffer.Bytes()); err != nil {
				syncErr = fmt.Errorf("write failed: %w", err)
			} else if err := s.walFile.Sync(); err != nil {
				syncErr = fmt.Errorf("sync failed: %w", err)
			} else {
				atomic.AddUint64(&s.syncCount, 1)
			}

			// Phase 4: Fan-out callback
			for _, txn := range validTxns {
				select {
				case txn.ResultChan <- syncErr:
				default:
				}
			}
		}()
	}
}
