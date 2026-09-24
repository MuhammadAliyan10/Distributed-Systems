// redis/internal/storage/memory/store.go
package memory

import (
	"redis/internal/storage"
	"sync"
	"time"
)

const shardCount = 256

type Shard struct {
				mu sync.RWMutex
				data map[string]storage.Value
}

type Store struct {
	shards []*Shard
}


func NewStore() *Store{
	s := &Store{
		shards: make([]*Shard, shardCount),
	}

	for i :=0; i<shardCount; i++{
		s.shards[i] = &Shard{
			data: make(map[string]storage.Value, 1024),
		}
	}
	s.StartEvictionWorker()
	return s
}

func fnv32a(key string) uint32{
	const (
		offset32 = 2166136261
		prime32 = 16777619
	)
	var hash uint32 = offset32
	for i:=0; i<len(key); i++{
		hash ^= uint32(key[i])
		hash *= prime32
	}
	return hash
}


func (s *Store) getShard(key string) *Shard{
	hash := fnv32a((key))
	return s.shards[hash&(shardCount - 1)]
}
func (s *Store) Get(key string) (storage.Value, bool){
				shard := s.getShard(key)
				shard.mu.RLock()
				defer shard.mu.RUnlock()

				val, exists := shard.data[key]
				if !exists{
					return  storage.Value{}, false
				}
				if !val.ExpiresAt.IsZero() && time.Now().After(val.ExpiresAt){
					return storage.Value{}, false
				}
				return val, true
}

func (s *Store) Set(key string, value[]byte, ttl time.Duration){
	shard := s.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	var expiresAt time.Time

	if ttl > 0{
		expiresAt = time.Now().Add(ttl)
	}
	shard.data[key] = storage.Value{
		Data: value,
		Type: "string",
		ExpiresAt: expiresAt,
	}
}

func (s *Store) Del(key string) int{
	shard := s.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	if _, exists := shard.data[key];
	exists{
		delete(shard.data, key)
		return 1
	}
	return 0
}
