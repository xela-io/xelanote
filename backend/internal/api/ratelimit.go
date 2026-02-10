package api

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

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

// Allow checks if a request from the given IP is allowed.
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	client, exists := rl.clients[ip]
	if !exists {
		client = &clientLimiter{
			limiter:  rate.NewLimiter(rl.limit, rl.burst),
			lastSeen: time.Now(),
		}
		rl.clients[ip] = client
	} else {
		client.lastSeen = time.Now()
	}

	return client.limiter.Allow()
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

			if !limiter.Allow(ip) {
				w.Header().Set("Retry-After", "60")
				respondError(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
