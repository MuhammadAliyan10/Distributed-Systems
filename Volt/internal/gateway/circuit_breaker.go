// internal/gateway/circuit_breaker.go
package gateway

import (
	"sync"
	"sync/atomic"
	"time"
)

type CircuitState int32

const (
	CircuitClosed CircuitState = 0
	CircuitOpen   CircuitState = 1
)

const (
	FAILURE_THRESHOLD     = 5
	SLOW_RESPONSE_TIMEOUT = 500 * time.Millisecond
	RECOVERY_WAIT         = 10 * time.Second
)

type CircuitBreaker struct {
	state       int32
	loadCounter int64
	trippedAt   time.Time
	mu          sync.Mutex
}

func NewCircuitBreaker() *CircuitBreaker {
	return &CircuitBreaker{}
}

func (cb *CircuitBreaker) IsOpen() bool {
	if atomic.LoadInt32(&cb.state) == int32(CircuitClosed) {
		return false
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if time.Since(cb.trippedAt) > RECOVERY_WAIT {
		atomic.StoreInt32(&cb.state, int32(CircuitClosed))
		atomic.StoreInt64(&cb.loadCounter, 0)
		return false
	}

	return true
}

func (cb *CircuitBreaker) RecordSuccess() {
	atomic.StoreInt64(&cb.loadCounter, 0)
}

func (cb *CircuitBreaker) RecordFailure() {
	count := atomic.AddInt64(&cb.loadCounter, 1)
	if count >= FAILURE_THRESHOLD {
		cb.mu.Lock()
		defer cb.mu.Unlock()

		if atomic.LoadInt32(&cb.state) == int32(CircuitClosed) {
			atomic.StoreInt32(&cb.state, int32(CircuitOpen))
			cb.trippedAt = time.Now()
		}
	}
}
