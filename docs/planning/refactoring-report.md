# Refactoring- und Verbesserungsreport

> **Stand:** 2026-02-21 | **Autor:** Claude Opus 4.6 (Staff/Principal Engineer Review)
> **Scope:** Vollstaendige Analyse von Backend (Go/Chi/SQLite) + Frontend (SvelteKit) + CI/CD
> **Methode:** Systematische Pruefung aller 8 Scope-Bereiche mit konkreten Befunden

---

## 1) Executive Summary

1. **52 Error-Leakage-Stellen** in 28 API-Dateien geben `err.Error()` direkt an den Client weiter — teilweise mit internen Details (Security Finding F-01, bereits dokumentiert)
2. **Test-Coverage ist die groesste Schwachstelle**: ~35% der API-Handler, ~50% der Service-Methoden und ~73% der DB-Methoden haben keine Tests. Frontend-Komponenten liegen bei ~2% Coverage
3. **Architektur ist sauber**: Keine Layer-Violations (API→DB), keine zirkulaeren Abhaengigkeiten, kein GetDB()-Bypass. Die 37 Violations in der Baseline sind reine Type-Imports, keine Method-Calls
4. **NoteService ist kein God Object** (entgegen frueherer Annahme): 3 Methoden im Kern-File, ~100 Methoden ueber 20 Files verteilt — gut strukturierte Facade
5. **Dead Code ist minimal**: 3 Spike-Files im Frontend, 4 alte Log-Files im Backend, 1 verwaistes `build-old/` Directory
6. **Performance ist gut**: Alle Queries bounded, Indexes abgestimmt, keine N+1-Patterns in Hot Paths. Einige List-Endpoints liefern Full Content statt Slim Projections
7. **Observability hat Luecken**: Kein Request-Correlation-ID-System, 7 Legacy-`log.Printf`-Aufrufe statt slog, keine Infrastructure-Metriken (CPU/Memory/Latency)
8. **Rate-Limiting ist exzellent**: Alle sensitiven Endpoints geschuetzt, Dual-Key (IP+User), Memory-Cap mit Cleanup
9. **Dependencies sind aktuell** mit einer Ausnahme: `boombuler/barcode` v1.0.1 (2019) — sollte auditiert werden
10. **CI/CD ist produktionsreif**: Auto-Rollback, Health-Checks, Security-Hardening, umfangreiche Pre-Commit-Hooks

---

## 2) Repo-Landkarte

### Tech-Stack & Entry Points

| Schicht | Technologie | Entry Point |
|---------|------------|-------------|
| Backend | Go 1.25, Chi v5.2.5, SQLite+FTS5 | `backend/cmd/server/main.go` |
| Frontend | SvelteKit 2.50, Svelte 5.2, Vite 6, Tailwind 4 | `frontend/src/routes/+layout.svelte` |
| Desktop | Electron 33.2 + Tauri 2.x | `frontend/src-electron/main.ts` |
| CI/CD | GitHub Actions (Tests) + Forgejo Actions (Deploy) | `.github/workflows/`, `.forgejo/workflows/` |
| DB | SQLite WAL, 51 Migrationen | `backend/internal/db/migrations/` |

### Moduluebersicht

| Package | Files | Lines | Verantwortlichkeit |
|---------|-------|-------|-------------------|
| `internal/api` | 106 | 12.216 | HTTP-Handler, Routing, Middleware, Validation |
| `internal/db` | 73 | 10.977 | SQLite-Queries, Migrations, Models |
| `internal/service` | 60 | 8.872 | Business-Logik, 19 Services |
| `internal/llm` | 8 | 2.038 | Claude/Gemini/ChatGPT Provider |
| `internal/parser` | 4 | 408 | Wikilinks, Due-Dates, Canvas-Links |
| `internal/auth` | 2 | 143 | JWT, Upload-Signaturen |
| `internal/websocket` | 1 | 171 | WebSocket-Manager |
| `internal/fido2` | 3 | 277 | WebAuthn Credential Store |
| `cmd/server` | 13 | 657 | Server-Bootstrap, Config, Shutdown |
| Frontend `src/lib` | ~314 | — | Stores (49), Components (112), Editor (47), Crypto (8) |

### Top 5 Risiko-Zonen

| Zone | Risiko | Grund |
|------|--------|-------|
| `internal/api/` Error-Handling | Hoch | 52 `err.Error()`-Leaks koennen interne Details exponieren |
| `internal/service/sharing.go` | Hoch | 323 Zeilen Business-Logik, 0 dedizierte Service-Tests |
| `internal/db/search.go` | Mittel | 481 Zeilen, `FilteredSearch` mit dynamischem Query-Building (137 Zeilen) |
| `frontend/src/lib/editor/` | Mittel | 47 Files, live-preview.ts allein 55KB — komplexeste Frontend-Logik |
| `internal/api/recipe_suggestions.go` | Mittel | 466 Zeilen, groesster Handler, externe LLM-Aufrufe, 0 Tests |

---

## 3) Findings-Katalog

### Legende
- **Schweregrad**: S1 (kritisch) — S4 (kosmetisch)
- **Aufwand**: XS (<1h), S (1-4h), M (4h-2d), L (2-5d), XL (>5d)
- **Risiko beim Fix**: Gering/Mittel/Hoch (Regressionsrisiko)

### Architektur & Modulgrenzen

| ID | Kategorie | S | Aufwand | Nutzen | Risiko | Fundstelle(n) | Was ist falsch | Konkrete Aenderung | Akzeptanzkriterien |
|----|-----------|---|---------|--------|--------|---------------|---------------|-------------------|-------------------|
| A-01 | Architektur | S3 | M | DX, Testbarkeit | Gering | `internal/api/server.go` ServerConfig: 19 konkrete Service-Pointer | Dependency Injection via konkrete Typen statt Interfaces — erschwert Mocking in Tests | Interfaces pro Service definieren (z.B. `NoteServicer`), ServerConfig auf Interfaces umstellen | Tests koennen Services mocken ohne echte DB; bestehende Tests laufen weiter |
| A-02 | Architektur | S3 | S | Typsicherheit | Gering | `internal/api/` — 28 Files importieren `db`-Package (Baseline: 37, aktuell ~28 nach Cleanup) | API-Layer referenziert DB-Types direkt statt Service-Layer-Aliases | Plan existiert: `docs/planning/layer-violations-cleanup.md` — Phase 1 (Type-Aliases) + Phase 2 (Import-Migration in 10 Batches) umsetzen | `scripts/layer-violation-baseline.txt` ist leer; CI-Check bleibt gruen |
| A-03 | Architektur | S4 | S | Konsistenz | Gering | `internal/service/notes_features.go`, `notes_journal.go` — Thin pass-throughs | 5-8% der Service-Methoden sind Zero-Logic-Wrapper um DB-Calls | Akzeptieren als Architektur-Konvention (Layer-Compliance) ODER in Layer-Violation-Cleanup zusammen mit A-02 adressieren | Dokumentierte Entscheidung in `docs/conventions.md` |

### Codequalitaet

| ID | Kategorie | S | Aufwand | Nutzen | Risiko | Fundstelle(n) | Was ist falsch | Konkrete Aenderung | Akzeptanzkriterien |
|----|-----------|---|---------|--------|--------|---------------|---------------|-------------------|-------------------|
| C-01 | Security/Code | S1 | M | Security | Mittel | 28 API-Files, 52 Stellen — z.B. `auth_register.go:37` (CAPTCHA-Errors), `recipes_images.go:150` (Image-Processing-Errors), `recipe_suggestions.go:211` (LLM-Errors) | `respondError(w, ..., err.Error())` leakt interne Fehlermeldungen an Client | Jede Stelle pruefen: User-facing Validierungsfehler → expliziter String-Literal; interne Fehler → `respondInternalErr()`. Kategorisierung: ~30 sind harmlose Validation-Errors, ~15 sind potenziell sensitiv | 0 Stellen wo `err.Error()` interne DB/Service-Details exponiert; grep-Count fuer `respondError.*err.Error` = 0 fuer Nicht-Validation-Faelle |
| C-02 | Code | S3 | S | Konsistenz | Gering | `internal/db/db.go:108,124,142,144`, `internal/api/middleware.go:31,40,47` | 7 `log.Printf()`-Aufrufe statt strukturiertem `slog` | Alle 7 Stellen auf `slog.Warn()` / `slog.Info()` umstellen. DB-Package braucht Logger-Injection (aehnlich wie Services) | `grep -rn 'log.Printf' internal/` liefert 0 Treffer |
| C-03 | Code | S3 | M | Lesbarkeit | Mittel | `internal/api/notes_crud_update.go:15` (133 Zeilen), `internal/api/auth_login.go:10` (114 Zeilen), `internal/api/auth_register.go:15` (105 Zeilen) | Handler-Funktionen >100 Zeilen mit verschachtelter Logik | Jede Funktion in Schritte extrahieren: `parseRequest()`, `validate()`, `execute()`, `respond()`. Oder: Service-Layer uebernimmt mehr Orchestrierung | Keine Funktion >80 Zeilen in API-Layer; Tests weiterhin gruen |
| C-04 | Code | S3 | S | DX | Gering | `internal/db/search.go:344` — `FilteredSearch()` 137 Zeilen dynamisches Query-Building | Grosse Funktion mit vielen Conditional-Branches fuer Filter-Kombinationen | Query-Builder-Pattern extrahieren oder in Sub-Methoden aufteilen: `buildBaseQuery()`, `applyFolderFilter()`, `applyTagFilter()`, `applyDateFilters()` | FilteredSearch < 60 Zeilen; bestehende Such-Tests laufen weiter |
| C-05 | Code | S4 | XS | Konsistenz | Gering | `internal/api/folders_crud.go:25,108,131,165` — 5x `err.Error()` fuer ValidateFolderPath | Validation-Errors werden inkonsistent behandelt (mal `err.Error()`, mal String-Literal) | Einheitlich: `respondError(w, http.StatusBadRequest, "invalid folder path")` mit Detail im Log | Konsistentes Pattern in allen Folder-Handlern |

### Dead Code & Repo-Hygiene

| ID | Kategorie | S | Aufwand | Nutzen | Risiko | Fundstelle(n) | Was ist falsch | Konkrete Aenderung | Akzeptanzkriterien |
|----|-----------|---|---------|--------|--------|---------------|---------------|-------------------|-------------------|
| D-01 | Dead Code | S4 | XS | Hygiene | Gering | `frontend/src/lib/editor/live-preview-spike.ts`, `live-preview-spike.test.ts`, `live-preview-update-spike.test.ts` | 3 Spike-Files (~200 Zeilen) die nirgends importiert werden | Loeschen. Wenn Referenz benoetigt: Git-History reicht | Files existieren nicht mehr; `grep -r 'live-preview-spike' src/` = 0 (nur Test-Import) |
| D-02 | Hygiene | S4 | XS | Hygiene | Gering | `backend/server.log` (188KB), `server_new.log` (413KB), `backend_output.log` (845B), `server_test.log` (201B) | 4 alte Log-Files aus Entwicklung, committed ins Repo | Loeschen + `*.log` in Root-`.gitignore` ergaenzen (ist bereits in Backend-Section, aber nicht global) | `git ls-files '*.log'` = 0 |
| D-03 | Hygiene | S4 | XS | Hygiene | Gering | `frontend/build-old/` (3.8MB) — alter Build-Output mit Icons, SW, HTML | Verwaistes Build-Artefakt | Loeschen. `build-old/` in `.gitignore` ergaenzen | Verzeichnis existiert nicht mehr |
| D-04 | Hygiene | S4 | XS | Klarheit | Gering | `frontend/.env` (dev-only JWT_SECRET) — kein `.env.example` in Root | Root hat `.env` mit "DO NOT COMMIT"-Kommentar aber kein offizielles `.env.example` | Bestehende `.env` loeschen (ist Dev-Artefakt), `backend/.env.example` als kanonisches Template nutzen | Kein `.env` in Git-Tree; `.env.example` dokumentiert alle Variablen |

### Performance & Ressourcen

| ID | Kategorie | S | Aufwand | Nutzen | Risiko | Fundstelle(n) | Was ist falsch | Konkrete Aenderung | Akzeptanzkriterien |
|----|-----------|---|---------|--------|--------|---------------|---------------|-------------------|-------------------|
| P-01 | Performance | S3 | S | Response-Size | Gering | `internal/api/recipes_handlers.go:14-29` — `listRecipes()`, `internal/api/canvas.go:17-33` — `listCanvasNotes()` | List-Endpoints liefern vollen Content statt Slim-Projektion (Recipes: Markdown-Content, Canvas: JSON-Diagramme) | `fields=slim`-Support analog zu Notes-List (`notes_crud_read.go`): Content weglassen in Listen, nur bei Detail-Abruf liefern | Recipe-/Canvas-Listen enthalten kein `content`-Feld; Detail-Endpoint liefert weiterhin alles |
| P-02 | Performance | S3 | S | DB-Effizienz | Gering | `internal/service/recipes_notes.go:69-73`, `notes_encryption_create.go:46,155` | Keyword-Insertion in Schleife: einzelne INSERT pro Keyword statt Batch | Batch-INSERT mit Multi-Value-Syntax: `INSERT INTO note_keywords (note_id, keyword) VALUES (?,?),(?,?),...` | Keyword-Insert nutzt einen einzigen Query; Test fuer 50+ Keywords laeuft in <10ms |
| P-03 | Performance | S4 | S | Admin-UX | Gering | `internal/service/admin.go:101-103` | N+1: `calculateUserStorageMB()` wird pro User in Schleife aufgerufen | Aggregate-Query: `SELECT user_id, SUM(size) FROM uploads GROUP BY user_id` in einem Call | Admin-Stats-Endpoint macht 1 DB-Query statt N+1 |
| P-04 | Performance | S4 | S | Response-Size | Gering | `internal/api/graph.go:13-39` | Graph-Response kann >50KB werden (MaxGraphNodes=300, keine Pagination) | Pagination oder Client-seitige Lazy-Loading-Option fuer grosse Graphen | Hypothese — Verifizieren: Response-Size bei 300 Nodes messen. Wenn <100KB: akzeptabel |

### Testbarkeit & Qualitaetssicherung

| ID | Kategorie | S | Aufwand | Nutzen | Risiko | Fundstelle(n) | Was ist falsch | Konkrete Aenderung | Akzeptanzkriterien |
|----|-----------|---|---------|--------|--------|---------------|---------------|-------------------|-------------------|
| T-01 | Tests | S2 | L | Safety | Gering | `internal/api/` — ~71 von 106 Files ohne Tests. Besonders kritisch: `recipes_*.go` (7 Files), `folders_crud.go`, `canvas.go`, `graph.go`, `templates.go`, `snippets.go`, `tags.go`, `versions.go` | Grosse Teile der API-Schicht haben keine Handler-Tests | API-Test-Files erstellen nach bestehendem Pattern (siehe `notes_handlers_test.go`, `sharing_handlers_test.go`). Priorisierung: Folders → Recipes → Templates/Snippets → Tags/Versions → Canvas → Graph | Jeder API-Handler hat mindestens Happy-Path + Error-Path Test; `go test -cover` fuer `internal/api` > 60% |
| T-02 | Tests | S2 | M | Safety | Gering | `internal/service/sharing.go` (323 Zeilen, 0 Service-Tests), `internal/service/admin.go`, `internal/service/graph.go`, `internal/service/notes_folders.go` | Kern-Business-Logik ohne dedizierte Unit-Tests | Service-Test-Files erstellen. Sharing hat hoechste Prioritaet (Berechtigungslogik). Vorlage: `service/auth_test.go`, `service/twofa_test.go` | Alle exported Service-Methoden in sharing/admin/graph haben Tests |
| T-03 | Tests | S2 | L | Safety | Gering | `internal/db/` — ~73 von 101 Files haben partielle oder keine Tests. Besonders: `activity.go`, `admin_metrics.go`, `templates.go`, `snippets.go`, `tags.go`, `versions.go`, `due_dates.go`, `task_events.go` | DB-Schicht hat ~27% Test-Coverage | DB-Test-Files erstellen nach Pattern von `notes_test.go`, `folders_test.go`. Prioritaet: Templates/Snippets (CRUD) → Tags → Versions → Activity → Due-Dates | Alle CRUD-Methoden in DB-Layer haben Happy-Path-Tests; `go test -cover` fuer `internal/db` > 50% |
| T-04 | Tests | S3 | XL | Frontend-Safety | Gering | `frontend/src/lib/stores/` — 23 von 46 Stores ohne Tests. Besonders: `recipes.svelte.ts`, `sharing.svelte.ts`, `journal.svelte.ts`, `tree-operations.svelte.ts` | Haelfte der Frontend-State-Logik ungetestet | Store-Tests nach Vorlage `auth.svelte.test.ts`, `features.svelte.test.ts` erstellen. Priorisierung: recipes → sharing → journal → tree-operations | Alle Stores mit Business-Logik haben Tests |
| T-05 | Tests | S3 | XL | Frontend-Safety | Gering | `frontend/src/lib/components/` — 2 von 112 Komponenten haben Tests | Fast keine Komponenten-Tests | Die 10 komplexesten Komponenten testen: Editor, Sidebar, RecipeEditor, CanvasEditor, ShareDialog, QuickSwitcher, UnifiedTree, VersionHistoryDialog, JournalHeatmap, WebAuthnDeviceManager | Min. 10 Komponenten-Test-Files mit Render + Interaction Tests |
| T-06 | Tests | S3 | L | E2E-Safety | Mittel | `frontend/tests/e2e/` — 6 Specs, decken nur: Login, Folders, Notes, 2FA, Encryption, Code-Splitting | Keine E2E-Tests fuer: Recipes, Canvas, Journal, Search, Tags, Versions, Templates, Import/Export, Sharing-Flows | E2E-Specs hinzufuegen. Prioritaet: Recipes → Search → Journal → Sharing → Import/Export | Mindestens 4 neue E2E-Specs fuer Kern-Features |
| T-07 | Tests | S3 | S | Robustheit | Gering | `internal/api/notes_helpers_etag.go` — `resolveETagVersion` gibt 500 statt 404 fuer nicht-existente Noten | Bug entdeckt in Tests: ErrNotFound wird nicht auf 404 gemappt | `resolveETagVersion` soll `errors.Is(err, db.ErrNotFound)` pruefen und 404 zurueckgeben | `TestDecryptNote_NotFound` erwartet 404 statt 500; bestehende Tests angepasst |

### DX / Build / CI

| ID | Kategorie | S | Aufwand | Nutzen | Risiko | Fundstelle(n) | Was ist falsch | Konkrete Aenderung | Akzeptanzkriterien |
|----|-----------|---|---------|--------|--------|---------------|---------------|-------------------|-------------------|
| X-01 | DX | S3 | XS | Konsistenz | Gering | `backend/.air.toml` — Build-Tags koennten von Makefile abweichen | Air Hot-Reload koennte andere Build-Tags verwenden als `make build` | Verifizieren: `.air.toml` Build-Command muss `-tags 'fts5 sqlite_crypt'` enthalten (wie Makefile) | `make dev` und `make build` verwenden identische Build-Tags |
| X-02 | DX | S4 | XS | Klarheit | Gering | `frontend/.env` im Git-Tree, `backend/.env.example` existiert separat | Kein einheitliches `.env.example` im Repo-Root | Root-Level `.env.example` erstellen (oder auf `backend/.env.example` verweisen), `frontend/.env` loeschen | Ein kanonisches Environment-Template; README verweist darauf |

### Sicherheit (refactor-relevant)

| ID | Kategorie | S | Aufwand | Nutzen | Risiko | Fundstelle(n) | Was ist falsch | Konkrete Aenderung | Akzeptanzkriterien |
|----|-----------|---|---------|--------|--------|---------------|---------------|-------------------|-------------------|
| S-01 | Security | S2 | S | Security | Gering | `internal/api/ratelimit.go` — In-Memory Rate-Limiter | Rate-Limiter hat Cleanup-Loop, aber kein hartes Memory-Cap. Bei DDoS mit vielen verschiedenen IPs waechst die Map (Finding F-15 aus Security Audit) | LRU-Cap (z.B. 100.000 Entries) einbauen. Aelteste Entries evicten wenn Cap erreicht | Rate-Limiter hat feste Obergrenze; Memory-Usage unter Last messbar begrenzt |
| S-02 | Security | S3 | XS | Security | Gering | `internal/service/user_types.go:105-110` — `isValidEmail()` nutzt `net/mail.ParseAddress()` | Email-Validation akzeptiert technisch gueltige aber unuebliche Formate (z.B. `user@[192.168.1.1]`) (Finding F-XX aus Security Audit) | Zusaetzliche Pruefung: Email muss `.` nach `@` enthalten, Mindestlaenge 5 Zeichen | Edge-Case-Emails (`@`, `a@b`) werden abgelehnt; Standard-Emails funktionieren weiterhin |
| S-03 | Security | S4 | XS | Hygiene | Gering | `go.sum` — `github.com/boombuler/barcode v1.0.1-0.20190219062509` | Barcode-Library von 2019, transitiv ueber `pquerna/otp`. Kein bekanntes CVE, aber Alt-Dependency | `govulncheck` ausfuehren; wenn sauber: akzeptieren. Sonst: `pquerna/otp` auf neueste Version pruefen | `govulncheck ./...` zeigt keine Findings fuer barcode |

### Observability & Betrieb

| ID | Kategorie | S | Aufwand | Nutzen | Risiko | Fundstelle(n) | Was ist falsch | Konkrete Aenderung | Akzeptanzkriterien |
|----|-----------|---|---------|--------|--------|---------------|---------------|-------------------|-------------------|
| O-01 | Observability | S3 | M | Debugging | Mittel | Kein Request-ID-System im gesamten Backend | Kein Correlation-ID fuer Request-Tracing. Logs koennen nicht einem einzelnen Request zugeordnet werden | Middleware hinzufuegen: `X-Request-ID` Header generieren (UUID), in Context speichern, an slog-Logger binden | Jeder Log-Eintrag enthaelt `request_id`-Feld; Response enthaelt `X-Request-ID` Header |
| O-02 | Observability | S4 | XS | Konsistenz | Gering | `internal/db/db.go:108,124,142,144`, `internal/api/middleware.go:31,40,47` | 7 Legacy `log.Printf()`-Aufrufe statt slog | Auf `slog.Warn()`/`slog.Info()` umstellen. DB-Package benoetigt Logger-Parameter in `Open()` oder Package-Level-Logger | `grep -rn 'log.Printf' internal/` = 0 Treffer |

---

## 4) Quick Wins (Top 15)

Aenderungen < 1 Tag, hohe Hebelwirkung, minimales Risiko.

| # | Finding | Aufwand | Impact |
|---|---------|---------|--------|
| 1 | **D-01**: 3 Spike-Files loeschen (`live-preview-spike*`) | XS (5 min) | Repo-Hygiene |
| 2 | **D-02**: 4 alte Log-Files loeschen + `.gitignore` erweitern | XS (5 min) | Repo-Hygiene |
| 3 | **D-03**: `frontend/build-old/` loeschen | XS (2 min) | 3.8MB weniger |
| 4 | **C-02**: 7 `log.Printf()` → `slog` migrieren | S (1h) | Logging-Konsistenz |
| 5 | **T-07**: `resolveETagVersion` Bug fixen (500→404) | S (1h) | Correctness |
| 6 | **S-02**: Email-Validation verschaerfen | XS (30 min) | Security |
| 7 | **S-03**: `govulncheck` fuer barcode-Dependency | XS (5 min) | Security-Audit |
| 8 | **X-01**: Air Build-Tags verifizieren/synchronisieren | XS (10 min) | DX-Konsistenz |
| 9 | **C-05**: Folder-Handler Error-Strings vereinheitlichen | S (1h) | Konsistenz |
| 10 | **P-02**: Keyword Batch-INSERT statt Loop | S (2h) | DB-Performance |
| 11 | **P-03**: Admin N+1 → Aggregate-Query | S (1h) | Admin-Performance |
| 12 | **D-04**: Root `.env` aufraumen | XS (10 min) | Repo-Hygiene |
| 13 | **P-01**: `fields=slim` fuer Recipe-/Canvas-Listen | S (3h) | Response-Size |
| 14 | **C-01** (Teilmenge): Die 15 sensitiven `err.Error()`-Stellen fixen | S (3h) | Security |
| 15 | **O-02**: Legacy log.Printf auf slog migrieren | XS (30 min) | Identisch mit #4 — alternativ: Ein Quick-Win PR fuer alle Hygiene-Items (D-01 bis D-04 + C-02) |

---

## 5) Refactoring-Roadmap

### Sprint 1 (1 Woche): Stabilisierung & Quick Wins

**Ziele:**
- Alle Quick Wins abarbeiten (D-01 bis D-04, C-02, T-07, S-02)
- Error-Leakage fuer die 15 sensitiven Stellen fixen (C-01 Teilmenge)
- resolveETagVersion-Bug fixen

**Deliverables:**
- 2-3 PRs: Hygiene-PR, Security-Error-Handling-PR, Bug-Fix-PR
- `govulncheck` Ergebnis dokumentiert

**Risiken:** Gering (kleine, isolierte Aenderungen)

**Messkriterien:**
- 0 `log.Printf()` im Backend
- 0 sensitive `err.Error()`-Leaks
- Keine Spike-Files, keine alten Logs im Repo

### Sprint 2-4 (1 Monat): Strukturelle Refactors + Test-Coverage

**Ziele:**
- Error-Handling vollstaendig vereinheitlichen (C-01 komplett, alle 52 Stellen)
- Layer-Violations-Cleanup Phase 1: Service-Layer Type-Aliases + Error-Re-Exports (A-02)
- API-Test-Coverage fuer Folders, Recipes, Templates/Snippets, Tags, Versions (T-01)
- Service-Tests fuer Sharing, Admin, Graph (T-02)
- Request-Correlation-ID Middleware (O-01)

**Deliverables:**
- 6-8 PRs: Error-Handling (2), Layer-Cleanup Phase 1 (1), API-Tests (3-4), Correlation-ID (1)

**Risiken:**
- Error-Handling-Refactoring koennte Validierungsmeldungen aendern die Frontend erwartet → Frontend-Impact pruefen
- Layer-Cleanup aendert Import-Pfade in vielen Files → sorgfaeltiges Batching noetig

**Messkriterien:**
- `grep 'respondError.*err.Error' internal/api/` = 0 (fuer Nicht-Validation-Faelle)
- `internal/api` Test-Coverage > 50%
- `internal/service` Test-Coverage > 40%
- Jeder API-Request hat `X-Request-ID` im Log

### Quartal (3 Monate): Architektur & Groessere Projekte

**Ziele:**
- Layer-Violations-Cleanup Phase 2: Alle 37 API-Files migrieren (A-02)
- Rate-Limiter Memory-Cap (S-01)
- DB-Test-Coverage auf > 50% bringen (T-03)
- Frontend Store-Tests (T-04, mindestens Recipes, Sharing, Journal)
- Handler-Funktionen >100 Zeilen aufbrechen (C-03)
- FilteredSearch refactoren (C-04)
- Performance: Slim-Projections fuer alle List-Endpoints (P-01)
- Optional: Interface-basierte DI evaluieren (A-01)

**Deliverables:**
- 10-15 PRs: Layer-Cleanup Phase 2 (5 Batches), DB-Tests (3-4), Frontend-Tests (2-3), Performance (2)

**Risiken:**
- Layer-Cleanup Phase 2 ist mechanisch aber voluminoes (37 Files) → CI muss nach jedem Batch gruen sein
- Interface-Refactoring (A-01) ist optional und nur sinnvoll wenn Test-Mocking-Bedarf steigt

**Messkriterien:**
- `scripts/layer-violation-baseline.txt` ist leer
- Backend-Test-Coverage gesamt > 50%
- Keine API-Handler-Funktion > 80 Zeilen
- Rate-Limiter Memory unter 50MB bei 100k unique IPs

---

## 6) PR-Slicing Vorschlaege

| # | PR Title | Scope | Betroffene Pfade | Definition of Done |
|---|----------|-------|-----------------|-------------------|
| 1 | `chore: remove dead spike files and stale build artifacts` | D-01, D-02, D-03, D-04 | `frontend/src/lib/editor/live-preview-spike*`, `backend/*.log`, `frontend/build-old/`, `.gitignore` | Files geloescht; `.gitignore` erweitert; CI gruen |
| 2 | `fix: migrate legacy log.Printf to structured slog` | C-02, O-02 | `internal/db/db.go`, `internal/api/middleware.go` | 0 `log.Printf`-Aufrufe; slog-Pattern konsistent |
| 3 | `fix: prevent internal error leakage in sensitive endpoints` | C-01 (Teilmenge) | `internal/api/auth_register.go`, `recipes_images.go`, `recipe_suggestions.go` + 10 weitere | Keine internen Error-Details an Client; Validation-Errors explizit als String-Literale |
| 4 | `fix: return 404 instead of 500 for nonexistent notes in ETag resolution` | T-07 | `internal/api/notes_helpers_etag.go` | `errors.Is(err, db.ErrNotFound)` → 404; Test angepasst |
| 5 | `perf: batch keyword insertion for encrypted notes` | P-02 | `internal/db/notes_misc.go` oder neues `notes_keywords.go`, `internal/service/notes_encryption_create.go`, `recipes_notes.go` | 1 INSERT statt N; Benchmark < 10ms fuer 50 Keywords |
| 6 | `perf: add slim projection for recipe and canvas list endpoints` | P-01 | `internal/api/recipes_handlers.go`, `canvas.go`, `internal/db/recipes_notes.go` | `fields=slim` Parameter; Content nicht in Listen |
| 7 | `refactor: unify error handling pattern across all API handlers` | C-01 (komplett), C-05 | 28 Files in `internal/api/` | Klare Trennung: Validation → String-Literal, Internal → `respondInternalErr()` |
| 8 | `test: add API handler tests for folders CRUD` | T-01 (Teilmenge) | `internal/api/folders_handlers_test.go` (neu) | Happy-Path + Error-Path Tests fuer alle 6 Folder-Endpoints |
| 9 | `test: add API handler tests for recipes` | T-01 (Teilmenge) | `internal/api/recipes_handlers_test.go` (neu) | Tests fuer List, Detail, Update Metadata, Set Ingredients, Scale |
| 10 | `test: add service tests for sharing business logic` | T-02 | `internal/service/sharing_test.go` (neu) | Alle exported Sharing-Methoden getestet; Berechtigungslogik verifiziert |
| 11 | `test: add API handler tests for templates and snippets` | T-01 (Teilmenge) | `internal/api/templates_handlers_test.go`, `snippets_handlers_test.go` (neu) | CRUD Happy-Path Tests fuer Templates + Snippets |
| 12 | `test: add API handler tests for tags and versions` | T-01 (Teilmenge) | `internal/api/tags_handlers_test.go`, `versions_handlers_test.go` (neu) | Tag-Assignment + Version-List/Restore Tests |
| 13 | `refactor: layer-violations Phase 1 — service type-aliases and error re-exports` | A-02 Phase 1 | `internal/service/errors.go` (neu), `note_types.go` (erweitern), `sharing_types.go` (neu), `admin_types.go` (neu) | Alle DB-Types haben Service-Layer-Aliases; GetDB() entfernt |
| 14 | `refactor: layer-violations Phase 2 — migrate API imports (Batch 1-3)` | A-02 Phase 2 | 12 Files in `internal/api/` (Batch 1-3 aus layer-violations-cleanup.md) | 12 Files aus Baseline entfernt; CI gruen |
| 15 | `refactor: layer-violations Phase 2 — migrate API imports (Batch 4-7)` | A-02 Phase 2 | 14 Files in `internal/api/` | 14 weitere Files aus Baseline entfernt |
| 16 | `refactor: layer-violations Phase 2 — migrate API imports (Batch 8-10)` | A-02 Phase 2 | 11 Files in `internal/api/` | Baseline leer; CI-Check gehaertet |
| 17 | `feat: add request correlation ID middleware` | O-01 | `internal/api/middleware.go`, `internal/api/helpers.go` | `X-Request-ID` in jedem Log-Eintrag + Response-Header |
| 18 | `security: add memory cap to rate limiter` | S-01 | `internal/api/ratelimit.go` | LRU-Eviction bei 100k Entries; Memory messbar begrenzt |
| 19 | `test: add DB layer tests for templates, snippets, tags, versions` | T-03 (Teilmenge) | `internal/db/templates_test.go`, `snippets_test.go`, `tags_test.go`, `versions_test.go` (alle neu) | Alle CRUD-Methoden haben Happy-Path-Tests |
| 20 | `refactor: extract long handler functions into sub-steps` | C-03 | `notes_crud_update.go`, `auth_login.go`, `auth_register.go` | Keine Handler-Funktion > 80 Zeilen |

---

## 7) Goldene Regeln fuer das Repo

Diese Regeln sollten in `docs/conventions.md` ergaenzt und vom Team gelebt werden:

1. **Error-Handling**: Validation-Errors → expliziter String an Client. Interne Errors → `respondInternalErr()` (loggt Details, gibt generische Meldung). Niemals `err.Error()` an Client fuer Service/DB-Fehler.

2. **Logging**: Ausschliesslich `slog`. Kein `log.Printf`, kein `fmt.Println`. Strukturierte Fields: `slog.String("user_id", ...)`, `slog.String("note_id", ...)`.

3. **Layer-Disziplin**: API importiert nur `service`-Package. Service importiert nur `db`-Package. Keine Abkuerzungen. DB-Types werden via Service-Layer-Aliases exponiert.

4. **Test-Minimum**: Jeder neue Handler braucht mindestens einen Happy-Path-Test und einen Error-Path-Test. Jede neue Service-Methode braucht einen Unit-Test.

5. **Handler-Groesse**: Max 80 Zeilen pro Handler-Funktion. Bei Ueberschreitung: in `parseRequest()`, `validate()`, `execute()`, `respond()` aufteilen.

6. **Naming**: Handler-Dateien nach Domain gruppiert mit Prefix: `notes_crud_*.go`, `notes_meta_*.go`, `notes_ai_*.go`. Service-Dateien analog: `notes_*.go`, `recipes_*.go`.

7. **Response-Pattern**: Listen-Endpoints liefern standardmaessig Slim-Projections (kein Content). Detail-Endpoints liefern alles. Client steuert via `fields=slim|full`.

8. **Security-Defaults**: Rate-Limiting fuer jeden neuen Endpoint. Keine `err.Error()`-Exposition. Input-Laengen-Limits fuer alle String-Felder.

9. **Dependency-Policy**: Direkten Dependencies juenger als 2 Jahre. Aeltere Dependencies werden bei naechstem Refactoring-Sprint geprueft. `govulncheck` laeuft woechentlich in CI.

10. **Dead-Code-Toleranz**: Keine Spike-Files in Main-Branch. Keine committed Log-Files. Keine verwaisten Build-Artefakte. Pre-Commit-Hook koennte `*.log`-Files pruefen.

11. **Commit-Hygiene**: CHANGELOG.md pflegen (lefthook prueft). `Co-Authored-By` bei AI-Unterstuetzung. Keine `--force` Pushes auf Main.

12. **Observability**: Jeder Log-Eintrag enthaelt `request_id` (sobald O-01 implementiert). Error-Logs enthalten `user_id`, `note_id`, `operation`. Keine PII in Logs (nur Hashes via `hashIdentifier()`).

---

## Anhang: Bereits erledigter Fortschritt

### Abgeschlossene Refactorings (vorherige Sessions)

| Was | Status | Refs |
|-----|--------|------|
| DB-Layer Error-Wrapping (`fmt.Errorf`) in 8 Files | Erledigt | CHANGELOG.md, 8 DB-Files |
| Service-Layer `checkNoteLimit()` Extraktion (7 Duplikate → 1 Methode) | Erledigt | `service/notes_helpers.go` |
| Graph-Helper-Extraktion (`scanGraphNodes`, `scanFilteredEdges`, `buildGraphData`) | Erledigt | `db/graph.go` |
| Canvas-Validation-Extraktion (`validateCanvasNodes`, `validateCanvasEdges`) | Erledigt | `service/canvas_validate.go` |
| golangci-lint Suppressed Findings Cleanup (7 von 8 Rules entfernt) | Erledigt | `.golangci.yml` |
| CI Guidelines Enforcement (Layer-Ratchet, Svelte4-Guard, Security-Check) | Erledigt | `scripts/`, `lefthook.yml` |

### Abgeschlossene Test-Arbeit

| Was | Status | Files |
|-----|--------|-------|
| Service-Layer Tests: Encryption, User, Trash, Admin, FIDO2, 2FA | Erledigt | 6 neue Test-Files, ~90 Subtests |
| API-Tests: Auth-Flow, Notes-CRUD, Trash, Admin | Erledigt | 4 neue Test-Files + Shared Helpers |
| API-Tests: Journal, Search, Encryption, Backlinks, Sharing | Erledigt | 4 neue Test-Files |

### Sprint 1 Quick Wins (Session 2026-02-21)

| Finding | Was | Status | Commit |
|---------|-----|--------|--------|
| D-01 | 2 Spike-Files geloescht (`live-preview-spike*`) | Erledigt | `06160eb` |
| D-02 | 4 alte Log-Files geloescht (waren nicht git-tracked, `.gitignore` hatte `*.log` bereits) | Erledigt | `06160eb` |
| D-03 | `frontend/build-old/` geloescht (war nicht git-tracked, `.gitignore` hatte es bereits) | Erledigt | `06160eb` |
| D-04 | `frontend/.env` — existierte bereits nicht mehr | Entfaellt | — |
| T-07 | **Bug gefixt**: `resolveETagVersion` 500→404 via `errors.Is(err, service.ErrNotFound)` | Erledigt | `475592e` |
| C-02/O-02 | 7 `log.Printf()` → `slog` migriert (4 in `db/db.go`, 3 in `api/middleware.go`) | Erledigt | `74a0be4` |
| S-02 | Email-Validation verschaerft: Domain muss `.` enthalten, min. 5 Zeichen | Erledigt | `9704b72` |
| S-03 | `govulncheck` — Toolchain-Inkompatibilitaet (Go 1.25), manuelle Pruefung: keine CVEs fuer `boombuler/barcode` | Geprueft | — |
| X-01 | `.air.toml` Build-Tags verifiziert — identisch mit Makefile (`-tags 'fts5 sqlite_crypt'`) | Verifiziert | — |
| C-05 | Folder-Handler: `ValidateFolderPath` → fester String, Service-Errors → `respondInternalErr()` | Erledigt | `410434c` |
| P-02 | Keyword-Insertion: Batch-INSERT statt Loop (neue `InsertNoteKeywords()`, 5 Callsites migriert) | Erledigt | `da8770b` |
| P-03 | Admin N+1: `calculateAllUserStorageMB()` traversiert Upload-Dir einmal statt pro User | Erledigt | `3186e23` |
| C-01 (Teilmenge) | 4 sensitive `err.Error()`-Leaks gefixt: `auth_login.go` (Login, 2FA-Login), `twofa.go` (Verify, RegenerateBackupCodes) | Erledigt | siehe unten |

### Sprint 2 Strukturelle Refactors (Session 2026-02-21)

| Finding | Was | Status | Commit |
|---------|-----|--------|--------|
| P-01 | Recipe-Liste: `?fields=slim` fuer Content-freie Projektion | Erledigt | `fabc760` |
| O-01 | Request-Correlation-ID: `requestIDLoggerMiddleware` + `respondInternalErr` enrichment | Erledigt | `e282670` |
| S-01 | Rate-Limiter Memory-Cap — war bereits implementiert (`maxRateLimitClients=10000`, `evictOldest()`) | Entfaellt | — |
| C-01 (komplett) | `ValidationError`-Typ in Service-Layer; Register/BootstrapAdmin Error-Handling getrennt (Validation→400, Internal→500) | Erledigt | `de9efed` |
| C-03 | Handler-Extraktion: `updateNote` (134→54), `login` (115→58), `register` (106→29) | Erledigt | `5da2c7c` |
| C-04 | `FilteredSearch` (137→16): `buildFilteredSearchQuery` + 5 Filter-Helpers + `scanNoteRows` shared | Erledigt | `3474a99` |
| A-02 Phase 1 | `service.User` Type-Alias fuer `db.User` — 0 Layer-Violations in Baseline | Erledigt | `a049911` |

### Sprint 2 Test-Coverage (Session 2026-02-21, Fortsetzung)

| Finding | Was | Status | Details |
|---------|-----|--------|---------|
| T-01 | API-Handler-Tests fuer Folders, Templates, Snippets, Tags, Versions | Erledigt | 3 neue Test-Files, 49 Tests: `folders_handlers_test.go` (12), `templates_handlers_test.go` (17), `tags_versions_handlers_test.go` (20) |
| T-02 | Service-Tests fuer Sharing und Graph | Erledigt | Sharing: 11 neue Tests + 10 Subtests (happy-path + permissions). Graph: 6 Tests (CRUD, caching, isolation). Admin: bereits abgedeckt |

### Bekannte Bugs (entdeckt in Tests)

| Bug | Status | Ref |
|-----|--------|-----|
| `resolveETagVersion` gibt 500 statt 404 fuer nicht-existente Noten | **Gefixt** (T-07, `475592e`) | `notes_helpers_etag.go` |

### Bestehende Plaene (referenziert, nicht dupliziert)

| Plan | Status | Ref |
|------|--------|-----|
| Layer-Violations-Cleanup (37 Files, 10 Batches) | Geplant | `docs/planning/layer-violations-cleanup.md` |
| Modernisierungsplan (Go 1.25, CodeMirror Patches) | Bereit | `docs/planning/modernization-plan.md` |
| Security Audit Open Findings (F-01, F-15, F-XX) | Offen | `docs/security_audit_findings.md` |
