# Security Audit Plan - Kritisch Revalidiert und Umgesetzt

Stand: 2026-02-12

## 1. Ergebnis der Revalidierung

Alle priorisierten Security-Maßnahmen aus dem Audit wurden technisch umgesetzt und verifiziert.

Es bleiben zwei bewusst dokumentierte Restpunkte:
- `SEC-005`: Actions sind auf feste Versionen gepinnt und mutable Refs werden CI-seitig blockiert; ein kompletter Wechsel auf Commit-SHAs fuer jede Action ist als naechster Hardening-Schritt dokumentiert.
- `SEC-CSP-001`: `style-src` wurde gehaertet; `script-src 'unsafe-inline'` bleibt aktuell fuer den bestehenden Svelte-Bootstrap-Flow erforderlich und ist als Restrisiko markiert.

## 2. Umsetzungsstatus (strukturiert)

| ID | Prioritaet | Status | Umgesetzt in |
|---|---|---|---|
| SEC-001 Trusted Proxy/XFF | P0 | Erledigt | `backend/internal/api/middleware.go`, `backend/cmd/server/main.go`, `backend/internal/api/ratelimit_test.go` |
| SEC-005 CI Supply Chain | P0 | Erledigt (Policy-Baseline) | `.github/workflows/security.yml`, `.github/workflows/quality.yml`, `scripts/check-action-pinning.sh` |
| SEC-006 Logging PII | P1 | Erledigt | `backend/internal/api/helpers.go`, diverse API-Handler, `backend/internal/service/captcha.go` |
| SEC-RT-001 Refresh Token Reuse | P1 | Erledigt | `backend/internal/db/auth.go`, `backend/internal/service/auth.go`, `backend/internal/api/auth_tokens.go`, Migration `044` |
| SEC-003 Electron Hardening | P2 | Erledigt | `frontend/src-electron/main.ts`, `frontend/src-electron/windows/main-window.ts`, `frontend/src-electron/modules/ipc-handlers.ts` |
| SEC-004 Tauri Fallback Hardening | P2 | Erledigt | `frontend/src-tauri/src/keyring.rs` |
| SEC-CSP-001 CSP Hardening | P2 | Erledigt (teilweise Restrisiko) | `backend/internal/api/security.go`, `backend/internal/api/security_test.go`, `frontend/src-tauri/tauri.conf.json` |
| SEC-RL-001 Rate-Limit Keying | P2 | Erledigt | `backend/internal/api/ratelimit.go`, `backend/internal/api/twofa.go` |
| SEC-FTS-001 Search Abuse Guardrails | P2 | Erledigt | `backend/internal/db/search_test.go` |
| SEC-002 Registration Default | P3 | Erledigt | `backend/internal/db/migrations/018_system_settings.sql`, Migration `045`, `backend/internal/db/settings.go`, `backend/internal/service/auth.go`, `backend/internal/api/auth_register.go` |
| SEC-007 Binary Hygiene | P0 | Erledigt | `.gitignore`, `backend/cmd/server/server` (enttracked), `scripts/check-binary-hygiene.sh` |

## 3. Technische Umsetzung pro Bereich

1. Netzwerk- und Request-Vertrauen
- Default Trusted Proxies auf Loopback reduziert.
- XFF-Auswertung auf right-to-left trusted chain umgestellt.
- In Production (`XELANOTE_ENV=production`) ist `TRUSTED_PROXIES` verpflichtend.

2. Session- und Token-Sicherheit
- Refresh-Token-Familienmodell eingefuehrt (`family_id`, `consumed_at`, `replaced_by`, `revoked_at`).
- Reuse Detection implementiert.
- Bei erkannter Reuse wird die gesamte Familie revokiert und als Security-Event geloggt.

3. Logging und Datenschutz
- Roh-IP Logging entfernt und auf gehashte Felder umgestellt (`remote_ip_hash`).
- Security-relevante Logs nutzen zentrale Helper.

4. Desktop-Hardening
- Electron ohne `--no-sandbox`, Renderer mit `sandbox: true`.
- Unsichere Header/CORS-Bypass-Logik im Main-Prozess entfernt.
- IPC-Inputvalidierung fuer URL/Token/KEK-Handover gehaertet.
- Tauri-Fallback-Key nun zufaellig/persistent statt machine-id-deriviert, Dateirechte auf `0600`.

5. CI- und Repo-Hygiene
- Mutable Action-Refs (`@main`, `@master`) via CI-Script blockiert.
- Trivy Action nicht mehr auf `master`.
- Binary-Guard in CI und getracktes Binary aus Git entfernt.

## 4. Verifikation (durchgefuehrt)

1. Action Pinning Policy
```bash
bash scripts/check-action-pinning.sh
```
Ergebnis: bestanden.

2. Binary Hygiene
```bash
bash scripts/check-binary-hygiene.sh
```
Ergebnis: bestanden.

3. Backend Test-Suite
```bash
cd backend
GOCACHE=/tmp/go-cache GOMODCACHE=/tmp/go-mod-cache /usr/local/go/bin/go test -tags "fts5 sqlite_crypt" ./...
```
Ergebnis: bestanden.

4. Tauri Build-Validierung
```bash
cd frontend/src-tauri
cargo check
```
Ergebnis: bestanden.

5. Electron Lint fuer geaenderte Hardening-Dateien
```bash
cd frontend
npx eslint src-electron/main.ts src-electron/windows/main-window.ts src-electron/modules/ipc-handlers.ts
```
Ergebnis: bestanden.

## 5. Restrisiko-Notizen

1. `SEC-005` (weiteres Hardening moeglich)
- Aktuell: stabile Versionspins + CI-Block fuer mutable Refs.
- Optional naechster Schritt: alle externen Actions auf Commit-SHAs pinnen.

2. `SEC-CSP-001` (funktionales Restrisiko)
- `style-src` ist gehaertet.
- `script-src 'unsafe-inline'` bleibt bis zur Umstellung des Frontend-Bootstrap-Flows aktiv.
