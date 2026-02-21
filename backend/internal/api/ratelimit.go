package api

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// maxRateLimitClients is the hard cap on tracked clients per limiter to prevent
// memory exhaustion during DDoS with many distinct IPs (F-15).
const maxRateLimitClients = 10000

// RateLimiter provides per-IP rate limiting using token bucket algorithm.
type RateLimiter struct {
	mu              sync.RWMutex
	clients         map[string]*clientLimiter
	limit           rate.Limit
	burst           int
	cleanupInterval time.Duration
	maxAge          time.Duration
	stopChan        chan struct{}
}

type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewRateLimiter creates a new rate limiter with the specified rate and burst.
// The rate is specified as events per duration (e.g., 10 events per 15 minutes).
// The cleanup goroutine removes stale entries to prevent memory leaks.
func NewRateLimiter(events int, per time.Duration, burst int) *RateLimiter {
	rl := &RateLimiter{
		clients:         make(map[string]*clientLimiter),
		limit:           rate.Limit(float64(events) / per.Seconds()),
		burst:           burst,
		cleanupInterval: time.Minute * 5,
		maxAge:          time.Hour,
		stopChan:        make(chan struct{}),
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// Allow checks if a request for the given key is allowed.
// Key can be IP-only or a composite key such as "user:<id>|ip:<ip>".
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	client, exists := rl.clients[key]
	if !exists {
		// F-15: Evict oldest entry if at capacity to prevent memory exhaustion
		if len(rl.clients) >= maxRateLimitClients {
			rl.evictOldest()
		}
		client = &clientLimiter{
			limiter:  rate.NewLimiter(rl.limit, rl.burst),
			lastSeen: time.Now(),
		}
		rl.clients[key] = client
	} else {
		client.lastSeen = time.Now()
	}

	return client.limiter.Allow()
}

// evictOldest removes the least recently seen client entry.
// Must be called while holding rl.mu.
func (rl *RateLimiter) evictOldest() {
	var oldestKey string
	var oldestTime time.Time
	first := true

	for key, client := range rl.clients {
		if first || client.lastSeen.Before(oldestTime) {
			oldestKey = key
			oldestTime = client.lastSeen
			first = false
		}
	}

	if !first {
		delete(rl.clients, oldestKey)
	}
}

// cleanup removes stale entries from the clients map.
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for ip, client := range rl.clients {
				if now.Sub(client.lastSeen) > rl.maxAge {
					delete(rl.clients, ip)
				}
			}
			rl.mu.Unlock()
		case <-rl.stopChan:
			return
		}
	}
}

// Stop stops the cleanup goroutine.
func (rl *RateLimiter) Stop() {
	close(rl.stopChan)
}

// ClientCount returns the number of tracked clients (for testing/monitoring).
func (rl *RateLimiter) ClientCount() int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	return len(rl.clients)
}

// rateLimitMiddleware creates a middleware that rate limits requests using the provided limiter.
func rateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getClientIPSafe(r)
			key := "ip:" + ip
			if userID, ok := getUserID(r); ok {
				// Dual keying limits abuse from shared IPs while preserving per-user controls.
				key = "user:" + strconv.Itoa(userID) + "|ip:" + ip
			}

			if !limiter.Allow(key) {
				w.Header().Set("Retry-After", "60")
				respondError(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
