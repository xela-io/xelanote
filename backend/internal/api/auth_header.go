package api

import "strings"

// parseBearerToken parses Authorization header values in the form:
// "Bearer <token>" (case-insensitive scheme, extra whitespace tolerated).
func parseBearerToken(authHeader string) (string, bool) {
	parts := strings.Fields(strings.TrimSpace(authHeader))
	if len(parts) != 2 {
		return "", false
	}
	if !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	if parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func hasBearerAuthorizationHeader(authHeader string) bool {
	_, ok := parseBearerToken(authHeader)
	return ok
}
