package utils

import "sync"

type IDStore struct {
	ids []string
	mu  sync.RWMutex
}

func (s *IDStore) Add(ids ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ids = append(s.ids, ids...)
}

func (s *IDStore) Get() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ids
}
