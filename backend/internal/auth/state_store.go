package auth

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

type StateStore struct {
	mu      sync.Mutex
	ttl     time.Duration
	entries map[string]time.Time
}

func NewStateStore(ttl time.Duration) *StateStore {
	return &StateStore{
		ttl:     ttl,
		entries: make(map[string]time.Time),
	}
}

func (s *StateStore) Create(now time.Time) (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	state := base64.RawURLEncoding.EncodeToString(bytes)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[state] = now.Add(s.ttl)
	s.cleanupLocked(now)

	return state, nil
}

func (s *StateStore) Consume(state string, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	expiresAt, ok := s.entries[state]
	if !ok {
		return false
	}
	delete(s.entries, state)

	s.cleanupLocked(now)
	return now.Before(expiresAt)
}

func (s *StateStore) cleanupLocked(now time.Time) {
	for state, expiresAt := range s.entries {
		if now.After(expiresAt) {
			delete(s.entries, state)
		}
	}
}
