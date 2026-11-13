package main

import (
	"fmt"
	"sync"
	"time"
)

type entry struct {
	kind      byte // s: string, l: list
	str       string
	list      []string
	expiresAt time.Time // zero value means no expiry
}

const (
	kindString byte = 's'
	kindList   byte = 'l'
)

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
		str:  value,
		kind: kindString,
	}
	if ttl > 0 {
		entry.expiresAt = time.Now().Add(ttl)
	}
	s.data[key] = entry
}

func (s *Store) RPush(key string, value ...string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ok := s.data[key]
	if !ok || s.isExpired(data) {
		// create
		e := entry{
			list: append([]string(nil), value...),
			kind: kindList,
		}
		s.data[key] = e
		return len(e.list), nil
	}
	if data.kind != kindList {
		return 0, fmt.Errorf("WRONGTYPE Operation against a key holding the wrong kind of value")
	}
	// append
	data.list = append(data.list, value...)
	s.data[key] = data
	return len(data.list), nil
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ok := s.data[key]
	if s.isExpired(data) {
		delete(s.data, key)
		return "", false
	}
	return data.str, ok
}

func (s *Store) isExpired(data entry) bool {
	return !data.expiresAt.IsZero() && time.Now().After(data.expiresAt)
}
