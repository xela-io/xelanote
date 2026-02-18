# REFACTORING_REPORT

## Phase 1: Analyse & Bestandsaufnahme

### 1) Repository-Uebersicht

#### Sprachen, Frameworks, Libraries
- Backend:
  - Sprache: Go (`backend/go.mod`)
  - HTTP: Chi v5
  - DB: SQLite + FTS5 (`mattn/go-sqlite3`)
  - Auth/Security: JWT, WebAuthn, TOTP, CSRF, Rate Limiting
- Frontend:
  - Sprache: TypeScript + Svelte (`.svelte`, `.ts`)
  - Framework: SvelteKit 2 + Svelte 5 Runes (`frontend/package.json`)
  - Build: Vite 6
  - Styling: Tailwind CSS 4
  - Tests: Vitest + Playwright
  - Desktop: Electron/Tauri (Quellstruktur vorhanden)
- Weitere relevante Artefakte:
  - SQL-Migrationen (`backend/internal/db/migrations/*.sql`)
  - Shell/Quality-Skripte (`scripts/*.sh`)

#### Projektstruktur & Architektur-Pattern
- Klare 3-Layer-Architektur im Backend:
  - API: `backend/internal/api/`
  - Service: `backend/internal/service/`
  - DB: `backend/internal/db/`
- Initialisierung/Komposition:
  - `backend/cmd/server/main.go`
- Frontend-Domains:
  - API-Client: `frontend/src/lib/api/*`
  - Stores: `frontend/src/lib/stores/*.svelte.ts`
  - Editor-Subsystem: `frontend/src/lib/editor/*`
  - Routen: `frontend/src/routes/*`

#### Einstiegspunkte
- Backend Runtime: `backend/cmd/server/main.go`
- Backend API Server Konstruktion: `backend/internal/api/api.go`
- Frontend App Shell/Layout: `frontend/src/routes/+layout.svelte`
- Frontend Root-Load: `frontend/src/routes/+layout.ts`
- Electron Main Process: `frontend/src-electron/main.ts`

#### Tests, Linting, CI/CD
- Tests:
  - Go-Testdateien: 69
  - Frontend Test/Spezifikationsdateien: 113
- Lint/Format/Type:
  - Go: `go vet`, `gofmt`, `golangci-lint` (`.golangci.yml`, `Makefile`)
  - Frontend: ESLint (`frontend/eslint.config.js`), Prettier, svelte-check
  - Git Hooks: `lefthook.yml`
- CI/CD:
  - Workflows: 3 (`.github/workflows/ci.yml`, `quality.yml`, `security.yml`)
  - Sicherheitsjobs: govulncheck, npm audit, dependency-review

### 2) Identifizierte Problembereiche (mit Fundstellen)

## 🔴 KRITISCH

1. Architekturverletzung (API -> DB direkt) trotz definierter Layer-Regel
- Fundstelle: `backend/internal/api/notes_crud_create.go:9`, `backend/internal/api/notes_crud_create.go:38`
- Befund:
  - API-Layer importiert `internal/db` und greift auf `db.AllowedNoteTypes` zu.
  - Das widerspricht der dokumentierten Schichtentrennung (API -> Service -> DB).
  - Bestehender Policy-Check meldet neuen Verstoß (`scripts/check-layer-violations.sh` schlug fehl).
- Risiko:
  - Weitere schleichende Layer-Erosion, schwerere Refactors, höheres Bug-Risiko bei Domain-Änderungen.

## 🟡 WICHTIG

1. Sehr lange, komplexe Handler/Methoden (God Functions)
- Fundstellen (Beispiele):
  - `backend/internal/api/auth_login.go:11` (196 Zeilen)
  - `backend/internal/api/notes_crud_create.go:14` (193 Zeilen)
  - `backend/internal/api/notes_crud_update.go:15` (134 Zeilen)
  - `backend/cmd/server/main.go:20` (170 Zeilen)
  - `frontend/src/routes/+layout.svelte:1` (585 Zeilen)
  - `frontend/src/lib/components/Editor.svelte:1` (1289 Zeilen)
- Risiko:
  - Geringe Testbarkeit, hohe kognitive Last, erhöhte Regressionswahrscheinlichkeit.

2. Duplizierte Auth-/Login-Abläufe
- Fundstellen:
  - `backend/internal/api/auth_login.go:67-110`
  - `backend/internal/api/auth_login.go:163-205`
- Befund:
  - Identische Blöcke für User-Lookup, Salt-Lookup, Cookie-Setzen, CSRF-Token, Response-Aufbau in zwei Zweigen.
- Risiko:
  - Divergentes Verhalten bei zukünftigen Änderungen, erhöhte Wartungskosten.

3. Inkonsistente/überholte Kommentare zur Auth-Persistenz
- Fundstelle: `frontend/src/lib/stores/auth.svelte.ts:210-213`
- Befund:
  - Kommentar sagt „Web lädt aus sessionStorage“, Implementierung nutzt primär HttpOnly-Cookies und nur Timestamp-Metadaten in sessionStorage.
- Risiko:
  - Fehlannahmen bei Security-Änderungen und Onboarding.

4. TODOs in produktiv relevanten Pfaden (offene funktionale Kanten)
- Fundstellen:
  - `frontend/src/lib/components/RecipeEditor.svelte:57` (Readonly-Role nicht umgesetzt)
  - `frontend/src/lib/stores/encryption.svelte.ts:408` (Auto-Lock Timer Stop TODO)
  - `backend/internal/db/search.go:22-23` (Search Timeout nicht konfigurierbar)
  - `backend/internal/db/migrations/020_e2e_encryption.sql:109` (keywords_enabled TODO)
- Risiko:
  - Teilfunktionalität, technische Schulden, uneinheitliches Laufzeitverhalten.

5. Fehlerbehandlung teils „log-only“ statt kontrollierter Degradation/Transaktion
- Fundstellen:
  - `backend/internal/api/notes_crud_create.go:181-190`
- Befund:
  - Nach erfolgreicher Notizanlage werden Link-/DueDate-Folgeschritte bei Fehler nur geloggt; Client bekommt trotzdem `201`.
- Risiko:
  - Inkonsistenter Zustand zwischen Client-Erwartung und Persistenz.

6. Typsicherheitsausnahmen (`any`) in Runtime-Code
- Fundstelle: `frontend/src/lib/editor/task-sortable.ts:25-35`
- Befund:
  - Zugriff auf interne CM-Observer via `(view as any)` + ESLint-Disable.
- Risiko:
  - Fragile Kopplung an interne APIs, Bruch bei Upgrades.

7. Lint-Status aktuell nicht sauber
- Fundstelle: `frontend/src/lib/components/Editor.svelte:2:1`
- Befund:
  - ESLint-Fehler (`simple-import-sort/imports`).
- Risiko:
  - Qualitätspipeline rot, erhöhte Merge-Reibung.

8. Security-Hardening verbesserbar (nicht akut exploitable, aber Angriffsfläche)
- CSP mit `'unsafe-inline'` und `'wasm-unsafe-eval'`:
  - `backend/internal/api/security.go:18-20`
- CORS permissiver Modus bei leerer Allowlist:
  - `backend/internal/api/api.go:95-109`
- Risiko:
  - Höhere XSS-/Policy-Risiken im Vergleich zu nonce/hash-basierter CSP und strikt konfigurierter Origin-Liste.

9. CI Action Pinning uneinheitlich
- Fundstellen:
  - Ungepinnte Actions z. B. `actions/checkout@v4` in `ci.yml`, `quality.yml`, `security.yml`
  - Teilweise gepinnt (positiv): z. B. Docker Buildx in `ci.yml:179`
- Risiko:
  - Supply-Chain-Risiko durch Tag-Moves.

10. Namens-/Dateikonventions-Inkonsistenzen
- Fundstellen:
  - `backend/internal/service/user_apikey.go`
  - `backend/internal/service/user_apikeys.go`
  - `backend/internal/api/users_apikeys.go`
- Befund:
  - Singular/Plural-Mix erschwert Auffindbarkeit und mentale Modellbildung.

## 🟢 NICE-TO-HAVE

1. Konsolidierung von Magic Numbers/Konfigurierbarkeit
- Beispiele:
  - Upload-Limit `10MB`: `backend/internal/api/uploads.go:20`
  - Search Timeout `5s`: `backend/internal/db/search.go:23`
- Empfehlung:
  - Zentral in Config/Constants + Doku.

2. Logging-Konsistenz Frontend
- Fundstellen:
  - Mehrere `console.log/warn/error` in `frontend/src/lib/stores/auth.svelte.ts`
- Empfehlung:
  - Einheitlicher Logger/Debug-Flags statt direkter Konsolenaufrufe.

3. Weitere Modularisierung sehr großer Svelte-Komponenten
- Fundstellen:
  - `frontend/src/lib/components/Editor.svelte`
  - `frontend/src/routes/+layout.svelte`
- Empfehlung:
  - Weitere Aufteilung nach Feature-Slices/Orchestratoren.

### 3) Explizite Kategorieabdeckung (Anforderungsliste)

- Code Smells:
  - Lange Methoden/Funktionen: vorhanden (siehe oben)
  - Tiefe Verschachtelung: vorhanden in großen Handlern/Layout-Orchestrierung
  - Duplikation: vorhanden (`auth_login.go`)
  - God Classes/Module: tendenziell in großen Orchestrator-Dateien
- Tote/unerreichbare Code-Abschnitte:
  - Keine harten „unreachable code“-Blöcke eindeutig nachweisbar im Scan.
  - Technische „quasi-dead“ Hinweise: mehrere TODOs in produktiven Pfaden.
- Inkonsistente Namensgebung:
  - API key Dateien singular/plural gemischt.
- Fehlerbehandlung:
  - Mehrere log-only Fehlerpfade ohne explizite Partial-Failure-Semantik.
- Hardcodierte Werte:
  - Mehrere sicherheits-/performance-relevante Werte fest kodiert.
- Sicherheit:
  - Kein direkter SQL-Injection-Hinweis im gescannten DB-Code (parametrisierte Queries).
  - Upload-Path-Traversal Schutz vorhanden.
  - CSP/CORS-Härtung dennoch ausbaufähig.
- Performance:
  - Große orchestrierende Komponenten/Handler deuten auf Optimierungspotenzial hin.
  - Caching/FTS vorhanden (positiv), aber weitere Granularität möglich.
- Veraltete Dependencies/deprecated APIs:
  - Lokale Prüfung in dieser Umgebung nicht vollständig möglich:
    - `go` CLI nicht verfügbar
    - `npm outdated` konnte wegen Umgebungsrestriktionen nicht abgeschlossen werden

### 4) Aktuelle Tool-Checks in dieser Session

- Erfolgreich:
  - `scripts/check-security-patterns.sh` -> OK
  - `scripts/check-svelte4-imports.sh` -> OK
  - `scripts/check-binary-hygiene.sh` -> OK
  - `frontend npm run check` -> 0 Errors/Warnings
- Fehlgeschlagen:
  - `scripts/check-layer-violations.sh` -> neuer Layer-Verstoß (`internal/api/notes_crud_create.go`)
  - `frontend npm run lint` -> Import-Sort-Fehler in `frontend/src/lib/components/Editor.svelte`
- Nicht ausführbar in Umgebung:
  - Go-basierte Checks (`go test`, `go vet`) wegen fehlendem `go` Binary

---

## Priorisierte Gesamtliste

### 🔴 KRITISCH
1. Layer-Verstoß API -> DB in `backend/internal/api/notes_crud_create.go`.

### 🟡 WICHTIG
1. Lange/God-Funktionen in Backend und Frontend-Orchestrierung.
2. Duplizierte Login-Logik in `auth_login.go`.
3. Inkonsistente/veraltete Auth-Kommentare.
4. Offene TODOs in Security/Role/Encryption-nahen Bereichen.
5. Log-only Fehlerpfade bei Folgeoperationen nach Notizanlage.
6. `any`-Bypass in Runtime-TS-Code.
7. Aktueller ESLint-Fehler.
8. CSP/CORS-Hardening ausbaufähig.
9. Uneinheitliches Action-Pinning in CI.
10. Namenskonventions-Inkonsistenzen.

### 🟢 NICE-TO-HAVE
1. Weitere Externalisierung von Magic Numbers.
2. Konsistentes Frontend-Logging-Konzept.
3. Weitere Zerlegung großer Svelte-Dateien.
