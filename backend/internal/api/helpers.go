package api

import (
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"net/http"
)

// hashIdentifier returns a truncated SHA-256 hash for log correlation
// without exposing PII. 16 hex chars = 8 bytes = sufficient for correlation
// with negligible collision risk at expected log volumes.
func hashIdentifier(identifier string) string {
	h := sha256.Sum256([]byte(identifier))
	return hex.EncodeToString(h[:8])
}

func securityIPAttr(r *http.Request) slog.Attr {
	return slog.String("remote_ip_hash", hashIdentifier(getClientIPSafe(r)))
}
