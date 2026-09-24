package memory

import (
	"time"
)


func (s *Store) StartEvictionWorker() {
	go func(){
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for range ticker.C{
			s.evictCycle()
		}
	}()
}

func (s *Store) evictCycle(){
	const sampleSize = 20

	for _, shard := range s.shards{
		sampleCount := 0

		shard.mu.Lock()

		for key, val := range shard.data{
			if sampleCount >= sampleSize{
				break
			}
			if !val.ExpiresAt.IsZero() && time.Now().After(val.ExpiresAt){
				delete(shard.data, key)
			}
			sampleCount++
		}
		shard.mu.Unlock()
	}


}
