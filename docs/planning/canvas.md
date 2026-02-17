# Canvas Feature RFC (Obsidian-Canvas-aehnlich)

## Status

- Datum: 2026-02-17
- Status: Draft
- Ziel: Implementierbarer End-to-End-Plan fuer ein Infinite-Canvas-Feature mit Karten, Verbindungen und Import/Export.

---

## 1. Zielbild

Wir fuehren ein neues Feature `canvas` ein, das Nutzern erlaubt, visuelle Boards aus Elementen und Kanten zu bauen (aehnlich Obsidian Canvas), inklusive:

- Infinite Canvas (Pan/Zoom)
- Karten-Typen: Text, Note-Referenz, URL, Datei/Attachment, Gruppe
- Verbindungen mit Labels/Styles
- Undo/Redo
- Import/Export im JSON-Canvas-kompatiblen Format
- Optional E2EE fuer Canvas-Inhalte (analog Notizen-Encryption)

## 2. Nicht-Ziele (v1)

- Echtzeit-Kollaboration mit Presence/Cursors
- CRDT-basierte Merge-Engine
- Public Share Links fuer Canvas
- Vollstaendige Multi-Page-Design-Tool-Funktionen (Bezier Editing, Advanced Snap Rules)

---

## 3. Architekturentscheidung

### Entscheidung: eigenes Domain-Modul `canvas` (nicht `note_type='canvas'`)

Begruendung:

- Das bestehende Note-System ist stark auf `note/journal/recipe` optimiert.
- Canvas hat andere Persistenzstruktur (viele Elemente + Kanten + Viewport + Z-Order).
- Eigene API/DB-Schicht vermeidet Sonderfaelle in bestehender Notes-Logik.
- Entspricht Conventions (Backend: API -> Service -> DB sauber trennbar).

---

## 4. Datenmodell (Backend/SQLite)

### 4.1 Tabellen

1. `canvases`
- `id TEXT PRIMARY KEY`
- `user_id INTEGER NOT NULL`
- `title TEXT NOT NULL`
- `folder_path TEXT NOT NULL DEFAULT '/'`
- `version INTEGER NOT NULL DEFAULT 1`
- `view_x REAL NOT NULL DEFAULT 0`
- `view_y REAL NOT NULL DEFAULT 0`
- `zoom REAL NOT NULL DEFAULT 1`
- `created_at TEXT NOT NULL`
- `updated_at TEXT NOT NULL`
- `is_deleted INTEGER NOT NULL DEFAULT 0`

2. `canvas_elements`
- `id TEXT PRIMARY KEY`
- `canvas_id TEXT NOT NULL REFERENCES canvases(id) ON DELETE CASCADE`
- `user_id INTEGER NOT NULL`
- `type TEXT NOT NULL` (`text|note|url|file|group`)
- `x REAL NOT NULL`
- `y REAL NOT NULL`
- `width REAL NOT NULL`
- `height REAL NOT NULL`
- `rotation REAL NOT NULL DEFAULT 0`
- `z_index INTEGER NOT NULL DEFAULT 0`
- `style_json TEXT` (Farben/Border etc.)
- `data_json TEXT NOT NULL` (typ-spezifischer Payload)
- `created_at TEXT NOT NULL`
- `updated_at TEXT NOT NULL`

3. `canvas_edges`
- `id TEXT PRIMARY KEY`
- `canvas_id TEXT NOT NULL REFERENCES canvases(id) ON DELETE CASCADE`
- `user_id INTEGER NOT NULL`
- `from_element_id TEXT NOT NULL REFERENCES canvas_elements(id) ON DELETE CASCADE`
- `to_element_id TEXT NOT NULL REFERENCES canvas_elements(id) ON DELETE CASCADE`
- `from_handle TEXT` (`top|right|bottom|left|center`)
- `to_handle TEXT`
- `label TEXT`
- `style_json TEXT`
- `created_at TEXT NOT NULL`
- `updated_at TEXT NOT NULL`

4. `canvas_versions` (Undo/History + Conflict-Fallback)
- `id INTEGER PRIMARY KEY AUTOINCREMENT`
- `canvas_id TEXT NOT NULL`
- `user_id INTEGER NOT NULL`
- `version INTEGER NOT NULL`
- `snapshot_json TEXT NOT NULL`
- `created_at TEXT NOT NULL`

### 4.2 Indizes

- `idx_canvases_user_updated` auf `(user_id, is_deleted, updated_at DESC)`
- `idx_canvas_elements_canvas` auf `(canvas_id, z_index)`
- `idx_canvas_edges_canvas` auf `(canvas_id)`
- `idx_canvas_versions_canvas_version` auf `(canvas_id, version DESC)`

### 4.3 Migration

- Neue Migration: `048_canvas.sql` (naechste freie Nummer nach `047_analytics_events.sql`)
- `db.go` in `runMigrations()` erweitern.
- `schema.sql` aktualisieren.

---

## 5. API-Design (Backend)

Neue Route-Gruppe: `/api/canvases`

1. `GET /api/canvases`
- Zweck: Liste der Canvases des Users (Metadaten, kein Voll-Snapshot)
- Query: `folder`, `limit`, `cursor`

2. `POST /api/canvases`
- Zweck: Canvas erstellen
- Body: `{ title, folder_path }`

3. `GET /api/canvases/{id}`
- Zweck: Vollstaendige Canvas-Daten laden
- Response: Canvas + Elemente + Kanten + Metadaten

4. `PATCH /api/canvases/{id}`
- Zweck: inkrementelles Update
- Body:
  - `base_version`
  - `viewport` (optional)
  - `ops` (Array von Create/Update/Delete fuer Elemente/Kanten)
- Konflikt: `409` bei `base_version`-Mismatch

5. `POST /api/canvases/{id}/import`
- Zweck: JSON Canvas importieren
- Body: JSON-Datei-Content oder Upload-Referenz

6. `GET /api/canvases/{id}/export`
- Zweck: JSON Canvas exportieren
- Response: Spezifikationskompatibles JSON

### 5.1 Fehlerverhalten

- API-Layer mappt Errors ueber bestehende `respondError()`/`respondInternalErr()`.
- Keine internen SQL-Details im Client-Fehler.
- `404` fuer fremde/nicht vorhandene Canvas.
- `413` fuer zu grosse Payloads.

---

## 6. Service- und DB-Schicht

### Service (`backend/internal/service/`)

- `canvas.go` (CRUD + Patch-Apply + Validation)
- `canvas_import_export.go`
- `canvas_types.go`

Wichtige Regeln:

- Ownership in jeder Operation pruefen (`user_id`).
- Optimistic Locking ueber `version`.
- `ops` atomar in einer DB-Transaktion anwenden.
- Snapshot in `canvas_versions` nur bei sinnvollen Intervallen (z. B. alle N Writes oder explicit save).

### DB (`backend/internal/db/`)

- `canvas_models.go`
- `canvas_crud.go`
- `canvas_patch.go`
- `canvas_import_export.go`

---

## 7. Frontend-Architektur

### 7.1 Route-Struktur

- `frontend/src/routes/canvas/+page.svelte` (Liste/Startpunkt)
- `frontend/src/routes/canvas/[id]/+page.svelte` (Editor)

### 7.2 API-Layer

- `frontend/src/lib/api/canvas.ts`
- `frontend/src/lib/api.ts` Barrel erweitern
- Nutzung des zentralen `request()`-Clients (`client.ts`) inkl. Offline-Queue-Optionen

### 7.3 Store (Svelte 5 Runes)

- `frontend/src/lib/stores/canvas.svelte.ts`
- Modul-Split fuer groessere Logik:
  - `stores/canvas/state.ts`
  - `stores/canvas/ops.ts`
  - `stores/canvas/history.ts`
  - `stores/canvas/viewport.ts`

### 7.4 Komponenten

- `CanvasViewport.svelte` (Pan/Zoom, Grid)
- `CanvasElement.svelte` (Renderer pro Typ)
- `CanvasEdgeLayer.svelte`
- `CanvasToolbar.svelte`
- `CanvasMiniMap.svelte`
- `CanvasInspector.svelte`

---

## 8. JSON Canvas Kompatibilitaet

Ziel:

- Import aus Obsidian-Canvas-Dateien
- Export in weitgehend kompatibles JSON

Mapping-Strategie:

- Standard-Felder direkt uebernehmen (Nodes, Edges, Position/Size).
- Nicht abbildbare XelaNote-Sonderfelder unter `x_xelanote` namespace kapseln.
- Beim Re-Export Originalfelder stabil halten, soweit moeglich.

Referenz:

- https://jsoncanvas.org/spec/1.0/

---

## 9. Security, Privacy, Encryption

- Kein Auth-Token im localStorage (bestehende Policy unveraendert).
- Canvas-Inhalte wie Notizinhalte behandeln (keine sensiblen Logs).
- Bei aktivierter E2EE:
  - Elementinhalte (`text`, URL-Metadaten, Labels) verschluesseln.
  - Position/Dimension optional im Klartext (fuer technische Funktion) oder mit separater Policy konfigurierbar.
- Import-Validation:
  - Strikte Schema-Pruefung
  - Limits fuer Element-/Kantenanzahl
  - Max Payload Groesse

---

## 10. Performance-Ziele

Initiale Zielwerte (Richtwerte):

- 60 FPS bei Interaktionen bis 300 Elemente / 500 Kanten
- Ladezeit `< 1.5s` fuer Canvas mit 200 Elementen auf Desktop
- PATCH-Roundtrip `< 250ms` im LAN/normalen Setup

Technische Massnahmen:

- Viewport Culling (nur sichtbare Elemente rendern)
- Batched Updates via `requestAnimationFrame`
- Debounced Autosave + operation batching
- Lazy Rendering fuer schwere Karten (z. B. Datei-Previews)

---

## 11. Tests

### Backend

- DB-Tests: Constraints, Ownership, Cascade, Versioning
- Service-Tests: Patch-Validierung, Konflikte, Import-Mapping
- API-Tests: Statuscodes, Validation Errors, 409 Konflikte

### Frontend

- Store-Tests: Ops-Reducer, Undo/Redo, Conflict-Rebase
- Komponenten-Tests: Selection, Keyboard, Drag/Resize
- E2E: Erstellen, Verbinden, Speichern, Reload, Import/Export

---

## 12. Rollout-Plan

### Phase A (MVP, 2-3 Wochen)

- DB + API CRUD
- Basis-Editor (Pan/Zoom, Text/Note Karten, Kanten)
- Persistenz mit Optimistic Locking

### Phase B (2-3 Wochen)

- Import/Export JSON Canvas
- Gruppenkarten, Minimap, Inspector
- Undo/Redo + bessere Tastaturbedienung

### Phase C (2-4 Wochen)

- Offline-Sync Integration
- Performance-Tuning fuer grosse Boards
- Verschluesselungs-Pfad und Beta->GA Rollout

---

## 13. Offene Entscheidungen

1. Soll Canvas als eigener Baumknoten in der Sidebar erscheinen oder im Note-Tree integriert werden?
2. Sollen Canvas-Elemente auf Notizen hard-linken (ID) oder soft-linken (Titel-Fallback)?
3. E2EE-Modus: Position/Size verschluesseln ja/nein?
4. Maximal-Limits pro Canvas (Elemente/Kanten) fuer v1?

---

## 14. Konkreter Implementierungs-Start (erste PR)

1. Migration `048_canvas.sql` + DB-Modelle + DB-CRUD
2. Service `canvas.go` mit `GetCanvas`, `CreateCanvas`, `PatchCanvas`
3. API-Routes `routes_canvas.go` + Handler
4. Frontend `lib/api/canvas.ts` + `routes/canvas/[id]/+page.svelte` mit Basis-Viewport
5. Tests fuer Create/Get/Patch/409

---

## 15. Ticket-Zerlegung (umsetzbar)

Hinweis:

- Reihenfolge ist auf minimale Risiken optimiert.
- Alle Tickets folgen Backend-Layer-Regel: API -> Service -> DB.
- Schaetzung in "Hands-on" ohne Review-Wartezeiten.

### Epic C-0: Canvas v1 Foundation

- Ziel: Baseline fuer Canvas-Feature inkl. Persistenz, Editor und Import/Export.
- DoD: Tickets C-1 bis C-10 abgeschlossen.

### C-1: DB Migration + Schema

- Scope:
  - Migration `048_canvas.sql`
  - Tabellen: `canvases`, `canvas_elements`, `canvas_edges`, `canvas_versions`
  - Indizes laut Abschnitt 4.2
  - `runMigrations()` + `schema.sql` aktualisieren
- Dateien:
  - `backend/internal/db/migrations/048_canvas.sql`
  - `backend/internal/db/db.go`
  - `backend/internal/db/schema.sql`
- Akzeptanzkriterien:
  - [ ] Migration laeuft auf leerer und bestehender DB ohne Fehler
  - [ ] Indizes vorhanden
  - [ ] `make test` bleibt gruen
- Aufwand: 0.5-1 Tag

### C-2: DB Layer Canvas CRUD

- Scope:
  - Modelle + CRUD fuer Canvas-Metadaten
  - Voll-Ladequery fuer Elemente + Kanten
  - Soft-Delete fuer Canvas
- Dateien:
  - `backend/internal/db/canvas_models.go`
  - `backend/internal/db/canvas_crud.go`
  - Tests in `backend/internal/db/canvas_test.go`
- Akzeptanzkriterien:
  - [ ] Create/Get/List/Delete funktionieren inkl. Ownership-Filter
  - [ ] Cascade fuer Elemente/Kanten funktioniert
  - [ ] DB-Tests decken Success + NotFound + Fremdzugriff ab
- Aufwand: 1 Tag

### C-3: Service Layer Canvas Core

- Scope:
  - Business-Methoden: `CreateCanvas`, `GetCanvas`, `ListCanvases`, `DeleteCanvas`
  - Fehler-Mapping auf `db.Err*`
  - Validierung fuer Title/FolderPath
- Dateien:
  - `backend/internal/service/canvas.go`
  - `backend/internal/service/canvas_types.go`
  - Tests: `backend/internal/service/canvas_test.go`
- Akzeptanzkriterien:
  - [ ] Keine HTTP-Typen im Service
  - [ ] Ownership erzwungen
  - [ ] Service-Tests bestehen
- Aufwand: 0.5-1 Tag

### C-4: API Routes + Handler (Canvas CRUD)

- Scope:
  - Neue Routes in `routes_*.go`
  - Handler fuer GET/POST/LIST/DELETE
  - Response- und Error-Mapping
- Dateien:
  - `backend/internal/api/routes_canvas.go`
  - `backend/internal/api/canvas.go`
  - `backend/internal/api/api.go` (Server wiring)
- Akzeptanzkriterien:
  - [ ] Endpunkte funktionieren mit Auth
  - [ ] 401/404/500 Verhalten konsistent
  - [ ] API-Tests fuer zentrale Pfade
- Aufwand: 1 Tag

### C-5: Patch API + Optimistic Locking

- Scope:
  - `PATCH /api/canvases/{id}` mit `base_version` + `ops[]`
  - Atomic apply in Transaktion
  - 409 bei Version-Konflikt
- Dateien:
  - `backend/internal/db/canvas_patch.go`
  - `backend/internal/service/canvas_patch.go`
  - `backend/internal/api/canvas_patch.go` (oder in `canvas.go`)
- Akzeptanzkriterien:
  - [ ] Konkurrenz-Update liefert 409
  - [ ] Erfolgreiches Patch inkrementiert Version
  - [ ] Teilfehler rollt komplette Transaktion zurueck
- Aufwand: 1.5-2 Tage

### C-6: Frontend API Modul + Typen

- Scope:
  - `lib/api/canvas.ts` fuer alle CRUD/PATCH Calls
  - Typen in `lib/api/types.ts`
  - Barrel-Export `lib/api.ts`
- Dateien:
  - `frontend/src/lib/api/canvas.ts`
  - `frontend/src/lib/api/types.ts`
  - `frontend/src/lib/api.ts`
- Akzeptanzkriterien:
  - [ ] API-Modul mit zentralem `request()` Client
  - [ ] TypeScript strict ohne `any`
  - [ ] Fehler ueber `ApiError` behandelbar
- Aufwand: 0.5 Tag

### C-7: Canvas Store (Svelte 5 Runes)

- Scope:
  - State fuer Canvas, Elemente, Kanten, Selection, Viewport
  - Ops-Queue lokal + Save Action
  - Undo/Redo Basis (History Stack)
- Dateien:
  - `frontend/src/lib/stores/canvas.svelte.ts`
  - optional Split: `frontend/src/lib/stores/canvas/*.ts`
- Akzeptanzkriterien:
  - [ ] Nur Svelte 5 Runes
  - [ ] Load/Save/Patch integriert
  - [ ] Undo/Redo fuer add/move/delete von Elementen
- Aufwand: 1-1.5 Tage

### C-8: Canvas Route + Basis-Editor

- Scope:
  - Route `canvas/[id]`
  - Basis-Viewport (Pan/Zoom)
  - Element Render (Text + Note)
  - Kanten Render + Selection
- Dateien:
  - `frontend/src/routes/canvas/[id]/+page.svelte`
  - `frontend/src/lib/components/canvas/*`
- Akzeptanzkriterien:
  - [ ] 60 FPS bei kleinen/mittleren Boards
  - [ ] Drag/Move/Resize funktioniert
  - [ ] Persistenz nach Reload intakt
- Aufwand: 2-3 Tage

### C-9: JSON Canvas Import/Export

- Scope:
  - Import-Endpoint + Mapping
  - Export-Endpoint + kompatibles JSON
  - Frontend UI fuer Import/Export Action
- Dateien:
  - `backend/internal/service/canvas_import_export.go`
  - `backend/internal/db/canvas_import_export.go`
  - `frontend/src/lib/components/canvas/CanvasToolbar.svelte`
- Akzeptanzkriterien:
  - [ ] Import einer gueltigen JSON Canvas Datei erfolgreich
  - [ ] Export ist wieder importierbar
  - [ ] Invalid JSON fuehrt zu 400 (ohne interne Details)
- Aufwand: 1.5-2 Tage

### C-10: Test- und Quality-Gate

- Scope:
  - DB-/Service-/API Tests ergaenzen
  - Frontend Store + Interaktions-Tests
  - ggf. e2e happy path
- Dateien:
  - `backend/internal/db/canvas_test.go`
  - `backend/internal/service/canvas_test.go`
  - `backend/internal/api/canvas_test.go`
  - `frontend/src/lib/stores/canvas.test.ts`
- Akzeptanzkriterien:
  - [ ] `make test` gruen
  - [ ] `make test-frontend` gruen
  - [ ] Keine Policy-Verletzung (`make check-policy`)
- Aufwand: 1-1.5 Tage

### C-11: Sidebar/Navigation Integration

- Scope:
  - Einstieg im UI (z. B. "Canvases" Bereich)
  - Create-Dialog + Listenansicht
  - Routing aus Tree/Sidebar
- Akzeptanzkriterien:
  - [ ] Canvas in Navigation erreichbar
  - [ ] Neue Canvas erstellbar aus UI
  - [ ] i18n Keys in `de/en` vorhanden
- Aufwand: 1 Tag

### C-12: Offline Queue Integration (optional fuer v1, empfohlen fuer v1.1)

- Scope:
  - Canvas-Patch Operationen `_offlineAllowed`
  - Queue-Konflikte bei Reconnect behandeln
- Akzeptanzkriterien:
  - [ ] Offline-Aenderungen werden gepuffert
  - [ ] Reconnect fuehrt zu konsistentem State
  - [ ] 409-Flow fuer Canvas analog Notes
- Aufwand: 1.5-2 Tage

### C-13: E2EE Erweiterung Canvas (v1.1)

- Scope:
  - Verschluesselung fuer sensible Elementinhalte
  - Schluesselhandhabung analog Note-E2EE
- Akzeptanzkriterien:
  - [ ] Keine Klartext-Inhalte im DB-Dump (bei aktiver E2EE)
  - [ ] Editor kann verschluesselte Canvas laden/speichern
  - [ ] Tests fuer Encrypt/Decrypt-Pfad
- Aufwand: 2-3 Tage

### C-14: Performance Hardening (v1.1)

- Scope:
  - Viewport Culling
  - Debounced/Batched Rendering
  - Profiling fuer grosse Boards
- Akzeptanzkriterien:
  - [ ] Interaktionen bleiben fluessig bei 300/500 (Elemente/Kanten)
- [ ] Keine sichtbaren Frame-Drops bei normalem Dragging
- [ ] Messwerte dokumentiert
- Aufwand: 1.5-2 Tage

---

## 16. PR-1 Scope (C-1 bis C-4)

### Ziel von PR-1

PR-1 liefert ein belastbares Backend-Fundament fuer Canvas:

- Datenmodell und Migrationen sind produktionsfaehig
- CRUD-API fuer Canvas-Metadaten + Voll-Laden ist nutzbar
- Layer-Konvention bleibt eingehalten (API -> Service -> DB)

PR-1 liefert bewusst noch keinen visuellen Canvas-Editor im Frontend.

### Enthalten (in scope)

1. `C-1` DB Migration + Schema
2. `C-2` DB Layer Canvas CRUD
3. `C-3` Service Layer Canvas Core
4. `C-4` API Routes + Handler (CRUD)

### Nicht enthalten (out of scope)

1. Patch/Ops-Engine (`C-5`)
2. Frontend API/Store/Editor (`C-6` bis `C-8`)
3. JSON Canvas Import/Export (`C-9`)
4. Offline/E2EE/Performance Hardening (`C-12` bis `C-14`)

### Geplante Commits (Reihenfolge)

1. `feat(db): add canvas schema migration 048 and register in db bootstrap`
- Inhalt:
  - `backend/internal/db/migrations/048_canvas.sql`
  - `backend/internal/db/db.go` (migrations list)
  - `backend/internal/db/schema.sql`
- Check:
  - `cd backend && go test -tags "fts5 sqlite_crypt" ./internal/db/...`

2. `feat(db): implement canvas models and CRUD queries with ownership checks`
- Inhalt:
  - `backend/internal/db/canvas_models.go`
  - `backend/internal/db/canvas_crud.go`
  - `backend/internal/db/canvas_test.go`
- Check:
  - `cd backend && go test -tags "fts5 sqlite_crypt" ./internal/db/...`

3. `feat(service): add canvas service core methods and validations`
- Inhalt:
  - `backend/internal/service/canvas.go`
  - `backend/internal/service/canvas_types.go`
  - `backend/internal/service/canvas_test.go`
- Check:
  - `cd backend && go test -tags "fts5 sqlite_crypt" ./internal/service/...`

4. `feat(api): expose canvas CRUD routes and handlers`
- Inhalt:
  - `backend/internal/api/routes_canvas.go`
  - `backend/internal/api/canvas.go`
  - `backend/internal/api/api.go` (wiring)
  - `backend/internal/api/canvas_test.go`
- Check:
  - `cd backend && go test -tags "fts5 sqlite_crypt" ./internal/api/...`

5. `docs: update api docs and changelog for canvas backend foundation`
- Inhalt:
  - `docs/api.md` (neue Endpoint-Sektion)
  - `CHANGELOG.md` (`[Unreleased]`)
- Check:
  - `make quality`

### Definition of Done fuer PR-1

- [ ] Migration 048 laeuft auf neuer und bestehender DB
- [ ] CRUD-Endpunkte vorhanden:
  - `POST /api/canvases`
  - `GET /api/canvases`
  - `GET /api/canvases/{id}`
  - `DELETE /api/canvases/{id}`
- [ ] Ownership-Checks verhindern Cross-User-Zugriff
- [ ] API-Fehlercodes konsistent (`401`, `404`, `500`)
- [ ] Unit-Tests fuer DB/Service/API sind gruen
- [ ] `make test` und `make check-policy` sind gruen

### Empfohlene Reihenfolge fuer Review

1. Migration + Schema
2. DB Query-Korrektheit und Constraints
3. Service-Validierung und Error-Mapping
4. API-Verhalten und Authz
5. Tests + Docs/Changelog

### Cut-Lines (bei Zeitdruck)

Wenn PR-1 zu gross wird, zuerst folgende Teile verschieben:

1. `DELETE /api/canvases/{id}` (kann in PR-1b)
2. Cursor-Pagination fuer `GET /api/canvases` (erstmal einfacher `limit`)
3. `canvas_versions` Snapshot-Write (nur Tabelle jetzt, Nutzung spaeter)

### Risiken in PR-1

1. Schema-Drift zwischen `schema.sql` und Migrationen.
2. Ownership-Filter in einzelnen Queries vergessen.
3. API-Wiring ohne Rate-Limit/Policy-Anschluss.

Mitigation:

- Tests pro Layer + negative Cases (fremde `user_id`, not found, soft-deleted).
- Review-Checklist pro Query (`WHERE user_id = ?`).
- Vor Merge `make quality && make test && make check-policy`.

---

## 17. SOTA-Maximalarchitektur (Kosten/Zeit egal)

Dieser Abschnitt beschreibt die "bestmoegliche" Zielarchitektur fuer Performance, Usability und Design-Qualitaet.  
Sie ersetzt nicht PR-1, sondern definiert das Endziel nach der Foundation.

### 17.1 Leitprinzipien

1. Local-first als Standard (offline zuerst, online als Replikation)
2. Echtzeit-Kollaboration ohne manuelle Konfliktaufloesung
3. Rendering-Engine fuer grosse Boards (nicht nur kleine CRUD-Boards)
4. Harte Trennung von:
   - kollaborativen Daten
   - grossen Assets (Bilder/Videos)
   - praesentationsbezogener UI-State

### 17.2 Ziel-Stack

1. Kollaborationskern:
- CRDT-Dokument pro Canvas (Yjs oder Automerge)
- Presence/Awareness Kanal fuer Cursor, Selection, Namen

2. Transport:
- WebSocket Multiplexing pro Canvas-Room
- Delta-Updates + Awareness getrennt
- Reconnect/Resync mit Resume-Token

3. Persistenz:
- Event-Log (append-only updates) + periodische Snapshots
- Backend-Storage fuer CRDT-Updates, Materialized Read Models fuer Listen

4. Rendering:
- Svelte-UI shell bleibt
- Canvas-Engine als eigener Renderer (WebGL/WebGPU faehig, mit 2D-Fallback)
- Viewport Culling + Spatial Index (R-Tree/Quadtree)
- Offscreen Rendering/Worker fuer teure Berechnungen

5. Assets:
- Asset Uploads getrennt vom Realtime-Kanal
- Signed URLs + CDN/Edge Cache
- Thumbnails/Preview-Pipeline asynchron

### 17.3 Warum das SOTA ist

1. Performance:
- CRDT + delta sync skaliert besser als Request-Response Patch bei hoher Parallelitaet.
- Culling + GPU/Canvas Renderer verhindert DOM-Bottlenecks bei grossen Boards.

2. Usability:
- Kollaboration ohne Locking-Pain.
- Nahtloses Multi-Device Verhalten (same board, same state, gleiche Cursor/Selection Semantik).

3. Design:
- Intentional Interaction-Layer (Selection Handles, Snapping, Keyboard-first).
- Konsistente UX fuer Maus, Touch und Pen.

### 17.4 Referenz-Features fuer UX-Qualitaet

Minimum fuer "best-in-class" Canvas:

1. Multi-Select Marquee + Lasso
2. Smart Guides + Snaplines
3. Inline Text Editing ohne Modalwechsel
4. Keyboard-first Workflow (Move/Align/Distribute/Duplicate)
5. Minimap + Command Palette + Search-in-canvas
6. Presence (remote cursor, remote selection, follow-user)
7. Zustaende fuer Draft vs Confirmed (optimistische Edits sichtbar)

### 17.5 Sicherheits- und Privacy-Modell (SOTA)

1. E2EE (optional, aber vollstaendig):
- Elementinhalte verschluesselt
- Edge-Labels verschluesselt
- Awareness optional unverschluesselt (Konfigurationsentscheidung)

2. Metadata-Minimierung:
- Server sieht nur noetige Routing-Metadaten
- Keine sensitiven Canvas-Inhalte in Logs

3. Revisions-Sicherheit:
- Signierte Snapshot-Metadaten
- Auditfaehiger Event-Stream

### 17.6 Empfohlene Zielarchitektur (konkret)

Backend:

1. Room-Service fuer Realtime Sessions
2. CRDT-Update Store (`canvas_updates`)
3. Snapshot Store (`canvas_snapshots`)
4. Read Model (`canvases`) fuer Liste/Suche/Rechte

Frontend:

1. CRDT-Store pro Canvas
2. Renderer-Engine Modul (`lib/canvas-engine/*`)
3. Interaction-System (`selection`, `transform`, `guides`, `history`)
4. Asset-System (`upload`, `cache`, `preview`)

### 17.7 Migrationspfad von aktuellem Plan -> SOTA

1. Phase S1: Foundation (PR-1 bis PR-2)
- Bestehender Plan C-1..C-5
- Ziel: stabiles CRUD + Versioning

2. Phase S2: Local-first Datamodell
- Introduce CRDT-Dokument parallel zum CRUD-Modell
- Dual-write + Vergleichsmetriken

3. Phase S3: Realtime Rooms
- WebSocket Room Layer, Awareness, Presence
- Konfliktlogik von 409/PATCH auf CRDT verschieben

4. Phase S4: Renderer Upgrade
- Einfuehrung Engine-Layer mit Culling und Spatial Index
- Grosse Boards Benchmark gegen DOM-Renderer

5. Phase S5: Full SOTA
- Event-Sourcing + Snapshots + Recovery Tools
- E2EE fuer Canvas Payloads
- Kollaborative UX-Polish (follow mode, branch/history UX)

### 17.8 KPI-Ziele fuer SOTA-Track

1. 120 FPS Ziel auf High-End, stabile 60 FPS auf Mid-Range bei 2k+ Shapes
2. P95 lokale Interaktion < 16ms
3. Reconnect + Resync < 2s bei 10k Events
4. Merge-free user experience (keine manuellen Konfliktdialoge im Standardfall)

### 17.9 Entscheidungsfazit

Wenn "bestmoeglich" wirklich Prioritaet hat:

1. Den aktuellen PR-1-Plan als Fundament behalten.
2. Ab PR-2 nicht weiter in reines CRUD/PATCH investieren.
3. Frueh auf CRDT + Realtime + Engine-Layer umstellen.

Das ist die robusteste Route zu maximaler Performance, Usability und Design-Qualitaet.

---

## 18. Externe Referenzen (SOTA-Basis)

1. Yjs Awareness/Presence:
- https://docs.yjs.dev/api/about-awareness
- https://docs.yjs.dev/getting-started/adding-awareness
- https://docs.yjs.dev/ecosystem/connection-provider/y-websocket

2. JSON Canvas Spezifikation:
- https://jsoncanvas.org/spec/1.0/

3. Canvas-Engine Optimierungen (Culling, Multiplayer-Ansatz):
- https://tldraw.dev/sdk-features/culling
- https://tldraw.dev/features/composable-primitives/multiplayer-collaboration

4. Web Platform Baseline/Compatibility Kontext:
- https://developer.mozilla.org/en-US/docs/Glossary/Baseline/Compatibility
- https://developer.mozilla.org/en-US/docs/Web/API/GPUSupportedFeatures
