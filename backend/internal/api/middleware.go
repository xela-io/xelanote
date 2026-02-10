package api

import (
	"context"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/xela-io/xelanote/internal/auth"
)

// Default trusted CIDR ranges for reverse proxies (private networks)
var defaultTrustedCIDRs = []string{
	"127.0.0.1/32",   // localhost IPv4
	"::1/128",        // localhost IPv6
	"10.0.0.0/8",     // Private class A
	"172.16.0.0/12",  // Private class B
	"192.168.0.0/16", // Private class C
}

// trustedProxyNets holds the parsed trusted CIDR networks
var trustedProxyNets []*net.IPNet

// InitTrustedProxies initializes the trusted proxy CIDRs.
// If trustedProxiesEnv is empty, default private network ranges are used.
// The format is comma-separated CIDRs: "10.0.0.0/8,192.168.1.0/24"
func InitTrustedProxies(trustedProxiesEnv string) {
	var cidrs []string

	if trustedProxiesEnv == "" {
		cidrs = defaultTrustedCIDRs
		log.Printf("Using default trusted proxies: %v", cidrs)
	} else {
		parts := strings.Split(trustedProxiesEnv, ",")
		for _, part := range parts {
			cidr := strings.TrimSpace(part)
			if cidr != "" {
				cidrs = append(cidrs, cidr)
			}
		}
		log.Printf("Using custom trusted proxies: %v", cidrs)
	}

	trustedProxyNets = make([]*net.IPNet, 0, len(cidrs))
	for _, cidr := range cidrs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			log.Printf("Warning: invalid trusted proxy CIDR '%s': %v", cidr, err)
			continue
		}
		trustedProxyNets = append(trustedProxyNets, ipNet)
	}
}

// isTrustedProxy checks if the given IP address is from a trusted proxy.
func isTrustedProxy(ip string) bool {
	// Parse the IP address
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// Check if IP is in any trusted network
	for _, ipNet := range trustedProxyNets {
		if ipNet.Contains(parsedIP) {
			return true
		}
	}

	return false
}

// getClientIPSafe extracts the real client IP from the request.
// It only trusts X-Forwarded-For or X-Real-IP headers if the immediate
// connection comes from a trusted proxy.
func getClientIPSafe(r *http.Request) string {
	// Get the immediate connection IP (without port)
	remoteAddr := r.RemoteAddr
	remoteIP, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		// If SplitHostPort fails, the address might not have a port
		remoteIP = remoteAddr
	}

	// Only trust forwarded headers if request comes from a trusted proxy
	if isTrustedProxy(remoteIP) {
		// Check X-Forwarded-For first (most common)
		xff := r.Header.Get("X-Forwarded-For")
		if xff != "" {
			// X-Forwarded-For can contain multiple IPs: "client, proxy1, proxy2"
			// The leftmost IP is the original client
			parts := strings.Split(xff, ",")
			if len(parts) > 0 {
				clientIP := strings.TrimSpace(parts[0])
				if clientIP != "" {
					return clientIP
				}
			}
		}

		// Check X-Real-IP as fallback
		xri := r.Header.Get("X-Real-IP")
		if xri != "" {
			return strings.TrimSpace(xri)
		}
	}

	// Return the direct connection IP if not from trusted proxy
	return remoteIP
}

// Context key for user ID
type contextKey string

const userIDKey contextKey = "userID"

// authMiddleware validates JWT access tokens and attaches user ID to request context
// This middleware should be applied to all protected routes
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var tokenString string

		// Attempt 1: Authorization Header (preferred)
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString = parts[1]
			}
		}

		// Attempt 2: access_token Cookie (fallback)
		if tokenString == "" {
			tokenString = getAccessTokenFromCookie(r)
		}

		// No auth found
		if tokenString == "" {
			respondError(w, http.StatusUnauthorized, "missing authorization")
			return
		}

		// Validate JWT access token
		claims, err := auth.ValidateAccessToken(tokenString, s.jwtSecret)
		if err != nil {
			respondError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		// Attach user ID to request context
		ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)

		// Call next handler with enriched context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// getUserID extracts the authenticated user ID from request context
// Returns (userID, true) if authenticated, (0, false) if not authenticated
func getUserID(r *http.Request) (int, bool) {
	userID, ok := r.Context().Value(userIDKey).(int)
	return userID, ok
}

// adminMiddleware checks if the authenticated user is an admin
// Must be used after authMiddleware
func (s *Server) adminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserID(r)
		if !ok {
			respondError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		isAdmin, err := s.adminService.IsUserAdmin(userID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to check admin status")
			return
		}

		if !isAdmin {
			respondError(w, http.StatusForbidden, "admin access required")
			return
		}

		next.ServeHTTP(w, r)
	})
}
