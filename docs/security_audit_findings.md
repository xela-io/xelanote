# Security Audit Findings

## SEC-001: JWT Access Token in WebSocket-URL leakt ueber Request-Logs
- Severity: High
- Betroffene Komponenten: `backend/internal/api/websocket.go`, `backend/internal/api/api.go`, `frontend/src/lib/stores/websocket.svelte.ts`
- Beleg:
  - Token via Query-Param: `handleWebSocket` liest `r.URL.Query().Get("token")` in `backend/internal/api/websocket.go`
  - Request Logging aktiv: `r.Use(middleware.Logger)` in `backend/internal/api/api.go`
  - Client haengt Token an URL: `new WebSocket(\`${WS_URL}?token=...\`)` in `frontend/src/lib/stores/websocket.svelte.ts`
- Exploit-Szenario:
  1) User verbindet sich per WebSocket.
  2) `middleware.Logger` schreibt Request-URI inkl. `token=...` in App-Logs oder Reverse-Proxy-Logs.
  3) Angreifer mit Log-Zugriff extrahiert Token und verwendet ihn fuer API-Calls.
- Impact: Konto-Uebernahme fuer Token-Lebensdauer, Datenzugriff, Write-Aktionen
- Likelihood: Medium-High
- Root Cause: Auth-Token im Query-String + globales Request-Logging
- Fix: WebSocket-Auth ueber HttpOnly-Cookie (bereits gesetzt) oder sec-websocket-protocol; Query-Token entfernen, optional Logger redaction
- Patch:
```diff
--- a/backend/internal/api/websocket.go
+++ b/backend/internal/api/websocket.go
@@
-    // Get JWT token from query parameter
-    token := r.URL.Query().Get("token")
+    // Prefer HttpOnly cookie to avoid token in URL/logs
+    token := getAccessTokenFromCookie(r)
+    if token == "" {
+        // Backward-compat fallback (consider removing after client update)
+        token = r.URL.Query().Get("token")
+    }
```

```diff
--- a/frontend/src/lib/stores/websocket.svelte.ts
+++ b/frontend/src/lib/stores/websocket.svelte.ts
@@
-    ws = new WebSocket(`${WS_URL}?token=${encodeURIComponent(token)}`);
+    ws = new WebSocket(WS_URL); // rely on HttpOnly cookie
```
- Tests:
  - WebSocket handshake ohne Query-Token, Cookie-Auth aktiv -> Verbindung erfolgreich
  - Server-Logs pruefen: kein `token=` mehr in URL
- Follow-ups:
  - Optional: `middleware.Logger` durch redacting logger ersetzen fuer alle sensiblen Query-Parameter

## SEC-002: Rate-Limit Bypass durch Spoofing von X-Real-IP/X-Forwarded-For
- Severity: Medium
- Betroffene Komponenten: `backend/internal/api/ratelimit.go`
- Beleg: `getClientIP` nimmt `X-Real-IP`/`X-Forwarded-For` ohne Trusted-Proxy-Check in `backend/internal/api/ratelimit.go`
- Exploit-Szenario:
  1) Angreifer sendet viele Login-Requests mit variierendem `X-Real-IP`.
  2) Rate-Limiter behandelt jeden Request als neue IP, Limit greift nicht.
- Impact: Erleichtertes Brute-Forcing von Passwoertern/Backup-Codes
- Likelihood: Medium
- Root Cause: Untrusted Header werden fuer Security-Entscheidung genutzt
- Fix: `getClientIPSafe` aus `backend/internal/api/middleware.go` verwenden (Trusted Proxies)
- Patch:
```diff
--- a/backend/internal/api/ratelimit.go
+++ b/backend/internal/api/ratelimit.go
@@
-    ip := getClientIP(r)
+    ip := getClientIPSafe(r)
```
- Tests:
  - Requests mit gefaelschtem `X-Real-IP` von untrusted RemoteAddr -> gleiche IP im Limiter
  - Trusted proxy gesetzt (`TRUSTED_PROXIES`) -> echte Client-IP aus XFF genutzt
- Follow-ups:
  - Dokumentation: TRUSTED_PROXIES muss gesetzt sein, wenn Reverse Proxy eine Public IP nutzt

## SEC-003: User-Delete loescht falsches Upload-Verzeichnis; Uploads bleiben oeffentlich erreichbar
- Severity: Medium
- Betroffene Komponenten: `backend/internal/service/admin.go`, `backend/internal/api/api.go`, `backend/internal/api/uploads.go`
- Beleg:
  - Pfadfehler: `filepath.Join(..., string(rune(targetUserID)))` in `DeleteUser` und `calculateUserStorageMB` (`backend/internal/service/admin.go`)
  - Public Uploads ohne Auth: Routes `r.Get("/uploads/{user_id}/{filename}", s.serveUploadPublic)` in `backend/internal/api/api.go`
- Exploit-Szenario:
  1) User laedt Bild hoch, URL ist `/api/uploads/<user_id>/<uuid>.png`.
  2) Admin loescht User. Wegen `string(rune(userID))` wird falsches Verzeichnis geloescht.
  3) Bild bleibt im Upload-Ordner und ist weiterhin oeffentlich per URL erreichbar.
- Impact: Privacy Leak (Datenreste), moegliche Loeschung falscher Uploads
- Likelihood: Medium
- Root Cause: Falsche String-Konvertierung von userID fuer Dateipfade
- Fix: `strconv.Itoa(userID)` verwenden; optional Uploads nur authentifiziert ausliefern oder signierte URLs
- Patch:
```diff
--- a/backend/internal/service/admin.go
+++ b/backend/internal/service/admin.go
@@
-    uploadDir := filepath.Join(s.dataDir, "uploads", string(rune(targetUserID)))
+    uploadDir := filepath.Join(s.dataDir, "uploads", strconv.Itoa(targetUserID))
@@
-    uploadDir := filepath.Join(s.dataDir, "uploads", string(rune(userID)))
+    uploadDir := filepath.Join(s.dataDir, "uploads", strconv.Itoa(userID))
```
- Tests:
  - Upload erstellen, User per Admin loeschen -> Upload-Ordner entfernt
  - Sicherstellen, dass `/api/uploads/<user_id>/<file>` 404 liefert
- Follow-ups:
  - Optional: `serveUploadPublic` abschalten oder auf Auth umstellen

## SEC-004: Storage-Limit Setting wird nicht durch Upload-API enforced (DoS/Quota Bypass)
- Severity: Medium
- Betroffene Komponenten: `backend/internal/api/uploads.go`, `backend/internal/service/settings.go`, `backend/internal/db/migrations/018_system_settings.sql`
- Beleg:
  - Setting existiert: `max_storage_mb_per_user` in `backend/internal/service/settings.go` / Migration
  - Upload-Endpoint enthaelt keine Quota-Pruefung in `backend/internal/api/uploads.go`
- Exploit-Szenario:
  1) Admin setzt `max_storage_mb_per_user` auf z.B. 10 MB.
  2) User laedt viele 10 MB Dateien hoch -> alle 200 OK.
  3) Disk wird voll, Service beeintraechtigt/ausfall.
- Impact: Denial of Service, Disk Exhaustion
- Likelihood: Medium
- Root Cause: Setting existiert, aber Upload-Handler ignoriert es
- Fix: Vor dem Speichern Quota aus Settings pruefen und aktuelle Nutzung ermitteln
- Patch:
```diff
--- a/backend/internal/api/uploads.go
+++ b/backend/internal/api/uploads.go
@@
+    maxStorageMB, err := s.settingsService.GetMaxStorageMBPerUser()
+    if err != nil {
+        respondError(w, http.StatusInternalServerError, "failed to check storage limit")
+        return
+    }
+    if maxStorageMB > 0 {
+        usedMB := s.adminService.GetUserStorageMB(userID)
+        if header.Size > 0 {
+            estimatedMB := usedMB + (float64(header.Size) / (1024 * 1024))
+            if estimatedMB > float64(maxStorageMB) {
+                respondError(w, http.StatusForbidden, "storage limit exceeded")
+                return
+            }
+        }
+    }
```

```diff
--- a/backend/internal/service/admin.go
+++ b/backend/internal/service/admin.go
@@
+// GetUserStorageMB exposes per-user storage usage for enforcement
+func (s *AdminService) GetUserStorageMB(userID int) float64 {
+    return s.calculateUserStorageMB(userID)
+}
```
- Tests:
  - Setting `max_storage_mb_per_user=1`, Upload 2MB -> 403
  - Uploads bis Limit -> 200
- Follow-ups:
  - Periodische Quota-Checks fuer bestehende Daten (Admin-Bericht)

## SEC-005: TOCTOU Race Condition in Upload Quota Enforcement (FIXED)
- Severity: HIGH
- Status: **FIXED** (2026-01-23)
- Betroffene Komponenten: `backend/internal/api/uploads.go`
- Beleg:
  - Original Implementation: Quota-Check erfolgte NACH Schreiben der Datei auf Disk (Post-write check only)
  - Race Condition Window: Zwischen File-Write und Quota-Check konnten concurrent Uploads das Limit umgehen
- Exploit-Szenario:
  1) User hat 9 MB von 10 MB Quota genutzt.
  2) User startet 5 parallele Uploads à 1 MB gleichzeitig.
  3) Alle 5 Dateien werden auf Disk geschrieben (Thread 1-5 parallel).
  4) Post-write Quota-Check laueft fuer jeden Upload einzeln, aber Dateien sind bereits gespeichert.
  5) Resultat: User hat nun 14 MB gespeichert trotz 10 MB Limit.
- Impact: Storage Quota Bypass, potentieller Denial of Service durch Disk Exhaustion
- Likelihood: Medium-High (trivial auszunutzen mit concurrent requests)
- Root Cause: Time-of-Check Time-of-Use (TOCTOU) vulnerability
  - Check (Quota-Validierung) erfolgte NACH Use (File-Write)
  - Keine atomare Operation zwischen Check und Write
- Fix (2026-01-23):
  - **Pre-write Quota Check**: Validierung mit `header.Size` VOR File-Write (Zeilen 84-100)
  - Blockiert Uploads die Quota ueberschreiten wuerden BEVOR Daten auf Disk geschrieben werden
  - Post-write Check bleibt als Fallback fuer chunked uploads (wo `header.Size` = 0 sein kann)
- Patch:
```diff
--- a/backend/internal/api/uploads.go
+++ b/backend/internal/api/uploads.go
@@
+    // Pre-write quota check using header size to prevent TOCTOU race condition
+    // This blocks concurrent uploads that would exceed quota before writing to disk
+    maxStorageMB, err := s.settingsService.GetMaxStorageMBPerUser()
+    if err != nil {
+        respondError(w, http.StatusInternalServerError, "failed to check storage limit")
+        return
+    }
+
+    if maxStorageMB > 0 && header.Size > 0 {
+        currentUsageMB := s.adminService.GetUserStorageMB(userID)
+        fileSizeMB := float64(header.Size) / (1024 * 1024)
+
+        if currentUsageMB+fileSizeMB > float64(maxStorageMB) {
+            respondError(w, http.StatusForbidden, "storage limit would be exceeded")
+            return
+        }
+    }
+
     // Save file
     filePath := filepath.Join(userUploadDir, filename)
```
- Tests:
  - Sequential upload bis Limit -> Letzte Upload rejected mit 403
  - Concurrent uploads die Quota ueberschreiten wuerden -> Fruehzeitig geblockt
  - Chunked upload (header.Size=0) -> Post-write check greift als Fallback
- Security Impact:
  - **HIGH**: Verhindert Storage Quota Bypass komplett
  - **Mitigation**: Concurrent uploads koennen Quota nicht mehr umgehen
  - **Defense in Depth**: Dual-check approach (pre-write + post-write) für maximale Sicherheit

---

## Security Audit 2026-02-03 (Production Hardening)

**Scope:** Full-stack audit + Hetzner Production Server

### Findings - All FIXED

#### SEC-2026-01: Missing Rate Limiting on 2FA Endpoints
- **Severity:** Medium
- **Status:** FIXED (commit 8fb52f8)
- **Issue:** `/2fa/setup`, `/2fa/` (DELETE), `/2fa/backup-codes/regenerate` had no rate limiting
- **Fix:** Added `tfaVerifyLimiter` (5/15min) and `backupCodeLimiter` (3/15min)
- **File:** `backend/internal/api/api.go`

#### SEC-2026-02: Missing Input Length Validation
- **Severity:** Low
- **Status:** FIXED (commit f05ed71)
- **Issue:** No max length validation for username, email, password, note title/content
- **Fix:** Added limits: Username 100, Email 255, Password 128, Title 500, Content 10MB
- **Files:** `backend/internal/service/auth.go`, `backend/internal/api/notes.go`

#### SEC-2026-03: Docker Compose External Port Binding
- **Severity:** Low
- **Status:** FIXED (commit 071db86)
- **Issue:** Port bound to all interfaces (`8081:8080`) instead of localhost
- **Fix:** Changed to `127.0.0.1:8081:8080`, added `cap_drop: ALL`
- **File:** `docker-compose.yml`

#### SEC-2026-04: X11Forwarding Enabled in SSH
- **Severity:** Medium
- **Status:** FIXED (server config)
- **Issue:** X11Forwarding was enabled in SSH config
- **Fix:** Added `/etc/ssh/sshd_config.d/99-hardening.conf` with `X11Forwarding no`

#### SEC-2026-05: Missing HTTP Security Headers in Caddy
- **Severity:** Medium
- **Status:** FIXED (server config)
- **Issue:** HSTS, X-Frame-Options, X-Content-Type-Options missing from reverse proxy
- **Note:** Headers were already set by backend, Caddy now also sets them for defense-in-depth

#### SEC-2026-06: Docker Container Missing cap_drop
- **Severity:** Low
- **Status:** FIXED (server config)
- **Issue:** Container ran without `--cap-drop=ALL`
- **Fix:** Added `--cap-drop=ALL` to production docker run command

#### SEC-2026-07: Unnecessary Firewall Ports Open
- **Severity:** Low
- **Status:** FIXED (server config)
- **Issue:** Ports 6090 (YaCy) and 51822 (WireGuard) were open but unused
- **Fix:** Removed UFW rules for these ports

### Verified Security Controls (PASS)

| Control | Status |
|---------|--------|
| JWT Algorithm Validation (HS256) | ✅ |
| Refresh Token SHA256 Hashing | ✅ |
| CSRF Double-Submit Pattern | ✅ |
| HttpOnly + SameSite=Strict Cookies | ✅ |
| bcrypt Cost Factor 12 | ✅ |
| Account Lockout (5 attempts, exponential backoff) | ✅ |
| Path Traversal Prevention | ✅ |
| MIME Type Validation | ✅ |
| TOTP Replay Protection | ✅ |
| Constant-Time Backup Code Comparison | ✅ |
| JWT_SECRET 64+ Character Validation | ✅ |
| DOMPurify XSS Prevention | ✅ |
| Parameterized SQL Queries | ✅ |
| Non-Root Container User | ✅ |
| Fail2ban (24h ban, 3 attempts) | ✅ |
| UFW Firewall (deny incoming default) | ✅ |
| Auto Security Updates | ✅ |

### Accepted Risks

1. **CSP `unsafe-inline`** - Required by SvelteKit, mitigated by DOMPurify
2. **30-day Refresh Token Lifetime** - Mitigated by rotation, HttpOnly, SameSite=Strict

### Production Server Hardening Summary

```
SSH:       Port <SSH_PORT>, Key-only, Root disabled, X11Forwarding disabled
Firewall:  80, 443, <SSH_PORT>, <FORGEJO_SSH_PORT> (Forgejo SSH)
Fail2ban:  24h ban after 3 failed attempts (21 IPs banned total)
Docker:    512MB RAM, 1 CPU, 200 PIDs, cap_drop=ALL, no-new-privileges
Secrets:   ~/.xelanote.env (chmod 600)
Backups:   Daily at 03:00 UTC
SSL:       Cloudflare Full (strict) + Caddy auto-TLS
```
