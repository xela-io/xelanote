package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

const (
	streamStoreMaxEntries     = 100
	streamStoreMaxContentSize = 512 * 1024 // 512 KiB per entry
	streamStoreTTL            = 60 * time.Second
	streamStoreCleanupEvery   = 30 * time.Second
)

type streamContentStore struct {
	mu      sync.Mutex
	entries map[string]*streamContentEntry
}

type streamContentEntry struct {
	userID           int
	noteID           string
	plaintextContent string
	expiresAt        time.Time
}

func newStreamContentStore() *streamContentStore {
	s := &streamContentStore{
		entries: make(map[string]*streamContentEntry),
	}
	go s.cleanupLoop()
	return s
}

func (s *streamContentStore) store(token string, userID int, noteID, content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.entries) >= streamStoreMaxEntries {
		return fmt.Errorf("stream content store full")
	}

	s.entries[token] = &streamContentEntry{
		userID:           userID,
		noteID:           noteID,
		plaintextContent: content,
		expiresAt:        time.Now().Add(streamStoreTTL),
	}
	return nil
}

func (s *streamContentStore) get(token string) (*streamContentEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.entries[token]
	if !ok || time.Now().After(entry.expiresAt) {
		delete(s.entries, token)
		return nil, fmt.Errorf("token not found or expired")
	}
	delete(s.entries, token) // One-time use
	return entry, nil
}

func (s *streamContentStore) cleanupLoop() {
	ticker := time.NewTicker(streamStoreCleanupEvery)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for token, entry := range s.entries {
			if now.After(entry.expiresAt) {
				delete(s.entries, token)
			}
		}
		s.mu.Unlock()
	}
}

func generateStreamToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
