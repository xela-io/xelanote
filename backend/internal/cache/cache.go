package cache

import (
	"sync"
	"time"
)

// Cache is a thread-safe in-memory cache with TTL
type Cache struct {
	items      sync.Map
	ttl        time.Duration
	stopChan   chan struct{}
	stopOnce   sync.Once
}

// cacheItem represents a cached value with expiration
type cacheItem struct {
	value      interface{}
	expiration time.Time
}

// New creates a new cache with the specified TTL
func New(ttl time.Duration) *Cache {
	c := &Cache{
		ttl:      ttl,
		stopChan: make(chan struct{}),
	}
	go c.cleanupExpired()
	return c
}

// Get retrieves a value from the cache if it exists and hasn't expired
func (c *Cache) Get(key string) (interface{}, bool) {
	if item, ok := c.items.Load(key); ok {
		cached := item.(cacheItem)
		if time.Now().Before(cached.expiration) {
			return cached.value, true
		}
		c.items.Delete(key) // Expired - remove it
	}
	return nil, false
}

// Set stores a value in the cache with TTL
func (c *Cache) Set(key string, value interface{}) {
	c.items.Store(key, cacheItem{
		value:      value,
		expiration: time.Now().Add(c.ttl),
	})
}

// Delete removes a value from the cache
func (c *Cache) Delete(key string) {
	c.items.Delete(key)
}

// DeleteByPrefix removes all cache entries whose keys start with the given prefix
func (c *Cache) DeleteByPrefix(prefix string) {
	c.items.Range(func(key, _ interface{}) bool {
		if keyStr, ok := key.(string); ok && len(keyStr) >= len(prefix) && keyStr[:len(prefix)] == prefix {
			c.items.Delete(key)
		}
		return true
	})
}

// Close stops the cleanup goroutine.
func (c *Cache) Close() {
	c.stopOnce.Do(func() {
		close(c.stopChan)
	})
}

// cleanupExpired runs periodically to remove expired entries
func (c *Cache) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			now := time.Now()
			c.items.Range(func(key, value interface{}) bool {
				if now.After(value.(cacheItem).expiration) {
					c.items.Delete(key)
				}
				return true
			})
		case <-c.stopChan:
			return
		}
	}
}
