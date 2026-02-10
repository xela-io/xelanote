# Architektur

Dieses Dokument beschreibt die System-Architektur von xelanote, wichtige Design-Entscheidungen und die Implementierungs-Details.

## Inhaltsverzeichnis

- [Überblick](#überblick)
- [Tech Stack](#tech-stack)
- [Backend-Architektur](#backend-architektur)
- [Datenbankschema](#datenbankschema)
- [Wikilink-Parser](#wikilink-parser)
- [Link-Resolution](#link-resolution)
- [Ordner-System](#ordner-system)
- [Trash & Undo/Redo System](#trash--undoredo-system)
  - [Soft Delete & Trash](#soft-delete--trash)
  - [Command Pattern](#command-pattern)
  - [History Store](#history-store)
  - [Toast Notifications](#toast-notifications)
- [Frontend-Architektur](#frontend-architektur)
- [Deployment](#deployment)

---

## Überblick

Xelanote ist eine selbst-gehostete Notiz-Anwendung mit Wiki-style Verlinkung. Die Architektur folgt einem klassischen Client-Server-Modell:

```
┌─────────────────┐         HTTP/REST          ┌──────────────────┐
│                 │ ────────────────────────▶  │                  │
│  SvelteKit SPA  │                            │   Go Backend     │
│  (Frontend)     │ ◀────────────────────────  │   (Chi Router)   │
│                 │         JSON               │                  │
└─────────────────┘                            └────────┬─────────┘
                                                        │
                                                        │ SQL
                                                        ▼
                                               ┌────────────────┐
                                               │  SQLite + FTS5 │
                                               │  (Database)    │
                                               └────────────────┘
```

**Design-Philosophie:**
- **Einfachheit**: Eine einzige SQLite-Datenbank, keine externe Dependencies
- **Self-Contained**: Go Binary serviert das kompilierte Frontend (Single Binary Deployment)
- **Performance**: FTS5 für schnelle Volltextsuche, `journal_mode=DELETE` für Docker-Stabilität
- **Robustheit**: Optimistic Locking mit ETag, Transaktionen für Konsistenz

---

## Tech Stack

### Backend

| Komponente | Technologie | Zweck |
|------------|-------------|-------|
| **Sprache** | Go 1.24+ | Statisches Typsystem, native Performance, einfaches Deployment |
| **HTTP Router** | Chi v5 | Minimaler, idiomatischer Router mit Middleware-Support |
| **Datenbank** | SQLite 3 + FTS5 | Embedded Database, keine separate DB-Installation nötig |
| **UUID** | google/uuid | Generierung eindeutiger IDs für Notizen |

**Warum Go?**
- Single Binary Deployment (Backend + Frontend in einem Executable)
- Exzellente Concurrency-Unterstützung
- Minimale Runtime-Dependencies
- Cross-Platform Compilation

**Warum SQLite?**
- Zero Configuration
- Atomic Transactions
- FTS5 für performante Volltextsuche
- `journal_mode=DELETE` für robuste Docker-Deployments (Tradeoff bei Write-Concurrency)
- Perfekt für Self-Hosting (kein separater DB-Server)

### Frontend

| Komponente | Technologie | Zweck |
|------------|-------------|-------|
| **Framework** | SvelteKit 2 | Reaktive UI mit minimaler Bundle-Größe |
| **Sprache** | TypeScript | Type-Safety im Frontend |
| **Styling** | Tailwind CSS v4 | Utility-First CSS Framework |
| **Editor** | CodeMirror 6 | Performanter Markdown-Editor mit Extensions |
| **UI Components** | bits-ui, lucide-svelte | Accessible Components, Icons |
| **Markdown** | markdown-it | Markdown-Rendering für Preview |

**Warum SvelteKit?**
- Minimale Bundle-Größe (wichtig für Self-Hosting)
- Adapter-Static für Static Site Generation
- Reactive Stores für State Management
- TypeScript First-Class Support

---

## Backend-Architektur

### Package-Struktur

```
backend/
├── cmd/server/         # Einstiegspunkt (main.go)
│   └── static/         # Eingebettetes Frontend (embed.FS)
├── internal/
│   ├── api/            # HTTP Handler (Chi Router)
│   │   ├── api.go      # Server-Setup, Middleware, Routes
│   │   ├── notes.go    # CRUD Endpoints
│   │   ├── jobs.go     # Job Status Endpoints (NEU)
│   │   ├── search.go   # Search Endpoints
│   │   └── export.go   # Export Endpoint
│   ├── cache/          # Caching Layer (NEU)
│   │   └── cache.go    # In-Memory Cache mit TTL
│   ├── jobs/           # Background Jobs (NEU)
│   │   ├── jobs.go     # JobManager + Worker Pool
│   │   └── handlers.go # Job Handler Implementations
│   ├── db/             # Datenbank-Layer
│   │   ├── db.go       # Connection, Migration
│   │   ├── notes.go    # Note CRUD
│   │   ├── links.go    # Link Management
│   │   ├── search.go   # FTS5 Queries
│   │   ├── errors.go   # Custom Errors
│   │   └── schema.sql  # Schema Definition
│   ├── parser/         # Wikilink Parser
│   │   └── wikilink.go # Scanner-basierter Parser
│   └── service/        # Business Logic
│       ├── notes.go    # Note Service (CRUD + Link Processing + Cache)
│       └── rename.go   # Rename + Refactoring Logic
└── go.mod
```

### Layer-Architektur

Das Backend folgt einer **3-Schicht-Architektur**:

1. **API Layer** (`internal/api/`)
   - HTTP Request/Response Handling
   - Input Validation
   - Error Mapping zu HTTP Status Codes
   - CORS, Logging, Recovery Middleware

2. **Service Layer** (`internal/service/`)
   - Business Logic
   - Link-Processing nach Note Creation/Update
   - Transaktions-Koordination (z.B. Rename)
   - Orchestrierung mehrerer DB-Calls

3. **Data Layer** (`internal/db/`)
   - SQL Queries
   - Datenbank-Transaktionen
   - Schema-Migrations
   - Row-to-Struct Mapping

**Vorteile dieser Struktur:**
- Klare Trennung von Concerns
- Service Layer kann Business Logic testen ohne HTTP
- DB Layer ist wiederverwendbar
- Einfaches Mocking für Unit Tests

### Request Flow Beispiel

```
Client Request: PUT /api/notes/abc-123
        │
        ▼
[api.updateNote]          # Validiert Request, extrahiert If-Match Header
        │
        ▼
[service.UpdateNote]      # Business Logic: Update + Link Reprocessing
        │
        ├─▶ [db.UpdateNote]        # SQL: UPDATE notes
        │
        └─▶ [service.updateLinks]  # Parse Content → Update links/unresolved_links
                │
                └─▶ [db.SetLinks]  # SQL: DELETE + INSERT in Transaction
        │
        ▼
[api.respondJSON]         # Return updated Note mit ETag Header
```

---

### Realtime Updates (WebSocket)

- Endpoint: `GET /api/ws?token=<access_token>` (JWT im Query-Param beim Upgrade)
- Server pusht Events: `note.created`, `note.updated`, `note.deleted`
- Frontend verarbeitet Updates in der Notes-Store (Toast + List-Update)

---

## Datenbankschema

### ER-Diagramm

```
┌─────────────────┐         ┌─────────────────┐
│     notes       │         │      tags       │
├─────────────────┤         ├─────────────────┤
│ note_rowid (PK) │         │ id (PK)         │
│ id (UUID)       │         │ user_id         │
│ user_id         │         │ name            │
│ title           │         │ name_norm       │
│ title_norm      │         └─────────────────┘
│ content         │                  │
│ folder_path     │                  │
│ display_order   │                  │
│ version         │         ┌────────▼────────┐
│ created_at      │         │   note_tags     │
│ updated_at      │         ├─────────────────┤
│ is_deleted      │◀────────│ note_id (FK)    │
│ deleted_at      │         │ tag_id (FK)     │
└────────┬────────┘         └─────────────────┘
         │                  └─────────────────┘
         │
         │ 1:N
         │
    ┌────▼────────────────────────────┐
    │          links                  │
    ├─────────────────────────────────┤
    │ source_id (FK) → notes.id       │
    │ target_id (FK) → notes.id       │
    │ PRIMARY KEY (source_id, target) │
    └─────────────────────────────────┘
         │
         │ 1:N
         │
    ┌────▼────────────────────────────┐
    │     unresolved_links            │
    ├─────────────────────────────────┤
    │ source_id (FK)                  │
    │ target_ref                      │
    │ target_ref_norm                 │
    │ PRIMARY KEY (source, target)    │
    └─────────────────────────────────┘

    ┌─────────────────────────────────┐
    │       notes_fts (FTS5)          │
    ├─────────────────────────────────┤
    │ rowid → notes.note_rowid        │
    │ title (indexed)                 │
    │ content (indexed)               │
    └─────────────────────────────────┘
```

### Tabellen-Beschreibung

#### `notes`

Haupttabelle für alle Notizen.

```sql
CREATE TABLE notes (
    note_rowid INTEGER PRIMARY KEY,     -- SQLite ROWID für FTS5 Mapping
    id TEXT UNIQUE NOT NULL,            -- UUID für API
    user_id INTEGER,                    -- Owner (multi-user, app-level enforced)
    title TEXT NOT NULL,
    title_norm TEXT NOT NULL,           -- LOWER(TRIM(title)) für Case-Insensitive Matching
    content TEXT NOT NULL,
    folder_path TEXT DEFAULT '/',
    display_order INTEGER NOT NULL DEFAULT 0,
    version INTEGER NOT NULL DEFAULT 1, -- Optimistic Locking
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    is_deleted INTEGER DEFAULT 0,       -- Soft Delete
    deleted_at TEXT                     -- Trash Timestamp
);
```

**Design-Entscheidungen:**

- **`note_rowid`**: SQLite's ROWID wird explizit definiert, da FTS5 nur mit ROWID-basierten Tables arbeiten kann (nicht mit UUIDs)
- **`id` (UUID)**: String-basierte UUIDs für die API (besser für externe Referenzen, URLs)
- **`user_id`**: Multi-User-Isolation (Queries sind user-scoped)
- **`title_norm`**: Vorberechnetes normalisiertes Title-Feld für schnelles Case-Insensitive Matching (user-scoped Unique Index).
- **`display_order`**: Custom Sortierung fuer Drag & Drop (optional genutzt)
- **`version`**: Ermöglicht Optimistic Locking (Client sendet `If-Match: <version>`, Server prüft vor Update)
- **`is_deleted` / `deleted_at`**: Soft Delete statt physischem DELETE (wichtig für Audit-Trail, Trash-UI)

#### `notes_fts`

FTS5 Virtual Table für Volltextsuche.

```sql
CREATE VIRTUAL TABLE notes_fts USING fts5(
    title,
    content,
    content='notes',                          -- External Content Table
    content_rowid='note_rowid',               -- ROWID Mapping
    tokenize='unicode61 remove_diacritics 2'  -- Unicode Tokenizer
);
```

**Design-Entscheidungen:**

- **External Content Table**: FTS5-Index referenziert `notes` Table, speichert Content nicht doppelt (spart Speicher)
- **Triggers**: `AFTER INSERT/UPDATE/DELETE` Triggers halten FTS-Index automatisch synchron
- **Tokenizer**: `unicode61` mit `remove_diacritics` ermöglicht Suche nach "cafe" findet auch "café"

#### `links`

Resolved Links zwischen existierenden Notizen.

```sql
CREATE TABLE links (
    source_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    target_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    PRIMARY KEY (source_id, target_id)
);
```

**Warum separate Links-Tabelle?**
- Ermöglicht effiziente Backlink-Queries (`WHERE target_id = ?`)
- Many-to-Many Beziehung zwischen Notizen
- CASCADE DELETE: Wenn Notiz gelöscht wird, werden alle ihre Links automatisch entfernt

#### `unresolved_links`

Links zu Notizen, die noch nicht existieren.

```sql
CREATE TABLE unresolved_links (
    source_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    target_ref TEXT NOT NULL,           -- Original Title aus [[Link]]
    target_ref_norm TEXT NOT NULL,      -- Normalisiert für Matching
    PRIMARY KEY (source_id, target_ref)
);
```

**Warum Unresolved Links tracken?**

Problem: User schreibt `[[Feature Request]]` bevor die Notiz existiert.

Lösung:
1. Beim Speichern: `[[Feature Request]]` wird in `unresolved_links` eingetragen
2. Beim Erstellen von "Feature Request": System findet `unresolved_links WHERE target_ref_norm = 'feature request'`
3. Alle referenzierenden Notizen werden reprocessed → Unresolved wird zu Resolved Link

Dies ermöglicht **bidirektionale Links ohne Enforcement** (User kann auf nicht-existierende Notizen verlinken).

#### Weitere Tabellen (Migrations)

- **`folders`**: Ordner-Hierarchie (`path`, `parent_id`, `user_id`, `display_order`)
- **`tags` / `note_tags`**: User-scoped Tags und Zuordnung zu Notes
- **`users` / `refresh_tokens`**: Auth und Token-Rotation
- **`note_versions`**: Version History (Snapshots pro Note)
- **`templates` / `snippets`**: Wiederverwendbare Inhalte (user-scoped, `name_norm`)

---

## Wikilink-Parser

### Warum Custom Parser?

**Anforderungen:**
- Parse `[[Title]]` und `[[Title|Alias]]`
- Ignoriere Links in Code Fences (\`\`\`)
- Ignoriere Links in Inline Code (\`)
- Ignoriere Escaped Brackets (`\[\[`)
- Keine Newlines in Links erlauben
- Position-Information für Rename-Refactoring

**Alternative Ansätze:**
1. Regex: Scheitert an verschachtelten Strukturen (Code Fences vs. Inline Code)
2. Markdown Parser Libraries: Zu heavy-weight, keine Position-Info, kein Custom Syntax Support
3. **Custom Scanner** ✓: Vollständige Kontrolle, minimale Dependencies

### Parser-Implementierung

Der Parser in `internal/parser/wikilink.go` ist ein **Zustandsbasierter Scanner**:

```go
type parser struct {
    content string    // Input
    pos     int       // Current position
    result  ParseResult
}

func (p *parser) parse() {
    for p.pos < len(p.content) {
        switch {
        case p.matchCodeFence():
            p.skipCodeFence()      // Überspringe ```...```
        case p.matchInlineCode():
            p.skipInlineCode()     // Überspringe `...`
        case p.matchEscapedBracket():
            p.pos += 2             // Überspringe \[
        case p.matchWikiLink():
            p.parseWikiLink()      // Parse [[...]]
        default:
            p.pos++                // Normale Zeichen überspringen
        }
    }
}
```

### Code Fence Handling

**Problem**: Links innerhalb von Code Blocks sollen nicht parsed werden.

```markdown
Normaler Text [[Link1]] <- wird geparsed

```go
fmt.Println("[[Link2]]")  <- wird NICHT geparsed
```

[[Link3]] <- wird geparsed
```

**Lösung**:

```go
func (p *parser) skipCodeFence() {
    p.pos += 3  // Skip opening ```

    // Skip to end of line (language identifier)
    for p.pos < len(p.content) && p.content[p.pos] != '\n' {
        p.pos++
    }

    // Find closing ```
    for p.pos < len(p.content) {
        if p.matchCodeFence() {
            p.pos += 3
            return
        }
        p.pos++
    }
}
```

### Inline Code Handling

**Problem**: Inline Code kann mehrere Backticks haben.

```markdown
Normal `[[Link]]` <- wird nicht geparsed
Mit Double: ``[[Link]]`` <- wird nicht geparsed
Mit Triple: ```[[Link]]``` <- wird als Code Fence behandelt
```

**Lösung**: Zähle Opening Backticks, suche matching Closing Backticks.

```go
func (p *parser) skipInlineCode() {
    // Count opening backticks
    numBackticks := 0
    for p.pos < len(p.content) && p.content[p.pos] == '`' {
        numBackticks++
        p.pos++
    }

    // Find matching closing
    for p.pos < len(p.content) {
        if /* count matches numBackticks */ {
            return
        }
        p.pos++
    }
}
```

### Wikilink Parsing

**Features:**
- Extrahiere Title und optionalen Alias
- Speichere Byte-Offsets (SpanStart, SpanEnd) für Refactoring
- Handle Edge Cases (leere Links, Newlines, verschachtelte Brackets)

```go
type WikiLink struct {
    TargetRaw   string  // "Title|Alias" oder "Title"
    TargetTitle string  // "Title" (normalisiert, getrimmt)
    Alias       string  // "Alias" oder ""
    SpanStart   int     // Byte offset von [[
    SpanEnd     int     // Byte offset nach ]]
}
```

**Test Coverage**: Siehe `testdata/parser/` für Testvektoren.

---

## Link-Resolution

### Workflow: Note Creation

```
User speichert Note mit Content: "See [[Feature A]] and [[Feature B]]"
                │
                ▼
        [service.CreateNote]
                │
                ├──▶ [db.CreateNote]           # INSERT INTO notes
                │
                ├──▶ [service.updateLinks]     # Parse + Resolve Links
                │      │
                │      ├─▶ [parser.Parse]      # Extract [[Feature A]], [[Feature B]]
                │      │
                │      ├─▶ [db.GetNoteByTitle] # Check if "Feature A" exists
                │      │          │
                │      │          ├─ EXISTS    → Add to resolvedIDs
                │      │          └─ NOT FOUND → Add to unresolvedRefs
                │      │
                │      └─▶ [db.SetLinks]        # Transaktion:
                │                               # DELETE old links
                │                               # INSERT INTO links (resolved)
                │                               # INSERT INTO unresolved_links
                │
                └──▶ [service.resolveUnresolvedLinks]
                       │
                       └─▶ [db.GetUnresolvedBacklinks]  # Find notes waiting for this title
                              │
                              └─▶ Reprocess each → Unresolved → Resolved
```

### Workflow: Note Rename

**Herausforderung**: Wenn "Feature A" → "Feature Alpha" umbenannt wird, müssen alle `[[Feature A]]` Links in anderen Notizen aktualisiert werden.

```
User: POST /api/notes/{id}/rename { "newTitle": "Feature Alpha" }
                │
                ▼
    [service.RenameNote]  # Alles in einer Transaktion!
                │
                ├─▶ UPDATE notes SET title = "Feature Alpha", version++
                │
                ├─▶ SELECT source_id FROM links WHERE target_id = ?
                │   UNION
                │   SELECT source_id FROM unresolved_links WHERE target_ref_norm = ?
                │   # Finde ALLE Notizen, die auf alte Title verweisen
                │
                ├─▶ Für jede referenzierende Note:
                │   │
                │   ├─▶ SELECT content FROM notes WHERE id = source_id
                │   │
                │   ├─▶ [rewriteWikilinks]
                │   │   │
                │   │   ├─▶ [parser.Parse] # Extract all links
                │   │   │
                │   │   ├─▶ For each link where target = "Feature A":
                │   │   │   │
                │   │   │   └─▶ Replace [[Feature A]] → [[Feature Alpha]]
                │   │   │       Replace [[Feature A|Alias]] → [[Feature Alpha|Alias]]
                │   │   │       (preserviere Alias!)
                │   │   │
                │   │   └─▶ Return new content
                │   │
                │   └─▶ UPDATE notes SET content = new_content, version++
                │
                └─▶ COMMIT Transaktion
                    │
                    └─▶ Außerhalb Transaktion: Reprocess links aller affected notes
```

**Wichtige Details:**

1. **Transaktional**: Rename + alle Content-Updates in einer Transaktion (Konsistenz!)
2. **Alias-Preservation**: `[[Feature A|My Alias]]` wird zu `[[Feature Alpha|My Alias]]` (nicht `[[Feature Alpha]]`)
3. **Reverse Order Replacement**: Links werden rückwärts ersetzt (von hinten nach vorne), um Byte-Offsets valide zu halten
4. **Version Increments**: Jede modifizierte Note bekommt `version++` (Clients merken, dass Content sich geändert hat)

### E2E-verschluesselte Notizen

**Problem**: Bei E2E-verschluesselten Notizen kann der Server den Content nicht lesen und daher keine `[[Wikilinks]]` parsen.

**Loesung**: Client-seitige Link-Extraktion

```
User erstellt verschluesselte Notiz mit Content: "See [[Feature A]]"
                │
                ▼
        [Frontend: notes.svelte.ts]
                │
                ├──▶ extractWikiLinks(content)    # Regex-basierte Extraktion
                │         │
                │         └─▶ ["feature a"]       # Normalisiert, dedupliziert
                │
                ├──▶ encryptContent(content)      # AES-256-GCM Verschluesselung
                │
                └──▶ POST /api/notes
                       Body: {
                         title: "...",
                         encrypted_content: "base64...",
                         links: ["feature a"]     # Links als Metadaten
                       }
                │
                ▼
        [Backend: api/notes.go]
                │
                ├──▶ Validierung (max 500 Links, max 200 Zeichen)
                │
                └──▶ [service.UpdateLinksFromClient]
                       │
                       ├─▶ [db.GetNoteByTitle]    # Check ob Target existiert
                       │
                       └─▶ [db.SetLinks]          # Speichere resolved/unresolved
```

**Wichtige Details:**

1. **Vor Verschluesselung**: Links werden im Frontend extrahiert bevor `encryptContent()` aufgerufen wird
2. **Case-Insensitive Deduplication**: `Set<string>` mit `toLowerCase()` verhindert Duplikate
3. **Server-Validierung**: Limits verhindern Missbrauch (500 Links max, 200 Zeichen max)
4. **Transparente Integration**: Backlinks und Graph funktionieren identisch zu unverschluesselten Notizen

**Security-Ueberlegungen:**

- Server sieht Link-Titel (z.B. "Feature A"), aber nicht den Kontext
- Inhalt der Notiz bleibt vollstaendig verschluesselt
- Trade-off: Minimale Metadata-Leakage vs. funktionierende Backlinks

### Link Resolution Performance

**Problem**: Bei vielen Notes kann Link-Resolution langsam werden.

**Optimierungen:**

1. **Index auf `title_norm`**: O(log n) Lookup statt Full Table Scan
2. **Index auf `target_ref_norm`**: Schnelles Finden von Unresolved Links
3. **Deduplication**: `seen map[string]bool` verhindert doppelte Eintraege bei mehrfachen Links
4. **Batch Processing**: `SetLinks()` loescht + inserted alle Links in einer Transaktion

**Trade-off**:
- Mehr Speicher (Links + Unresolved Links Tables)
- Schnellere Backlink-Queries (wichtig fuer UI)

---

## Ordner-System

Xelanote organisiert Notizen in einer **expliziten Ordnerstruktur** (`folders` Tabelle) plus `folder_path` in `notes`.

### Datenmodell

**Folders-Tabelle** (Migration `002_folders_table.sql`, Erweiterungen `003_add_order_fields.sql`, `005_add_user_ownership.sql`):

```sql
CREATE TABLE folders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    path TEXT NOT NULL,
    parent_id INTEGER REFERENCES folders(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    display_order INTEGER NOT NULL DEFAULT 0,
    user_id INTEGER
);

CREATE UNIQUE INDEX idx_folders_user_path ON folders(user_id, path);
```

**Root-Row (Status)**:
- **Update (Migration 025):** Hardcoded Root-Folder id=1 wurde eliminiert.
- Top-Level-Folders haben jetzt `parent_id=NULL` (virtueller Root).
- Per-User-Isolation funktioniert ohne explizite Root-Row.

**Pfad-Format (Validierung)**:
- Root: `/`
- Unterordner: `/Projekte`
- Verschachtelt: `/Projekte/xelanote/Backend`
- Keine Trailing Slashes ausser Root (`backend/internal/utils/paths.go`).

### API

**Ordnerliste (explizit)**:

```http
GET /api/folders
```

Antwort liefert Folder-Objekte inkl. `note_count`, `display_order`, `parent_id` (`backend/internal/api/folders.go`, `backend/internal/db/folders.go`).

**Ordnerverwaltung**:
- `POST /api/folders` (Create)
- `PUT /api/folders/{id}/move`
- `PUT /api/folders/{id}/rename`
- `DELETE /api/folders/{id}`
- `POST /api/folders/reorder`

**Legacy-Endpoint**:

```http
GET /api/folders-legacy
```

Liefert Pfade+Counts aus der Notes-Tabelle fuer aeltere Clients
(`backend/internal/api/search.go`, `backend/internal/db/notes.go`).

### API: Notiz verschieben

**Endpoint:**

```http
PUT /api/notes/:id
If-Match: "5"
Content-Type: application/json

{
  "title": "API Design",
  "content": "...",
  "folder_path": "/Projekte/xelanote"
}
```

**Validierung im Service Layer:**

```go
// internal/service/notes.go
if folderPath != "" && folderPath[0] != '/' {
    return fmt.Errorf("folder path must start with /")
}
if folderPath != "" {
    folderPath = normalizeFolderPath(folderPath) // trim trailing slash
}
```

**Auto-Normalisierung:**
- Trailing Slash wird entfernt
- Input `/Projekte/xelanote/` → gespeichert als `/Projekte/xelanote`

### Frontend: Folder + Tree Stores

- `frontend/src/lib/stores/folders.svelte.ts` laedt Ordner ueber `GET /api/folders` (Dialoge/Filter).
- `frontend/src/lib/stores/tree.svelte.ts` baut den Unified Tree (Folders + Notes) fuer `UnifiedTree.svelte`.
- Expanded-State wird im Tree-Store in `localStorage` persistiert (`xelanote_tree_expanded`).

**Beispiel (Tree-Load):**

```typescript
const [foldersResult, notesResult] = await Promise.all([
  api.getFolders(),
  api.listNotes({ limit: 1000 })
]);
treeData = buildTree(foldersResult.folders ?? [], notesResult.notes ?? []);
```

### Workflow: Notiz verschieben (aktuell)

```
User waehlt Zielordner im Move-Dialog
            │
            ▼
[Frontend] PUT /api/notes/:id
  Body: { ..., "folder_path": "/Projekte/xelanote" }
  Header: If-Match: "<version>"
            │
            ▼
[Backend: service.UpdateNote]
  - validiert leading "/"
  - normalisiert trailing "/" (entfernt)
            │
            ▼
[Response] ETag: "<newVersion>"
```

**Hinweis:** Ordner werden explizit ueber `/api/folders` verwaltet; Notes erzeugen keine Folder-Row automatisch.

### Ordner-Loeschung (explizit)

- `DELETE /api/folders/{id}` scheitert, wenn Notes oder Subfolder existieren.
- Root (`/`) kann nicht geloescht werden.
- Implementierung: `backend/internal/db/folders.go`.

### Performance & Limits

- Folder-Counts werden per JOIN berechnet (`backend/internal/db/folders.go`).
- Tree-Store laedt Notes mit Limit 1000 (`frontend/src/lib/stores/tree.svelte.ts`).

### Weitere Details

- [API Dokumentation](api.md#folders)

## Trash & Undo/Redo System

Das Trash & Undo/Redo System ermöglicht sicheres Löschen mit Wiederherstellungsmöglichkeit und vollständige Undo/Redo-Funktionalität für Property-Änderungen.

### Soft Delete & Trash

#### Database Schema

```sql
-- notes table
deleted_at TEXT DEFAULT NULL    -- Timestamp when deleted
is_deleted INTEGER DEFAULT 0    -- Soft delete flag

-- Index for efficient trash queries
CREATE INDEX idx_notes_deleted ON notes(is_deleted, deleted_at DESC)
  WHERE is_deleted = 1;
```

**Soft Delete Flow**:

```
User clicks Delete
       ↓
DELETE Command executed
       ↓
UPDATE notes SET
  is_deleted = 1,
  deleted_at = NOW(),
  updated_at = NOW()
       ↓
Note verschwindet aus Liste
Note erscheint im Trash
```

**Vorteile**:
- **Sicher**: Kein Datenverlust bei versehentlichem Löschen
- **Audit Trail**: `deleted_at` zeigt wann gelöscht wurde
- **Performance**: Index nur für deleted notes (WHERE is_deleted = 1)
- **Undo möglich**: Snapshot wird im Command gespeichert

#### Trash API

```go
// Database Layer
func (db *DB) ListDeletedNotes(userID, limit, cursor) ([]Note, string, error)
func (db *DB) RestoreNote(userID, id) (*Note, error)
func (db *DB) PermanentlyDeleteNote(userID, id) error
func (db *DB) GetDeletedNotesCount(userID) (int, error)
func (db *DB) EmptyTrash(userID) (int, error)
```

**Endpoints**:
- `GET /api/trash` - Liste gelöschter Notizen (cursor-basiert)
- `GET /api/trash/count` - Anzahl für Badge
- `POST /api/notes/:id/restore` - Wiederherstellen
- `DELETE /api/notes/:id/permanent` - Hard Delete (Safety: nur wenn `is_deleted=1`)
- `DELETE /api/trash` - Alle permanent löschen

### Command Pattern

Das Command Pattern entkoppelt Ausführung von Undo-Logik und ermöglicht vollständige Wiederherstellung.

#### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      User Action                            │
│           (Delete, Create, Rename, Move)                    │
└────────────────────┬────────────────────────────────────────┘
                     ↓
┌─────────────────────────────────────────────────────────────┐
│              Command Pattern Layer                          │
│   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │
│   │DeleteCommand │  │CreateCommand │  │RenameCommand │    │
│   └──────────────┘  └──────────────┘  └──────────────┘    │
└────────────────────┬────────────────────────────────────────┘
                     ↓
┌─────────────────────────────────────────────────────────────┐
│              History Store (Undo/Redo)                      │
│  undoStack: [cmd1, cmd2, cmd3]  ← execute() adds here      │
│  redoStack: [cmd4, cmd5]        ← undo() moves here        │
│                                                             │
│  localStorage: Max 50 commands, 500KB limit                 │
└────────────────────┬────────────────────────────────────────┘
                     ↓
┌─────────────────────────────────────────────────────────────┐
│                    API Layer                                │
│  /api/notes/:id (DELETE, PUT, POST)                        │
└────────────────────┬────────────────────────────────────────┘
                     ↓
┌─────────────────────────────────────────────────────────────┐
│                    Database                                 │
└─────────────────────────────────────────────────────────────┘
```

#### Command Interface

```typescript
interface Command {
  execute(): Promise<boolean>;    // Führt Command aus
  undo(): Promise<boolean>;       // Macht Command rückgängig
  serialize(): CommandData;       // Für localStorage
  getDescription(): string;       // Für UI ("Delete note X")
}
```

#### Concrete Commands

**DeleteCommand**:
```typescript
class DeleteCommand implements Command {
  private snapshot: {
    noteId: string;
    title: string;
    content: string;
    folder_path: string;
    version: number;
  };

  async execute() {
    await api.deleteNote(this.snapshot.noteId);  // Soft delete
  }

  async undo() {
    const note = await api.restoreNote(this.snapshot.noteId);
    const fresh = await api.getNote(this.snapshot.noteId);  // Fresh version!
    await api.updateNote(
      this.snapshot.noteId,
      this.snapshot,  // Restore exact state
      fresh.version   // Use current version (Optimistic Locking)
    );
  }
}
```

**CreateCommand**:
```typescript
class CreateCommand implements Command {
  async execute() {
    const note = await api.createNote({...});
    this.noteId = note.id;  // Store for undo
  }

  async undo() {
    await api.deleteNote(this.noteId);  // Soft-delete created note
  }
}
```

**RenameTitleCommand**:
```typescript
class RenameTitleCommand implements Command {
  private oldTitle: string;
  private newTitle: string;

  async execute() {
    await api.renameNote(noteId, this.newTitle);  // Mit Link Refactoring!
  }

  async undo() {
    await api.renameNote(noteId, this.oldTitle);  // Zurück zu altem Titel
  }
}
```

**Warum kein Content Undo?**

Content Edits (Markdown Änderungen) werden NICHT im Command Pattern gehandhabt:
- ✅ CodeMirror hat **eigenes Undo/Redo System** (built-in)
- ✅ Viel performanter (keine API Calls)
- ✅ Funktioniert granular (jeder Tastendruck)
- ❌ Snapshot-basiertes Undo würde zu viele API Calls verursachen

**Command Pattern ist nur für**:
- Property Changes (Title, Folder)
- Lifecycle Operations (Delete, Create)
- Multi-Note Operations (Rename mit Link Refactoring)

### History Store

#### State Management

```typescript
// history.svelte.ts
let undoStack = $state<Command[]>([]);
let redoStack = $state<Command[]>([]);
const MAX_HISTORY_SIZE = 50;
const STORAGE_KEY = 'xelanote_command_history';
const MAX_STORAGE_SIZE = 500 * 1024;  // 500KB
```

#### Core Functions

```typescript
export async function executeCommand(cmd: Command): Promise<boolean> {
  const success = await cmd.execute();
  if (success) {
    undoStack.push(cmd);
    redoStack = [];  // Clear redo on new action!
    if (undoStack.length > MAX_HISTORY_SIZE) {
      undoStack.shift();  // LRU eviction
    }
    saveHistory();  // Persist to localStorage
  }
  return success;
}

export async function undo(): Promise<boolean> {
  const cmd = undoStack.pop();
  if (!cmd) return false;

  const success = await cmd.undo();
  if (success) {
    redoStack.push(cmd);
    saveHistory();
  } else {
    undoStack.push(cmd);  // Failed, put back
  }
  return success;
}

export async function redo(): Promise<boolean> {
  const cmd = redoStack.pop();
  if (!cmd) return false;

  const success = await cmd.execute();
  if (success) {
    undoStack.push(cmd);
    saveHistory();
  } else {
    redoStack.push(cmd);  // Failed, put back
  }
  return success;
}
```

#### localStorage Persistenz

```typescript
function saveHistory() {
  const historyData = {
    undo: undoStack.map(cmd => cmd.serialize()),
    redo: redoStack.map(cmd => cmd.serialize())
  };

  const json = JSON.stringify(historyData);

  // Size check
  if (json.length > MAX_STORAGE_SIZE) {
    while (undoStack.length > 0 && json.length > MAX_STORAGE_SIZE) {
      undoStack.shift();  // Remove oldest
    }
  }

  localStorage.setItem(STORAGE_KEY, json);
}

export function loadHistory() {
  const stored = localStorage.getItem(STORAGE_KEY);
  if (!stored) return;

  const data = JSON.parse(stored);
  undoStack = data.undo.map(deserializeCommand);
  redoStack = data.redo.map(deserializeCommand);
}
```

**Vorteile**:
- ✅ History überlebt Page Reloads
- ✅ User kann nach Neustart noch Undo machen
- ✅ LRU Eviction verhindert unbegrenztes Wachstum
- ✅ Graceful Degradation bei QuotaExceededError

#### Keyboard Shortcuts

```typescript
// +layout.svelte
function handleKeydown(e: KeyboardEvent) {
  // Ctrl+Z: Undo
  if ((e.ctrlKey || e.metaKey) && e.key === 'z' && !e.shiftKey) {
    if (history.canUndo()) {
      e.preventDefault();
      history.undo();
    }
  }

  // Ctrl+Shift+Z or Ctrl+Y: Redo
  if (
    ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === 'z') ||
    ((e.ctrlKey || e.metaKey) && e.key === 'y')
  ) {
    if (history.canRedo()) {
      e.preventDefault();
      history.redo();
    }
  }
}
```

### Toast Notifications

#### Toast Store

```typescript
interface Toast {
  id: string;
  type: 'success' | 'error' | 'info' | 'warning';
  message: string;
  duration: number;
  action?: { label: string; handler: () => void };
}

export function undoToast(message: string, onUndo: () => void): string {
  return addToast({
    type: 'success',
    message,
    duration: 10000,  // 10s für Undo
    action: { label: 'Undo', handler: onUndo }
  });
}
```

#### Integration mit Delete

```typescript
// Editor.svelte
async function handleDelete() {
  const snapshot = { /* ... */ };
  const deleteCmd = new DeleteCommand(snapshot);

  await history.executeCommand(deleteCmd);

  toast.undoToast('Note moved to trash', async () => {
    const success = await history.undo();
    if (success) {
      toast.success('Note restored');
      await notes.loadNotes();
    }
  });

  goto('/');
}
```

**UX Flow**:
1. User löscht Notiz
2. Toast erscheint: "Note moved to trash" mit "Undo" Button (10s)
3. User kann:
   - Undo Button klicken → Sofortige Wiederherstellung
   - Ctrl+Z drücken → Sofortige Wiederherstellung
   - Nichts tun → Toast verschwindet, Notiz bleibt im Trash
4. Trash Count Badge aktualisiert sich automatisch

### Edge Cases & Lösungen

#### Version Conflicts (Optimistic Locking)

**Problem**: Undo mit stale version → 409 Conflict

**Lösung**: Fresh Version Fetch vor jedem Update
```typescript
async undo() {
  const fresh = await api.getNote(noteId);  // Get current version
  await api.updateNote(noteId, data, fresh.version);  // Use fresh version
}
```

#### localStorage Quota Exceeded

**Problem**: Command History zu groß (>5MB)

**Lösung**:
```typescript
if (json.length > MAX_STORAGE_SIZE) {
  while (undoStack.length > 0 && json.length > MAX_STORAGE_SIZE) {
    undoStack.shift();  // LRU eviction
  }
}
```

**Graceful Degradation**:
```typescript
try {
  localStorage.setItem(STORAGE_KEY, json);
} catch (error) {
  if (error.name === 'QuotaExceededError') {
    clearHistory();  // Reset wenn Quota exceeded
  }
}
```

#### Concurrent Edits

**Problem**: Notiz während Deletion geändert → Snapshot veraltet

**Lösung**:
- Bei Restore: Check ob Content geändert wurde
- Wenn ja: Warnung + Latest Version behalten (nicht Snapshot)
- Future: Conflict Resolution Dialog mit Diff View

---

## Performance-Optimierungen

Xelanote implementiert drei unabhängige Performance-Optimierungs-Strategien für schnelle User Experience und effiziente Backend-Performance.

### 1. Optimistic UI Updates (Frontend)

**Ziel**: Instant UI-Feedback ohne Netzwerk-Latenz

**Pattern**: Snapshot → Optimistic Update → API Call → Revert bei Fehler

**Implementierung** (`frontend/src/lib/stores/`):

```typescript
// notes.svelte.ts: Note Rename
export async function renameCurrentNote(newTitle: string) {
  if (!currentNote) return;

  // 1. Snapshot für Rollback
  const snapshot = {
    note: { ...currentNote },
    notesList: [...notes]
  };

  // 2. Optimistische UI-Aktualisierung (sofort sichtbar!)
  currentNote = { ...currentNote, title: newTitle };
  notes = notes.map(n => n.id === currentNote!.id ? currentNote! : n);
  isDirty = false;

  try {
    // 3. API Call im Hintergrund
    const result = await api.renameNote(snapshot.note.id, newTitle);
    currentNote = result.note;
    await loadNotes();
  } catch (e) {
    // 4. Revert bei Fehler
    currentNote = snapshot.note;
    notes = snapshot.notesList;
    error = e instanceof Error ? e.message : 'Failed to rename note';
    throw e;
  }
}
```

**Optimierte Operationen**:
- Note Rename (`notes.svelte.ts:176-252`)
- Note Move (`notes.svelte.ts:254-291`)
- Folder Create (`tree.svelte.ts:300-336`)
- Folder Rename (`tree.svelte.ts:349-369`)
- Folder Move (`tree.svelte.ts:371-413`)
- Folder Delete (`tree.svelte.ts:415-449`)

**Vorteile**:
- ✅ UI fühlt sich instant an (0ms perceived latency)
- ✅ Funktioniert bei schlechtem Netzwerk
- ✅ Automatischer Rollback bei Fehlern
- ✅ Kein Flicker während API-Calls

### 2. Backend Caching Layer

**Ziel**: 20-100x Speedup für Folder-Operationen

**Architektur**: In-Memory Cache zwischen Service und DB Layer

**Cache Implementation** (`backend/internal/cache/cache.go`):

```go
type Cache struct {
    items sync.Map        // Thread-safe storage
    ttl   time.Duration   // Time-to-live (5 minutes)
}

func (c *Cache) Get(key string) (interface{}, bool) {
    if item, ok := c.items.Load(key); ok {
        cached := item.(cacheItem)
        if time.Now().Before(cached.expiration) {
            return cached.value, true
        }
        c.items.Delete(key) // Expired
    }
    return nil, false
}
```

**Cache Keys**:
- `cache:folders:{userID}:all` - Alle Folders für User
- `cache:folder:{userID}:{path}` - Folder by Path
- `cache:folder_id:{userID}:{id}` - Folder by ID

**Integration in NoteService** (`backend/internal/service/notes.go`):

```go
func (s *NoteService) GetAllFolders(userID int) ([]db.Folder, error) {
    key := fmt.Sprintf("cache:folders:%d:all", userID)

    // Cache hit?
    if cached, ok := s.cache.Get(key); ok {
        return cached.([]db.Folder), nil
    }

    // Cache miss - query database
    folders, err := s.db.GetAllFolders(userID)
    if err != nil {
        return nil, err
    }

    // Store in cache
    s.cache.Set(key, folders)
    return folders, nil
}
```

**Cache Invalidierung** (bei Mutations):

```go
func (s *NoteService) invalidateFolderCache(userID int) {
    // Broad invalidation strategy (safe against stale data)
    s.cache.DeleteByPrefix(fmt.Sprintf("cache:folder:%d:", userID))
    s.cache.DeleteByPrefix(fmt.Sprintf("cache:folder_id:%d:", userID))
    s.cache.Delete(fmt.Sprintf("cache:folders:%d:all", userID))
}

func (s *NoteService) CreateFolder(...) (*db.Folder, error) {
    folder, err := s.db.CreateFolder(userID, path, parentID)
    if err != nil {
        return nil, err
    }
    s.invalidateFolderCache(userID)  // Cache invalidieren!
    return folder, nil
}
```

**Performance-Metriken**:
| Operation | Ohne Cache | Mit Cache | Speedup |
|-----------|------------|-----------|---------|
| GetAllFolders | 50-500ms | <5ms | 20-100x |
| GetFolderByPath | 10-100ms | <1ms | 50-100x |
| GetFolderByID | 10-100ms | <1ms | 50-100x |

**Cache-Properties**:
- TTL: 5 Minuten (configurable)
- Auto-Cleanup: Goroutine räumt expired entries auf
- Thread-Safe: sync.Map für concurrent access
- Invalidation: Breite Invalidierung bei Mutations (sicher)

### 3. Background Jobs Infrastructure

**Ziel**: Async Processing für Long-Running Operations (>100 Backlinks)

**Architektur**: Producer-Consumer Pattern mit Worker Pool

**JobManager** (`backend/internal/jobs/jobs.go`):

```go
type JobManager struct {
    jobs     sync.Map        // map[string]*Job
    queue    chan *Job       // Buffered channel (1000 capacity)
    workers  int             // Worker pool size (4 workers)
    ctx      context.Context
    cancel   context.CancelFunc
    handlers map[JobType]JobHandler
}

func (jm *JobManager) Submit(job *Job) error {
    job.Status = JobStatusPending
    jm.jobs.Store(job.ID, job)
    jm.queue <- job  // Non-blocking enqueue
    return nil
}

func (jm *JobManager) worker() {
    for {
        select {
        case <-jm.ctx.Done():
            return
        case job := <-jm.queue:
            jm.executeJob(job)  // Execute in worker goroutine
        }
    }
}
```

**Job Types & States**:

```go
type JobType string
const (
    JobTypeRenameNote JobType = "rename_note"
    // Weitere: bulk_import, folder_rename, etc.
)

type JobStatus string
const (
    JobStatusPending   JobStatus = "pending"
    JobStatusRunning   JobStatus = "running"
    JobStatusCompleted JobStatus = "completed"
    JobStatusFailed    JobStatus = "failed"
)
```

**Async API Endpoint** (`backend/internal/api/notes.go`):

```go
func (s *Server) renameNote(w http.ResponseWriter, r *http.Request) {
    // Check für async mode
    asyncMode := r.URL.Query().Get("async") == "true"

    if asyncMode {
        // Submit job und return sofort mit 202 Accepted
        jobID := fmt.Sprintf("job_%d_%d", userID, time.Now().UnixNano())
        job := &jobs.Job{
            ID:     jobID,
            Type:   jobs.JobTypeRenameNote,
            UserID: userID,
            Metadata: map[string]interface{}{
                "noteID":   id,
                "newTitle": req.NewTitle,
            },
        }

        s.jobManager.Submit(job)

        respondJSON(w, http.StatusAccepted, map[string]interface{}{
            "job_id": jobID,
            "status": "pending",
        })
        return
    }

    // Sync mode (default)
    result, err := s.noteService.RenameNote(r.Context(), userID, id, req.NewTitle)
    // ...
}
```

**Job Status API** (`backend/internal/api/jobs.go`):

```http
GET /api/jobs/{id}
Authorization: Bearer <token>
```

Response:
```json
{
  "id": "job_123_1234567890",
  "type": "rename_note",
  "user_id": 1,
  "status": "completed",
  "progress": 1.0,
  "result": {
    "note": { /* ... */ },
    "updated_note_count": 15
  },
  "created_at": "2026-01-17T15:00:00Z",
  "updated_at": "2026-01-17T15:00:05Z"
}
```

**Frontend Polling** (`frontend/src/lib/stores/notes.svelte.ts`):

```typescript
async function pollJobCompletion(jobId: string, maxAttempts = 60): Promise<Job> {
  for (let i = 0; i < maxAttempts; i++) {
    const job = await api.getJobStatus(jobId);
    if (job.status === 'completed' || job.status === 'failed') {
      return job;
    }
    // Wait 1 second between polls
    await new Promise(resolve => setTimeout(resolve, 1000));
  }
  throw new Error('Job timeout - operation took too long');
}

export async function renameCurrentNote(newTitle: string) {
  if (!currentNote) return;

  // Heuristic: >100 backlinks → async mode
  const backlinksResult = await api.getBacklinks(currentNote.id);
  const useAsync = backlinksResult.backlinks.length > 100;

  if (useAsync) {
    toast.info('Renaming note in background...');
    const { job_id } = await api.renameNoteAsync(currentNote.id, newTitle);

    // Poll until complete
    const job = await pollJobCompletion(job_id);

    if (job.status === 'completed') {
      currentNote = job.result.note;
      toast.success('Note renamed successfully');
    } else {
      toast.error(job.error || 'Failed to rename note');
    }
  } else {
    // Sync mode for fast operations
    const result = await api.renameNote(currentNote.id, newTitle);
    currentNote = result.note;
  }
}
```

**Use Cases**:
- Note Rename mit 100+ Backlinks (5+ Sekunden)
- Bulk Import (viele Notizen gleichzeitig)
- Folder Rename mit vielen Child-Notes

**Worker Pool Configuration**:
- Default: 4 Workers (parallel job execution)
- Queue Capacity: 1000 Jobs
- Graceful Shutdown: context.CancelFunc

---

## Frontend-Architektur

### Package-Struktur

```
frontend/src/
├── lib/
│   ├── components/         # Svelte Components
│   │   ├── Editor.svelte       # CodeMirror Wrapper
│   │   ├── Sidebar.svelte      # Note List
│   │   ├── QuickSwitcher.svelte # Ctrl+P Modal
│   │   └── ...
│   ├── stores/             # Svelte Stores (State)
│   │   ├── notes.ts            # Notes Store + API Calls
│   │   ├── search.ts           # Search State
│   │   └── ui.ts               # UI State (Modals, Theme)
│   ├── editor/             # CodeMirror Setup
│   │   ├── setup.ts            # Extensions, Keymaps
│   │   └── wikilink.ts         # Wikilink Syntax Highlighting
│   └── api.ts              # API Client (fetch wrapper)
├── routes/
│   ├── +page.svelte        # Main Page
│   ├── +layout.svelte      # Global Layout
│   └── notes/[id]/+page.svelte  # Note Detail
└── app.html
```

### State Management

Xelanote verwendet **Svelte Stores** für globalen State:

```typescript
// lib/stores/notes.ts
export const notes = writable<Note[]>([]);
export const currentNote = writable<Note | null>(null);

export async function loadNote(id: string) {
    const response = await fetch(`/api/notes/${id}`);
    const note = await response.json();
    currentNote.set(note);
    return note;
}
```

**Vorteile:**
- Reaktiv: UI updated automatisch bei Store-Änderungen
- Type-Safe: TypeScript Interfaces für alle Stores
- Simpel: Kein Redux Boilerplate

### CodeMirror Integration

**Setup** (`lib/editor/setup.ts`):

```typescript
import { EditorView, basicSetup } from 'codemirror';
import { markdown } from '@codemirror/lang-markdown';

export function createEditor(parent: HTMLElement, content: string) {
    return new EditorView({
        doc: content,
        extensions: [
            basicSetup,
            markdown(),
            wikilinkHighlighter(),  // Custom Extension
            keymap.of([
                { key: 'Mod-s', run: saveNote }  // Ctrl+S
            ])
        ],
        parent
    });
}
```

**Wikilink Highlighting**:

Benutzerdefinierte Syntax-Hervorhebung für `[[Links]]`:

```typescript
const wikilinkHighlighter = ViewPlugin.fromClass(class {
    decorations: DecorationSet;

    update(update: ViewUpdate) {
        // Parse content für [[...]]
        // Erstelle Decorations (CSS classes)
        // Rendere als klickbare Links
    }
});
```

### API Client

Zentralisierter API Client mit Error Handling:

```typescript
// lib/api.ts
export async function apiCall(url: string, options?: RequestInit) {
    const response = await fetch(url, {
        ...options,
        headers: {
            'Content-Type': 'application/json',
            ...options?.headers
        }
    });

    if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Request failed');
    }

    return response.json();
}

// Verwendung in Stores:
export async function createNote(title: string, content: string) {
    return apiCall('/api/notes', {
        method: 'POST',
        body: JSON.stringify({ title, content })
    });
}
```

---

## Deployment

### Single Binary Deployment

Go's `embed` Package ermöglicht Single-File Deployment:

```go
//go:embed static/*
var staticFS embed.FS

func main() {
    // API Routes
    router := chi.NewRouter()
    server := api.NewServer(noteService)

    // Embed Frontend
    staticSub, _ := fs.Sub(staticFS, "static")
    fileServer := http.FileServer(http.FS(staticSub))

    // SPA Routing: Alle Requests → index.html
    router.Handle("/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if _, err := staticFS.Open("static" + r.URL.Path); err == nil {
            fileServer.ServeHTTP(w, r)  // File exists
        } else {
            r.URL.Path = "/"
            fileServer.ServeHTTP(w, r)  // Fallback to index.html
        }
    }))

    http.ListenAndServe(":8080", router)
}
```

**Build Process**:

```bash
# 1. Build Frontend
cd frontend && npm run build
# Output: frontend/build/*

# 2. Copy Frontend to Backend
cp -r frontend/build/* backend/cmd/server/static/

# 3. Build Backend (embeds static/)
cd backend && CGO_ENABLED=1 go build -tags "fts5 sqlite_crypt" -o xelanote ./cmd/server
# Output: Single Binary mit embedded Frontend
# Ohne SQLCipher: -tags "fts5"
```

### Docker Deployment

Multi-Stage Build für minimales Image:

```dockerfile
# Stage 1: Build Frontend
FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: Build Backend
FROM golang:1.24-alpine AS backend
WORKDIR /app
COPY backend/ ./backend/
COPY --from=frontend /app/frontend/build ./backend/cmd/server/static/
WORKDIR /app/backend
RUN CGO_ENABLED=1 go build -tags "fts5 sqlite_crypt" -o /xelanote ./cmd/server

# Stage 3: Runtime
FROM alpine:latest
RUN apk --no-cache add ca-certificates sqlcipher
COPY --from=backend /xelanote /usr/local/bin/
ENTRYPOINT ["xelanote"]
```

**Image Größe**: ~30MB (Alpine + Go Binary + Frontend)

### Datenbank-Konfiguration

```bash
# Umgebungsvariablen
JWT_SECRET=...                     # Required
XELANOTE_DB=/app/data/xelanote.db  # Database path
XELANOTE_DB_KEY_FILE=/run/secrets/xelanote_db_key  # Optional SQLCipher key

# Docker Volume für Persistenz
docker run -d \
  -p 8080:8080 \
  -v xelanote-data:/app/data \
  xelanote:latest
```

**SQLite Journal Mode (DELETE)**:

Explizit gesetzt bei DB-Init:

```go
PRAGMA foreign_keys = ON
PRAGMA journal_mode = DELETE
PRAGMA synchronous = FULL
```

**Grund**:
- Stabileres Verhalten in Docker-Umgebungen
- Tradeoff: weniger Write-Concurrency als WAL

---

## Design-Entscheidungen im Überblick

### Warum keine externe Datenbank?

**Anforderung**: Self-Hosting soll einfach sein (Ein Command, fertig).

**Trade-off**:
- Vorteil: Zero Configuration, Single Binary
- Nachteil: Nicht horizontal skalierbar

**Entscheidung**: SQLite ist perfekt für Personal Note-Taking (tausende Notes, ein User). Für Multi-User würde man Postgres bevorzugen.

### Warum Custom Parser statt Regex?

**Problem**: Regex kann verschachtelte Strukturen nicht korrekt handhaben.

```markdown
`[[Not a link]]` <- Inline Code
```

Regex würde matchen, Scanner überspringt korrekt den Inline Code Block.

### Warum Soft Delete?

**Anforderung**: Möglichkeit für Undo, Audit-Trail.

**Alternative**: Hard Delete + Backup
**Entscheidung**: Soft Delete mit `is_deleted` Flag ist einfacher und ermöglicht spätere Features (Papierkorb, Restore).

### Warum ETag statt Timestamp für Locking?

**Problem**: Timestamp-basiertes Locking ist fehleranfällig (Clock Skew, Race Conditions).

**Lösung**: Optimistic Locking mit Version Counter:

```http
GET /api/notes/123
→ Response: { ..., "version": 5 }
   Header: ETag: "5"

PUT /api/notes/123
   Header: If-Match: "5"
   Body: { ... }

→ Server: Check version = 5 → Update + Increment
→ Response: { ..., "version": 6 }
```

Falls concurrent Update: `409 Conflict`.

---

## Performance-Überlegungen

### Datenbank-Optimierungen

1. **FTS5 Tokenizer**: `unicode61 remove_diacritics` für bessere Suchqualität
2. **Selective Index**: `CREATE INDEX ON notes(user_id, title_norm) WHERE is_deleted = 0` (kleinerer Index, user-scoped)
3. **LIMIT für Pagination**: Verhindert Large Result Sets
4. **Prepared Statements**: DB caching für häufige Queries

### Frontend-Optimierungen

1. **Code Splitting**: SvelteKit lädt nur Pages on-demand
2. **Lazy Loading**: CodeMirror Extensions werden asynchron geladen
3. **Debounced Search**: Quick Switcher wartet 200ms bevor API Call
4. **Virtual Scrolling**: Suche/Trash virtualisiert; Tree/List noch TODO

### Link-Resolution-Optimierungen

1. **Batch Processing**: `SetLinks()` macht DELETE + INSERT in einer Transaktion
2. **Index Lookups**: `title_norm` Index für O(log n) Lookup
3. **Deduplication**: Verhindert redundante Link-Einträge
4. **Async Reprocessing**: Nach Rename erfolgt Link-Reprocessing außerhalb der Transaktion

---

## Zukünftige Erweiterungen

### Geplante Features

1. **Graph View**: Visualisierung des Note-Netzwerks (D3.js)
2. **Tag System**: Bereits im Schema vorhanden (`tags`, `note_tags`)
3. **Plugin System**: Lua/WASM Scripts für Custom Transformations
4. **Sync**: Optional Sync zu Git Repository (Auto-Commit Changes)
5. **Real-time Collaboration**: WebSockets + Operational Transform

### Mögliche Optimierungen

1. **Caching Layer**: Redis Cache für häufige Queries
2. **Background Jobs**: Queue für Link-Reprocessing (bei vielen Notes)
3. **Sharding**: Bei >100k Notes → Split Database
4. **CDN**: Frontend Assets über CDN (bei vielen Usern)

---

## Weitere Ressourcen

- [API Dokumentation](api.md) - Detaillierte Endpoint-Beschreibungen
- [Development Guide](development.md) - Setup, Testing, Contributing
- [SQLite FTS5 Docs](https://www.sqlite.org/fts5.html) - FTS5 Query Syntax
- [CodeMirror 6 Docs](https://codemirror.net/docs/) - Editor Extensions
