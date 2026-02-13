# Plan: Layer Violations Baseline reduzieren

## Context

Die Architektur-Konvention verlangt API → Service → DB (keine Schicht überspringen). Aktuell importieren 37 API-Dateien direkt das `db`-Package. Ein CI-Ratchet (`scripts/check-layer-violations.sh`) verhindert neue Violations und erzwingt Baseline-Updates bei Fixes. Ziel: Baseline auf 0 reduzieren + Ratchet härten.

## Kritische Analyse

### Was der Ratchet prüft
Das Script (`scripts/check-layer-violations.sh:16-27`) sucht nach `import "internal/db"` in API-Dateien. Es erkennt NICHT:
- Indirekte DB-Zugriffe via `s.noteService.GetDB().Method()` (Hintertür)
- `journal.go` und `features.go` haben **8 direkte DB-Aufrufe** via GetDB(), sind aber NICHT in der Baseline

### Kategorien der 37 Violations

| Kategorie | Anzahl | Beschreibung |
|-----------|--------|-------------|
| TYPE_ONLY | 4 | Nur `db.SomeType` in Deklarationen |
| ERROR_ONLY | 2 | Nur `db.ErrNotFound` etc. |
| TYPE_AND_ERROR | 28 | Kombination aus Types + Errors, keine direkten DB-Calls |
| DIRECT_CALL | 3 | `s.noteService.GetDB().Method()` + db-Import |

**Kernproblem:** 93% der Violations (34/37) sind reine Type-/Error-Referenzen — kein Code ruft DB-Methoden direkt auf. Die 3 echten Layer-Verletzungen plus die 8 GetDB()-Calls in journal.go/features.go sind das eigentliche Problem.

### Ansatz

Type-Aliases + Error-Re-Exports im Service-Layer. Ist kosmetisch für Types, aber:
1. Erfüllt den Ratchet-Check (kein `db`-Import in API)
2. Etabliert die Convention dass API nur über Service kommuniziert
3. Späterer Refactor zu echten DTOs wird einfacher
4. Echte Violations (GetDB()-Calls) werden substantiell behoben

## Umsetzungsplan

### Phase 1: Fundament legen

**1.1 — `service/errors.go` erstellen (neu)**
```go
var (
    ErrNotFound        = db.ErrNotFound
    ErrVersionMismatch = db.ErrVersionMismatch
    ErrInvalidQuery    = db.ErrInvalidQuery
)
```

**1.2 — Type-Aliases pro Domain erstellen (jeweils neue Dateien)**

Aufteilung nach fachlicher Zugehörigkeit statt einer Sammeldatei:

- **`service/note_types.go`** — `Note`, `NoteVersion`, `Backlink`, `GraphData`, `SearchFilters`, `FolderInfo`
- **`service/sharing_types.go`** — `NoteShare`, `FolderShare`, `SharedNote`, `SharedFolder`, `UserSearchResult`
- **`service/admin_types.go`** — `DailyCount`, `DailyFloat`, `ActivityLog`, `ActivityFilter`

Typen die bereits in bestehenden Type-Dateien leben, dort ergänzen:
- **`service/recipes_types.go`** (existiert) — `RecipeMetadata`, `RecipeIngredient`, `RecipeDetail` hinzufügen
- **`service/user_types.go`** (existiert) — `WebAuthnCredential` hinzufügen

**1.3 — GetDB()-Aufrufe durch Service-Methoden ersetzen + Tests**

Neue Service-Methoden (jeweils thin wrapper um DB-Call):

| Methode | Aufrufer | Service-Datei | Test |
|---------|----------|---------------|------|
| `NoteService.SetNoteDueDates(noteID, userID, dueDates)` | notes_crud_create.go:156, notes_crud_update.go:125 | `notes_crud.go` | Unit-Test: Aufruf delegiert korrekt |
| `NoteService.RecordTaskEvent(event)` | task_events.go:103 | `notes_helpers.go` | Unit-Test: Delegation + Error-Propagation |
| `NoteService.GetUserFeature(userID, feature)` | journal.go:15, features.go:36 | neue `notes_features.go` | Unit-Test: Feature found + not found |
| `NoteService.ListUserFeatures(userID)` | features.go:18 | `notes_features.go` | Unit-Test |
| `NoteService.SetUserFeature(userID, feature, enabled, settings)` | features.go:63 | `notes_features.go` | Unit-Test |
| `NoteService.JournalExistsForDate(userID, date)` | journal.go:54 | neue `notes_journal.go` | Unit-Test |
| `NoteService.ListJournalEntries(userID)` | journal.go:81 | `notes_journal.go` | Unit-Test |
| `NoteService.ListJournalDates(userID, year, month)` | journal.go:143 | `notes_journal.go` | Unit-Test |
| `NoteService.ListJournalDatesForYear(userID, year)` | journal.go:192 | `notes_journal.go` | Unit-Test |

**1.4 — `GetDB()` aus NoteService entfernen**
Nachdem alle Aufrufer migriert sind, `GetDB()` in `notes_service.go:41` löschen.

**1.5 — Ratchet härten: GetDB()-Check in CI**

`scripts/check-layer-violations.sh` erweitern um zweiten Check:
```bash
# Check 2: No GetDB() bypass in API layer
GETDB_VIOLATIONS=$(grep -rn '\.GetDB()' "$BACKEND"/internal/api/*.go | grep -v '_test.go' || true)
if [ -n "$GETDB_VIOLATIONS" ]; then
  echo "FAIL: Direct DB access via GetDB() in API layer:"
  echo "$GETDB_VIOLATIONS"
  echo "Refactor to use service methods instead."
  exit 1
fi
```

Damit werden künftige GetDB()-Bypasses vom CI abgefangen.

### Phase 2: API-Dateien migrieren (kleine Batches à 3-4 Dateien)

Pro Batch: `db.X` → `service.X` ersetzen, db-Import entfernen, Datei aus Baseline streichen, Build + Test.

**Batch 1 (4 Dateien) — Error-only + Type-only einfachste:**
`snippets.go`, `templates.go`, `graph.go`, `notes_meta_backlinks.go`

**Batch 2 (4 Dateien) — Type-only Rest:**
`notes_helpers_types.go`, `recipes_types.go`, `recipes_images_signing.go`, `users_types.go`

**Batch 3 (4 Dateien) — Notes CRUD:**
`notes_crud_create.go`, `notes_crud_read.go`, `notes_crud_update.go`, `notes_crud_delete.go`

**Batch 4 (4 Dateien) — Notes Meta:**
`notes_encryption.go`, `notes_meta_color.go`, `notes_meta_rename.go`, `notes_misc_titles.go`

**Batch 5 (3 Dateien) — Notes Rest:**
`notes_trash_actions.go`, `notes_ai_suggest.go`, `versions.go`

**Batch 6 (3 Dateien) — Auth/Users:**
`auth_helpers.go`, `auth_user.go`, `users_encryption.go`

**Batch 7 (3 Dateien) — Users + Admin:**
`users_preferences.go`, `users_security.go`, `admin.go`

**Batch 8 (4 Dateien) — Sharing:**
`sharing_folders.go`, `sharing_notes.go`, `sharing_placements.go`, `sharing_search.go`

**Batch 9 (3 Dateien) — Search + Import:**
`search.go`, `import.go`, `task_events.go`

**Batch 10 (5 Dateien) — Recipes:**
`recipes_collection_shares.go`, `recipes_collections.go`, `recipes_handlers.go`, `recipes_images.go`, `recipes_shared.go`

### Phase 3: Aufräumen

- `scripts/layer-violation-baseline.txt` leeren
- Prüfen ob `GetDB()` komplett entfernt ist
- CHANGELOG.md aktualisieren
- `docs/planning/layer-violations-cleanup.md` synchronisieren

## Kritische Dateien

| Datei | Aktion |
|-------|--------|
| `backend/internal/service/errors.go` | Neu erstellen |
| `backend/internal/service/note_types.go` | Neu erstellen |
| `backend/internal/service/sharing_types.go` | Neu erstellen |
| `backend/internal/service/admin_types.go` | Neu erstellen |
| `backend/internal/service/recipes_types.go` | Type-Aliases ergänzen |
| `backend/internal/service/user_types.go` | Type-Alias ergänzen |
| `backend/internal/service/notes_features.go` | Neu: Feature-Delegation |
| `backend/internal/service/notes_journal.go` | Neu: Journal-Delegation |
| `backend/internal/service/notes_crud.go` | SetNoteDueDates() hinzufügen |
| `backend/internal/service/notes_service.go` | GetDB() entfernen |
| `scripts/check-layer-violations.sh` | GetDB()-Check ergänzen |
| `scripts/layer-violation-baseline.txt` | Pro Batch aktualisieren |
| `backend/internal/api/journal.go` | GetDB()-Calls ersetzen |
| `backend/internal/api/features.go` | GetDB()-Calls ersetzen |
| 37 Dateien in `backend/internal/api/` | db-Import → service-Import |

## Verifikation

**Nach jedem Batch:**
1. `go build -tags "fts5 sqlite_crypt" ./...` — Kompiliert
2. `make test` — Alle Backend-Tests bestehen
3. `bash scripts/check-layer-violations.sh` — Baseline-Check bestanden

**Nach Phase 1 (neue Service-Methoden):**
- Gezielte Unit-Tests für alle 9 neuen Service-Wrapper laufen
- `grep -rn '\.GetDB()' backend/internal/api/` liefert 0 Treffer

**Am Ende:**
- `make lint-golangci` — Keine neuen Lint-Fehler
- `scripts/layer-violation-baseline.txt` ist leer
- CI-Check erkennt sowohl db-Imports als auch GetDB()-Bypasses
