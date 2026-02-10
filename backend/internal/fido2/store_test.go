package fido2

import (
	"testing"
	"time"
)

func TestInMemorySessionStore_StoreAndGet(t *testing.T) {
	store := NewInMemorySessionStore()
	defer store.Stop()

	// Store a value
	err := store.Store("key1", "value1", 5*time.Minute)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Get the value (one-time)
	data, err := store.Get("key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if data != "value1" {
		t.Errorf("Expected 'value1', got '%v'", data)
	}

	// Second Get should fail (one-time use)
	_, err = store.Get("key1")
	if err != ErrSessionNotFound {
		t.Errorf("Expected ErrSessionNotFound on second Get, got %v", err)
	}
}

func TestInMemorySessionStore_NotFound(t *testing.T) {
	store := NewInMemorySessionStore()
	defer store.Stop()

	_, err := store.Get("nonexistent")
	if err != ErrSessionNotFound {
		t.Errorf("Expected ErrSessionNotFound, got %v", err)
	}
}

func TestInMemorySessionStore_TTLExpiry(t *testing.T) {
	store := NewInMemorySessionStore()
	defer store.Stop()

	// Store with very short TTL
	err := store.Store("expiring", "data", 10*time.Millisecond)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Force expiry without sleeping
	store.mu.Lock()
	if entry, ok := store.sessions["expiring"]; ok {
		entry.expiresAt = time.Now().Add(-1 * time.Second)
	}
	store.mu.Unlock()

	// Should be expired
	_, err = store.Get("expiring")
	if err != ErrSessionNotFound {
		t.Errorf("Expected ErrSessionNotFound for expired entry, got %v", err)
	}
}

func TestInMemorySessionStore_Delete(t *testing.T) {
	store := NewInMemorySessionStore()
	defer store.Stop()

	store.Store("deleteme", "data", 5*time.Minute)

	err := store.Delete("deleteme")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = store.Get("deleteme")
	if err != ErrSessionNotFound {
		t.Errorf("Expected ErrSessionNotFound after delete, got %v", err)
	}
}

func TestInMemorySessionStore_Overwrite(t *testing.T) {
	store := NewInMemorySessionStore()
	defer store.Stop()

	store.Store("key", "first", 5*time.Minute)
	store.Store("key", "second", 5*time.Minute)

	data, err := store.Get("key")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if data != "second" {
		t.Errorf("Expected 'second' (overwritten), got '%v'", data)
	}
}

func TestInMemorySessionStore_StructData(t *testing.T) {
	store := NewInMemorySessionStore()
	defer store.Stop()

	type PendingLogin struct {
		UserID   int
		Username string
	}

	pending := &PendingLogin{UserID: 42, Username: "testuser"}
	store.Store("pending:token123", pending, 5*time.Minute)

	data, err := store.Get("pending:token123")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	retrieved, ok := data.(*PendingLogin)
	if !ok {
		t.Fatal("Type assertion failed")
	}
	if retrieved.UserID != 42 || retrieved.Username != "testuser" {
		t.Errorf("Expected {42, testuser}, got {%d, %s}", retrieved.UserID, retrieved.Username)
	}
}
