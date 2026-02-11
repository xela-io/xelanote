package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiter_AllowBurst(t *testing.T) {
	// Create a limiter that allows 5 requests per hour with burst of 5
	rl := NewRateLimiter(5, time.Hour, 5)
	defer rl.Stop()

	ip := "192.168.1.1"

	// All burst requests should be allowed
	for i := 0; i < 5; i++ {
		if !rl.Allow(ip) {
			t.Errorf("Request %d should be allowed (within burst)", i+1)
		}
	}

	// Next request should be blocked
	if rl.Allow(ip) {
		t.Error("Request 6 should be blocked (burst exceeded)")
	}
}

func TestRateLimiter_DifferentIPs(t *testing.T) {
	// Create a limiter that allows 2 requests per hour with burst of 2
	rl := NewRateLimiter(2, time.Hour, 2)
	defer rl.Stop()

	// Each IP should have its own bucket
	ips := []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"}

	for _, ip := range ips {
		// Each IP should be allowed burst requests
		for i := 0; i < 2; i++ {
			if !rl.Allow(ip) {
				t.Errorf("Request from %s should be allowed", ip)
			}
		}
	}

	// Verify client count
	if count := rl.ClientCount(); count != 3 {
		t.Errorf("Expected 3 clients, got %d", count)
	}
}

func TestGetClientIP(t *testing.T) {
	// Initialize trusted proxies for this test
	InitTrustedProxies("")

	// Security note: X-Forwarded-For is checked first, then X-Real-IP.
	// For X-Forwarded-For, we take the FIRST IP (original client).
	// Headers are only trusted if RemoteAddr is from a trusted proxy.
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expected   string
	}{
		{
			name:       "RemoteAddr only",
			headers:    nil,
			remoteAddr: "192.168.1.1:12345",
			expected:   "192.168.1.1",
		},
		{
			name:       "X-Forwarded-For single IP from trusted proxy",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.195"},
			remoteAddr: "10.0.0.1:12345", // 10.x.x.x is trusted
			expected:   "203.0.113.195",
		},
		{
			name:       "X-Forwarded-For multiple IPs takes FIRST (original client)",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.195, 70.41.3.18, 150.172.238.178"},
			remoteAddr: "10.0.0.1:12345",
			expected:   "203.0.113.195", // First IP = original client
		},
		{
			name:       "X-Real-IP from trusted proxy",
			headers:    map[string]string{"X-Real-IP": "198.51.100.178"},
			remoteAddr: "10.0.0.1:12345",
			expected:   "198.51.100.178",
		},
		{
			name: "X-Forwarded-For takes precedence over X-Real-IP",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.195",
				"X-Real-IP":       "198.51.100.178",
			},
			remoteAddr: "10.0.0.1:12345",
			expected:   "203.0.113.195", // X-Forwarded-For is checked first
		},
		{
			name:       "X-Forwarded-For with spaces takes FIRST trimmed",
			headers:    map[string]string{"X-Forwarded-For": "  203.0.113.195  ,  70.41.3.18  "},
			remoteAddr: "10.0.0.1:12345",
			expected:   "203.0.113.195", // First IP, trimmed
		},
		{
			name:       "RemoteAddr without port",
			headers:    nil,
			remoteAddr: "192.168.1.1",
			expected:   "192.168.1.1",
		},
		{
			name:       "Headers ignored from untrusted remote",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.195"},
			remoteAddr: "8.8.8.8:12345", // Public IP, not trusted
			expected:   "8.8.8.8",
		},
		{
			name:       "Invalid X-Forwarded-For falls back to remote addr",
			headers:    map[string]string{"X-Forwarded-For": "not-an-ip"},
			remoteAddr: "10.0.0.1:12345",
			expected:   "10.0.0.1",
		},
		{
			name: "Invalid X-Forwarded-For uses valid X-Real-IP fallback",
			headers: map[string]string{
				"X-Forwarded-For": "not-an-ip",
				"X-Real-IP":       "198.51.100.178",
			},
			remoteAddr: "10.0.0.1:12345",
			expected:   "198.51.100.178",
		},
		{
			name:       "Invalid X-Real-IP falls back to remote addr",
			headers:    map[string]string{"X-Real-IP": "not-an-ip"},
			remoteAddr: "10.0.0.1:12345",
			expected:   "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			ip := getClientIPSafe(req)
			if ip != tt.expected {
				t.Errorf("getClientIP() = %s, expected %s", ip, tt.expected)
			}
		})
	}
}

func TestRateLimitMiddleware_AllowsRequest(t *testing.T) {
	rl := NewRateLimiter(10, time.Minute, 10)
	defer rl.Stop()

	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	middleware := rateLimitMiddleware(rl)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	if !called {
		t.Error("Handler should have been called")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestRateLimitMiddleware_BlocksExcessiveRequests(t *testing.T) {
	rl := NewRateLimiter(2, time.Hour, 2)
	defer rl.Stop()

	callCount := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	})

	middleware := rateLimitMiddleware(rl)
	wrappedHandler := middleware(handler)

	ip := "192.168.1.1:12345"

	// First 2 requests should pass
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
		req.RemoteAddr = ip
		rec := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Request %d: Expected status %d, got %d", i+1, http.StatusOK, rec.Code)
		}
	}

	// Third request should be blocked
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	req.RemoteAddr = ip
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status %d, got %d", http.StatusTooManyRequests, rec.Code)
	}

	// Verify Retry-After header
	if rec.Header().Get("Retry-After") == "" {
		t.Error("Expected Retry-After header to be set")
	}

	// Handler should have been called only twice
	if callCount != 2 {
		t.Errorf("Handler should have been called 2 times, was called %d times", callCount)
	}
}
