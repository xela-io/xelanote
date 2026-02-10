package cache

import (
	"strings"
	"sync"
	"time"
)

// Cache is a thread-safe in-memory cache with TTL support
type Cache struct {
	items sync.Map
	ttl   time.Duration
}

type cacheItem struct {
	value      interface{}
	expiration time.Time
}

// NewCache creates a new cache with the specified TTL
func NewCache(ttl time.Duration) *Cache {
	c := &Cache{ttl: ttl}
	go c.cleanupExpired() // Start cleanup goroutine
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
		if strings.HasPrefix(key.(string), prefix) {
			c.items.Delete(key)
		}
		return true
	})
}

// cleanupExpired runs periodically to remove expired entries
func (c *Cache) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		c.items.Range(func(key, value interface{}) bool {
			if now.After(value.(cacheItem).expiration) {
				c.items.Delete(key)
			}
			return true
		})
	}
}
