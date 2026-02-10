# Refactoring Report -- xelanote

**Datum:** 2026-02-10
**Analysierte Dateien:** ~90 Go-Quelldateien, ~100 Svelte/TS-Dateien, 43 DB-Migrationen
**Gesamtbewertung:** Solide Architektur mit starker Security-Basis -- Hauptprobleme bei Code-Duplikation und God-Files

---

## 0. Phase 1 Re-Analyse (2026-02-10)

Hinweis: Diese Re-Analyse basiert auf statischem Code-Review (kein Runtime/Load-Test). Findings sind kritisch gegengeprueft und mit dem bestehenden Report abgeglichen.

### 0.1 Projekt-Uebersicht (Stack, Architektur, Entry Points)

**Sprachen**
- Go (Backend)
- TypeScript/JavaScript + Svelte (Frontend)
- Rust (Tauri Desktop)
- SQL (Migrationen/Schema)
- Shell/YAML/Markdown/HTML/CSS (Build/CI/Docs)

**Backend**
- Go 1.24, Chi Router, SQLite + FTS5, gorilla/websocket, JWT, WebAuthn, Testify
- Architektur: 3-Schichten (API -> Service -> DB), WebSocket-Updates, Jobs, Caching

**Frontend**
- SvelteKit 2 + Svelte 5, Vite 6, Tailwind 4, CodeMirror 6, libsodium (WASM), svelte-i18n
- Tests: Vitest + Playwright
- Desktop: Electron + Tauri

**Entry Points**
- Backend: `backend/cmd/server/main.go`
- Frontend (SvelteKit): `frontend/src/routes/+layout.svelte`, `frontend/src/routes/+page.svelte`, `frontend/src/app.html`
- Electron: `frontend/src-electron/main.ts`
- Tauri: `frontend/src-tauri/src/main.rs`

**Projektstruktur (Hauptordner)**
- `backend/` (Go API/Service/DB, Migrationen, Jobs)
- `frontend/` (SvelteKit App, Desktop Integrationen, Tests)
- `docs/` (Architektur, API, Runbooks)
- `scripts/` (Build/Checks)

### 0.2 Tests, Linting, CI/CD

**Tests**
- Go: `backend/internal/*` mit `_test.go` (DB/Service/API/Parser/Auth)
- Frontend: Vitest (`frontend/src/lib/**/*.test.ts`)
- E2E: Playwright (`frontend/tests/e2e/*.spec.ts`)
- Bekanntes Problem: `e2e-feature.test.ts > Scenario 2` ist laut `TESTING.md` fehlschlagend

**Lint/Format**
- Go: `go vet` (CI/Make), `.golangci.yml` vorhanden
- Frontend: ESLint (flat config), Prettier, markdownlint
- Hooks: `lefthook.yml`
- `.editorconfig` vorhanden

**CI/CD**
- GitHub Actions: `ci.yml`, `quality.yml`, `security.yml`
- Forgejo Deploy Workflows: `.forgejo/workflows/deploy-staging.yml`, `.forgejo/workflows/deploy-production.yml`

### 0.3 Findings (Priorisiert)

#### 🔴 KRITISCH
Keine eindeutigen kritischen Findings aus statischer Analyse. (Wenn du Runtime-Checks oder Security-Audits willst: bitte freigeben.)

#### 🟡 WICHTIG

**W-1: God-Files / God-Components (Wartbarkeit, Testbarkeit)**
- Backend: `backend/internal/api/notes_*.go`, `backend/internal/db/notes_*.go`, `backend/cmd/server/*.go`, `backend/internal/api/api.go`
- Frontend: `frontend/src/lib/api.ts`, `frontend/src/lib/components/Editor.svelte`, `frontend/src/lib/components/Sidebar.svelte`, `frontend/src/routes/settings/+page.svelte`, `frontend/src/lib/stores/notes.svelte.ts`

**W-2: Fehlerbehandlung wird in produktivem Code teilweise ignoriert**
Aktuell keine offenen Punkte (bereinigte Error-Handling-Hotspots).

**W-3: Harte Konfigurationen im Code statt Konfiguration/Flags**
- `backend/cmd/server/server_jobs.go`: Worker-Count (4), Versions-Pruning (100)
- `backend/internal/api/api.go`: Rate-Limit Defaults hart codiert
- `frontend/src/lib/config.ts`: `DEFAULT_SERVER` = `https://xelanote.com` (Desktop Default)

**W-4: Typ-Sicherheit mit `any`/unsicheren Casts**
- `frontend/src/lib/crypto/sodium.ts` (`sodium: any`)
- `frontend/src/lib/stores/websocket.svelte.ts` (`payload: any`)
- `frontend/src/lib/editor/markdown.ts` (Renderer-Optionen `any`)
- `frontend/src/lib/stores/history.svelte.ts` (`data as any`)

**W-5: Nicht-source Artefakt im Repo**
- `backend/cmd/server/server` ist ein ELF-Binary im Repo (nicht in `.gitignore`), kann Releases/Reviews verfaelschen.

#### 🟢 NICE-TO-HAVE

**N-1: TODOs in produktivem Code**
- `backend/internal/db/search.go` (Timeout konfigurierbar machen)
- `frontend/src/lib/components/RecipeEditor.svelte` (Readonly-Checks TODO)

**N-2: Potentiell teure Full-Reloads bei WebSocket-Events**
- `frontend/src/lib/stores/websocket.svelte.ts`: `tree.loadTree()` auf jedem Note-Event (ggf. inkrementell moeglich)

### 0.4 Abgleich mit bestehendem Report (kritisch geprueft)

**Bestaetigt**
- God-Files im Backend/Frontend (entsprechen bisherigen W-1 bis W-3)
- WebSocket nutzt `getWsBaseUrl()` (K-3 wirkt umgesetzt)
- Gemini API-Key via Header (`x-goog-api-key`) (K-5 wirkt umgesetzt)

**Abweichung/Teilweise**
- CORS-Fix (K-4): In `backend/internal/api/api.go` ist der Fatal-Check nur aktiv wenn `XELANOTE_ENV` nicht leer ist. Leeres `XELANOTE_ENV` bleibt permissiv. Das widerspricht dem Anspruch "nicht-explizites Env == streng". Ich werte das als noch offen/unklar.

**Nicht verifiziert (nur statisch, keine historische Commit-Pruefung)**
- K-1/K-2/K-6 und einige W-4+ Items aus dem Alt-Report habe ich nicht separat gegen historische Commits geprueft. Wenn du willst, kann ich die konkrete Implementierung/Regressionen tief pruefen.

---

## Phase 2 Fortschritt (Strukturelles Refactoring)

### Erledigt
- **Backend API: Rate-Limit-Konfiguration extrahiert**
  - Neue Datei: `backend/internal/api/rate_limits.go`
  - `NewServer` nutzt jetzt `buildRateLimitConfig()` fuer Limits (keine Verhaltensaenderung).

- **Backend API: Server-Struct ausgelagert**
  - Neue Datei: `backend/internal/api/server.go`
  - `Server`-Struct und zugehoerige Methoden aus `api.go` verschoben.

### Offen
- Weitere Aufteilung der God-Files (Backend/Frontend) nach Domänen/Responsibilities.
- Modularisierung `frontend/src/lib/api.ts` (Feature-spezifische Clients).
- Zerlegung `frontend/src/lib/components/Editor.svelte` in Sub-Komponenten.

## Phase 3 Fortschritt (Code-Qualitaet im Detail)

### Erledigt
- **Type-Safety: CommandData als discriminated union**
  - `frontend/src/lib/commands/types.ts`
  - `frontend/src/lib/stores/history.svelte.ts` entfernt `any` Casts.

- **Type-Safety: libsodium Wrapper typisiert**
  - `frontend/src/lib/crypto/sodium.ts` statt `any`.

- **Type-Safety: Markdown-It Renderer Args typisiert**
  - `frontend/src/lib/editor/markdown.ts` mit `MarkdownIt.Options/Env/Renderer`.

- **Type-Safety: WebSocket Payloads ohne `any`**
  - `frontend/src/lib/stores/websocket.svelte.ts` mit `unknown` + Helper-Typen.
- **Backend: Fehlerbehandlung & Validierung gehaertet**
  - `backend/internal/api/journal.go` validiert `year`/`month` strikt (inkl. Range-Checks).
  - `backend/internal/api/notes_helpers.go`, `backend/internal/api/notes_crud.go`, `backend/internal/api/import.go` behandeln WS-JSON-Encode-Errors (loggen statt ignorieren).
  - `backend/internal/db/notes_crud.go`, `backend/internal/db/notes_list.go`, `backend/internal/db/notes_trash.go`, `backend/internal/db/notes_encryption.go`, `backend/internal/db/journal.go` pruefen RFC3339-Timestamps strikt (Parse-Errors werden retourniert).
  - `backend/internal/api/import.go` validiert File-Anzahl, leere Inhalte, Notiz-/Folder-Felder und begrenzt Error-Listen.
  - `backend/internal/service/notes_crud.go`, `backend/internal/service/notes_encryption.go` loggen Snapshot-Query-Fehler (und snapshoten konservativ).
  - `backend/internal/api/admin.go` behandelt Fehler beim Laden von User-Details mit klaren HTTP-Antworten.

### Offen
- Weitere `any`/unsaubere Casts (z.B. `history.svelte.ts` JSON parse, andere Stores).

## Phase 4 Fortschritt (Testing & Dokumentation)

### Erledigt
- **Unit-Tests fuer History Store**
  - `frontend/src/lib/stores/history.test.ts`
  - Test-Namen folgen dem `should ... when ...` Schema.

- **Dokumentation: Test-Namenskonvention**
  - `TESTING.md` ergaenzt (Namensschema).

- **CHANGELOG aktualisiert**
  - `CHANGELOG.md` Eintrag fuer neue Tests.

- **Backend-Service Tests: Sharing-Validierung**
  - `backend/internal/service/sharing_test.go`
  - Abdeckung: Self-Share, Non-Owner, Encrypted Note/Folder, Folder mit verschluesselten Notizen, Duplicate.
  - Hinweis: Tests laufen mit `-tags "fts5"` (SQLite FTS5 erforderlich).

- **Backend-Service Tests: Auth Refresh/Logout**
  - `backend/internal/service/auth_test.go`
  - Abdeckung: Refresh-Token-Rotation + Logout invalidiert Refresh-Token.
  - Hinweis: Tests laufen mit `-tags "fts5"` (SQLite FTS5 erforderlich).

- **Backend-Service Tests: Notes Snapshots/Validation**
  - `backend/internal/service/notes_test.go`
  - Abdeckung: invalides `folderPath`, Snapshot-Threshold (kein Doppel-Snapshot), Snapshot nach Threshold, Note-Limit.
  - Hinweis: Tests laufen mit `-tags "fts5"` (SQLite FTS5 erforderlich).

- **Backend-Service Tests: Notes Link-Resolution**
  - `backend/internal/service/notes_links_test.go`
  - Abdeckung: Unresolved-Links Cleanup, Backlinks ignorieren geloeschte Source-Notes, geloeschte Target-Notes, User-Scoped Backlinks.
  - Hinweis: Tests laufen mit `-tags "fts5"` (SQLite FTS5 erforderlich).

- **Dokumentation fuer neuere Refactorings**
  - Modul-Snapshot + How-To in `REFACTORING_REPORT.md` sowie API-Struktur in `README.md`/`docs/development.md`.

### Offen
- Keine (optional abgeschlossen).

## Phase 5 Fortschritt (Linting & Formatierung)

### Ergebnis
- Tooling ist bereits vorhanden und konsistent konfiguriert:
  - Go: `go vet`, `.golangci.yml`, `gofmt` via Makefile/CI
  - Frontend: ESLint (flat), Prettier, markdownlint
  - Hooks: `lefthook.yml`
  - `.editorconfig` vorhanden
- Import-Sortierung fuer Frontend via ESLint hinzugefuegt.

### Optional (nach Wunsch)
- `lint-staged`/Husky zusaetzlich zu Lefthook (falls zusaetzliche Hook-Logik gewuenscht).
- Import-Sortierung (Go: `gofumpt`/`goimports`, Frontend: `eslint-plugin-simple-import-sort`).

## Phase 2.1 Fortschritt (Backend Reliability Pass)

### Erledigt
- **Encrypted note decode/marshal guardrails**
  - `backend/internal/api/notes_crud.go`, `backend/internal/api/notes_encryption.go` (Base64-Decode Fehler + JSON Marshal Fehler behandelt)

- **LLM Client: Error Body Read Handling**
  - `backend/internal/llm/claude.go`
  - `backend/internal/llm/gemini.go`

- **Error Report Service: Error Body Read Handling**
  - `backend/internal/service/errorreport.go`

### Offen
- Weitere Error-Handling-Hotspots (z.B. `api/journal.go`, `db/*` RowsAffected/LastInsertId).

## Phase 3 Fortschritt (Frontend-Modularisierung)

### Status
- **Abgeschlossen:** `frontend/src/lib/api.ts` ist jetzt reine Facade mit Re-Exports.
- **Neue Module:** `api/client.ts`, `api/types.ts`, `api/notes.ts`, `api/search.ts`, `api/folders.ts`, `api/tags.ts`,
  `api/auth.ts`, `api/uploads.ts`, `api/imports.ts`, `api/trash.ts`, `api/versions.ts`, `api/preferences.ts`,
  `api/admin.ts`, `api/ai.ts`, `api/sharing.ts`, `api/recipes.ts`, `api/journal.ts`, `api/features.ts`,
  `api/encryption.ts`, `api/config.ts`, `api/due-dates.ts`, `api/graph.ts`, `api/templates.ts`, `api/snippets.ts`.
- **Typecheck:** Lauf angestossen; bestehende FE-Typfehler bleiben (nicht durch Modularisierung verursacht).

### Modulstruktur (Snapshot)
```
frontend/src/lib/api/
  admin.ts
  ai.ts
  auth.ts
  client.ts
  config.ts
  due-dates.ts
  encryption.ts
  features.ts
  folders.ts
  graph.ts
  imports.ts
  journal.ts
  notes.ts
  preferences.ts
  recipes.ts
  search.ts
  sharing.ts
  snippets.ts
  tags.ts
  templates.ts
  trash.ts
  types.ts
  uploads.ts
  versions.ts
```

### How-To: Neues API-Modul hinzufuegen
1. Neue Datei unter `frontend/src/lib/api/` anlegen (z.B. `foo.ts`) und Funktionen mit `request()` aus `api/client.ts` nutzen.
2. Falls neue Response-Types benoetigt werden: in `api/types.ts` definieren.
3. In `frontend/src/lib/api.ts` per `export * from './api/foo';` re-exporten.
4. Bestehende Call-Sites bleiben auf `import * as api from '$lib/api'`.

### Nächste Schritte (Resume)
1. FE-Typecheck-Fehler bereinigen (siehe Typecheck-Log).
2. Lint/Format fuer neue Module im Frontend wurde ausgefuehrt (Prettier + ESLint --fix).

## Fortschritt

| Phase | Beschreibung | Commit | Status |
|-------|-------------|--------|--------|
| Phase 1 | Kritische Issues (K-1 bis K-6) | `c799968` | ERLEDIGT |
| Phase 2 | Strukturelles Refactoring (7 Tasks) | `7856b6b` | ERLEDIGT |
| Phase 3a | Code Quality (5 Tasks) | `9f2c607` | ERLEDIGT |
| Phase 3b | Code Quality (4 Tasks) | `8d067c5` | ERLEDIGT |
| Phase 4 | Testing & Documentation | -- | OFFEN |
| Phase 5 | Linting & Formatting | -- | OFFEN |

**Erledigt: 22 von 47 Findings (6 kritisch, 13 wichtig, 3 backlog)**

---

## Inhaltsverzeichnis

1. [Projekt-Uebersicht](#1-projekt-uebersicht)
2. [Kritische Probleme (SOFORT)](#2--kritische-probleme--sofort-beheben)
3. [Wichtige Probleme (DIESE WOCHE)](#3--wichtige-probleme--diese-woche)
4. [Nice-to-Have (BACKLOG)](#4--nice-to-have--backlog)
5. [Statistik](#5-statistik)
6. [Aktionsplan](#6-aktionsplan)

---

## 1. Projekt-Uebersicht

| Bereich | Technologie | Version |
|---------|-------------|---------|
| Backend | Go + Chi + SQLite/FTS5 | Go 1.24, Chi v5 |
| Frontend | SvelteKit + Svelte 5 + Tailwind v4 | SK 2.50, Svelte 5.2, TW 4.0 |
| Editor | CodeMirror 6 | v6.35 |
| Crypto | libsodium (WASM) + @noble/hashes | v0.8.1 / v2.0.1 |
| Build | Vite 6 + Docker Multi-Stage | v6.0 |
| CI/CD | Forgejo Actions + Lefthook | SHA-pinned |
| Tests | Go testing + Vitest + Playwright | 36 Go-Test-Dateien, 12 FE-Unit, 6 E2E |

**Architektur:** Drei-Schichten (API -> Service -> DB) mit E2E-Encryption, Offline-Queue, WebSocket-Updates, PWA-Support.

**Staerken:**
- Zero-Knowledge E2E-Encryption (AES-256-GCM + Argon2id)
- Konstant-Zeit-Auth (bcrypt Dummy-Hash fuer nicht-existierende User)
- Docker-Haertung (read-only FS, cap_drop ALL, PID-Limit)
- Zero ESLint-Warnings (von 260+ reduziert)
- Parametrisierte SQL-Queries durchgehend
- Pre-Commit-Hooks (gofmt, go-vet, ESLint, Prettier, Markdownlint)

---

## 2. KRITISCHE PROBLEME -- SOFORT BEHEBEN

### K-1: Inkonsistente Signatur-Parameter bei Upload-URLs (Bug) -- ERLEDIGT (Phase 1, `c799968`)

**Dateien:**
- `backend/internal/api/uploads.go:145` -- generiert `?signature=...&expires=...`
- `backend/internal/api/recipes.go:966` -- generiert `?sig=...&expires=...`
- `backend/internal/api/uploads.go:169` -- liest nur `signature`, nicht `sig`

**Problem:** `uploadImage()` und `signImageURL()` verwenden unterschiedliche Query-Parameter-Namen. `serveUpload()` liest nur `signature`. Rezeptbilder fallen auf Cookie-Fallback zurueck; ohne Cookie (z.B. `<img>` in externen Clients) schlaegt Zugriff fehl.

**Fix:** `sig` in `recipes.go:966` auf `signature` geaendert.

---

### K-2: Admin-Handler ignorieren getUserID-Fehler (Audit-Trail) -- ERLEDIGT (Phase 1, `c799968`)

**Datei:** `backend/internal/api/admin.go:164, 207, 318`

**Problem:** `adminID, _ := getUserID(r)` ignoriert Fehler. Bei Fehlschlag ist `adminID = 0`, und Admin-Aktionen (User loeschen, Admin-Toggle, Settings aendern) werden mit falscher Admin-ID geloggt.

**Fix:** Fehlerbehandlung hinzugefuegt, bei Fehler HTTP 401.

---

### K-3: WebSocket ignoriert `getWsBaseUrl()` -- Desktop-Apps kaputt -- ERLEDIGT (Phase 1, `c799968`)

**Datei:** `frontend/src/lib/stores/websocket.svelte.ts:23-25`

**Problem:** WebSocket-URL war hart kodiert. `config.ts:98-110` bietet `getWsBaseUrl()` fuer Desktop-Apps (Tauri/Electron), wurde aber nicht genutzt.

**Fix:** `WS_URL` durch `getWsBaseUrl()` ersetzt.

---

### K-4: CORS erlaubt jede Origin bei nicht-explizitem `XELANOTE_ENV` -- ERLEDIGT (Phase 1, `c799968`)

**Datei:** `backend/internal/api/api.go:557-573, 88-96`

**Problem:** Fatal-Check greift nur bei `env == "production"`. Staging oder leerer `XELANOTE_ENV` laeuft mit vollständig permissivem CORS.

**Fix:** Fatal-Check auf `env != "development" && env != "test"` geaendert.

---

### K-5: Gemini API-Key im URL-Query-Parameter (Logging-Risiko) -- ERLEDIGT (Phase 1, `c799968`)

**Datei:** `backend/internal/llm/gemini.go:187, 294`

**Problem:** API-Key wurde als `?key=...` uebergeben. URL-Parameter werden in Server-Logs gespeichert.

**Fix:** Auf `x-goog-api-key` Header umgestellt.

---

### K-6: Typ-Assertion ohne Check in twofa.go/fido2.go (Panic-Risiko) -- ERLEDIGT (Phase 1, `c799968`)

**Dateien:**
- `backend/internal/api/twofa.go:54, 79, 112, 196, 228`
- `backend/internal/api/fido2.go:48, 67, 101, 114`

**Problem:** `userID := r.Context().Value(userIDKey).(int)` ohne Typ-Check.

**Fix:** Durch `getUserID(r)` mit Fehlerbehandlung ersetzt.

---

## 3. WICHTIGE PROBLEME -- DIESE WOCHE

### Strukturelle Probleme (God-Files / SOLID-Verletzungen)

#### W-1: God-Struct `Server` mit 30+ Feldern -- ERLEDIGT

**Datei:** `backend/internal/api/api.go:31-84`

21-Parameter-Konstruktor. Vereint 15 Service-Referenzen, 14 Rate-Limiter und Infrastructure-Referenzen in einer flachen Struktur.

**Fix:** `ServerConfig` eingefuehrt und `NewServer` auf einen Config-Struct umgestellt (Aufrufer in `main.go` und Tests aktualisiert).

---

#### W-2: God-Files im Backend -- IN ARBEIT

| Datei | Zeilen | Verantwortlichkeiten |
|-------|--------|---------------------|
| `backend/internal/api/notes_meta.go` | neu | Rename, Backlinks, Color, Async-Job |
| `backend/internal/api/notes_ai.go` | neu | LLM/Summary/AI-Endpoints |
| `backend/internal/api/notes_trash.go` | neu | Trash-Endpoints |
| `backend/internal/api/notes_helpers.go` | neu | Validation, ETag, request types, helper utils |
| `backend/internal/api/notes_encryption.go` | neu | Decrypt + DEK-Reencrypt Endpoints |
| `backend/internal/api/notes_misc.go` | neu | listNoteTitles + reorderNotes |
| `backend/internal/api/notes_crud.go` | neu | CRUD (list/create/get/update/delete) |
| `backend/internal/service/notes_service.go` | neu | NoteService struct + ctor |
| `backend/internal/service/notes_cache.go` | neu | Cache-Invalidierung |
| `backend/internal/service/notes_helpers.go` | neu | Note-Service Helper/Validation/Cache-Keys |
| `backend/internal/service/notes_crud.go` | neu | Note-Service CRUD |
| `backend/internal/service/notes_search.go` | neu | Search, Backlinks, Titles |
| `backend/internal/service/notes_links.go` | neu | Link-Parsing + Update |
| `backend/internal/service/notes_rename.go` | neu | Rename + Wikilink-Update |
| `backend/internal/service/notes_trash.go` | neu | Trash-Operationen |
| `backend/internal/service/notes_versions.go` | neu | Versionen + Restore |
| `backend/internal/service/notes_encryption.go` | neu | Encrypt/Decrypt + DEK-Rewrap |
| `backend/internal/service/notes_ai.go` | neu | AI-Flag Handling |
| `backend/internal/service/notes_folders.go` | neu | Folder-Operationen + Defaults |
| `backend/internal/service/notes_tags.go` | neu | Tag-Operationen |
| `backend/internal/db/notes_models.go` | neu | Note-Modelle/DTOs |
| `backend/internal/db/notes_crud.go` | neu | Note-CRUD + Titel-Queries |
| `backend/internal/db/notes_list.go` | neu | Listen + Folder-Listing |
| `backend/internal/db/notes_folders.go` | neu | Folder-Queries + Counts |
| `backend/internal/db/notes_trash.go` | neu | Trash-Operationen |
| `backend/internal/db/notes_encryption.go` | neu | Encrypt/Decrypt + Keywords |
| `backend/internal/db/notes_rewrap.go` | neu | DEK-Rewrap (Bulk) |
| `backend/internal/db/notes_summary.go` | neu | Summary + Content-Hash |
| `backend/internal/db/notes_ai.go` | neu | AI-Flags + Titel-Query |
| `backend/internal/db/notes_color.go` | neu | Farb-Update |
| `backend/internal/db/notes_misc.go` | neu | ReorderNotes |
| `backend/cmd/server/main.go` | Entry | Flag-Parsing + Orchestrierung |
| `backend/cmd/server/server_config.go` | neu | Env/Paths/JWT/Origins |
| `backend/cmd/server/server_logger.go` | neu | Logger |
| `backend/cmd/server/server_services.go` | neu | Core Services + Maintenance |
| `backend/cmd/server/server_llm.go` | neu | LLM Router + Summarize |
| `backend/cmd/server/server_jobs.go` | neu | JobManager + Version-Pruning |
| `backend/cmd/server/server_error_report.go` | neu | Forgejo Error Reports |
| `backend/cmd/server/server_websocket.go` | neu | WebSocket Manager |
| `backend/cmd/server/server_webauthn.go` | neu | FIDO2/WebAuthn |
| `backend/cmd/server/server_turnstile.go` | neu | Turnstile |
| `backend/cmd/server/server_static.go` | neu | Static SPA + Cache Headers |
| `backend/cmd/server/server_pprof.go` | neu | pprof Server |
| `backend/cmd/server/server_shutdown.go` | neu | Graceful Shutdown |

**Status:** NoteService + Note-DB + Server-Init aufgeteilt. Offene God-Files: `backend/internal/api/api.go`.

---

#### W-3: God-Files im Frontend -- OFFEN

| Datei | Zeilen | Verantwortlichkeiten |
|-------|--------|---------------------|
| `frontend/src/lib/api.ts` | 3135 | 264 Exporte: Types, Token-Refresh, CSRF, Offline-Queue, 60+ API-Funktionen |
| `frontend/src/routes/settings/+page.svelte` | 2276 | 7 Tabs mit eigenem State und Logik |
| `frontend/src/lib/components/Editor.svelte` | 2102 | 15+ Verantwortlichkeiten (CM6, Toolbar, Preview, Upload, AI, etc.) |
| `frontend/src/lib/stores/notes.svelte.ts` | 1361 | 20+ Exports, Auto-Save, Conflict-Detection, Encryption |
| `frontend/src/lib/components/Sidebar.svelte` | 1115 | Mobile + Desktop mit ~300 Zeilen Template-Duplikation |
| `frontend/src/routes/+layout.svelte` | 724 | 170-Zeilen `initializeAsync`, 15+ Subsysteme |

**Fortschritt:**
- `frontend/src/lib/components/Editor.svelte`: Toolbar in `frontend/src/lib/components/editor/EditorToolbar.svelte` ausgelagert (erster Schnitt).
- `frontend/src/lib/components/Editor.svelte`: Task-Sort/Toggle-Logik nach `frontend/src/lib/editor/task-toggle.ts` ausgelagert.
- `frontend/src/lib/components/Editor.svelte`: Image-Upload/Paste/Drop-Logik nach `frontend/src/lib/editor/image-upload.ts` ausgelagert.
- `frontend/src/lib/components/Editor.svelte`: AI-Transform/Selection-Logik nach `frontend/src/lib/editor/ai-actions.ts` ausgelagert.
- `frontend/src/lib/components/Editor.svelte`: Find/Replace-UI-Flow nach `frontend/src/lib/editor/find-replace-ui.ts` ausgelagert.
- `frontend/src/lib/components/Editor.svelte`: Split-Resize-Handler nach `frontend/src/lib/editor/split-resize.ts` ausgelagert.
- `frontend/src/lib/components/Editor.svelte`: Lazy-Load Dialog-Loader nach `frontend/src/lib/editor/dialog-loaders.ts` ausgelagert.
- `frontend/src/lib/components/Editor.svelte`: Task-Insert + Indent/Outdent nach `frontend/src/lib/editor/task-insert.ts` und `frontend/src/lib/editor/indentation.ts` ausgelagert.
- `frontend/src/lib/components/Editor.svelte`: Panels (Backlinks/Summary/Tags/Links) nach `frontend/src/lib/components/editor/EditorPanels.svelte` ausgelagert.
- `frontend/src/lib/components/Editor.svelte`: Preview-Click/TOC-Handler nach `frontend/src/lib/editor/preview-interactions.ts` ausgelagert.

---

### Code-Duplikation Backend

#### W-4: Duplizierte `cleanMarkdownCodeBlock` / `cleanJSONResponse` -- ERLEDIGT (Phase 2, `7856b6b`)

**Dateien:**
- `backend/internal/llm/gemini.go:235-251`
- `backend/internal/service/recipe_suggestions.go:500-512`

**Fix:** `cleanJSONResponse()` in `recipe_suggestions.go` durch `llm.CleanMarkdownCodeBlock()` ersetzt, lokale Duplikat-Funktion geloescht.

---

#### W-5: Duplizierte Link/DueDate-Validierung in createNote/updateNote -- ERLEDIGT (Phase 2, `7856b6b`)

**Datei:** `backend/internal/api/notes_helpers.go`

**Fix:** Helper-Funktionen `validateLinks()` und `validateDueDates()` extrahiert.

---

#### W-6: Dupliziertes API-Key-Management (Claude/Gemini) -- ERLEDIGT (Phase 3a, `9f2c607`)

**Datei:** `backend/internal/api/users.go:519-701`

**Fix:** 6 duplizierte Handler durch 3 generische Factory-Funktionen ersetzt (`handleSetAPIKey`, `handleDeleteAPIKey`, `handleGetAPIKeyStatus`) mit `apiKeyProvider`-Struct. ~180 Zeilen auf ~95 Zeilen reduziert.

---

#### W-7: Duplizierte Ownership/Permission/Encryption-Checks in Recipes -- ERLEDIGT (Phase 3b, `8d067c5`)

**Datei:** `backend/internal/service/recipes.go`

**Fix:** `checkRecipeWriteAccess()` und `checkRecipeReadAccess()` Helper extrahiert. 7 Methoden vereinfacht.

---

#### W-8: Duplizierte LLM-Client-Implementierung -- OFFEN

**Dateien:**
- `backend/internal/llm/claude.go` (352 Zeilen)
- `backend/internal/llm/gemini.go` (374 Zeilen)

4 nahezu identische HTTP-Request/Response-Handling-Bloecke (Generate + GenerateWithImage, je 2x). Bug-Fixes muessen an 4 Stellen gleichzeitig gemacht werden.

---

#### W-9: Duplizierte Konstanten MaxLinksPerNote / MaxLinkTitleLength -- ERLEDIGT (Phase 2, `7856b6b`)

**Dateien:**
- `backend/internal/api/notes_helpers.go:94-100`
- `backend/internal/service/notes_links.go:13-17`

**Fix:** Konstanten in ein gemeinsames Paket verschoben.

---

### Code-Duplikation Frontend

#### W-10: Wikilink-Extraktion 5x identisch kopiert -- ERLEDIGT (Phase 2, `7856b6b`)

**Datei:** `frontend/src/lib/stores/notes.svelte.ts`

**Fix:** `extractUniqueLinks(content)` Helper extrahiert und an allen 5 Stellen verwendet.

---

#### W-11: Encrypt/Decrypt-Pattern 6x dupliziert -- ERLEDIGT (Phase 2, `7856b6b`)

**Datei:** `frontend/src/lib/stores/notes.svelte.ts`

**Fix:** `decryptNoteResponse()` Helper extrahiert und an allen 6 Stellen verwendet.

---

#### W-12: Paranoid-Mode-Offline-Guard 4x kopiert -- ERLEDIGT (Phase 2, `7856b6b`)

**Datei:** `frontend/src/lib/stores/notes.svelte.ts`

**Fix:** `assertOnlineForParanoidMode()` Helper extrahiert und an allen 4 Stellen verwendet.

---

### Sicherheit & Informations-Offenlegung

#### W-13: 408 console.log-Statements, teils mit sensitiven Daten -- ERLEDIGT (Phase 3b, `8d067c5`)

**76 Dateien betroffen.** Kritisch:
- `auth.svelte.ts:507-518` -- loggte Encryption-Salt-Informationen
- `encryption.svelte.ts` -- loggte KEK-Persistence-Details
- `kdf.worker.ts` -- loggte KDF-Operationen
- `search-index.svelte.ts` -- loggte verschluesselte Notizen-Anzahl
- `+layout.svelte` -- loggte KEK-Restore-Status

**Fix:** Alle sensitiven Crypto/KEK/Salt-Logs aus 8 Frontend-Dateien entfernt. Verbleibende Logs enthalten keine sensitiven Daten oder sind hinter `import.meta.env.DEV` geschuetzt.

---

#### W-14: Raw `err.Error()` in HTTP-500-Responses -- ERLEDIGT (Phase 2, `7856b6b`)

**83 Instanzen in 13 API-Dateien.**

**Fix:** `respondInternalErr(w, msg, err)` Helper in `api.go` eingefuehrt. Loggt Fehler serverseitig via `slog.Error()`, gibt generisches `"internal server error"` an Client zurueck. Alle 83 Instanzen ersetzt.

---

#### W-15: Authentifizierte User-Enumeration via /api/users/search -- ERLEDIGT (Phase 3a, `9f2c607`)

**Datei:** `backend/internal/db/sharing.go`

**Fix:** E-Mail aus Suche entfernt (nur noch Username), Ergebnis-Limit von 10 auf 5 reduziert.

---

#### W-16: Debug-Logging im Produktionscode -- ERLEDIGT (Phase 2, `7856b6b`)

**Datei:** `backend/cmd/server/server_static.go`

**Fix:** Debug-Logging (`"Checking embedded files..."`, `"Root entries: X"`) entfernt.

---

### TypeScript / Type-Safety

#### W-17: `any`-Typen an 15+ kritischen Stellen -- TEILWEISE ERLEDIGT (Phase 3b, `8d067c5`)

| Datei | Status | Aenderung |
|-------|--------|-----------|
| `frontend/src/lib/crypto/fido2.ts:60,82,100,116` | ERLEDIGT | `ServerPublicKeyOptions`/`ServerCredentialDescriptor` Interfaces definiert |
| `frontend/src/lib/stores/settings.svelte.ts:124,155,198,277` | ERLEDIGT | `catch (err: any)` durch `catch (err: unknown)` mit `instanceof Error` Check ersetzt |
| `frontend/src/lib/crypto/fido2.ts:151,190` | ERLEDIGT | `catch (err: any)` durch `catch (err: unknown)` mit `instanceof DOMException` ersetzt |
| `frontend/src/lib/stores/websocket.svelte.ts:98` | OFFEN | `payload: any` -- WebSocket-Protokoll muesste komplett typisiert werden |
| `frontend/src/lib/crypto/sodium.ts:5` | OFFEN | `let sodium: any` -- Abhaengig von libsodium-wrappers Typ-Definitionen |
| `frontend/src/lib/editor/markdown.ts:460-462` | OFFEN | `options: any, env: any, _self: any` -- markdown-it Plugin-API |

---

#### W-18: Duplikat-Import in Login-Page -- ERLEDIGT (Phase 2, `7856b6b`)

**Datei:** `frontend/src/routes/login/+page.svelte:4, 15`

**Fix:** Duplikat-Import `auth_store` entfernt, alle Referenzen auf `auth` umgestellt.

---

### Performance

#### W-19: `IsAvailable()` macht echten API-Call bei jedem Aufruf -- ERLEDIGT (Phase 3b, `8d067c5`)

**Dateien:**
- `backend/internal/llm/claude.go` (IsAvailable)
- `backend/internal/llm/gemini.go` (IsAvailable)

**Fix:** 5-Minuten-TTL-Cache in beide Clients eingebaut (`availableCache` + `availableCacheExpiry` Felder). Verhindert wiederholte externe HTTP-Calls bei Status-Checks.

---

### Fehlende Tests

#### W-20: Kritische Backend-Pakete ohne Tests -- OFFEN

| Paket/Datei | Zeilen |
|-------------|--------|
| `backend/internal/llm/claude.go` | 352 |
| `backend/internal/llm/gemini.go` | 374 |
| `backend/internal/llm/router.go` | 291 |
| `backend/internal/service/summarize.go` | 974 |
| `backend/internal/service/recipes.go` | 764 |
| `backend/internal/service/recipe_suggestions.go` | 513 |
| `backend/internal/cache/cache.go` | 76 |
| `backend/internal/jobs/jobs.go` | 145 |
| `backend/internal/crypto/apikey.go` | 169 |
| `backend/internal/api/export.go` | 153 |
| `backend/internal/api/import.go` | 213 |

---

#### W-21: Kritische Frontend-Stores ohne Tests -- OFFEN

| Datei | Zeilen |
|-------|--------|
| `frontend/src/lib/stores/notes.svelte.ts` | 1361 |
| `frontend/src/lib/stores/auth.svelte.ts` | 568 |
| `frontend/src/lib/stores/encryption.svelte.ts` | 401 |
| `frontend/src/lib/stores/websocket.svelte.ts` | 216 |
| `frontend/src/lib/stores/settings.svelte.ts` | 362 |
| `frontend/src/lib/api.ts` | 3135 |

---

## 4. NICE-TO-HAVE -- BACKLOG

### N-1: Cache und WebSocket-Manager ohne Shutdown-Mechanismus -- OFFEN

- `backend/internal/cache/cache.go:23` -- Goroutine-Leak
- `backend/internal/websocket/manager.go` -- Endlose `for-select` ohne Context

### N-2: JobManager-Jobs werden nie aufgeraeumt (Memory Leak) -- OFFEN

`backend/internal/jobs/jobs.go` -- `jm.jobs` (sync.Map) waechst unbegrenzt.

### N-3: Error-Vergleich via String-Match statt Sentinel Errors -- ERLEDIGT (Phase 3a, `9f2c607`)

`backend/internal/api/admin.go` -- `err.Error() == "cannot demote yourself"` / `"cannot delete yourself"`

**Fix:** `ErrSelfDemotion` und `ErrSelfDeletion` Sentinel Errors in `service/admin.go` definiert. `admin.go` nutzt jetzt `errors.Is()`.

### N-4: `respondJSON` ignoriert `json.Encode`-Fehler -- ERLEDIGT (Phase 3a, `9f2c607`)

`backend/internal/api/api.go`

**Fix:** Encode-Fehler wird jetzt via `slog.Error()` geloggt.

### N-5: Export laedt alle Notizen in den Speicher -- OFFEN

`backend/internal/api/export.go:12-51` -- bei vielen/grossen Notizen problematisch.

### N-6: Deprecated `NormalizeTitle` in db.go -- OFFEN

`backend/internal/db/db.go:219-224` -- Wrapper auf `utils.NormalizeTitle`.

### N-7: Frontmatter-Tags werden beim Import nicht verwendet -- OFFEN

`backend/internal/api/import.go:36-39` -- `Tags` geparsed aber nie angehaengt.

### N-8: Migrations-Nummern haben Luecken (025 -> 028) -- OFFEN

`backend/internal/db/db.go:140-181`

### N-9: Dupliziertes Null-zu-leeres-Slice Pattern -- OFFEN

10+ Stellen in `search.go`, `tags.go`, `recipes.go`, etc. `ensureNotes()` existiert, wird aber nicht konsistent verwendet.

### N-10: SetRecipeService Post-Init Pattern -- OFFEN

`backend/internal/api/api.go:197-204` -- Server kurzzeitig in unvollstaendigem Zustand.

### N-11: Error-Messages Deutsch/Englisch gemischt (import.go) -- OFFEN

`backend/internal/api/import.go:81, 88, 97, 107`

### N-12: Doppelt definiertes User-Interface im Frontend -- OFFEN

`frontend/src/lib/api.ts:41-47` vs. `frontend/src/lib/stores/auth.svelte.ts:13-19`

### N-13: Unused reactive Variable `_autoLockTimeout` -- ERLEDIGT (Phase 3a, `9f2c607`)

`frontend/src/lib/stores/encryption.svelte.ts:27`

**Fix:** Variable entfernt.

### N-14: 7 veraltete TODO-Kommentare ("Phase 2"/"Phase 3") -- ERLEDIGT (Phase 3a, `9f2c607`)

**Fix:** Alle veralteten TODO-Kommentare aus `encryption.svelte.ts`, `e2e.ts` und `settings.svelte.ts` entfernt.

### N-15: Legacy-Konstanten `IS_TAURI`, `IS_ELECTRON`, `IS_DESKTOP` -- OFFEN

`frontend/src/lib/config.ts:42-44` -- als "Legacy" markiert, Consumer nutzen Funktionsversionen.

### N-16: Nicht-lokalisierte Strings -- OFFEN

- `frontend/src/routes/+layout.svelte:503` -- `'Sie haben ungespeicherte Aenderungen'`
- `frontend/src/routes/+layout.svelte:631` -- `'Laden...'`
- `frontend/src/lib/stores/notes.svelte.ts:169, 256, 426, 789, 959` -- Deutsche Fehlermeldungen hardcoded

### N-17: Magic Numbers ohne benannte Konstanten -- OFFEN

- `notes.svelte.ts:143` -- `limit: 1000` ("MVP LIMIT")
- `+layout.svelte:48` -- `window.innerWidth < 768` (Mobile-Breakpoint)
- `websocket.svelte.ts:172` -- `reconnectAttempts >= 10`
- `main.go:137` -- `jobs.NewJobManager(4)` (4 Worker)
- `main.go:153` -- `PruneAllVersions(100)` (100 Versionen)
- `recipes.go:358` -- `baseServings = 4`
- `recipes.go:403` -- `len(existing) >= 50` (Max Images)

### N-18: `@html` mit i18n-Strings ohne DOMPurify -- OFFEN

`frontend/src/routes/settings/encryption/+page.svelte` (12 Stellen), `TwoFactorDisable.svelte:98`

Aktuell nur Developer-kontrollierte Strings. Kein unmittelbares Risiko, aber Defense-in-Depth-Verstoß.

### N-19: CSP erlaubt `unsafe-inline` fuer Scripts -- OFFEN

`backend/internal/api/security.go:16-17` -- Benoetigt durch SvelteKit adapter-static. Langfristig auf Nonce-basierte CSP migrieren.

### N-20: `@ts-expect-error` statt Typ-Erweiterung -- OFFEN

`frontend/src/lib/components/Editor.svelte:287, 291` -- Besser: Custom Event Types in `app.d.ts`.

---

## 5. Statistik

### Nach Schweregrad

| Schweregrad | Gesamt | Erledigt | Offen |
|-------------|--------|----------|-------|
| KRITISCH | 6 | **6** | 0 |
| WICHTIG | 21 | **13** | 8 |
| NICE-TO-HAVE | 20 | **3** | 17 |
| **Gesamt** | **47** | **22** | **25** |

### Nach Kategorie

| Kategorie | Gesamt | Erledigt | Offen |
|-----------|--------|----------|-------|
| Sicherheit | 9 | 7 | 2 |
| Code-Duplikation | 11 | 7 | 4 |
| SOLID / God-Files | 6 | 0 | 6 |
| Error Handling | 5 | 4 | 1 |
| Bugs | 1 | 1 | 0 |
| TypeScript / Types | 3 | 2 | 1 |
| Performance | 3 | 1 | 2 |
| Testing | 2 | 0 | 2 |
| Wartbarkeit | 7 | 0 | 7 |

### Verifizierte positive Security Controls (Auszug)

| Control | Status |
|---------|--------|
| Parametrisierte SQL-Queries | PASS |
| JWT-Algorithmus-Enforcement (kein `alg:none`) | PASS |
| JWT-Secret Mindestlaenge 64 Zeichen | PASS |
| Refresh-Token-Hashing (SHA-256) | PASS |
| Konstant-Zeit bcrypt fuer nicht-existierende User | PASS |
| Cookie-Flags: HttpOnly, Secure, SameSite=Strict | PASS |
| CSRF Double-Submit mit Constant-Time Compare | PASS |
| File-Upload: Magic-Byte Detection + UUID-Dateinamen | PASS |
| Pfad-Traversal-Schutz (Upload + Export) | PASS |
| HMAC-SHA256 signierte Upload-URLs mit Ablauf | PASS |
| Request-Body-Limits (1MB/16MB) | PASS |
| Rate-Limiting auf allen Auth-Endpoints | PASS |
| Account-Lockout (IP + Global) | PASS |
| Docker: Non-Root, Read-Only FS, Cap-Drop ALL | PASS |
| DOMPurify auf allen gerenderten Markdown-Inhalten | PASS |
| API-Keys verschluesselt at-rest (AES-256-GCM) | PASS |
| PII gehasht in Logs | PASS |

---

## 6. Aktionsplan

### Erledigt

| # | Finding | Phase | Commit | Beschreibung |
|---|---------|-------|--------|--------------|
| 1 | K-1 | 1 | `c799968` | `sig` -> `signature` in `recipes.go` |
| 2 | K-2 | 1 | `c799968` | `getUserID`-Fehler in `admin.go` geprüeft |
| 3 | K-3 | 1 | `c799968` | `WS_URL` durch `getWsBaseUrl()` ersetzt |
| 4 | K-4 | 1 | `c799968` | CORS-Fatal-Check fuer Non-Dev verschaerft |
| 5 | K-5 | 1 | `c799968` | Gemini API-Key in Header statt URL |
| 6 | K-6 | 1 | `c799968` | Type-Assertions durch `getUserID()` ersetzt |
| 7 | W-4 | 2 | `7856b6b` | `cleanJSON` ins `llm`-Paket verschoben |
| 8 | W-5 | 2 | `7856b6b` | `validateLinks()` / `validateDueDates()` extrahiert |
| 9 | W-9 | 2 | `7856b6b` | Duplizierte Konstanten zusammengefuehrt |
| 10 | W-10/11/12 | 2 | `7856b6b` | Frontend-Helper: `extractUniqueLinks()`, `decryptNoteResponse()`, `assertOnlineForParanoidMode()` |
| 11 | W-14 | 2 | `7856b6b` | 83x `err.Error()` durch `respondInternalErr` ersetzt |
| 12 | W-16 | 2 | `7856b6b` | Debug-Logging entfernt |
| 13 | W-18 | 2 | `7856b6b` | Duplikat-Import entfernt |
| 14 | W-6 | 3a | `9f2c607` | API-Key-Management generalisiert (Factory Pattern) |
| 15 | W-15 | 3a | `9f2c607` | User-Search auf Username-only beschraenkt |
| 16 | N-3 | 3a | `9f2c607` | Sentinel Errors statt String-Match |
| 17 | N-4 | 3a | `9f2c607` | `respondJSON` Encode-Fehler geloggt |
| 18 | N-13/14 | 3a | `9f2c607` | Unused Variable + stale TODOs entfernt |
| 19 | W-7 | 3b | `8d067c5` | `checkRecipeWriteAccess/ReadAccess` Helper extrahiert |
| 20 | W-19 | 3b | `8d067c5` | `IsAvailable()` mit 5-Min-TTL-Cache |
| 21 | W-13 | 3b | `8d067c5` | Sensitive Crypto/KEK-Logs aus 8 Dateien entfernt |
| 22 | W-17 | 3b | `8d067c5` | FIDO2-Typen + `catch(err: unknown)` in settings/fido2 |

### Naechste Schritte (offen)

| # | Finding | Aufwand | Beschreibung |
|---|---------|---------|--------------|
| 23 | W-1 | 4 Std | `Server`-Struct aufbrechen (Services, RateLimiters, Config) |
| 24 | W-8 | 4 Std | LLM-Client-Duplikation zusammenfuehren |
| 25 | W-2, W-3 | 3-5 Tage | God-Files aufteilen (Backend + Frontend) |
| 26 | W-20/21 | 3-5 Tage | Testabdeckung fuer kritische Pakete verbessern |
| 27 | N-1-20 | 2-4 Std | Verbleibende Backlog-Items abarbeiten |
