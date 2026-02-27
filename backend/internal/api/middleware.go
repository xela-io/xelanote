package api

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/xela-io/xelanote/internal/auth"
	"github.com/xela-io/xelanote/internal/service"
)

// Default trusted CIDR ranges for reverse proxies.
// Secure default: trust only loopback unless explicitly configured.
var defaultTrustedCIDRs = []string{
	"127.0.0.1/32", // localhost IPv4
	"::1/128",      // localhost IPv6
}

// trustedProxyNets holds the parsed trusted CIDR networks
var trustedProxyNets []*net.IPNet

// InitTrustedProxies initializes the trusted proxy CIDRs.
// If trustedProxiesEnv is empty, only loopback is trusted.
// The format is comma-separated CIDRs: "10.0.0.0/8,192.168.1.0/24"
func InitTrustedProxies(trustedProxiesEnv string) {
	var cidrs []string

	if trustedProxiesEnv == "" {
		cidrs = defaultTrustedCIDRs
		slog.Info("using default trusted proxies", slog.String("cidrs", fmt.Sprintf("%v", cidrs)))
	} else {
		parts := strings.Split(trustedProxiesEnv, ",")
		for _, part := range parts {
			cidr := strings.TrimSpace(part)
			if cidr != "" {
				cidrs = append(cidrs, cidr)
			}
		}
		slog.Info("using custom trusted proxies", slog.String("cidrs", fmt.Sprintf("%v", cidrs)))
	}

	trustedProxyNets = make([]*net.IPNet, 0, len(cidrs))
	for _, cidr := range cidrs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			slog.Warn("invalid trusted proxy CIDR", slog.String("cidr", cidr), slog.String("error", err.Error()))
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

	// Only trust forwarded headers if request comes from a trusted proxy.
	if isTrustedProxy(remoteIP) {
		xff := r.Header.Get("X-Forwarded-For")
		if xff != "" {
			if clientIP := extractClientIPFromXFF(xff, remoteIP); clientIP != "" {
				return clientIP
			}
		}

		// Check X-Real-IP as fallback
		xri := r.Header.Get("X-Real-IP")
		if xri != "" {
			clientIP := parseValidIP(strings.TrimSpace(xri))
			if clientIP != "" {
				return clientIP
			}
		}
	}

	// Return the direct connection IP if not from trusted proxy
	return remoteIP
}

// extractClientIPFromXFF parses an X-Forwarded-For chain and returns the first
// untrusted hop scanning from right to left.
//
// Example chain:
// client, proxy1, proxy2
//
// We append remoteIP as the right-most hop and then walk right-to-left,
// skipping trusted proxies. The first untrusted IP is treated as the client.
func extractClientIPFromXFF(xff, remoteIP string) string {
	parts := strings.Split(xff, ",")
	hops := make([]string, 0, len(parts)+1)
	validXFFHops := 0

	for _, raw := range parts {
		if ip := parseValidIP(strings.TrimSpace(raw)); ip != "" {
			hops = append(hops, ip)
			validXFFHops++
		}
	}

	// If XFF exists but has no valid IP entries, let caller try X-Real-IP fallback.
	if validXFFHops == 0 {
		return ""
	}

	normalizedRemote := parseValidIP(strings.TrimSpace(remoteIP))
	if normalizedRemote == "" {
		return ""
	}
	hops = append(hops, normalizedRemote)

	for i := len(hops) - 1; i >= 0; i-- {
		if !isTrustedProxy(hops[i]) {
			return hops[i]
		}
	}

	// If all hops are trusted, fall back to remote IP.
	return normalizedRemote
}

// parseValidIP normalizes and validates a candidate IP string.
// Returns empty string for invalid values.
func parseValidIP(candidate string) string {
	if candidate == "" {
		return ""
	}

	if ip := net.ParseIP(candidate); ip != nil {
		return ip.String()
	}

	// Accept host:port inputs and normalize to plain IP.
	host, _, err := net.SplitHostPort(candidate)
	if err != nil {
		return ""
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.String()
	}

	return ""
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
		if parsedToken, ok := parseBearerToken(authHeader); ok {
			tokenString = parsedToken
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

// requestIDLoggerMiddleware enriches the slog logger in the request context
// with the request ID generated by chi's middleware.RequestID.
// Must be placed after middleware.RequestID in the middleware chain.
func requestIDLoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetReqID(r.Context())
		if reqID != "" {
			ctx := context.WithValue(r.Context(), requestLoggerKey, reqID)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		next.ServeHTTP(w, r)
	})
}

const requestLoggerKey contextKey = "requestID"

// panicRecoveryMiddleware recovers from panics, logs them, enqueues an error
// report to Forgejo, and returns a 500 response. Replaces chi's Recoverer.
func (s *Server) panicRecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rv := recover(); rv != nil {
				stack := string(debug.Stack())
				msg := fmt.Sprintf("%v", rv)

				s.logger().Error("panic recovered",
					slog.String("panic", msg),
					slog.String("stack", stack),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
				)

				if s.errorReportService != nil {
					s.errorReportService.EnqueueReport(service.ErrorReport{
						Type:        "automatic",
						ErrorType:   "Panic",
						Message:     msg,
						Stack:       stack,
						Fingerprint: service.ComputeFingerprint("Panic", msg),
						URL:         r.Method + " " + r.URL.Path,
						Component:   "backend",
					})
				}

				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
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
