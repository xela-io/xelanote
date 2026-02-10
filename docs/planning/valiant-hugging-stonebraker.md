# Rezept-Feature Implementation Plan (v2 - aktualisiert 2026-02-08)

## Kontext

xelanote ist eine Notiz-App (Go/Chi/SQLite + SvelteKit). Das Rezept-Feature wird als erweiterter Notiz-Typ implementiert (`note_type='recipe'`), analog zum bestehenden Journal-Feature.

**Sharing ist bereits vollstaendig implementiert** (Migration 034-036, Note-Sharing + Folder-Sharing + Placements). Rezepte nutzen das bestehende Sharing-System direkt:
- Einzelne Rezepte koennen per Note-Share geteilt werden
- Der `/Rezepte`-Ordner kann per Folder-Share geteilt werden
- Empfaenger koennen geteilte Rezepte in eigene Ordner platzieren (Placements)
- Berechtigungskette: Note-Share > Folder-Share (Prioritaet)

## Architekturentscheidungen

| Entscheidung | Gewaehlt | Begruendung |
|-------------|---------|------------|
| Datenmodell | `note_type='recipe'` (wie Journal) | Nutzt bestehende Infrastruktur (Versioning, Search, Wikilinks, Tags, Sharing) |
| Zutaten | Strukturierte Rows in `recipe_ingredients` | Ermoeglicht Skalierung, Referenzierung, Einkaufslisten |
| Mengenangaben | `amount REAL` + `amount_text TEXT` + `scalable BOOL` | Exakte Berechnung + Bruch-Darstellung ("1/3") + nicht-skalierbare Zutaten ("1 Prise") |
| E2E-Encryption | **Optional verschluesselbar** | Nutzt bestehenden Encryption-Toggle. Verschluesselte Rezepte koennen nicht geteilt werden (bestehende Business-Regel). Unverschluesselte sind immer teilbar. |
| Editor | Formular-basiert mit Tabs | Zutaten-Formular + Markdown-Editor fuer Anleitung |
| Ordner | Nur `/Rezepte` | Wie `/Journal`, klare Trennung. Auto-erstellt beim Feature-Aktivieren. |
| Collections | Eigene Entitaet (`recipe_collections`) | Kochbuecher mit many-to-many Zuordnung |
| Sharing | **Bestehendes** `note_shares` + `folder_shares` System | Kein neuer Code noetig. Rezepte sind Notes, Sharing funktioniert generisch. |
| Kollaboration | Abwechselnd bearbeiten | Bestehende Versionskontrolle + Konflikterkennung (409 + WebSocket). Kein CRDT/OT noetig. |

---

## Phase 1: Database Migration + Backend (Rezepte Basis)

### 1.1 Migration: `backend/internal/db/migrations/037_recipes.sql`

```sql
-- Recipe Metadata (separate Tabelle, blaeht notes nicht auf)
CREATE TABLE IF NOT EXISTS recipe_metadata (
    note_id TEXT NOT NULL PRIMARY KEY REFERENCES notes(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    servings INTEGER DEFAULT 4,
    original_servings INTEGER DEFAULT 4,
    prep_time_minutes INTEGER,
    cook_time_minutes INTEGER,
    source_url TEXT,
    difficulty TEXT CHECK(difficulty IN ('easy', 'medium', 'hard')),
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);

-- Recipe Ingredients (strukturiert, geordnet)
CREATE TABLE IF NOT EXISTS recipe_ingredients (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    note_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount REAL,                        -- Numerisch fuer Berechnung (NULL = "nach Geschmack")
    amount_text TEXT,                   -- Anzeige-Text ("1/3", "1 1/2", "2-3")
    unit TEXT,                          -- 'g', 'ml', 'EL', 'TL', 'Stk', etc.
    name TEXT NOT NULL,
    group_name TEXT,                    -- 'Teig', 'Sauce', 'Topping'
    display_order INTEGER DEFAULT 0,
    optional INTEGER DEFAULT 0,
    scalable INTEGER DEFAULT 1,         -- 0 = nicht skalieren ("1 Prise Salz")
    created_at TEXT DEFAULT (datetime('now'))
);

-- Recipe Collections (Kochbuecher)
CREATE TABLE IF NOT EXISTS recipe_collections (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    color TEXT,
    display_order INTEGER DEFAULT 0,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    UNIQUE(user_id, name)
);

-- Collection-Recipe Join (many-to-many)
CREATE TABLE IF NOT EXISTS recipe_collection_items (
    collection_id INTEGER NOT NULL REFERENCES recipe_collections(id) ON DELETE CASCADE,
    note_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    display_order INTEGER DEFAULT 0,
    added_at TEXT DEFAULT (datetime('now')),
    PRIMARY KEY (collection_id, note_id)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_recipe_metadata_user ON recipe_metadata(user_id);
CREATE INDEX IF NOT EXISTS idx_recipe_ingredients_note ON recipe_ingredients(note_id, display_order);
CREATE INDEX IF NOT EXISTS idx_recipe_ingredients_user ON recipe_ingredients(user_id);
CREATE INDEX IF NOT EXISTS idx_recipe_collections_user ON recipe_collections(user_id, display_order);
CREATE INDEX IF NOT EXISTS idx_notes_recipe_type ON notes(user_id, note_type, updated_at)
    WHERE note_type = 'recipe' AND is_deleted = 0;

-- Trigger: updated_at
CREATE TRIGGER IF NOT EXISTS recipe_metadata_updated AFTER UPDATE ON recipe_metadata
BEGIN UPDATE recipe_metadata SET updated_at = datetime('now') WHERE note_id = NEW.note_id; END;

CREATE TRIGGER IF NOT EXISTS recipe_collections_updated AFTER UPDATE ON recipe_collections
BEGIN UPDATE recipe_collections SET updated_at = datetime('now') WHERE id = NEW.id; END;
```

**Kein Trigger-Konflikt:** Bestehende Journal-Triggers in Migration 029 erlauben `note_type='recipe'` + `journal_date=NULL` (Trigger prueft nur `note_type='journal'`).

### 1.2 Neue Backend-Dateien

| Datei | Inhalt |
|-------|--------|
| `backend/internal/db/recipes.go` | Go Structs (RecipeMetadata, RecipeIngredient, RecipeCollection, RecipeCollectionItem) + DB-Methoden (CRUD fuer Metadata, Ingredients, Collections, CreateRecipeNote, ListRecipeNotes) |
| `backend/internal/db/recipes_test.go` | DB-Tests |
| `backend/internal/api/recipes.go` | HTTP-Handler fuer alle Recipe-Endpoints |
| `backend/internal/service/recipes.go` | Business Logic (CreateRecipeNote mit Feature-Check, ScaleIngredients, Validation) |

### 1.3 Modifizierte Backend-Dateien

| Datei | Aenderung |
|-------|----------|
| `backend/internal/db/journal.go:8-12` | `NoteTypeRecipe = "recipe"` Konstante hinzufuegen |
| `backend/internal/db/notes.go` | `ListNotesByFolder()`: Neuer Branch fuer `/Rezepte` (zeigt nur `note_type='recipe'`, analog zu `/Journal`-Branch). `CreateRecipeNote()` DB-Methode (analog CreateJournalNote). |
| `backend/internal/api/notes.go` | `createNote()`: Recipe-Validation + Branching (analog Journal-Block). NoteRequest um Recipe-Felder erweitern. |
| `backend/internal/api/api.go` | Recipe-Routes registrieren (unter `/recipes` Route-Group) |

### 1.4 Go Structs (Kerndesign)

```go
type RecipeMetadata struct {
    NoteID          string  `json:"note_id"`
    UserID          int     `json:"user_id"`
    Servings        int     `json:"servings"`
    OriginalServings int    `json:"original_servings"`
    PrepTimeMinutes *int    `json:"prep_time_minutes,omitempty"`
    CookTimeMinutes *int    `json:"cook_time_minutes,omitempty"`
    SourceURL       *string `json:"source_url,omitempty"`
    Difficulty      *string `json:"difficulty,omitempty"`
}

type RecipeIngredient struct {
    ID           int      `json:"id"`
    NoteID       string   `json:"note_id"`
    Amount       *float64 `json:"amount,omitempty"`       // nil = "nach Geschmack"
    AmountText   *string  `json:"amount_text,omitempty"`  // "1/3", "1 1/2"
    Unit         *string  `json:"unit,omitempty"`
    Name         string   `json:"name"`
    GroupName    *string  `json:"group_name,omitempty"`
    DisplayOrder int      `json:"display_order"`
    Optional     bool     `json:"optional"`
    Scalable     bool     `json:"scalable"`               // false = "1 Prise Salz"
}

type RecipeCollection struct {
    ID           int     `json:"id"`
    UserID       int     `json:"user_id"`
    Name         string  `json:"name"`
    Description  *string `json:"description,omitempty"`
    Color        *string `json:"color,omitempty"`
    DisplayOrder int     `json:"display_order"`
    RecipeCount  int     `json:"recipe_count,omitempty"`  // Computed in Queries
}

type RecipeDetail struct {
    Note        Note               `json:"note"`
    Metadata    RecipeMetadata     `json:"metadata"`
    Ingredients []RecipeIngredient `json:"ingredients"`
    Collections []RecipeCollection `json:"collections"`
}
```

### 1.5 API-Endpoints (Rezepte)

| Method | Path | Beschreibung |
|--------|------|-------------|
| POST | `/api/notes` | Rezept erstellen (`note_type='recipe'` + metadata + ingredients im Body) |
| GET | `/api/recipes` | Alle Rezepte des Users (liste mit Metadata) |
| GET | `/api/recipes/{id}` | Rezept-Details (Note + Metadata + Ingredients + Collections) |
| PUT | `/api/recipes/{id}/metadata` | Metadata updaten |
| PUT | `/api/recipes/{id}/ingredients` | Zutaten ersetzen (Replace-All Strategie) |
| GET | `/api/recipes/{id}/scaled?servings=N` | Skalierte Zutaten berechnen |
| GET | `/api/recipes/collections` | Kochbuecher listen |
| POST | `/api/recipes/collections` | Kochbuch erstellen |
| PUT | `/api/recipes/collections/{id}` | Kochbuch updaten |
| DELETE | `/api/recipes/collections/{id}` | Kochbuch loeschen |
| POST | `/api/recipes/collections/{id}/items` | Rezept zu Kochbuch hinzufuegen |
| DELETE | `/api/recipes/collections/{id}/items/{noteId}` | Rezept aus Kochbuch entfernen |
| GET | `/api/recipes/collections/{id}/items` | Rezepte im Kochbuch listen |

**Sharing-Endpoints existieren bereits:**
- `POST /api/notes/{id}/shares` — Rezept teilen (generisch, funktioniert fuer alle note_types)
- `POST /api/folders/{id}/shares` — /Rezepte-Ordner teilen
- `GET /api/shared` — Geteilte Notes/Rezepte abrufen (filtern mit `?type=recipe`)
- `GET /api/shared/folders` — Geteilte Ordner abrufen

**Access-Control fuer Recipe-Endpoints:**
- `GET /api/recipes/{id}`: Owner ODER shared (viewer/editor) — nutzt bestehendes `GetSharePermission()`
- `PUT /api/recipes/{id}/metadata`: Owner ODER editor — analog zu `UpdateSharedNote()`
- `PUT /api/recipes/{id}/ingredients`: Owner ODER editor
- `GET /api/recipes/{id}/scaled`: Owner ODER shared (viewer/editor) — read-only, immer erlaubt

### 1.6 Service Layer Pattern

```go
// CreateRecipeNote - analog zu CreateJournalNote in service/notes.go:988-1036
func (s *RecipeService) CreateRecipeNote(userID int, title, content, folderPath string,
    metadata RecipeMetadata, ingredients []RecipeIngredient) (*Note, error) {
    // 1. Check note limit
    // 2. Check feature enabled: s.db.GetUserFeature(userID, "recipe")
    // 3. s.db.CreateRecipeNote(userID, title, content, "/Rezepte")
    // 4. s.db.SetRecipeMetadata(noteID, metadata)
    // 5. s.db.SetRecipeIngredients(noteID, ingredients)
    // 6. updateLinks() + invalidateCache()
    // return note
}

// ScaleIngredients - client-seitig moeglich, aber auch server-seitig fuer API-Konsistenz
func (s *RecipeService) ScaleIngredients(ingredients []RecipeIngredient,
    originalServings, targetServings int) []RecipeIngredient {
    // factor = targetServings / originalServings
    // Fuer jede Zutat: if scalable && amount != nil → amount * factor
    // amount_text wird NICHT skaliert (ist Anzeige-Hint, nicht Rechenquelle)
}
```

### 1.7 Tests (Phase 1)

**DB-Tests (`recipes_test.go`):**
- `TestCreateRecipeNote` — Note mit note_type='recipe' anlegen
- `TestCreateRecipeNote_AutoCreatesFolder` — /Rezepte-Ordner wird erstellt
- `TestRecipeNotIncludedInListNotes` — Rezepte tauchen nicht in normalen Ordnern auf
- `TestRecipeVisibleInRezepteFolder` — Rezepte sind unter /Rezepte sichtbar
- `TestCreateRecipeMetadata` / `TestGetRecipeMetadata` / `TestUpdateRecipeMetadata`
- `TestSetRecipeIngredients` / `TestGetRecipeIngredients` — Replace-All Strategie
- `TestRecipeCollections_CRUD` — Erstellen, Updaten, Loeschen
- `TestRecipeCollectionItems` — Rezept zu/aus Kochbuch
- `TestScaleIngredients` — Skalierung inkl. nicht-skalierbare Zutaten
- `TestScaleIngredients_NullAmount` — "nach Geschmack" bleibt NULL

**Sharing-Integration-Tests (in bestehenden Tests verifizieren):**
- `TestShareRecipeNote` — Bestehendes Sharing funktioniert mit recipe note_type
- `TestShareRezepteFolder` — Folder-Share gibt Zugriff auf alle Rezepte darin
- `TestSharedRecipeEditorCanUpdateMetadata` — Editor-Rolle erlaubt Metadata-Update
- `TestSharedRecipeViewerCannotUpdate` — Viewer-Rolle blockiert Updates
- `TestEncryptedRecipeCannotBeShared` — Verschluesselte Rezepte nicht teilbar

---

## Phase 2: Frontend — Recipe Editor + UI

### 2.1 Feature Store

**Modifizieren:** `frontend/src/lib/stores/features.svelte.ts`
- Recipe Feature Flag nach Journal-Pattern: `loadRecipeFeature()`, `toggleRecipeFeature()`, `resetRecipeFeature()`
- Getters: `getRecipeFeatureEnabled()`, `getRecipeFeatureLoaded()`

### 2.2 Neue Stores

| Store | Inhalt |
|-------|--------|
| `frontend/src/lib/stores/recipes.svelte.ts` | State: recipes[], currentRecipe (RecipeDetail), collections[]. Actions: loadRecipes(), loadRecipeDetail(), createRecipe(), updateMetadata(), setIngredients(), scaleIngredients(). Collections: loadCollections(), createCollection(), addToCollection(), removeFromCollection(). |

### 2.3 API Client

**Modifizieren:** `frontend/src/lib/api.ts`
- TypeScript Interfaces: `RecipeMetadata`, `RecipeIngredient`, `RecipeCollection`, `RecipeDetail`, `ScaledIngredient`
- Recipe API Funktionen: `getRecipes()`, `getRecipeDetail()`, `updateRecipeMetadata()`, `setRecipeIngredients()`, `getScaledIngredients()`, Collection CRUD

### 2.4 Neue Komponenten

**Rezept-Editor:**

| Komponente | Beschreibung |
|-----------|-------------|
| `RecipeEditor.svelte` | Haupteditor mit Tabs: Zutaten \| Anleitung \| Vorschau. Wraps Editor.svelte fuer den Anleitung-Tab. |
| `RecipeMetadataForm.svelte` | Portionen, Zeiten, Schwierigkeit, Quelle-URL |
| `RecipeIngredientEditor.svelte` | Zutaten-Liste mit Add/Remove Buttons, Gruppen-Handling, Drag-Reorder |
| `RecipeIngredientRow.svelte` | Einzelne Zutat (amount, unit, name, optional, scalable) |
| `RecipeScaleControl.svelte` | Portionsrechner (+/- Buttons, client-seitige Berechnung) |
| `RecipePreview.svelte` | Rezeptkarten-Ansicht (Vorschau-Tab) |

**Sidebar + Navigation:**

| Komponente | Beschreibung |
|-----------|-------------|
| `RecipeButton.svelte` | Sidebar-Button (wie JournalButton.svelte) — oeffnet /recipes Route |

**Collections:**

| Komponente | Beschreibung |
|-----------|-------------|
| `RecipeCollectionList.svelte` | Kochbuch-Liste (in Sidebar oder Rezept-Uebersicht) |
| `RecipeCollectionDialog.svelte` | Kochbuch CRUD Dialog |
| `AddToCollectionDialog.svelte` | Rezept zu Kochbuch(en) zuordnen |

**Neue Routes:**

| Route | Beschreibung |
|-------|-------------|
| `frontend/src/routes/recipes/+page.svelte` | Rezept-Uebersicht (Grid/Liste, Filter, Suche, Kochbuecher) |

### 2.5 Modifizierte Komponenten

| Komponente | Aenderung |
|-----------|----------|
| `Sidebar.svelte` | Rezept-Sektion nach Journal (RecipeButton, analog JournalButton). Feature-Flag-gesteuert. |
| `frontend/src/routes/note/[id]/+page.svelte` | `note_type='recipe'` erkennen → RecipeEditor statt Standard-Editor laden |
| `frontend/src/routes/settings/+page.svelte` | Recipe Feature Toggle (analog Journal-Toggle) |
| `frontend/src/lib/stores/websocket.svelte.ts` | Neue Message-Types: `recipe.metadata.updated`, `recipe.ingredients.updated` fuer Live-Updates bei geteilten Rezepten |

### 2.6 Editor-Integration

```
RecipeEditor.svelte
+-- Tab: Zutaten
|   +-- RecipeMetadataForm (Portionen, Zeiten, Schwierigkeit)
|   +-- RecipeScaleControl (Portionsrechner)
|   +-- RecipeIngredientEditor (strukturierte Zutaten-Liste)
+-- Tab: Anleitung
|   +-- Editor.svelte (bestehender CodeMirror Markdown Editor)
+-- Tab: Vorschau
    +-- RecipePreview (Rezeptkarte mit formatierten Zutaten + gerenderte Anleitung)
```

**Viewer-Modus** (fuer shared Rezepte mit Role=viewer):
- Editor read-only
- Zutaten nicht editierbar
- Portionsrechner funktioniert (read-only, client-seitig)
- Sharing-Badge zeigt "Geteilt von [Username]"

**Sharing-Integration** (nutzt bestehende Komponenten):
- `ShareNoteDialog.svelte` — bereits vorhanden, funktioniert generisch fuer Rezepte
- `EditorMoreMenu.svelte` — "Teilen" Menuepunkt bereits vorhanden
- Verschluesselte Rezepte: "Teilen" deaktiviert (bestehende Business-Regel)
- Geteilte Rezepte sichtbar in `/shared` Route (bereits implementiert)

### 2.7 i18n

**Modifizieren:** `frontend/src/lib/locales/de.json` + `en.json`
- Namespace: `recipes.*` (Editor, Metadata, Ingredients, Collections, Sidebar, Settings)
- Ca. 40-60 neue Keys pro Locale

---

## Phase 3: AI-Integration (optional, nach stabilem MVP)

- `POST /api/recipes/parse-ingredients` — LLM parst Freitext → strukturierte Zutaten
- Rezept-spezifische AI-Actions (Substitutionen, Einheiten-Konvertierung)
- Import von URL (Webseite → Rezept via LLM/Schema.org)

---

## Kritische Dateien (Referenz)

**Backend-Pattern (Journal als Vorlage):**
- `backend/internal/db/journal.go` — Note-Type Konstanten (`NoteTypeJournal`) + DB-Queries
- `backend/internal/db/notes.go:437-545` — `ListNotesByFolder()` mit /Journal Branch
- `backend/internal/db/notes.go:888-972` — `CreateJournalNote()` + `CreateEncryptedJournalNote()`
- `backend/internal/api/notes.go:37-54` — `NoteRequest` Struct mit note_type/journal_date
- `backend/internal/api/notes.go:186-259` — `createNote()` Handler mit Journal-Branching
- `backend/internal/api/api.go:392-396` — Journal Route-Registration
- `backend/internal/service/notes.go:988-1095` — `CreateJournalNote()` Service mit Feature-Check

**Sharing (bereits implementiert, nur nutzen):**
- `backend/internal/db/sharing.go` (938 Zeilen) — 40+ DB-Methoden, Note+Folder Sharing
- `backend/internal/service/sharing.go` (333 Zeilen) — Business Logic, 11 Error Types
- `backend/internal/api/sharing.go` (630 Zeilen) — 13 HTTP-Handler
- `backend/internal/db/migrations/034_note_sharing.sql` — note_shares + folder_shares Tabellen
- `backend/internal/db/migrations/036_shared_note_placements.sql` — Placements
- `frontend/src/lib/stores/sharing.svelte.ts` — Sharing Store
- `frontend/src/lib/components/ShareNoteDialog.svelte` — Share UI

**Frontend-Pattern (Journal als Vorlage):**
- `frontend/src/lib/stores/features.svelte.ts` — Feature Flag Pattern (loadJournalFeature)
- `frontend/src/lib/stores/journal.svelte.ts` — Store Pattern (openJournalForDate)
- `frontend/src/lib/components/JournalButton.svelte` — Sidebar Button Pattern
- `frontend/src/lib/components/Sidebar.svelte` — Feature-Section Integration

**WebSocket (fuer Live-Updates bei geteilten Rezepten):**
- `backend/internal/websocket/manager.go` — BroadcastToUser Pattern
- `frontend/src/lib/stores/websocket.svelte.ts` — Message-Handling
- `frontend/src/lib/stores/notes.svelte.ts` — handleRemoteUpdate() Conflict Detection

---

## Implementierungsreihenfolge

| # | Schritt | Phase | Abhaengigkeit |
|---|---------|-------|---------------|
| 1 | Migration 037 (Recipe Tables) + Go Structs + DB-Layer | 1 | - |
| 2 | DB-Tests (recipes_test.go) | 1 | #1 |
| 3 | Service-Layer (CreateRecipeNote, ScaleIngredients, Validation) | 1 | #1 |
| 4 | API-Handler + Route-Registration (Recipes) | 1 | #3 |
| 5 | Sharing-Integration-Tests (bestehendes Sharing mit Rezepten) | 1 | #4 |
| 6 | Frontend: Feature Store + API Client + Recipe Store | 2 | #4 |
| 7 | Frontend: RecipeEditor + Ingredients + Metadata + Scale Control | 2 | #6 |
| 8 | Frontend: Sidebar-Integration + Settings Toggle | 2 | #6 |
| 9 | Frontend: note/[id] Route — Recipe-Editor-Erkennung | 2 | #7 |
| 10 | Frontend: /recipes Uebersichtsseite | 2 | #6 |
| 11 | Frontend: Collections (Kochbuecher) UI | 2 | #10 |
| 12 | Frontend: WebSocket-Integration (recipe.metadata/ingredients.updated) | 2 | #9 |
| 13 | i18n fuer alle neuen Texte | 2 | #7-#11 |
| 14 | AI-Integration (optional) | 3 | #4 |

---

## Verifikation / Testing

1. **Backend-Tests:** `cd backend && go test -tags "fts5" ./internal/db/ -run "TestRecipe" -v`
2. **Backend-Build:** `cd backend && go build -tags "fts5" ./cmd/server`
3. **Frontend-Build:** `cd frontend && npm run build`
4. **Frontend-Tests:** `cd frontend && npm run test`
5. **Manueller Test — Rezepte:**
   - Settings → Rezepte aktivieren → /Rezepte Ordner wird erstellt
   - Neues Rezept erstellen → Zutaten hinzufuegen (inkl. Gruppen, optionale, nicht-skalierbare)
   - Anleitung als Markdown schreiben → Vorschau pruefen
   - Portionen skalieren → Mengen pruefen
   - Kochbuch erstellen → Rezept zuordnen
   - Rezept ueber Wikilink `[[Rezeptname]]` referenzieren
   - Rezept-Sektion in Sidebar pruefen
   - Rezept-Uebersichtsseite mit Filtern testen
6. **Manueller Test — Sharing (nutzt bestehendes System):**
   - Rezept mit User B teilen (ueber EditorMoreMenu → Teilen)
   - User B sieht Rezept in /shared
   - User B als Editor kann Zutaten + Metadata bearbeiten
   - User B als Viewer kann nur lesen + skalieren
   - /Rezepte-Ordner teilen → User B sieht alle Rezepte
   - Verschluesseltes Rezept teilen → 400 Error (bestehende Regel)
   - Rezept verschluesseln → bestehende Shares werden automatisch entfernt
7. **Manueller Test — Conflict Resolution:**
   - User A + B bearbeiten gleiches geteiltes Rezept
   - User A speichert → Version +1 → WebSocket an B
   - User B speichert → 409 Conflict → Toast mit Optionen
