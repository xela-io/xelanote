# Security Audit Report

Datum: 2026-02-16  
Projekt: xelanote  
Scope: Backend, Frontend, Infrastruktur, CI/CD

## Architekturübersicht (aus Code abgeleitet)
1. Backend: Go/Chi API mit Middleware-Stack (`Logger`, `Recoverer`, `RequestID`, `Compress`, `CORS`, Security Headers) in `backend/internal/api/routes.go:13`.
2. AuthN/AuthZ: JWT Access + Refresh, Cookies + CSRF, Admin-Middleware (`backend/internal/api/middleware.go:164`, `backend/internal/api/cookies.go:12`, `backend/internal/api/csrf.go:126`).
3. Storage: SQLite (inkl. Refresh-Token-Hashing, Sharing-Tabellen, Migrations) in `backend/internal/db/*`.
4. Frontend: SvelteKit SPA + Electron/Tauri Clients, zentrale API-Client-Logik in `frontend/src/lib/api/client.ts`.
5. Infrastruktur/Deployment: Docker + Forgejo/GitHub Workflows (`Dockerfile`, `.forgejo/workflows/*`, `.github/workflows/*`).

## Angriffsoberflächen
1. HTTP-API unter `/api/*` (Auth, Notes, Sharing, Uploads, Admin).
2. WebSocket `/api/ws` (`backend/internal/api/websocket.go`).
3. File Upload/Serve `/api/uploads/{user_id}/{filename}` (`backend/internal/api/uploads.go`).
4. Import/Export (`backend/internal/api/import.go`, `backend/internal/api/export.go`).
5. CI/CD Supply Chain (GitHub/Forgejo Actions).

## Threat Model (kompakt)
1. Assets: Access/Refresh-Tokens, Upload-Dateien, Notizinhalte, Admin-Funktionen, Secrets (JWT, CI).
2. Akteure: Anonymer User, authentifizierter User, Admin, Insider, MITM, Supply-Chain-Angreifer.
3. Relevante STRIDE-Risiken: Spoofing (Desktop-Header), Tampering (CI Action Tags), Information Disclosure (Token-Exposure), Elevation via session theft.

## Findings (nach Schweregrad)

### SEC-001
- Titel: Refresh-/Access-Tokens werden trotz Cookie-Auth in Response-Body und JS-State exponiert
- Severity: High (Token-Diebstahl bei XSS deutlich einfacher; Refresh-Token wieder extrahierbar)
- Betroffene Komponenten: `backend/internal/api/auth_types.go:38`, `backend/internal/api/auth_login.go:102`, `backend/internal/api/auth_register.go:115`, `backend/internal/api/auth_tokens.go:57`, `backend/internal/api/fido2.go:255`, `frontend/src/lib/stores/auth.svelte.ts:282`, `frontend/src/lib/api/client.ts:126`
- Beleg: Auth/Refresh Responses enthalten `access_token` + `refresh_token`; Frontend speichert beide im Memory-State.
- Exploit-Szenario:
1. Angreifer erreicht DOM-XSS im Frontend.
2. Liest `authState.refreshToken` bzw. fängt `/auth/refresh` JSON-Antwort ab.
3. Nutzt Token extern für Session-Übernahme.
- Impact: Session takeover, langlebiger Zugriff (Refresh-Token).
- Likelihood: Mittel bis hoch (XSS ist realistische Web-App-Klasse; Defense-in-depth wird unterlaufen).
- Root Cause: Hybrid-Design (Cookie + Body-Tokens) widerspricht eigener Cookie-only-Sicherheitsannahme.
- Fix: Web-Flow auf striktes Cookie-Session-Modell umstellen; Refresh-Token niemals an JS ausliefern.
- Patch (Beispiel):
```diff
diff --git a/backend/internal/api/auth_types.go b/backend/internal/api/auth_types.go
@@
 type AuthResponse struct {
-  AccessToken  string `json:"access_token,omitempty"`
-  RefreshToken string `json:"refresh_token,omitempty"`
+  // Tokens only via HttpOnly cookies (web).
   User UserResponse `json:"user,omitempty"`
 }
@@
 type TokenResponse struct {
-  AccessToken  string `json:"access_token"`
-  RefreshToken string `json:"refresh_token"`
+  Ok bool `json:"ok"`
 }

diff --git a/backend/internal/api/auth_tokens.go b/backend/internal/api/auth_tokens.go
@@
- respondJSON(w, http.StatusOK, TokenResponse{AccessToken: newAccessToken, RefreshToken: newRefreshToken})
+ respondJSON(w, http.StatusOK, TokenResponse{Ok: true})
```
- Tests:
1. API-Test: `/api/auth/login` enthält keine `refresh_token`-Property im JSON.
2. API-Test: `/api/auth/refresh` gibt nur `{ok:true}` zurück.
3. Frontend-Test: Auth-State enthält keinen Refresh-Token mehr.

### SEC-002
- Titel: CAPTCHA-Bypass über `X-Client-Type: desktop` in Reverse-Proxy-Topologien
- Severity: High (Brute-force/Account-Angriffe auf Login/Register ohne CAPTCHA)
- Betroffene Komponenten: `backend/internal/api/auth_helpers.go:28`, `backend/internal/api/auth_login.go:32`, `backend/internal/api/auth_register.go:44`, `docs/deployment.md:137`
- Beleg: Desktop-Erkennung basiert auf Header + `RemoteAddr` localhost; bei lokalem Proxy kann externer Traffic als localhost erscheinen.
- Exploit-Szenario:
1. Deployment hinter lokalem Reverse Proxy.
2. Angreifer sendet `X-Client-Type: desktop`.
3. Login/Register ohne CAPTCHA-Token wird akzeptiert.
- Impact: Erhöhte Erfolgswahrscheinlichkeit für Credential Stuffing/Enumeration.
- Likelihood: Mittel (abhängig von Proxy-Topologie).
- Root Cause: Vertrauensentscheidung auf leicht manipulierbaren Request-Merkmalen.
- Fix: Kein CAPTCHA-Bypass per Header; Desktop muss ebenfalls CAPTCHA liefern oder separaten lokal gebundenen Auth-Kanal nutzen.
- Patch (Beispiel):
```diff
diff --git a/backend/internal/api/auth_login.go b/backend/internal/api/auth_login.go
@@
- } else if !isDesktopClient(r) {
-   if err := s.turnstileService.Verify(r.Context(), "", clientIP); err != nil { ... }
+ } else {
+   if err := s.turnstileService.Verify(r.Context(), "", clientIP); err != nil { ... }
 }

diff --git a/backend/internal/api/auth_register.go b/backend/internal/api/auth_register.go
@@
- } else if !isDesktopClient(r) {
-   if err := s.turnstileService.Verify(r.Context(), "", clientIP); err != nil { ... }
+ } else {
+   if err := s.turnstileService.Verify(r.Context(), "", clientIP); err != nil { ... }
 }
```
- Tests:
1. Integrationstest: `X-Client-Type: desktop` ohne CAPTCHA => `400 captcha token required`.
2. Regression: gültiger CAPTCHA-Token weiterhin erfolgreich.

### SEC-003
- Titel: Signatur-Orakel für fremde Upload-Pfade über Recipe-Image-URLs
- Severity: Medium
- Betroffene Komponenten: `backend/internal/service/recipes_images.go:19`, `backend/internal/api/recipes_images_signing.go:27`, `backend/internal/api/recipes_handlers.go:60`, `backend/internal/api/uploads.go:167`
- Beleg: `AddRecipeImage` validiert nur Prefix `/api/uploads/`; Signierung erfolgt später ohne Ownership-Bindung.
- Exploit-Szenario:
1. User B kennt URL von User A-Datei (`/api/uploads/A/filename`).
2. User B speichert diese URL als Recipe-Image.
3. `GET /api/recipes/{id}` liefert signierte URL für A-Datei.
4. Dateiabruf via Signatur ohne A-Auth möglich.
- Impact: Unautorisierter Dateiabruf (wenn Dateiname bekannt).
- Likelihood: Mittel (Dateiname muss bekannt werden).
- Root Cause: Fehlende Owner-Konsistenzprüfung beim Speichern/Signieren von Image-URLs.
- Fix: Beim Add nur Upload-URLs des aufrufenden Users akzeptieren (oder explizite ACL-Prüfung).
- Patch (Beispiel):
```diff
diff --git a/backend/internal/service/recipes_images.go b/backend/internal/service/recipes_images.go
@@
- if !strings.HasPrefix(imageURL, "/api/uploads/") { return nil, ErrInvalidImageURL }
+ ownerID, filename, err := parseUploadURL(imageURL)
+ if err != nil || ownerID != callerUserID || filename == "" {
+   return nil, ErrInvalidImageURL
+ }
```
- Tests:
1. `AddRecipeImage` mit `/api/uploads/{andererUser}/...` => `400 invalid image_url`.
2. Positivtest: eigener Upload-URL bleibt erlaubt.

### SEC-004
- Titel: CI-Workflows nicht auf Commit-SHAs gepinnt (GitHub Actions Supply Chain)
- Severity: Medium
- Betroffene Komponenten: `.github/workflows/ci.yml:15`, `.github/workflows/quality.yml:14`, `.github/workflows/security.yml:19`, `scripts/check-action-pinning.sh:8`
- Beleg: Nutzung von `@v4/@v5` Tags; lokale Policy blockiert nur `main/master`.
- Exploit-Szenario:
1. Upstream Action-Tag wird kompromittiert/retagged.
2. CI führt manipulierten Action-Code aus.
3. Secrets/Artefakte können kompromittiert werden.
- Impact: CI-Secret-Abfluss, Build-Manipulation.
- Likelihood: Mittel.
- Root Cause: Unvollständige Pinning-Policy.
- Fix: Alle `uses:` auf commit SHA pinnen; Policy entsprechend härten.
- Patch (Beispiel):
```diff
diff --git a/scripts/check-action-pinning.sh b/scripts/check-action-pinning.sh
@@
- if [[ "$line" =~ uses:[[:space:]]*[^[:space:]]+@(main|master)$ ]]; then
+ if [[ "$line" =~ uses:[[:space:]]*[^[:space:]]+@[^[:space:]]+$ ]] && \
+    [[ ! "$line" =~ @([0-9a-f]{40}|https://) ]]; then
     bad_refs=1
   fi
```
- Tests:
1. Workflow mit `@v4` muss fehlschlagen.
2. Workflow mit 40-hex SHA muss bestehen.

### SEC-005
- Titel: `.env` mit JWT_SECRET im Repository
- Severity: Low (aktuell als Dev markiert, aber hohes Missbrauchspotenzial bei Copy/Paste)
- Betroffene Komponenten: `.env:5`, `.gitignore:2`
- Beleg: Secret-ähnlicher Wert versioniert trotz Ignore-Regel.
- Exploit-Szenario:
1. Operator übernimmt `.env` unverändert in Staging/Prod.
2. Vorhersagbares Secret kompromittiert JWT-Vertrauen.
- Impact: Vollständige Token-Fälschung bei Fehlkonfiguration.
- Likelihood: Niedrig bis mittel (operativer Fehler).
- Root Cause: Fehlende Trennung zwischen Beispiel-Konfig und realem Secret-File.
- Fix: `.env` aus Git entfernen, `/.env.example` ohne echte Werte bereitstellen, CI-Secret-Scanner aktivieren.
- Patch (Beispiel):
```diff
*** Delete File: .env
*** Add File: .env.example
+JWT_SECRET=replace-me-with-openssl-rand-hex-32
```
- Tests:
1. CI-Check: kein tracked `.env`.
2. Secret-Scan in PR-Workflow.

### SEC-006 (Hypothese)
- Titel: CSRF-Schutzlücke auf `/auth/refresh` und `/auth/logout` (öffentliche Routen)
- Severity: Low (durch `SameSite=Strict` stark mitigiert; relevant bei same-site Angreifer/Subdomain-Kompromiss)
- Betroffene Komponenten: `backend/internal/api/routes.go:46`, `backend/internal/api/routes.go:47`, `backend/internal/api/csrf.go:127`
- Beleg: CSRF-Middleware läuft nur in `registerProtectedRoutes`.
- Verifikationsschritte:
1. Testumgebung mit kompromittierter Same-Site-Origin aufsetzen.
2. Cross-origin POST gegen `/api/auth/logout`/`/api/auth/refresh` ausführen.
3. Prüfen, ob Sessionzustand änderbar ist ohne CSRF-Header.
- Fix: Entweder Endpunkte in CSRF-geschützte Gruppe verschieben oder Handler-seitig CSRF bei Cookie-Nutzung erzwingen.

## Fix-First-Plan

### Top 5 Quick Wins (<= 1 Tag)
1. CAPTCHA-Bypass entfernen (SEC-002).
2. `.env` aus Repo entfernen und `.env.example` einführen (SEC-005).
3. CI Action Pinning Policy verschärfen (SEC-004).
4. Recipe-Image-URL Owner-Check ergänzen (SEC-003).
5. CSRF auf `/auth/logout` mindestens erzwingen (SEC-006).

### Top 5 High-Impact Fixes
1. Token-Body-Ausgabe vollständig entfernen (SEC-001).
2. Frontend auf cookie-only Session umstellen (SEC-001).
3. Upload-Signierung an ACL/Owner binden (SEC-003).
4. Alle GitHub Actions auf SHA pinnen (SEC-004).
5. Security regression tests in CI als Pflicht-Gate.

### Hardening Backlog
1. ASVS L2 Mapping automatisieren (Security CI Job).
2. CSP weiter härten (schrittweise Reduktion `unsafe-inline` wo möglich).
3. Secret Scanning (z. B. gitleaks/trufflehog) in PR.
4. Security unit/integration tests für Auth/Captcha/Sharing erweitern.
5. Regelmäßige Threat-Model-Review pro Release.

### Risk Acceptance Liste
1. SEC-006 kann temporär akzeptiert werden, wenn `SameSite=Strict` + keine untrusted Subdomains + dokumentierter Restrisiko-Entscheid.
2. Electron-Dev-Sicherheitslockerungen nur akzeptieren, wenn strikt auf Dev begrenzt und nicht releasebar gebaut.

## Executive Summary
1. Auth-Tokens werden im JSON-Body und JS-State exponiert; das ist der größte AppSec-Risikotreiber.
2. CAPTCHA-Bypass-Logik ist in bestimmten Proxy-Topologien praktisch ausnutzbar.
3. Recipe-Image-Flow erlaubt ein Signatur-Orakel für Upload-URLs ohne Owner-Bindung.
4. CI/CD hat vermeidbares Supply-Chain-Risiko durch unpinned GitHub Actions.
5. Ein `.env` mit JWT_SECRET ist versioniert und begünstigt Fehlkonfigurationen.
6. CSRF-Schutz ist nicht konsistent auf allen session-mutierenden Auth-Endpunkten.
7. Positiv: Refresh-Tokens sind gehasht, viele SQL-Queries sind parameterisiert, Upload-Path-Traversal ist sauber mitigiert.
8. Positiv: Security Headers, Rate Limits, Lockout und Trusted-Proxy-Handling sind grundsätzlich gut umgesetzt.
9. Höchste Priorität: SEC-001 und SEC-002 vor nächstem produktiven Release schließen.
10. Release-Blocker: SEC-001, SEC-002.
11. Gesamtbewertung: rot (mehrere realistisch ausnutzbare Schwachstellen mit Account-/Datenauswirkung).
