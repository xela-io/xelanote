package fido2

import (
	"errors"
	"sync"
	"time"
)

// ErrSessionNotFound is returned when a session key doesn't exist or has expired
var ErrSessionNotFound = errors.New("session not found or expired")

// SessionStore defines an interface for storing WebAuthn challenges and pending login tokens.
// Values are stored with a TTL and retrieved exactly once (Get deletes the entry).
type SessionStore interface {
	Store(key string, data any, ttl time.Duration) error
	Get(key string) (any, error) // One-time: deletes after retrieval
	Delete(key string) error
}

type sessionEntry struct {
	data      any
	expiresAt time.Time
}

// InMemorySessionStore is a thread-safe in-memory session store with TTL cleanup.
type InMemorySessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*sessionEntry
	stopChan chan struct{}
}

// NewInMemorySessionStore creates a new in-memory session store with periodic cleanup.
func NewInMemorySessionStore() *InMemorySessionStore {
	s := &InMemorySessionStore{
		sessions: make(map[string]*sessionEntry),
		stopChan: make(chan struct{}),
	}
	go s.cleanup()
	return s
}

// Store saves data under the given key with a TTL.
func (s *InMemorySessionStore) Store(key string, data any, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[key] = &sessionEntry{
		data:      data,
		expiresAt: time.Now().Add(ttl),
	}
	return nil
}

// Get retrieves and deletes data for the given key (one-time use).
func (s *InMemorySessionStore) Get(key string) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.sessions[key]
	if !ok {
		return nil, ErrSessionNotFound
	}

	delete(s.sessions, key)

	if time.Now().After(entry.expiresAt) {
		return nil, ErrSessionNotFound
	}

	return entry.data, nil
}

// Delete removes data for the given key.
func (s *InMemorySessionStore) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, key)
	return nil
}

// Stop halts the cleanup goroutine.
func (s *InMemorySessionStore) Stop() {
	close(s.stopChan)
}

func (s *InMemorySessionStore) cleanup() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for key, entry := range s.sessions {
				if now.After(entry.expiresAt) {
					delete(s.sessions, key)
				}
			}
			s.mu.Unlock()
		case <-s.stopChan:
			return
		}
	}
}
