package store

import "sync"

type KVStore struct {
    mu   sync.RWMutex
    data map[string]string
}

func New() *KVStore {
    return &KVStore{
        data: make(map[string]string),
    }
}

func (s *KVStore) Set(key, value string) {
	// prevent concurrent writes from separate nodes
    s.mu.Lock()
    defer s.mu.Unlock()

    s.data[key] = value
}

func (s *KVStore) Get(key string) (string, bool) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    v, ok := s.data[key]

    return v, ok
}

func (s *KVStore) Snapshot() map[string]string {
    s.mu.RLock()
    defer s.mu.RUnlock()

    copy := make(map[string]string)

    for k, v := range s.data {
        copy[k] = v
    }

    return copy
}