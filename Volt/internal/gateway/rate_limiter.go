// internal/gateway/rate_limiter.go
package gateway

import (
	"sync"
	"time"
)

type TokenBucket struct {
	tokens              float64
	capacity            float64
	refillRatePerSecond float64
	lastRefillTimestamp time.Time
	mutex               sync.Mutex
}

type RateLimiter struct {
	buckets map[string]*TokenBucket
	mutex   sync.RWMutex
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]*TokenBucket),
	}
}

func (rl *RateLimiter) getBucket(user string) *TokenBucket {
	rl.mutex.RLock()
	bucket, exists := rl.buckets[user]
	rl.mutex.RUnlock()

	if exists {
		return bucket
	}

	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	bucket, exists = rl.buckets[user]
	if exists {
		return bucket
	}

	bucket = &TokenBucket{
		tokens:              10000.0,
		capacity:            10000.0,
		refillRatePerSecond: 10000.0,
		lastRefillTimestamp: time.Now(),
	}
	rl.buckets[user] = bucket
	return bucket
}

func (rl *RateLimiter) Allow(user string) bool {
	// FIX: Use getBucket to prevent nil pointer panics on new users
	bucket := rl.getBucket(user)

	bucket.mutex.Lock()
	defer bucket.mutex.Unlock()

	now := time.Now()
	elapsed := now.Sub(bucket.lastRefillTimestamp).Seconds()

	tokensToAdd := elapsed * bucket.refillRatePerSecond
	bucket.tokens = min(bucket.capacity, bucket.tokens+tokensToAdd)
	bucket.lastRefillTimestamp = now

	if bucket.tokens >= 1.0 {
		bucket.tokens -= 1.0
		return true
	}

	return false
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
