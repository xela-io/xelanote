# Architektur

System-Architektur von xelanote, Design-Entscheidungen und Implementierungs-Details.

> **Coding-Konventionen**: Siehe [conventions.md](conventions.md) fuer Regeln beim Schreiben von neuem Code.

## Inhaltsverzeichnis

- [Ueberblick](#ueberblick)
- [Tech Stack](#tech-stack)
- [Backend-Architektur](#backend-architektur)
- [Frontend-Architektur](#frontend-architektur)
- [Datenbankschema](#datenbankschema)
- [Wikilink-Parser](#wikilink-parser)
- [Link-Resolution](#link-resolution)
- [E2E Encryption](#e2e-encryption)
- [Offline Mode](#offline-mode)
- [Deployment](#deployment)

---

## Ueberblick

Xelanote ist eine selbst-gehostete Notiz-Anwendung mit Wiki-style Verlinkung. Klassisches Client-Server-Modell:

```
┌─────────────────┐       HTTP/REST + WS       ┌──────────────────┐
│  SvelteKit SPA  │ ◀─────────────────────────▶ │   Go Backend     │
│  (Frontend)     │         JSON                │   (Chi Router)   │
└─────────────────┘                             └────────┬─────────┘
                                                         │ SQL
                                                         ▼
                                                ┌────────────────┐
                                                │  SQLite + FTS5 │
                                                └────────────────┘
```

**Design-Philosophie:**

- **Einfachheit**: Eine einzige SQLite-Datenbank, keine externe Dependencies
- **Self-Contained**: Go Binary serviert das kompilierte Frontend (Single Binary Deployment)
- **Performance**: FTS5 fuer Volltextsuche, In-Memory Cache, Virtual Scrolling
- **Security**: E2E Encryption (XChaCha20-Poly1305), HttpOnly Cookies, CSRF, Rate Limiting, WebAuthn
- **Offline-First**: IndexedDB Queue, Background Sync, Conflict Resolution

---

## Tech Stack

### Backend

| Komponente | Technologie | Version |
|------------|-------------|---------|
| Sprache | Go | 1.24 |
| HTTP Router | Chi v5 | 5.1.0 |
| Datenbank | SQLite 3 + FTS5 | via go-sqlite3 1.14.24 |
| Auth | JWT (HS256) | golang-jwt/jwt v5.3.0 |
| WebAuthn | go-webauthn | 0.15.0 |
| 2FA | TOTP | pquerna/otp 1.5.0 |
| WebSocket | gorilla/websocket | 1.5.3 |
| Crypto | Argon2id, bcrypt | golang.org/x/crypto |

### Frontend

| Komponente | Technologie | Version |
|------------|-------------|---------|
| Framework | SvelteKit | 2.50.0 |
| UI Framework | Svelte 5 (Runes) | 5.2.0 |
| Build Tool | Vite | 6.0.0 |
| Styling | Tailwind CSS | 4.0.0 |
| Editor | CodeMirror 6 | 6.35.0 |
| Markdown | markdown-it | 14.1.0 |
| Icons | lucide-svelte | 0.462.0 |
| i18n | svelte-i18n | 4.0.0-next.1 |
| Crypto | @noble/hashes + libsodium | 2.0.1 / 0.8.1 |
| Graph | force-graph | 1.51.0 |
| PWA | vite-plugin-pwa + Workbox | 1.2.0 / 7.4.0 |
| Desktop | Electron | 33.2.0 |
| Virtualisierung | @tanstack/svelte-virtual | 3.13.18 |

---

## Backend-Architektur

### 3-Layer Pattern

```
API Handler (internal/api/)  -->  Service (internal/service/)  -->  DB (internal/db/)
     101 Dateien                      47 Dateien                     70 Dateien
```

1. **API Layer** (`internal/api/`): HTTP Request/Response, Validierung, Auth-Check, CSRF, Rate Limiting, WebSocket Events
2. **Service Layer** (`internal/service/`): Business Logic, Caching, Orchestrierung, Link Processing
3. **DB Layer** (`internal/db/`): SQL Queries, Modelle, 44 Migrationen (002-045)

### Package-Struktur

```
backend/
├── cmd/server/              # Einstiegspunkt (13 Dateien)
│   ├── main.go              # Server-Start, Flag-Parsing
│   ├── server_config.go     # Konfiguration aus Env
│   ├── server_services.go   # Service-Initialisierung
│   ├── server_jobs.go       # Background Job Setup
│   ├── server_llm.go        # AI Provider Setup
│   ├── server_logger.go     # Structured Logging (slog)
│   ├── server_shutdown.go   # Graceful Shutdown
│   ├── server_static.go     # Embedded Frontend (embed.FS)
│   ├── server_turnstile.go  # CAPTCHA Setup
│   ├── server_webauthn.go   # WebAuthn Setup
│   ├── server_websocket.go  # WebSocket Manager
│   ├── server_pprof.go      # Profiling (opt-in)
│   └── server_error_report.go
├── internal/
│   ├── api/                 # HTTP Handler (101 Dateien)
│   │   ├── routes*.go       # Route Registration (6 Dateien)
│   │   ├── auth_*.go        # Auth Endpoints (6 Dateien)
│   │   ├── users_*.go       # User Management (7 Dateien)
│   │   ├── notes_crud_*.go  # Notes CRUD (4 Dateien)
│   │   ├── notes_ai_*.go    # AI Actions (6 Dateien)
│   │   ├── notes_meta_*.go  # Backlinks, Rename, Color (4 Dateien)
│   │   ├── notes_trash_*.go # Trash (2 Dateien)
│   │   ├── folders_*.go     # Folder CRUD (6 Dateien)
│   │   ├── sharing_*.go     # Note/Folder Sharing (5 Dateien)
│   │   ├── recipes_*.go     # Rezepte (8 Dateien)
│   │   └── ...              # Graph, Search, Tags, Templates, etc.
│   ├── service/             # Business Logic (47 Dateien)
│   │   ├── notes_*.go       # Note Service (13 Dateien)
│   │   ├── recipes_*.go     # Recipe Service (7 Dateien)
│   │   ├── auth.go          # Auth Service
│   │   ├── sharing.go       # Sharing Service
│   │   ├── graph.go         # Graph Service
│   │   ├── summarize_*.go   # AI Summarize (4 Dateien)
│   │   └── ...              # Admin, Activity, Templates, etc.
│   ├── db/                  # Datenbank (70 Dateien)
│   │   ├── db.go            # Connection, Migration Runner
│   │   ├── schema.sql       # Base Schema fuer neue DBs
│   │   ├── errors.go        # Custom Error Types
│   │   ├── migrations/      # 44 SQL-Migrationen (002-045)
│   │   ├── notes_*.go       # Note Queries (11 Dateien)
│   │   ├── folders_*.go     # Folder Queries (10 Dateien)
│   │   ├── recipes_*.go     # Recipe Queries (10 Dateien)
│   │   ├── sharing_*.go     # Sharing Queries (7 Dateien)
│   │   ├── preferences_*.go # User Preferences (6 Dateien)
│   │   └── ...              # Auth, Tags, Search, Graph, etc.
│   ├── auth/                # JWT + Upload Signatures (2 Dateien)
│   ├── cache/               # In-Memory Cache mit TTL (1 Datei)
│   ├── crypto/              # API Key Generation (1 Datei)
│   ├── fido2/               # WebAuthn Manager + Store (3 Dateien)
│   ├── jobs/                # Background Job Scheduler (2 Dateien)
│   ├── llm/                 # AI Provider (Claude, Gemini) (6 Dateien)
│   ├── parser/              # Wikilink + Due Date Parser (3 Dateien)
│   ├── utils/               # Path + String Utilities (2 Dateien)
│   └── websocket/           # Connection Manager (1 Datei)
└── go.mod
```

**Gesamt: 252 Go-Dateien + 44 SQL-Migrationen**

### Request Flow

```
Client: PUT /api/notes/abc-123
  │
  ▼
[api.updateNote]              # Validiert Request, prueft If-Match ETag
  │
  ▼
[service.UpdateNote]          # Business Logic: Update + Link Reprocessing
  ├─▶ [db.UpdateNote]        # SQL UPDATE mit Version-Check
  └─▶ [service.updateLinks]  # Parse Content → Resolve/Unresolved Links
  │
  ▼
[WebSocket Broadcast]         # note.updated Event an verbundene Clients
  │
  ▼
[respondJSON + ETag]          # JSON Response mit neuem ETag
```

### Middleware Stack

```
Global:     Logger → Recoverer → RequestID → Compress → CORS → Security Headers
Protected:  authMiddleware → csrfMiddleware
Per-Route:  rateLimitMiddleware(limiter)
Admin:      adminMiddleware (zusaetzlich zu Protected)
```

### Caching

- In-Memory Cache (`internal/cache/`) mit 5min TTL
- Cache Keys: `cache:folders:{userID}:all`, `cache:note:{userID}:{noteID}`
- Invalidierung bei Mutations (breite Strategie, sicher gegen Stale Data)
- Thread-safe via `sync.Map`

### Background Jobs

- Worker Pool Pattern (4 Worker, Queue Capacity 1000)
- Async Mode fuer lange Operationen (Rename mit 100+ Backlinks)
- API: `POST ?async=true` → `202 Accepted` + Job-ID → `GET /api/jobs/{id}` fuer Status

---

## Frontend-Architektur

### Package-Struktur

```
frontend/src/
├── lib/
│   ├── api/                 # API Client Module (25 Dateien)
│   │   ├── client.ts        # Zentraler Client (Token Refresh, Offline Queue)
│   │   ├── notes.ts         # Notes API
│   │   ├── auth.ts          # Auth API
│   │   ├── folders.ts       # Folders API
│   │   ├── recipes.ts       # Recipes API
│   │   ├── sharing.ts       # Sharing API
│   │   └── ...              # +19 weitere Domain-Module
│   ├── api.ts               # Barrel Export (re-exportiert alle api/ Module)
│   ├── components/          # Svelte 5 Komponenten (78 Dateien)
│   │   ├── Editor.svelte    # CodeMirror 6 Wrapper (Hauptkomponente)
│   │   ├── Sidebar.svelte   # Navigation + Note List
│   │   ├── UnifiedTree.svelte # Folder/Note Tree mit Drag&Drop
│   │   ├── Recipe*.svelte   # Rezept-Komponenten (13 Dateien)
│   │   ├── Graph*.svelte    # Graph View (3 Dateien)
│   │   └── ...              # Dialoge, Toolbars, etc.
│   ├── stores/              # Svelte 5 Runes Stores (31 Dateien)
│   │   ├── notes.svelte.ts  # Notes State + CRUD
│   │   ├── auth.svelte.ts   # Auth State + Token Management
│   │   ├── encryption.svelte.ts # E2E Encryption State
│   │   ├── folders.svelte.ts
│   │   ├── recipes.svelte.ts
│   │   ├── sharing.svelte.ts
│   │   ├── tree.svelte.ts   # Unified Tree State
│   │   ├── settings.svelte.ts
│   │   ├── websocket.svelte.ts
│   │   └── ...              # +22 weitere Stores
│   ├── editor/              # CodeMirror 6 Setup (31 Dateien)
│   │   ├── codemirror.ts    # Editor-Instanz + Extensions
│   │   ├── markdown.ts      # Markdown Rendering (Preview)
│   │   ├── wikilink-autocomplete.ts
│   │   ├── find-replace*.ts # Suchen & Ersetzen (3 Dateien)
│   │   ├── task-*.ts        # Task/Checkbox Extensions (4 Dateien)
│   │   ├── image-*.ts       # Bild Upload + Resize (2 Dateien)
│   │   ├── focus-mode-extensions.ts
│   │   ├── spell-check.ts
│   │   └── ...              # AI Actions, Editor Actions, etc.
│   ├── crypto/              # E2E Encryption (8 Dateien)
│   │   ├── e2e.ts           # E2EEncryption Klasse (XChaCha20-Poly1305)
│   │   ├── sodium.ts        # libsodium Wrapper (@noble/hashes)
│   │   ├── kdf.worker.ts    # Argon2id Web Worker
│   │   ├── kek-persistence.ts # KEK in IndexedDB
│   │   ├── fido2.ts         # WebAuthn/FIDO2
│   │   └── webauthn.ts
│   ├── offline/             # Offline Mode (4 Dateien)
│   │   ├── offline-queue.ts # IndexedDB Operation Queue
│   │   ├── sync-manager.svelte.ts # Background Sync
│   │   ├── diff-utils.ts    # Conflict Resolution (3-Way Merge)
│   │   └── types.ts
│   ├── locales/             # i18n (2 Dateien)
│   │   ├── de.json          # ~600+ Keys
│   │   └── en.json
│   ├── themes/              # 23 Themes (11 Light, 12 Dark)
│   ├── config.ts            # Feature Flags
│   ├── i18n.ts              # svelte-i18n Setup
│   └── utils.ts
├── routes/                  # 13 Routes
│   ├── +layout.svelte       # Global Layout, Auth Guard, Encryption Modal
│   ├── +page.svelte         # Home (Zuletzt bearbeitet)
│   ├── note/[id]/           # Note Editor
│   ├── search/              # Volltextsuche
│   ├── graph/               # Graph View
│   ├── trash/               # Papierkorb
│   ├── shared/              # Geteilte Notizen
│   ├── recipes/             # Rezepte
│   ├── journal/             # Journal / Tagebuch
│   ├── due-dates/           # Faelligkeiten
│   ├── settings/            # Einstellungen
│   ├── admin/               # Admin Dashboard
│   ├── login/ + register/   # Auth
│   └── about/               # Ueber
├── app.html
├── app.css                  # Global Styles + Theme Definitions
└── service-worker.ts        # PWA Service Worker (Workbox)
```

### State Management: Svelte 5 Runes

Alle Stores verwenden **Svelte 5 Runes** (`$state`, `$derived`, `$effect`). Keine Svelte 4 Stores.

**Store-Pattern** (Module-Level State + Getter/Setter):

```typescript
// stores/example.svelte.ts
let items = $state<Item[]>([]);
let isLoading = $state(false);

export function getItems() { return items; }
export function getIsLoading() { return isLoading; }

export async function loadItems() {
  isLoading = true;
  try {
    items = await api.getItems();
  } finally {
    isLoading = false;
  }
}
```

**Component Props** (Svelte 5):

```svelte
<script lang="ts">
  interface Props {
    note: Note;
    isSelected: boolean;
  }
  const { note, isSelected }: Props = $props();
</script>
```

### API Layer

- **Modular**: Ein File pro Domain in `lib/api/` (notes.ts, auth.ts, folders.ts, ...)
- **Barrel Export**: `lib/api.ts` re-exportiert alle 25 Module
- **Zentraler Client**: `lib/api/client.ts` mit Token-Refresh (Mutex), Offline Queue, Retry
- **Error Handling**: `ApiError` Klasse mit `.status` - immer `instanceof ApiError`, nie String-Matching

### Realtime: WebSocket

- Verbindung ueber `GET /api/ws` (Cookie-Auth, kein Query-Token)
- Events: `note.created`, `note.updated`, `note.deleted`, `note.restored`
- Echo-Suppression: eigene Aenderungen werden ignoriert (via `isSaving` Check)
- Auto-Reconnect mit Exponential Backoff

### Internationalisierung (i18n)

- Library: `svelte-i18n` mit ICU MessageFormat
- 2 Locales: Deutsch (de), Englisch (en), ~600+ Keys pro Locale
- Template: `{$_('namespace.key')}`, TypeScript: `get(_)('namespace.key')`
- Namespaces: `page.*`, `component.*`, `dialog.*`, `nav.*`, `settings.*`

---

## Datenbankschema

### Kerntabellen

```sql
-- Notizen (Haupttabelle)
notes (
  note_rowid INTEGER PRIMARY KEY,    -- ROWID fuer FTS5 Mapping
  id TEXT UNIQUE NOT NULL,           -- UUID fuer API
  user_id INTEGER,                   -- Owner (Multi-User)
  title TEXT, title_norm TEXT,       -- Titel + normalisiert (LOWER/TRIM)
  content TEXT,                      -- Markdown Content
  encrypted_content BLOB,           -- E2E verschluesselt (wenn aktiv)
  wrapped_dek TEXT,                  -- Wrapped Data Encryption Key
  folder_path TEXT DEFAULT '/',
  display_order INTEGER DEFAULT 0,
  version INTEGER DEFAULT 1,        -- Optimistic Locking
  is_deleted INTEGER DEFAULT 0,     -- Soft Delete
  ...
)

-- Volltextsuche
notes_fts USING fts5(title, content, content='notes', content_rowid='note_rowid',
                     tokenize='unicode61 remove_diacritics 2')

-- Wiki-Links (resolved)
links (source_id TEXT, target_id TEXT, PRIMARY KEY (source_id, target_id))

-- Wiki-Links (unresolved - Target existiert noch nicht)
unresolved_links (source_id TEXT, target_ref TEXT, target_ref_norm TEXT)

-- Ordner (explizite Hierarchie, Virtual Root: parent_id=NULL)
folders (id INTEGER PRIMARY KEY, path TEXT, parent_id INTEGER, name TEXT,
         user_id INTEGER, display_order INTEGER)

-- Tags
tags (id INTEGER PRIMARY KEY, user_id INTEGER, name TEXT, name_norm TEXT)
note_tags (note_id TEXT, tag_id INTEGER)
```

### Auth & Security

```sql
-- Benutzer
users (id INTEGER PRIMARY KEY, username TEXT, email TEXT, password_hash TEXT,
       is_admin INTEGER, encryption_salt BLOB, ...)

-- Token Rotation (gehasht gespeichert)
refresh_tokens (id, user_id, token_hash TEXT, ...)

-- 2FA (TOTP)
two_factor_auth (user_id, secret_encrypted TEXT, backup_codes TEXT, ...)

-- WebAuthn / Passkeys
fido2_credentials (id, user_id, credential_id BLOB, public_key BLOB, ...)
```

### Features

```sql
-- Versionshistorie
note_versions (id, note_id, user_id, version INTEGER, title, content, snapshot_at)

-- Sharing (Notizen + Ordner)
note_shares (id, note_id, owner_user_id, shared_with_user_id, role TEXT)
folder_shares (id, folder_id, owner_user_id, shared_with_user_id, role TEXT)

-- Rezepte
recipes (id, user_id, title, servings, prep_time, cook_time, ...)
recipe_ingredients (id, recipe_id, name, amount, unit, ...)
recipe_collections (id, user_id, name, ...)

-- Templates & Snippets
templates (id, user_id, name, content, ...)
snippets (id, user_id, name, shortcut, content, ...)

-- System
migrations (id, name TEXT, applied_at)  -- Migration Tracking
system_settings (key TEXT PRIMARY KEY, value TEXT)
activity_logs (id, user_id, event_type TEXT, ...)
```

### Migrationen

44 SQL-Migrationen (002-045), forward-only. Alle verwenden `IF NOT EXISTS` / `IF EXISTS`.
Neue DB: `schema.sql` (Basis) + alle Migrationen. Bestehende DB: nur neue Migrationen.

---

## Wikilink-Parser

Custom Scanner-basierter Parser in `backend/internal/parser/wikilink.go`.

### Warum Custom Parser?

- Regex scheitert an verschachtelten Strukturen (Code Fences vs. Inline Code)
- Markdown-Parser Libraries liefern keine Byte-Offsets fuer Refactoring
- Custom Scanner: volle Kontrolle, minimale Dependencies

### Zustandsbasierter Scanner

```go
type parser struct {
    content string
    pos     int
    result  ParseResult
}

func (p *parser) parse() {
    for p.pos < len(p.content) {
        switch {
        case p.matchCodeFence():    p.skipCodeFence()     // ```...```
        case p.matchInlineCode():   p.skipInlineCode()    // `...`
        case p.matchEscapedBracket(): p.pos += 2          // \[
        case p.matchWikiLink():     p.parseWikiLink()     // [[...]]
        default:                    p.pos++
        }
    }
}
```

### WikiLink Struktur

```go
type WikiLink struct {
    TargetRaw   string  // "Title|Alias" oder "Title"
    TargetTitle string  // Normalisiert, getrimmt
    Alias       string  // Optional
    SpanStart   int     // Byte offset von [[
    SpanEnd     int     // Byte offset nach ]]
}
```

Features: Code Fence Skipping, Inline Code Handling (multi-backtick), Escaped Brackets, Alias-Support, Position-Tracking fuer Rename-Refactoring.

---

## Link-Resolution

### Note Creation

```
User speichert Note mit "See [[Feature A]] and [[Feature B]]"
  │
  ▼
[service.CreateNote]
  ├─▶ [db.CreateNote]                # INSERT
  ├─▶ [parser.Parse]                 # Extract [[Feature A]], [[Feature B]]
  ├─▶ [db.GetNoteByTitle]            # Check ob Target existiert
  │     ├─ EXISTS    → resolvedIDs
  │     └─ NOT FOUND → unresolvedRefs
  ├─▶ [db.SetLinks]                  # Transaktion: DELETE old + INSERT new
  └─▶ [service.resolveUnresolvedLinks] # Andere Notes die auf diesen Titel warten
```

### Note Rename (mit Link Refactoring)

```
"Feature A" → "Feature Alpha"
  │
  ▼
[service.RenameNote]  # Alles in einer Transaktion
  ├─▶ UPDATE notes SET title = "Feature Alpha", version++
  ├─▶ SELECT alle Notes die auf "Feature A" verlinken (links + unresolved_links)
  ├─▶ Fuer jede referenzierende Note:
  │     ├─▶ [parser.Parse] content
  │     ├─▶ Replace [[Feature A]] → [[Feature Alpha]] (Alias bleibt erhalten!)
  │     └─▶ UPDATE content, version++
  └─▶ COMMIT + Reprocess links ausserhalb Transaktion
```

Wichtig: Reverse-Order Replacement (von hinten nach vorne) haelt Byte-Offsets valide.

### E2E-verschluesselte Notizen

Server kann Content nicht lesen. Loesung: Client-seitige Link-Extraktion.

```
Frontend: extractWikiLinks(content)  →  ["feature a", "feature b"]
Frontend: encryptContent(content)    →  base64 Blob
POST /api/notes { encrypted_content: "...", links: ["feature a", "feature b"] }
Backend: Validierung (max 500 Links, max 200 Zeichen) + SetLinks()
```

Trade-off: Server sieht Link-Titel (Metadaten), aber nicht den Kontext. Backlinks + Graph funktionieren identisch.

---

## E2E Encryption

### Architektur

```
User Password
  │
  ▼ (Argon2id, Web Worker)
KEK (Key Encryption Key)
  │
  ├─▶ Persistent: IndexedDB (balanced/convenient) oder Memory-only (paranoid)
  ├─▶ Desktop: OS Keyring (Electron)
  └─▶ Optional: WebAuthn Biometric Unlock
  │
  ▼
Per-Note DEK (Data Encryption Key, 32 bytes random)
  │
  ├─▶ DEK wrapped mit KEK → wrapped_dek (in DB)
  └─▶ Content encrypted mit DEK → encrypted_content BLOB (XChaCha20-Poly1305)
```

### Implementierung

- `frontend/src/lib/crypto/e2e.ts`: `E2EEncryption` Klasse (Singleton)
- `frontend/src/lib/crypto/sodium.ts`: Crypto-Primitives via @noble/hashes
- `frontend/src/lib/crypto/kdf.worker.ts`: Argon2id in Web Worker (nicht-blockierend)
- `frontend/src/lib/stores/encryption.svelte.ts`: State Management (lock/unlock/setup)

### Security Levels

| Level | KEK Persistence | Auto-Lock |
|-------|----------------|-----------|
| Paranoid | Memory only | Bei Tab-Wechsel |
| Balanced | IndexedDB (verschluesselt) | Nach 30min Inaktivitaet |
| Convenient | IndexedDB + WebAuthn | Nach 24h |

### Encryption Toggle

Einzelne Notizen koennen entschluesselt werden (POST `/api/notes/{id}/decrypt`). Ordner koennen als "unverschluesselt" markiert werden (neue Notizen darin werden ohne Encryption erstellt). Beim Verschluesseln werden alle Shares automatisch entfernt.

---

## Offline Mode

### Phase 1: Note CRUD Offline

```
Online                              Offline
  │                                   │
  ▼                                   ▼
API Call → Server               Enqueue in IndexedDB
  │                               │
  ▼                               ▼ (bei Reconnect)
Response                        Sync Manager
                                  ├─▶ Web Locks API (Tab-Safety)
                                  ├─▶ Queue-Optimierung (merge/cancel/fold)
                                  ├─▶ Replay Operations sequentiell
                                  └─▶ Conflict Detection (HTTP 409)
                                        ├─▶ Auto-Resolve (trivial)
                                        └─▶ ConflictDialog (3-Way Merge)
```

### Komponenten

- `offline-queue.ts`: IndexedDB mit 3 Stores (operation_queue, local_note_cache, temp_id_mappings)
- `sync-manager.svelte.ts`: Background Sync, Progress Tracking, Web Locks
- `diff-utils.ts`: 3-Way Merge fuer Conflict Resolution
- Temp-ID-System: Offline-erstellte Notizen bekommen temporaere IDs, URL-Rewriting nach Sync

### Einschraenkungen (Phase 1)

Nur Note CRUD offline. Tags, Folders, Rename, Trash-Restore bleiben online-only.
Paranoid Security Mode: read-only offline (kein Klartext in IndexedDB).

---

## Deployment

### Single Binary

Go `embed.FS` bettet das kompilierte Frontend ein. Ein Binary, keine Dependencies.

```bash
make build  # Frontend build + Copy + Go build → bin/xelanote
```

### Docker (Multi-Stage)

```
Stage 1: node:22-alpine    → npm ci + npm run build
Stage 2: golang:1.24-alpine → go build -tags "fts5"
Stage 3: alpine:latest      → ~30MB Runtime Image
```

### CI/CD

Forgejo Actions Workflow (`deploy-staging.yml`):

1. **Quality Gate Backend**: go vet + go test (in Docker)
2. **Quality Gate Frontend**: ESLint + Prettier + Markdownlint (in Docker)
3. **Pre-flight Checks**: Env-File, JWT_SECRET Laenge, Docker Daemon
4. **Build**: Docker Image mit SHA-Tag
5. **Deploy**: Stop old → Start new (Zero-Downtime)
6. **Health Check**: 30 Versuche, 2s Intervall
7. **Auto-Rollback**: Bei Health-Check Failure → vorheriges Image

Security Hardening: `--read-only`, `--cap-drop ALL`, `--no-new-privileges`, `--tmpfs /tmp`.

### Umgebungen

| Umgebung | URL | Infra |
|----------|-----|-------|
| Staging | notes.over-cloud.de | Homelab |
| Production | xelanote.com | Hetzner |

Beide pullen von Forgejo (`git.over-cloud.de`). Vollstaendige Anleitung: `docs/deployment.md`.

---

## Design-Entscheidungen

### SQLite statt Postgres/MySQL

- Zero Configuration, Single Binary, perfekt fuer Self-Hosting
- FTS5 fuer Volltextsuche integriert
- `journal_mode=DELETE` fuer Docker-Stabilitaet (Tradeoff: weniger Write-Concurrency)
- Nicht horizontal skalierbar, aber fuer Personal Note-Taking (tausende Notes, wenige User) ideal

### Custom Wikilink-Parser statt Regex

- Regex kann verschachtelte Strukturen nicht korrekt handhaben (Code Fences in Code Fences)
- Byte-Offsets noetig fuer Rename-Refactoring (kein Markdown-Parser liefert das)

### Soft Delete statt Hard Delete

- Papierkorb mit Wiederherstellung, Audit Trail
- Command Pattern fuer Undo/Redo (Delete, Create, Rename)
- `is_deleted` Flag + `deleted_at` Timestamp

### Version-basiertes Optimistic Locking statt Timestamps

- Kein Clock Skew Problem
- ETag Header: `SHA256(noteID + version)[:8]`
- 409 Conflict bei concurrent Updates

### @noble/hashes statt libsodium-wasm

- Pure JavaScript, kein WASM (funktioniert in allen Environments inkl. Electron)
- Vom renommierten Crypto-Autor Paul Miller
- Argon2id fuer KDF, XChaCha20-Poly1305 via libsodium-wrappers fuer Encryption
