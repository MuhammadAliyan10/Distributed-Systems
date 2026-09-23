// internal/shard/recovery.go
package shard

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"strconv"
	"strings"
	"time"
)

func (s *Shard) recover() error {
	_, err := s.walFile.Seek(0, 0)
	if err != nil {
		return fmt.Errorf("failed to seek to beginning of wal: %w", err)
	}

	crcTable := crc32.MakeTable(crc32.Castagnoli)
	reader := bufio.NewReader(s.walFile)
	var validOffset int64 = 0

	for {
		header := make([]byte, 8)
		_, err := io.ReadFull(reader, header)
		if err == io.EOF {
			break
		}
		if err != nil {
			break // Corrupted or partial header, stop reading
		}

		payloadLen := binary.LittleEndian.Uint32(header[0:4])
		expectedCRC := binary.LittleEndian.Uint32(header[4:8])

		payload := make([]byte, payloadLen)
		_, err = io.ReadFull(reader, payload)
		if err != nil {
			break // Corrupted payload, stop reading
		}

		actualCRC := crc32.Checksum(payload, crcTable)
		if actualCRC != expectedCRC {
			break // Checksum mismatch, stop reading
		}

		validOffset += 8 + int64(payloadLen)

		// Process payload (Format: TxnID,User,Amount,State\n)
		parts := strings.Split(string(bytes.TrimSpace(payload)), ",")
		if len(parts) != 4 {
			continue
		}

		txnID := parts[0]
		user := parts[1]
		amount, _ := strconv.ParseInt(parts[2], 10, 64)
		state := parts[3]

		s.txnStatus[txnID] = state

		account := s.state[user]
		if account == nil {
			account = &Account{
				AvailableBalance: 0,
				PendingHolds:     make(map[string]int64),
				ExecutedTxns:     make(map[string]int64),
			}
			s.state[user] = account
		}

		now := time.Now().Unix()

		switch state {
		case "PENDING":
			if amount < 0 {
				account.AvailableBalance += amount
				account.PendingHolds[txnID] = amount
			} else {
				account.PendingHolds[txnID] = amount
			}
		case "COMMITTED":
			if amount > 0 {
				account.AvailableBalance += account.PendingHolds[txnID]
			}
			delete(account.PendingHolds, txnID)
			account.ExecutedTxns[txnID] = now
		case "ABORTED":
			if amount < 0 {
				account.AvailableBalance -= account.PendingHolds[txnID]
			}
			delete(account.PendingHolds, txnID)
			account.ExecutedTxns[txnID] = now
		}
	}

	// Truncate any partial/corrupted writes at the end of the file
	if err := s.walFile.Truncate(validOffset); err != nil {
		return fmt.Errorf("failed to truncate corrupted tail: %w", err)
	}

	// Seek back to the end of the valid data so new writes append correctly
	_, err = s.walFile.Seek(validOffset, 0)
	if err != nil {
		return fmt.Errorf("failed to seek to end of wal: %w", err)
	}

	return nil
}
