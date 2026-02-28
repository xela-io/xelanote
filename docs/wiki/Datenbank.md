# Datenbank (SQLite)

## Überblick

xelanote nutzt **SQLite** als einzige Datenbank — keine externen Abhängigkeiten wie PostgreSQL oder Redis nötig. Besondere Features:

- **WAL-Modus** (Write-Ahead Logging) für parallele Reads
- **FTS5** für Volltextsuche
- **Optionale SQLCipher-Verschlüsselung** der gesamten DB-Datei (nur lokale Builds)

Die DB-Datei liegt standardmäßig unter `./data/xelanote.db` (konfigurierbar via `XELANOTE_DB`).

## Migrationssystem

Migrationen liegen in `backend/internal/db/migrations/` als nummerierte SQL-Dateien:

```
001_create_notes.sql
002_create_folders.sql
003_add_display_order.sql
...
059_add_user_storage_limit.sql
```

Beim Serverstart werden alle noch nicht angewandten Migrationen **sequentiell** ausgeführt. Die bereits ausgeführten werden in einer internen Tabelle getrackt.

**Regel:** Migrationen sind immer **inkrementell** — bestehende Migrationen werden nie geändert, neue werden angehängt.

## Schema-Überblick

### Kern-Tabellen

```sql
-- Benutzer
users (
    id TEXT PRIMARY KEY,           -- UUID
    username TEXT UNIQUE,
    password_hash TEXT,            -- bcrypt
    is_admin BOOLEAN,
    storage_limit INTEGER,         -- Bytes, NULL = kein Limit
    created_at, updated_at
)

-- Notizen
notes (
    id TEXT PRIMARY KEY,           -- UUID
    user_id TEXT REFERENCES users,
    title TEXT,
    content TEXT,
    folder_path TEXT,              -- z.B. "/" oder "/Projekte/Web"
    version INTEGER,               -- Optimistic Locking
    display_order INTEGER,
    color TEXT,                    -- Hex-Farbcode
    note_type TEXT,                -- 'note' | 'journal' | 'recipe' | 'canvas'
    journal_date TEXT,             -- YYYY-MM-DD (nur für Journal)
    ai_enabled BOOLEAN,
    is_deleted BOOLEAN DEFAULT 0,  -- Soft-Delete
    deleted_at DATETIME,

    -- Verschlüsselung
    encrypted_content BLOB,
    content_encrypted BOOLEAN,
    encrypted_title TEXT,
    title_encrypted BOOLEAN,
    wrapped_dek TEXT,              -- Mit KEK verschlüsselter DEK
    encryption_metadata TEXT,      -- JSON (Algo, Nonce, etc.)
    encryption_version INTEGER,

    -- KI
    summary TEXT,
    summary_encrypted BOOLEAN,
    content_hash TEXT,             -- Für Cache-Invalidierung

    created_at, updated_at,
    UNIQUE(user_id, title, folder_path)  -- Titel pro Ordner eindeutig
)

-- Ordner
folders (
    id TEXT PRIMARY KEY,
    user_id TEXT REFERENCES users,
    name TEXT,
    parent_path TEXT,
    display_order INTEGER,
    color TEXT,
    ai_enabled BOOLEAN,
    encryption_default BOOLEAN,
    created_at, updated_at
)
```

### Links und Graph

```sql
-- Wiki-Links zwischen Notizen
links (
    source_note_id TEXT REFERENCES notes,
    target_note_id TEXT REFERENCES notes,
    PRIMARY KEY (source_note_id, target_note_id)
)

-- Unaufgelöste Links (Ziel-Notiz existiert nicht)
-- Gespeichert als JSON-Array im Note-Record: unresolved_link_refs
```

### Versionierung

```sql
-- Versions-Snapshots
note_versions (
    id TEXT PRIMARY KEY,
    note_id TEXT REFERENCES notes,
    user_id TEXT,
    title TEXT,
    content TEXT,
    encrypted_content BLOB,
    version INTEGER,
    created_at DATETIME
)
```

Bei jedem Speichern (wenn sich Inhalt/Titel geändert hat und genug Zeit vergangen ist) wird ein Snapshot in `note_versions` erstellt. Alte Versionen werden regelmäßig bereinigt.

### Auth und Sicherheit

```sql
-- Refresh Tokens
refresh_tokens (
    id TEXT PRIMARY KEY,
    user_id TEXT REFERENCES users,
    token_hash TEXT,             -- SHA-256 Hash (Klartext nie in DB!)
    family TEXT,                 -- Token-Familie für Rotation
    expires_at DATETIME,
    created_at DATETIME
)

-- 2FA
two_factor_auth (
    user_id TEXT PRIMARY KEY REFERENCES users,
    secret TEXT,                 -- TOTP-Secret (verschlüsselt)
    enabled BOOLEAN,
    backup_codes TEXT            -- JSON-Array (gehasht)
)

-- FIDO2/WebAuthn Passkeys
fido2_credentials (
    id TEXT PRIMARY KEY,
    user_id TEXT REFERENCES users,
    name TEXT,                   -- "YubiKey 5C" etc.
    credential_id BLOB,
    public_key BLOB,
    sign_count INTEGER,
    created_at DATETIME
)

-- Account-Sperrung (Brute-Force-Schutz)
account_lockouts (
    user_id TEXT PRIMARY KEY,
    failed_attempts INTEGER,
    locked_until DATETIME
)
```

### Tags und Templates

```sql
-- Tags (M:N-Beziehung)
tags (id, name, user_id, UNIQUE(name, user_id))
note_tags (note_id, tag_id)

-- Vorlagen
templates (id, user_id, title, content, created_at)

-- Snippets (wiederverwendbare Textbausteine)
snippets (id, user_id, title, content, trigger_text, created_at)
```

### Rezepte

```sql
recipe_metadata (
    note_id TEXT PRIMARY KEY REFERENCES notes,
    servings INTEGER,
    prep_time INTEGER,           -- Minuten
    cook_time INTEGER,
    source_url TEXT,
    difficulty TEXT               -- 'easy' | 'medium' | 'hard'
)

recipe_ingredients (
    id TEXT PRIMARY KEY,
    note_id TEXT REFERENCES notes,
    name TEXT,
    amount REAL,
    unit TEXT,
    group_name TEXT,             -- z.B. "Sauce", "Teig"
    display_order INTEGER
)

recipe_images (
    id TEXT PRIMARY KEY,
    note_id TEXT REFERENCES notes,
    filename TEXT,
    display_order INTEGER
)

recipe_collections (
    id TEXT PRIMARY KEY,
    user_id TEXT,
    name TEXT, description TEXT,
    color TEXT
)
-- + recipe_collection_items (M:N)
```

### Einkaufslisten

```sql
shopping_lists (
    id TEXT PRIMARY KEY,
    user_id TEXT REFERENCES users,
    name TEXT,
    color TEXT,
    is_archived BOOLEAN,
    created_at, updated_at
)

shopping_items (
    id TEXT PRIMARY KEY,
    list_id TEXT REFERENCES shopping_lists,
    name TEXT,
    quantity REAL,
    unit TEXT,
    category TEXT,               -- "Obst", "Milchprodukte", etc.
    category_order INTEGER,
    parent_id TEXT,              -- 1 Ebene Verschachtelung
    is_checked BOOLEAN,
    display_order INTEGER,
    added_by_user_id TEXT,
    source_recipe_id TEXT,       -- Von welchem Rezept importiert
    created_at, updated_at
)

shopping_favorites (
    id TEXT PRIMARY KEY,
    user_id TEXT,
    name TEXT,
    default_quantity REAL,
    default_unit TEXT,
    category TEXT,
    usage_count INTEGER          -- Häufigkeit für Sortierung
)

shopping_list_shares (
    id TEXT PRIMARY KEY,
    list_id TEXT,
    shared_with_user_id TEXT,
    role TEXT                    -- 'viewer' | 'editor'
)
```

### Teilen

```sql
note_shares (
    id TEXT PRIMARY KEY,
    note_id TEXT,
    owner_user_id TEXT,
    shared_with_user_id TEXT,
    role TEXT                    -- 'viewer' | 'editor'
)

folder_shares (
    id TEXT PRIMARY KEY,
    folder_id TEXT,
    owner_user_id TEXT,
    shared_with_user_id TEXT,
    role TEXT
)
```

## Volltextsuche (FTS5)

SQLite FTS5 ermöglicht schnelle Volltextsuche über alle Notizen:

```sql
-- Virtuelle FTS5-Tabelle
CREATE VIRTUAL TABLE notes_fts USING fts5(
    title,
    content,
    content=notes,              -- Inhalt aus notes-Tabelle
    content_rowid=rowid
);

-- Trigger halten FTS-Index synchron
CREATE TRIGGER notes_ai AFTER INSERT ON notes BEGIN
    INSERT INTO notes_fts(rowid, title, content)
    VALUES (new.rowid, new.title, new.content);
END;

-- Analog für UPDATE und DELETE
```

### Suchabfrage

```sql
SELECT n.*, snippet(notes_fts, 1, '<mark>', '</mark>', '...', 30)
FROM notes_fts
JOIN notes n ON notes_fts.rowid = n.rowid
WHERE notes_fts MATCH ?
  AND n.user_id = ?
  AND n.is_deleted = 0
ORDER BY rank
LIMIT 200;
```

`note_keywords_fts` existiert weiterhin aus Kompatibilitaetsgruenden, wird fuer verschluesselte Notizen aber nicht mehr befuellt.

## Weitere Tabellen

| Tabelle | Zweck |
|---------|-------|
| `activity_logs` | Aktivitätsprotokoll (wird automatisch bereinigt) |
| `settings` | System-weite Einstellungen (Key-Value) |
| `user_preferences` | Pro-User-Einstellungen (Theme, Sprache, etc.) |
| `user_features` | Feature-Flags pro User (Journal, Canvas, etc.) |
| `note_due_dates` | Extrahierte `@due(...)` Deadlines |
| `note_user_state` | Cursor/Scroll-Position pro User pro Notiz |
| `open_tabs` | Offene Editor-Tabs (für Tab-Persistenz) |
| `task_events` | Checkbox-Interaktionen (für Statistik) |
| `perf_metrics` | Performance-Metriken (opt-in) |
| `analytics_events` | Analytics-Events (opt-in) |
| `error_reports` | Fehlerberichte |
| `home_dashboard_layout` | Dashboard-Layout pro User |

## Nächste Seiten

- [Backend](Backend.md) — Wie die Services die DB nutzen
- [API-Referenz](API-Referenz.md) — Alle REST-Endpunkte
