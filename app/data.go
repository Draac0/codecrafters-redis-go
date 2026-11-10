package main

import (
	"sync"
	"time"
)

type entry struct {
	value     string
	expiresAt time.Time // zero value means no expiry
}

type Store struct {
	mu   sync.Mutex
	data map[string]entry
}

func NewStore() *Store {
	return &Store{data: make(map[string]entry)}
}

func (s *Store) Set(key, value string, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry := entry{
		value: value,
	}
	if ttl > 0 {
		entry.expiresAt = time.Now().Add(ttl)
	}
	s.data[key] = entry
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ok := s.data[key]
	if !data.expiresAt.IsZero() && time.Now().After(data.expiresAt) {
		delete(s.data, key)
		return "", false
	}
	return data.value, ok
}
