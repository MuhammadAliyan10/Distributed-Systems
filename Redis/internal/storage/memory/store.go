// redis/internal/storage/memory/store.go
package memory

import (
	"redis/internal/storage"
	"sync"
	"time"
)

type Store struct {
mu sync.RWMutex
data map[string]storage.Value
}


func NewStore() *Store{
	return &Store{
		data: make(map[string]storage.Value, 1024),
	}
}

func (s *Store) Get(key string) (storage.Value, bool){
				s.mu.Lock()
				defer s.mu.Unlock()

				val, exists := s.data[key]
				if !exists{
					return  storage.Value{}, false
				}
				if !val.ExpiresAt.IsZero() && time.Now().After(val.ExpiresAt){
					return storage.Value{}, false
				}
				return val, true
}

func (s *Store) Set(key string, value[]byte, ttl time.Duration){
	s.mu.Lock()
	defer s.mu.Unlock()

	var expiresAt time.Time

	if ttl > 0{
		expiresAt = time.Now().Add(ttl)
	}
	s.data[key] = storage.Value{
		Data: value,
		Type: "string",
		ExpiresAt: expiresAt,
	}
}

func (s *Store) Del(key string) int{
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.data[key];
	exists{
		delete(s.data, key)
		return 1
	}
	return 0
}
