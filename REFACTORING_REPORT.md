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
- Status-Update (2026-02-11): `make test-frontend` lief gruen; `src/test/e2e-feature.test.ts` inkl. Scenario 2 ist aktuell passing.

**Lint/Format**
- Go: `go vet` (CI/Make), `.golangci.yml` vorhanden
- Frontend: ESLint (flat config), Prettier, markdownlint
- Hooks: `lefthook.yml`
- `.editorconfig` vorhanden

**CI/CD**
- GitHub Actions: `ci.yml`, `quality.yml`, `security.yml`
- Forgejo Deploy Workflows: `.forgejo/workflows/deploy-staging.yml`, `.forgejo/workflows/deploy-production.yml`

### 0.3 Findings (Priorisiert)

Status-Hinweis: Die folgenden Punkte sind Baseline-Findings aus dem Re-Review. Der **aktuelle Status**
steht in den Phasen-Abschnitten bzw. in den Finding-Details weiter unten.

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

**Update**
- CORS-Fix (K-4): Strenger Default fuer leeres `XELANOTE_ENV` ist umgesetzt. In `backend/cmd/server/main.go` wird der Serverstart jetzt abgebrochen, wenn `XELANOTE_ENV` leer **und** `CORS_ALLOWED_ORIGINS` nicht gesetzt ist.

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
- Keine offenen Findings in Phase 2.
- Follow-up (optional): weitere Zerlegung von `frontend/src/lib/components/Editor.svelte` in Sub-Komponenten.

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
- **Type-Safety: History-Deserialisierung ohne unsichere Casts**
  - `frontend/src/lib/stores/history.svelte.ts` validiert persistierte Command-Daten strukturell statt blindem Cast.
  - `frontend/src/lib/stores/history.test.ts` deckt gueltige und fehlerhafte localStorage-Payloads ab.
- **Type-Safety: Store-LocalStorage Parse-Guards erweitert**
  - `frontend/src/lib/stores/autosave.svelte.ts` validiert AutoSave-Settings vor Anwendung.
  - `frontend/src/lib/stores/folders.svelte.ts` und `frontend/src/lib/stores/tree.svelte.ts` validieren Expanded-Path-Arrays.
  - `frontend/src/lib/stores/settings.svelte.ts` validiert boolesche Preferences vor dem Laden.
- **Type-Safety: Notes-Store Signaturen ohne `any` (Teilbereich)**
  - `frontend/src/lib/stores/notes/saver.ts`, `frontend/src/lib/stores/notes/creator.ts`, `frontend/src/lib/stores/notes/mutations.ts`, `frontend/src/lib/stores/notes/encryption-toggle.ts`
    nutzen jetzt konkrete API-/Crypto-Typen (`NotePayload`, `TaskEventPayload`, `RecipeMetadata`, `RecipeIngredient`, `EncryptedPayload`).
  - `frontend/src/lib/stores/notes/helpers.ts` enthaelt `parseEncryptionMetadata()` fuer validierte Metadata-Deserialisierung.
- **Type-Safety: Parse-Guards fuer Realtime/Index/Offline**
  - `frontend/src/lib/stores/websocket.svelte.ts` validiert WebSocket-Message-/Payload-Form vor Verarbeitung.
  - `frontend/src/lib/stores/search-index.svelte.ts` und `frontend/src/lib/offline/sync-manager.svelte.ts` verwenden validiertes Encryption-Metadata-Parsing.
  - `frontend/src/lib/stores/graph.svelte.ts` validiert persistiertes Layout aus localStorage.
  - Gemeinsamer Helper: `frontend/src/lib/stores/encryption-metadata.ts`.
- **Type-Safety: Weitere Store-Parse-Haertungen**
  - `frontend/src/lib/stores/auth.svelte.ts` validiert JWT-Payload-Struktur vor Claim-Auswertung.
  - `frontend/src/lib/stores/notes/task-events.ts` validiert persistierte Queue-Events aus `sessionStorage`.
  - `frontend/src/lib/stores/notes/loaders.ts` und `frontend/src/lib/stores/notes/remote-updates.ts` nutzen den zentralen Metadata-Parser.
- **Type-Safety: API-/Crypto-Parse-Haertungen**
  - `frontend/src/lib/api/ai.ts` validiert SSE-Event-Payloads (`token`, `cached`, `error`) vor Nutzung.
  - `frontend/src/lib/api/client.ts` nutzt defensives Request-Body-Parsing fuer Offline-Mutationspfade.
  - `frontend/src/lib/crypto/e2e.ts` validiert encrypted title payloads strukturell in `decryptTitle()`.
  - `frontend/src/lib/crypto/fido2.ts` ersetzt `as unknown as` durch Runtime-Guards fuer Serveroptionen und Browser-Credentials.
- **UI-Parse-Haertung (Drag&Drop + lokale Preferences)**
  - `frontend/src/routes/journal/+page.svelte` validiert den gespeicherten Collapse-Boolean aus `localStorage`.
  - `frontend/src/lib/components/FolderTree.svelte`, `frontend/src/lib/components/UnifiedTree.svelte` und
    `frontend/src/lib/components/sidebar/sidebar-dnd.ts` validieren Drag-Daten vor Move/Reorder-Operationen.
- **Lazy-Component-Loading ohne unsichere Double-Casts**
  - Neuer Helper: `frontend/src/lib/utils/lazy-component.ts`.
  - `frontend/src/routes/+layout.svelte`, `frontend/src/routes/note/[id]/+page.svelte`,
    `frontend/src/lib/components/RecipeEditor.svelte`, `frontend/src/lib/components/UnifiedTree.svelte`,
    `frontend/src/lib/editor/dialog-loaders.ts` nutzen jetzt validiertes Modul-Loading statt
    `as unknown as ComponentType`.
- **Verbleibende `as unknown as`-Stellen bereinigt**
  - `frontend/src/lib/components/GraphCanvas.svelte` validiert den `force-graph`-Factory-Export vor Nutzung.
  - `frontend/src/lib/desktop/electron-bridge.ts` nutzt eine typisierte `Window`-Erweiterung fuer `electronAPI`.
- **Frontend-Typecheck-Bereinigung**
  - `npm run typecheck` ist wieder gruen (0 Errors, 0 Warnings).
  - Fixes u.a. in `frontend/src/lib/api/client.ts`, `frontend/src/lib/crypto/fido2.ts`,
    `frontend/src/lib/stores/websocket.svelte.ts`, `frontend/src/routes/settings/+page.svelte`,
    `frontend/src/lib/routes/settings/tabs/AccountTab.svelte`, zugehoerige Tests.
- **Editor-Modularisierung (weiterer Schnitt)**
  - Neuer Helper: `frontend/src/lib/editor/editor-ui-actions.ts`.
  - `frontend/src/lib/components/Editor.svelte` nutzt ausgelagerte UI-Handler fuer
    More-Menu/Color-Picker/Markdown-Hilfe, Color-Tag-Insert und File-Input-Parsing.
- **Editor-Modularisierung (Split-Resize-Controller)**
  - `frontend/src/lib/editor/split-resize.ts` bietet jetzt `createSplitResizeController()`.
  - `frontend/src/lib/components/Editor.svelte` nutzt den Controller statt lokaler Wrapper-Funktionen.
- **Editor-Modularisierung (Find/Replace-State-Adapter)**
  - Neuer Helper: `frontend/src/lib/editor/find-replace-state.ts`.
  - `frontend/src/lib/editor/find-replace-ui.ts` exportiert `FindReplaceState`/`FindReplaceHandlers`.
  - `frontend/src/lib/components/Editor.svelte` nutzt `readFindReplaceState()`/`writeFindReplaceState()` statt mehrfacher manueller State-Assignments.
- **Editor-Modularisierung (Task/Image-Content-Handler)**
  - Neuer Helper: `frontend/src/lib/editor/editor-content-actions.ts`.
  - `frontend/src/lib/components/Editor.svelte` nutzt ausgelagerte Handler fuer Task-Reorder und Image-Resize.
- **Editor-Modularisierung (Editor-Mode-Readiness)**
  - Neuer Helper: `frontend/src/lib/editor/editor-mode.ts` mit `ensureEditorReady()`.
  - `frontend/src/lib/components/Editor.svelte` nutzt den Helper fuer Task-Insert, Indent und Outdent.
- **Backend: Fehlerbehandlung & Validierung gehaertet**
  - `backend/internal/api/journal.go` validiert `year`/`month` strikt (inkl. Range-Checks).
  - `backend/internal/api/notes_helpers.go`, `backend/internal/api/notes_crud.go`, `backend/internal/api/import.go` behandeln WS-JSON-Encode-Errors (loggen statt ignorieren).
  - `backend/internal/db/notes_crud.go`, `backend/internal/db/notes_list.go`, `backend/internal/db/notes_trash.go`, `backend/internal/db/notes_encryption.go`, `backend/internal/db/journal.go` pruefen RFC3339-Timestamps strikt (Parse-Errors werden retourniert).
  - `backend/internal/api/import.go` validiert File-Anzahl, leere Inhalte, Notiz-/Folder-Felder und begrenzt Error-Listen.
  - `backend/internal/service/notes_crud.go`, `backend/internal/service/notes_encryption.go` loggen Snapshot-Query-Fehler (und snapshoten konservativ).
  - `backend/internal/api/admin.go` behandelt Fehler beim Laden von User-Details mit klaren HTTP-Antworten.
- **Frontend Parse-Haertung (gezielte Follow-ups)**
  - `frontend/src/lib/stores/notes/encryption-toggle.ts` nutzt `parseRecipeContentPayload()` statt direktem `JSON.parse(... as ...)` fuer Recipe-Decrypt-Preprocessing.
  - `frontend/src/lib/api/ai.ts` validiert `stream_token` aus dem Prepare-Endpoint explizit vor Nutzung.

### Offen
- Keine offenen Findings in Phase 3.
- Follow-up (optional): verbleibende Parse-/Type-Haertungen in weniger kritischen Modulen.

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

- **Frontend Quality Gates ausgefuehrt**
  - `npm run lint` (nach Auto-Fix) sauber.
  - `npm run format` (Prettier) ausgefuehrt.
  - `make test-frontend` (Vitest) gruen.
  - `npm run typecheck` (svelte-check) gruen, Warnungen bereinigt.
  - Follow-up: verbleibende `simple-import-sort`/unused-var-Warnungen in Settings/Dialog-Panels bereinigt; `npm run lint` wieder gruen.
- **Playwright Follow-up (Status)**
  - `frontend/src/lib/components/ui/BaseDialog.svelte`: `script module` auf `lang="ts"` korrigiert (Vite/esbuild Parse-Fehler behoben).
  - Lokale Browser-Installation ist erfolgt (`PLAYWRIGHT_BROWSERS_PATH=./.playwright-browsers`).
  - `npm run test:e2e` im aktuellen Environment weiterhin blockiert: fehlende System-Library `libglib-2.0.so.0` fuer Chromium Headless.

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

- **DB Insert-ID Guardrails (ausgewaehlte Pfade)**
  - `backend/internal/db/helpers.go`: `validateLastInsertID()` eingefuehrt (positiv + int-overflow-safe).
  - `backend/internal/db/tags.go`, `backend/internal/db/templates.go`, `backend/internal/db/snippets.go` nutzen die Validierung nach `LastInsertId()`.
  - Testlauf: `go test -tags fts5 ./internal/db` gruen.
- **DB RowsAffected-Helper (ausgewaehlte Pfade)**
  - `backend/internal/db/helpers.go`: `ensureRowsAffected()` eingefuehrt.
  - `backend/internal/db/tags.go`, `backend/internal/db/templates.go`, `backend/internal/db/snippets.go` nutzen den Helper fuer konsistentes NotFound-Handling.
  - `backend/internal/db/auth.go` nutzt `ensureRowsAffected()` in User-Update-Pfaden und `validateLastInsertID()` bei `CreateUser()`.

### Offen
- Keine offenen Findings im Reliability-Pass.
- Follow-up (optional): verbleibende DB-Pfade schrittweise auf gemeinsame `LastInsertId`/`RowsAffected`-Helper vereinheitlichen.

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
1. Optionale weitere Modularisierung von `frontend/src/lib/components/Editor.svelte` (restliche lokale Handler/Effects).
2. Optionale End-to-End-Szenarien nachziehen (insb. Playwright-Suite ausserhalb der Vitest-E2E-Feature-Tests).

## Fortschritt

| Phase | Beschreibung | Commit | Status |
|-------|-------------|--------|--------|
| Phase 1 | Kritische Issues (K-1 bis K-6) | `c799968` | ERLEDIGT |
| Phase 2 | Strukturelles Refactoring (7 Tasks) | `7856b6b` | ERLEDIGT |
| Phase 3a | Code Quality (5 Tasks) | `9f2c607` | ERLEDIGT |
| Phase 3b | Code Quality (4 Tasks) | `8d067c5` | ERLEDIGT |
| Phase 4 | Testing & Documentation | -- | ERLEDIGT |
| Phase 5 | Linting & Formatting | -- | ERLEDIGT |

**Status-Update:** 47 von 47 Findings erledigt, 0 teilweise, 0 offen. Verbleibende Punkte sind als optionale Follow-ups markiert.

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

### K-4: CORS erlaubt jede Origin bei nicht-explizitem `XELANOTE_ENV` -- ERLEDIGT

**Datei:** `backend/internal/api/api.go:557-573, 88-96`

**Problem:** Fatal-Check greift nur bei `env == "production"`. Staging oder leerer `XELANOTE_ENV` laeuft mit vollständig permissivem CORS.

**Fix:** Strikter Guard in `backend/cmd/server/main.go`: Start wird mit `log.Fatal` abgebrochen, wenn
`XELANOTE_ENV` leer ist und `CORS_ALLOWED_ORIGINS` nicht gesetzt ist. Damit ist der Security-Default fuer
nicht-explizites Env nicht mehr permissiv.

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

#### W-2: God-Files im Backend -- ERLEDIGT (P1, 2026-02-10)

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

**Status:** NoteService + Note-DB + Server-Init aufgeteilt. API-Routen aus `api.go` in `routes.go` ausgelagert; JSON-Helpers in `response_json.go`.

**Update (2026-02-11): weitere God-Files im Backend zerlegt**

| Datei | Ergebnis |
|-------|----------|
| `backend/internal/db/recipes.go` | Aufgeteilt in: `recipes_models.go`, `recipes_notes.go`, `recipes_metadata.go`, `recipes_ingredients.go`, `recipes_data.go`, `recipes_images.go`, `recipes_collections.go`, `recipes_summaries.go`, `recipes_scaling.go`, `recipes_sharing.go` |
| `backend/internal/db/sharing.go` | Aufgeteilt in: `sharing_models.go`, `sharing_utils.go`, `sharing_notes.go`, `sharing_folders.go`, `sharing_placements.go`, `sharing_recipes.go`, `sharing_search.go` |
| `backend/internal/service/summarize.go` | Aufgeteilt in: `summarize_service.go`, `summarize_tags.go`, `summarize_links.go`, `summarize_spellcheck.go`, `summarize_ai_transform.go` |
| `backend/internal/api/recipes.go` | Aufgeteilt in: `recipes_types.go`, `recipes_handlers.go`, `recipes_collections.go`, `recipes_collection_shares.go`, `recipes_shared.go`, `recipes_images.go`, `recipes_images_signing.go` |
| `backend/internal/db/recipes_test.go` | Aufgeteilt in: `recipes_helpers_test.go`, `recipes_notes_test.go`, `recipes_metadata_test.go`, `recipes_ingredients_test.go`, `recipes_collections_test.go`, `recipes_scaling_test.go`, `recipes_data_test.go`, `recipes_encryption_test.go`, `recipes_validation_test.go`, `recipes_sharing_test.go`, `recipes_images_test.go` |
| `backend/internal/api/auth.go` | Aufgeteilt in: `auth_types.go`, `auth_helpers.go`, `auth_register.go`, `auth_login.go`, `auth_tokens.go`, `auth_user.go` |
| `backend/internal/api/sharing.go` | Aufgeteilt in: `sharing_types.go`, `sharing_notes.go`, `sharing_folders.go`, `sharing_placements.go`, `sharing_search.go` |
| `backend/internal/api/users.go` | Aufgeteilt in: `users_types.go`, `users_preferences.go`, `users_account.go`, `users_encryption.go`, `users_security.go`, `users_webauthn.go`, `users_apikeys.go` |
| `backend/internal/api/notes_ai.go` | Aufgeteilt in: `notes_ai_types.go`, `notes_ai_summary.go`, `notes_ai_suggest.go`, `notes_ai_format.go`, `notes_ai_transform.go`, `notes_ai_titles.go` |
| `backend/internal/api/notes_trash.go` | Aufgeteilt in: `notes_trash_list.go`, `notes_trash_actions.go` |
| `backend/internal/service/recipes.go` | Aufgeteilt in: `recipes_types.go`, `recipes_helpers.go`, `recipes_notes.go`, `recipes_images.go`, `recipes_collections.go`, `recipes_collections_owner.go` |
| `backend/internal/api/notes_misc.go` | Aufgeteilt in: `notes_misc_types.go`, `notes_misc_titles.go`, `notes_misc_reorder.go` |
| `backend/internal/api/notes_helpers.go` | Aufgeteilt in: `notes_helpers_types.go`, `notes_helpers_validate.go`, `notes_helpers_etag.go` |
| `backend/internal/api/notes_meta.go` | Aufgeteilt in: `notes_meta_types.go`, `notes_meta_rename.go`, `notes_meta_backlinks.go`, `notes_meta_color.go` |
| `backend/internal/service/user.go` | Aufgeteilt in: `user_types.go`, `user_preferences.go`, `user_account.go`, `user_recovery.go`, `user_webauthn.go`, `user_apikeys.go` |
| `backend/internal/db/folders.go` | Aufgeteilt in: `folders_models.go`, `folders_utils.go`, `folders_queries.go`, `folders_move.go`, `folders_delete.go`, `folders_rename.go`, `folders_reorder.go`, `folders_color.go`, `folders_ai.go`, `folders_encryption.go` |
| `backend/internal/db/preferences.go` | Aufgeteilt in: `preferences_types.go`, `preferences_core.go`, `preferences_encryption.go`, `preferences_security.go`, `preferences_webauthn.go`, `preferences_apikeys.go` |
| `backend/internal/api/folders.go` | Aufgeteilt in: `folders_types.go`, `folders_crud.go`, `folders_reorder.go`, `folders_color.go`, `folders_ai.go`, `folders_encryption.go` |
| `backend/internal/api/notes_crud.go` | Aufgeteilt in: `notes_crud_create.go`, `notes_crud_read.go`, `notes_crud_update.go`, `notes_crud_delete.go` |
| `backend/internal/db/admin.go` | Aufgeteilt in: `admin_types.go`, `admin_stats.go`, `admin_users.go`, `admin_metrics.go` |
| `backend/internal/service/notes_encryption.go` | Aufgeteilt in: `notes_encryption_create.go`, `notes_encryption_update.go`, `notes_encryption_decrypt.go`, `notes_encryption_batch.go` |

---

#### W-3: God-Files im Frontend -- ERLEDIGT

| Datei | Zeilen | Verantwortlichkeiten |
|-------|--------|---------------------|
| `frontend/src/lib/api.ts` | 27 | Re-Export-Layer (API-Module ausgelagert) |
| `frontend/src/routes/settings/+page.svelte` | 1204 | Tabs + Layout, Tab-Views ausgelagert |
| `frontend/src/lib/components/Editor.svelte` | 1135 | CM6 + UI-Rest, zentrale Flows ausgelagert |
| `frontend/src/lib/stores/notes.svelte.ts` | 757 | Store-Kern, Flows ausgelagert |
| `frontend/src/lib/components/Sidebar.svelte` | 1047 | Desktop/Mobile Layout, Helper ausgelagert |
| `frontend/src/routes/+layout.svelte` | 528 | Init/Guards ausgelagert |

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
- `frontend/src/lib/components/Editor.svelte`: Note-Actions (Export/Delete/Wikilink) nach `frontend/src/lib/editor/note-actions.ts` ausgelagert.
- `frontend/src/lib/components/Editor.svelte`: Save/Title/AutoSave/AI/Encryption-Handler nach `frontend/src/lib/editor/editor-actions.ts` ausgelagert.
- `frontend/src/lib/components/Editor.svelte`: CodeMirror-Init/Action nach `frontend/src/lib/editor/editor-init.ts` ausgelagert.
- `frontend/src/lib/components/Editor.svelte`: Task-Reorder + Image-Resize-Handler nach `frontend/src/lib/editor/editor-content-actions.ts` ausgelagert.
- `frontend/src/lib/components/Editor.svelte`: Edit-Mode/Readiness-Logik fuer Insert/Indent/Outdent nach `frontend/src/lib/editor/editor-mode.ts` ausgelagert.
- `frontend/src/lib/components/Sidebar.svelte`: Resize-Handler nach `frontend/src/lib/components/sidebar/sidebar-resize.ts` ausgelagert.
- `frontend/src/lib/components/Sidebar.svelte`: onMount-Init nach `frontend/src/lib/components/sidebar/sidebar-init.ts` ausgelagert.
- `frontend/src/lib/components/Sidebar.svelte`: Drag/Drop + Touch-Reorder nach `frontend/src/lib/components/sidebar/sidebar-dnd.ts` ausgelagert.
- `frontend/src/lib/components/Sidebar.svelte`: Create/Logout Actions nach `frontend/src/lib/components/sidebar/sidebar-actions.ts` ausgelagert.
- `frontend/src/lib/components/Sidebar.svelte`: Escape-Handler nach `frontend/src/lib/components/sidebar/sidebar-escape.ts` ausgelagert.
- `frontend/src/routes/settings/+page.svelte`: Claude/Gemini-API-Key-Flow nach `frontend/src/lib/routes/settings/ai-keys.ts` ausgelagert.
- `frontend/src/routes/settings/+page.svelte`: Migration-Stats-Logik nach `frontend/src/lib/routes/settings/migration-stats.ts` ausgelagert.
- `frontend/src/routes/settings/+page.svelte`: 2FA + Backup-Code-Flow nach `frontend/src/lib/routes/settings/two-factor.ts` ausgelagert.
- `frontend/src/routes/settings/+page.svelte`: Passwort-Change + Rewrap-Flow nach `frontend/src/lib/routes/settings/password-change.ts` ausgelagert.
- `frontend/src/routes/settings/+page.svelte`: Security-Preferences-Flow nach `frontend/src/lib/routes/settings/security-preferences.ts` ausgelagert.
- `frontend/src/routes/settings/+page.svelte`: Account-Email-Flow nach `frontend/src/lib/routes/settings/account-forms.ts` ausgelagert.
- `frontend/src/routes/settings/+page.svelte`: Import/Export-Flow nach `frontend/src/lib/routes/settings/import-export.ts` ausgelagert.
- `frontend/src/lib/stores/notes.svelte.ts`: Helper/Task-Queue nach `frontend/src/lib/stores/notes/helpers.ts` und `frontend/src/lib/stores/notes/task-events.ts` ausgelagert.
- `frontend/src/lib/stores/notes.svelte.ts`: Accessor-Layer nach `frontend/src/lib/stores/notes/accessors.ts` ausgelagert.
- `frontend/src/lib/stores/notes.svelte.ts`: Auto-Save nach `frontend/src/lib/stores/notes/auto-save.ts` ausgelagert.
- `frontend/src/lib/stores/notes.svelte.ts`: Create-Note-Flow nach `frontend/src/lib/stores/notes/creator.ts` ausgelagert.
- `frontend/src/lib/stores/notes.svelte.ts`: Delete/Move-Flow nach `frontend/src/lib/stores/notes/mutations.ts` ausgelagert.
- `frontend/src/lib/stores/notes.svelte.ts`: Encryption-Toggle nach `frontend/src/lib/stores/notes/encryption-toggle.ts` ausgelagert.
- `frontend/src/lib/stores/notes.svelte.ts`: Rename-Flow nach `frontend/src/lib/stores/notes/rename.ts` ausgelagert.
- `frontend/src/lib/stores/notes.svelte.ts`: WebSocket-Remote-Updates nach `frontend/src/lib/stores/notes/remote-updates.ts` ausgelagert.
- `frontend/src/lib/stores/notes.svelte.ts`: Note-State-Updates nach `frontend/src/lib/stores/notes/state-updates.ts` ausgelagert.
- `frontend/src/lib/stores/notes.svelte.ts`: Pending-Check fuer WebSocket-Updates nach `frontend/src/lib/stores/notes/remote-update-gate.ts` ausgelagert.
- `frontend/src/routes/+layout.svelte`: PWA-Update-Registrierung nach `frontend/src/lib/routes/layout/pwa.ts` ausgelagert.
- `frontend/src/routes/+layout.svelte`: App-Init-Flow nach `frontend/src/lib/routes/layout/initialize.ts` ausgelagert.
- `frontend/src/routes/+layout.svelte`: Interaction-Handler nach `frontend/src/lib/routes/layout/interactions.ts` ausgelagert.
- `frontend/src/routes/+layout.svelte`: Viewport/Keyboard-Handling nach `frontend/src/lib/routes/layout/viewport.ts` ausgelagert.
- `frontend/src/routes/+layout.svelte`: Navigation-Guard nach `frontend/src/lib/routes/layout/navigation-guards.ts` ausgelagert.
- `frontend/src/routes/+layout.svelte`: Auth-Redirect-Guard nach `frontend/src/lib/routes/layout/auth-guards.ts` ausgelagert.
- `frontend/src/routes/+layout.svelte`: beforeunload-Handler nach `frontend/src/lib/routes/layout/beforeunload.ts` ausgelagert.
- `frontend/src/routes/settings/+page.svelte`: Tabs in `frontend/src/lib/routes/settings/tabs/*` ausgelagert (Account/Security/AI).

**Status:** God-Files im Frontend deutlich reduziert (Tab-Views + Logik extrahiert).

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

#### W-8: Duplizierte LLM-Client-Implementierung -- ERLEDIGT (P0, 2026-02-10)

**Dateien:**
- `backend/internal/llm/claude.go` (352 Zeilen)
- `backend/internal/llm/gemini.go` (374 Zeilen)

4 nahezu identische HTTP-Request/Response-Handling-Bloecke (Generate + GenerateWithImage, je 2x).

**Fix:** Gemeinsamer HTTP/JSON-Helper `llm/doJSONRequest` eingefuehrt; Claude/Gemini nutzen denselben Pfad fuer Request/Response + Error-Parsing.

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

#### W-17: `any`-Typen an 15+ kritischen Stellen -- ERLEDIGT (Phase 3b, `8d067c5`)

| Datei | Status | Aenderung |
|-------|--------|-----------|
| `frontend/src/lib/crypto/fido2.ts:60,82,100,116` | ERLEDIGT | `ServerPublicKeyOptions`/`ServerCredentialDescriptor` Interfaces definiert |
| `frontend/src/lib/stores/settings.svelte.ts:124,155,198,277` | ERLEDIGT | `catch (err: any)` durch `catch (err: unknown)` mit `instanceof Error` Check ersetzt |
| `frontend/src/lib/crypto/fido2.ts:151,190` | ERLEDIGT | `catch (err: any)` durch `catch (err: unknown)` mit `instanceof DOMException` ersetzt |
| `frontend/src/lib/stores/websocket.svelte.ts:98` | ERLEDIGT | Payload auf `unknown` + Helper-Typen umgestellt |
| `frontend/src/lib/crypto/sodium.ts:5` | ERLEDIGT | libsodium Wrapper typisiert (`SodiumWrapper`) |
| `frontend/src/lib/editor/markdown.ts:460-462` | ERLEDIGT | markdown-it Renderer-Args typisiert |

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

#### W-20: Kritische Backend-Pakete ohne Tests -- ERLEDIGT

| Paket/Datei | Zeilen |
|-------------|--------|
| `backend/internal/utils/*` | 2 Dateien |
| `backend/internal/websocket/manager.go` | 1 Datei |

**Fortschritt (P1, 2026-02-10):** LLM-Client-Tests + ProviderRouter-Tests hinzugefuegt (Claude/Gemini: Generate, Error-Parsing, Cache, Image-Validation; Router: Provider-Auswahl, Fallback, Invalidation). Import-API Tests hinzugefuegt (empty files, skip/insert, preserve structure + dedup). API-Key Crypto Tests hinzugefuegt (roundtrip, no secret, invalid ciphertext, validation, mask). Cache/Jobs Tests hinzugefuegt (TTL, prefix delete, submit/execute, failure paths, cleanup). Export-API Test hinzugefuegt (ZIP-Response + Headers). Summarize-Service Tests hinzugefuegt (idempotent summary, encrypted guard, encrypted store, tag/link parsing). Recipe-Service Tests hinzugefuegt (feature flag, encryption guard, share permissions, ingredient validation, image URL validation). Recipe-Suggestions Tests hinzugefuegt (helper funcs, defaults/validation for generated recipes).
**Fix:** Utils-Pfad/Title-Helper mit Table-Tests abgedeckt; WebSocket-Manager Tests fuer Register/Broadcast/Unregister/Stop hinzugefuegt.

**Tests:** `go test -tags fts5 ./...` (2026-02-10) OK.

---

#### W-21: Kritische Frontend-Stores ohne Tests -- ERLEDIGT (P1, 2026-02-10)

**Fortschritt (P1, 2026-02-10):** WebSocket-Store Tests hinzugefuegt (connect, note.deleted handling, reconnect). Auth-Store Tests hinzugefuegt (setAuth, updateTokens-Guard, logout resets). Encryption-Store Tests hinzugefuegt (encrypt/decrypt, settings, restore error handling, security level update). Settings-Store Tests hinzugefuegt (load/save, mobile fallback, changeEmail mapping, password rewrap). Notes-Store Tests hinzugefuegt (create wiring, content update + dirty, AI toggle, clear resets, temp-id replacement). API-Client Tests hinzugefuegt (auth/csrf headers, 204 handling, offline enqueue, refresh retry, offline rejection).

---

## 4. NICE-TO-HAVE -- BACKLOG

### N-1: Cache und WebSocket-Manager ohne Shutdown-Mechanismus -- ERLEDIGT (P0, 2026-02-10)

**Fix:** `Cache.Close()` stoppt Cleanup-Goroutine; WebSocket-Manager hat `Stop()` mit `done`-Signal + Connection-Cleanup.

### N-2: JobManager-Jobs werden nie aufgeraeumt (Memory Leak) -- ERLEDIGT (P0, 2026-02-10)

**Fix:** Cleanup-Loop entfernt abgeschlossene Jobs nach Retention-Window (24h).

### N-3: Error-Vergleich via String-Match statt Sentinel Errors -- ERLEDIGT (Phase 3a, `9f2c607`)

`backend/internal/api/admin.go` -- `err.Error() == "cannot demote yourself"` / `"cannot delete yourself"`

**Fix:** `ErrSelfDemotion` und `ErrSelfDeletion` Sentinel Errors in `service/admin.go` definiert. `admin.go` nutzt jetzt `errors.Is()`.

### N-4: `respondJSON` ignoriert `json.Encode`-Fehler -- ERLEDIGT (Phase 3a, `9f2c607`)

`backend/internal/api/api.go`

**Fix:** Encode-Fehler wird jetzt via `slog.Error()` geloggt.

### N-5: Export laedt alle Notizen in den Speicher -- ERLEDIGT (P1, 2026-02-10)

**Fix:** Export streamt Notes seitenweise in ZIP statt alle Notizen im Speicher zu halten.

### N-6: Deprecated `NormalizeTitle` in db.go -- ERLEDIGT (P1, 2026-02-10)

**Fix:** Wrapper entfernt, direkte Nutzung von `utils.NormalizeTitle` in DB-Paket (inkl. Tests).

### N-7: Frontmatter-Tags werden beim Import nicht verwendet -- ERLEDIGT (P1, 2026-02-10)

**Fix:** Frontmatter-Tags werden beim Import auf die Notiz gesetzt (inkl. Tests).

### N-8: Migrations-Nummern haben Luecken (025 -> 028) -- ERLEDIGT (P1, 2026-02-10)

**Fix:** Fehlende Migrationen `026_notes_order_index.sql` und `027_graph_indexes.sql` in die Migrationsliste aufgenommen.

### N-9: Dupliziertes Null-zu-leeres-Slice Pattern -- ERLEDIGT (P1, 2026-02-10)

**Fix:** Generische `ensureSlice()`-Helper eingefuehrt, alle betroffenen API-Responses vereinheitlicht (search/tags/recipes/notes).

### N-10: SetRecipeService Post-Init Pattern -- ERLEDIGT (P1, 2026-02-10)

**Fix:** Recipe-Services direkt in `ServerConfig` aufgenommen; Post-Init Setter entfernt.

### N-11: Error-Messages Deutsch/Englisch gemischt (import.go) -- ERLEDIGT (P1, 2026-02-10)

**Fix:** Import-Errors auf konsistente englische Meldungen umgestellt.

### N-12: Doppelt definiertes User-Interface im Frontend -- ERLEDIGT (P1, 2026-02-10)

**Fix:** `auth.svelte.ts` nutzt jetzt das User-Type aus `api` (Alias).

### N-13: Unused reactive Variable `_autoLockTimeout` -- ERLEDIGT (Phase 3a, `9f2c607`)

`frontend/src/lib/stores/encryption.svelte.ts:27`

**Fix:** Variable entfernt.

### N-14: 7 veraltete TODO-Kommentare ("Phase 2"/"Phase 3") -- ERLEDIGT (Phase 3a, `9f2c607`)

**Fix:** Alle veralteten TODO-Kommentare aus `encryption.svelte.ts`, `e2e.ts` und `settings.svelte.ts` entfernt.

### N-15: Legacy-Konstanten `IS_TAURI`, `IS_ELECTRON`, `IS_DESKTOP` -- ERLEDIGT (P1, 2026-02-10)

**Fix:** Legacy-Exports entfernt (Funktionsversionen bleiben).

### N-16: Nicht-lokalisierte Strings -- ERLEDIGT (P1, 2026-02-10)

**Fix:** Strings lokalisiert (Layout + Notes/Aautosave-Messages) und i18n-Keys ergaenzt.

### N-17: Magic Numbers ohne benannte Konstanten -- ERLEDIGT (P1, 2026-02-10)

**Fix:** Benannte Konstanten eingefuehrt (Notes-Limit, Mobile-Breakpoint, WS-Reconnect, Job-Worker, Version-Pruning, Recipe-Defaults, Max-Images).

### N-18: `@html` mit i18n-Strings ohne DOMPurify -- ERLEDIGT (P1, 2026-02-10)

**Fix:** DOMPurify-Sanitizing fuer i18n-`@html` in Settings/TwoFactorDisable hinzugefuegt.

### N-19: CSP erlaubt `unsafe-inline` fuer Scripts -- ERLEDIGT (P1, 2026-02-10)

**Fix:** `unsafe-inline` aus `script-src` entfernt; CAPTCHA-Page nutzt CSP-Nonce fuer Inline-Script/Style.

### N-20: `@ts-expect-error` statt Typ-Erweiterung -- ERLEDIGT (P1, 2026-02-10)

**Fix:** `DocumentEventMap` fuer `spell-check-replace` erweitert, `@ts-expect-error` entfernt.

---

## 5. Statistik

### Nach Schweregrad

| Schweregrad | Gesamt | Erledigt | Teilweise | In Arbeit | Offen |
|-------------|--------|----------|-----------|----------|-------|
| KRITISCH | 6 | **5** | **1** | 0 | 0 |
| WICHTIG | 21 | **21** | 0 | 0 | 0 |
| NICE-TO-HAVE | 20 | **20** | 0 | 0 | 0 |
| **Gesamt** | **47** | **46** | **1** | 0 | 0 |

### Nach Kategorie

Legende (Mapping-Auszug):
- `K-1/K-3/K-6` -> Bugs
- `K-2/K-4/K-5` -> Sicherheit
- `W-1/W-2/W-3` -> SOLID / God-Files
- `W-4` bis `W-12` -> Code-Duplikation
- `W-13` bis `W-16` -> Sicherheit
- `W-17` -> TypeScript / Types
- `W-18` -> Wartbarkeit
- `W-19` -> Performance
- `W-20/W-21` -> Testing
- `N-1/N-2/N-5` -> Performance
- `N-3/N-4` -> Error Handling
- `N-6` bis `N-17` -> Wartbarkeit
- `N-18/N-19` -> Sicherheit
- `N-20` -> TypeScript / Types

| Kategorie | Gesamt | Erledigt | Teilweise | In Arbeit | Offen |
|-----------|--------|----------|-----------|----------|-------|
| Sicherheit | 9 | **8** | **1** | 0 | 0 |
| Code-Duplikation | 9 | **9** | 0 | 0 | 0 |
| SOLID / God-Files | 3 | **3** | 0 | 0 | 0 |
| Error Handling | 2 | **2** | 0 | 0 | 0 |
| Bugs | 3 | **3** | 0 | 0 | 0 |
| TypeScript / Types | 2 | **2** | 0 | 0 | 0 |
| Performance | 4 | **4** | 0 | 0 | 0 |
| Testing | 2 | **2** | 0 | 0 | 0 |
| Wartbarkeit | 13 | **13** | 0 | 0 | 0 |

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
| 4 | K-4 | 1 | `c799968` + Follow-up | CORS-Default gehaertet: bei leerem `XELANOTE_ENV` ohne `CORS_ALLOWED_ORIGINS` bricht der Start ab |
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
| 22 | W-17 | 3b | `8d067c5` | FIDO2-Typen + `catch(err: unknown)` sowie Typisierung in sodium/websocket/markdown |
| 23 | W-8 | P0 | -- | LLM-HTTP/JSON-Handling konsolidiert |
| 24 | N-1 | P0 | -- | Cache/WebSocket-Shutdown implementiert |
| 25 | N-2 | P0 | -- | Job-Cleanup mit Retention |
| 26 | N-5 | P1 | -- | Export streamt Notes seitenweise in ZIP |

### Naechste Schritte (historisch, bereits umgesetzt)

| # | Finding | Aufwand | Beschreibung |
|---|---------|---------|--------------|
| 27 | W-2, W-3 | 3-5 Tage | God-Files aufteilen (Backend + Frontend) - umgesetzt |
| 28 | W-20 | 3-5 Tage | Testabdeckung fuer kritische Pakete verbessern - umgesetzt |
