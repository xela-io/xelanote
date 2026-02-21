-- xelanote initial database schema (applied once on first run)
-- Ongoing changes are handled by migrations in internal/db/migrations/

-- Enable foreign keys
PRAGMA foreign_keys = ON;

-- Core: Notes
CREATE TABLE IF NOT EXISTS notes (
    note_rowid INTEGER PRIMARY KEY,
    id TEXT UNIQUE NOT NULL,              -- UUID für API
    title TEXT NOT NULL,
    title_norm TEXT NOT NULL,             -- LOWER(TRIM()) für Matching
    content TEXT NOT NULL,
    folder_path TEXT DEFAULT '/',
    version INTEGER NOT NULL DEFAULT 1,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    is_deleted INTEGER DEFAULT 0
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_notes_title_norm ON notes(title_norm) WHERE is_deleted = 0;
CREATE INDEX IF NOT EXISTS idx_notes_folder ON notes(folder_path);
CREATE INDEX IF NOT EXISTS idx_notes_id ON notes(id);

-- Full-Text Search
CREATE VIRTUAL TABLE IF NOT EXISTS notes_fts USING fts5(
    title, content,
    content='notes',
    content_rowid='note_rowid',
    tokenize='unicode61 remove_diacritics 2'
);

-- FTS Triggers (INSERT/UPDATE/DELETE)
CREATE TRIGGER IF NOT EXISTS notes_ai AFTER INSERT ON notes WHEN NEW.is_deleted = 0 BEGIN
    INSERT INTO notes_fts(rowid, title, content)
    VALUES (NEW.note_rowid, NEW.title, NEW.content);
END;

CREATE TRIGGER IF NOT EXISTS notes_au AFTER UPDATE ON notes BEGIN
    INSERT INTO notes_fts(notes_fts, rowid, title, content)
    SELECT 'delete', OLD.note_rowid, OLD.title, OLD.content
    WHERE OLD.is_deleted = 0;
    INSERT INTO notes_fts(rowid, title, content)
    SELECT NEW.note_rowid, NEW.title, NEW.content WHERE NEW.is_deleted = 0;
END;

CREATE TRIGGER IF NOT EXISTS notes_ad AFTER DELETE ON notes BEGIN
    INSERT INTO notes_fts(notes_fts, rowid, title, content)
    VALUES('delete', OLD.note_rowid, OLD.title, OLD.content);
END;

-- Resolved Links (between existing notes)
CREATE TABLE IF NOT EXISTS links (
    source_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    target_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    PRIMARY KEY (source_id, target_id)
);

CREATE INDEX IF NOT EXISTS idx_links_target ON links(target_id);

-- Unresolved Links (target note doesn't exist yet)
CREATE TABLE IF NOT EXISTS unresolved_links (
    source_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    target_ref TEXT NOT NULL,
    target_ref_norm TEXT NOT NULL,        -- Für Case-Insensitive Matching
    PRIMARY KEY (source_id, target_ref)
);

CREATE INDEX IF NOT EXISTS idx_unresolved_norm ON unresolved_links(target_ref_norm);

-- Tags
CREATE TABLE IF NOT EXISTS tags (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    name_norm TEXT UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS note_tags (
    note_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (note_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_note_tags_tag ON note_tags(tag_id);
