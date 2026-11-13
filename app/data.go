package main

import (
	"errors"
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
	e := entry{
		str:  value,
		kind: kindString,
	}
	if ttl > 0 {
		e.expiresAt = time.Now().Add(ttl)
	}
	s.data[key] = e
}

func (s *Store) RPush(key string, value ...string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ok := s.lookup(key)
	if !ok {
		// create
		e := entry{
			list: append([]string(nil), value...),
			kind: kindList,
		}
		s.data[key] = e
		return len(e.list), nil
	}
	if data.kind != kindList {
		return 0, WrongType
	}
	// append
	data.list = append(data.list, value...)
	s.data[key] = data
	return len(data.list), nil
}

func (s *Store) LRange(key string, start int, stop int) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ok := s.lookup(key)
	if !ok {
		return []string{}, nil
	}
	if data.kind != kindList {
		return []string{}, WrongType
	}
	l := len(data.list)
	if start < 0 {
		start += l
		if start < 0 {
			start = 0
		}
	}

	if stop < 0 {
		stop += l
	}

	if stop >= l {
		stop = l - 1
	}

	if start > stop || start >= l {
		return []string{}, nil
	}

	return append([]string(nil), data.list[start:stop+1]...), nil
}

func (s *Store) Get(key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.lookup(key)
	if !ok {
		return "", errors.New("")
	}
	if e.kind != kindString {
		return "", WrongType
	}
	return e.str, nil
}

func (s *Store) lookup(key string) (entry, bool) {
	e, ok := s.data[key]
	if ok && !e.expiresAt.IsZero() && time.Now().After(e.expiresAt) {
		delete(s.data, key)
		return entry{}, false
	}
	return e, ok
}
