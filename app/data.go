package main

import (
	"errors"
	"slices"
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
	mu      sync.Mutex
	data    map[string]entry
	waiters map[string][]chan string
}

func NewStore() *Store {
	return &Store{data: make(map[string]entry), waiters: make(map[string][]chan string)}
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
	if ok && data.kind != kindList {
		return 0, ErrWrongType
	}
	if !ok {
		e := entry{
			list: append([]string(nil), value...),
			kind: kindList,
		}
		s.data[key] = e
	} else {
		data.list = append(data.list, value...)
		s.data[key] = data
	}
	l := len(s.data[key].list)
	s.serveWaiters(key)
	return l, nil
}

func (s *Store) LPush(key string, values ...string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ok := s.lookup(key)
	if ok && data.kind != kindList {
		return 0, ErrWrongType
	}
	newItems := slices.Clone(values)
	slices.Reverse(newItems)
	if !ok {
		e := entry{
			list: append([]string(nil), newItems...),
			kind: kindList,
		}
		s.data[key] = e
	} else {
		data.list = append(newItems, data.list...)
		s.data[key] = data
	}

	l := len(s.data[key].list)
	s.serveWaiters(key)
	return l, nil
}

func (s *Store) LLen(key string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ok := s.lookup(key)
	if ok && data.kind != kindList {
		return 0, ErrWrongType
	}
	if !ok {
		return 0, nil
	}
	return len(data.list), nil
}

func (s *Store) lpopLocked(key string, count int) ([]string, bool, error) {
	data, ok := s.lookup(key)
	var removedElements []string
	if ok && data.kind != kindList {
		return removedElements, true, ErrWrongType
	}
	if !ok {
		return removedElements, false, nil
	}
	popSliceLen := min(count, len(data.list))
	removedElements = append([]string(nil), data.list[:popSliceLen]...)
	for i := 0; i < popSliceLen; i++ {
		data.list[i] = ""
	}
	data.list = data.list[popSliceLen:]
	s.data[key] = data
	if len(data.list) == 0 {
		delete(s.data, key)
	}
	return removedElements, true, nil
}

func (s *Store) LPop(key string, count int) ([]string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lpopLocked(key, count)
}

func (s *Store) serveWaiters(key string) {
	waiters, ok := s.waiters[key]
	if !ok || len(waiters) == 0 {
		return
	}
	data, ok := s.lookup(key)
	if !ok {
		return
	}
	loopLen := min(len(data.list), len(waiters))
	elements, _, _ := s.lpopLocked(key, loopLen)
	for i := range loopLen {
		waiters[i] <- elements[i]
	}
	s.waiters[key] = waiters[loopLen:]
	if len(s.waiters[key]) == 0 {
		delete(s.waiters, key)
	}
}

func (s *Store) blpopTimer(key string, timeout int) ([]string, bool, error) {
	// Called with s.mu held; releases it before waiting.
	ch := make(chan string, 1)
	duration := time.Duration(timeout * int(time.Millisecond))
	timer := time.NewTimer(duration)
	s.waiters[key] = append(s.waiters[key], ch)
	s.mu.Unlock()
	select {
	case elem := <-ch:
		timer.Stop()
		return []string{key, elem}, false, nil
	case <-timer.C: // timeout
		s.mu.Lock()
		waiter := s.waiters[key]
		i := slices.Index(waiter, ch)
		if i != -1 {
			s.waiters[key] = append(waiter[:i], waiter[i+1:]...)
			if len(s.waiters[key]) == 0 {
				delete(s.waiters, key)
			}
			s.mu.Unlock()
			return nil, true, nil
		}
		s.mu.Unlock()
		elem := <-ch
		return []string{key, elem}, false, nil
	}
}

func (s *Store) blpopIndefinite(key string) ([]string, bool, error) {
	// Called with s.mu held; releases it before waiting.
	ch := make(chan string, 1)
	s.waiters[key] = append(s.waiters[key], ch)
	s.mu.Unlock()
	element := <-ch // waits here until ch has a value
	return []string{key, element}, false, nil
}

func (s *Store) BLPop(key string, timeout int) ([]string, bool, error) { // [key, element], is timed out, error msg
	s.mu.Lock()
	data, ok := s.lookup(key)
	if ok && data.kind != kindList {
		s.mu.Unlock()
		return nil, false, ErrWrongType
	}
	if ok && len(data.list) > 0 {
		element, _, err := s.lpopLocked(key, 1)
		s.mu.Unlock()
		if err != nil {
			return nil, false, err
		}
		return []string{key, element[0]}, false, nil
	}
	if timeout > 0 {
		return s.blpopTimer(key, timeout)
	}
	return s.blpopIndefinite(key)
}

func (s *Store) LRange(key string, start int, stop int) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ok := s.lookup(key)
	if !ok {
		return nil, nil
	}
	if data.kind != kindList {
		return nil, ErrWrongType
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
		return nil, nil
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
		return "", ErrWrongType
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
