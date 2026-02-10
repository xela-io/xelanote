-- Migration 020: End-to-End Encryption Support
-- Adds encryption salt to users, encryption fields to notes, and keyword search table

-- ============================================================
-- 1. Add encryption salt to users table
-- ============================================================
-- Salt is generated per-user on registration/first login
-- Used for Argon2id key derivation on client-side
ALTER TABLE users ADD COLUMN encryption_salt BLOB;

-- ============================================================
-- 2. Add encryption fields to notes table
-- ============================================================
-- SQLite doesn't support DROP COLUMN before 3.35.0
-- Instead, we use nullable columns and mark old data with flags

-- Encrypted content (BLOB for binary data)
ALTER TABLE notes ADD COLUMN encrypted_content BLOB;

-- Flag: Is content encrypted? (0=plaintext, 1=encrypted)
ALTER TABLE notes ADD COLUMN content_encrypted INTEGER DEFAULT 0;

-- Encrypted title (JSON payload if title encrypted, NULL otherwise)
ALTER TABLE notes ADD COLUMN encrypted_title TEXT;

-- Flag: Is title encrypted? (0=plaintext, 1=encrypted)
ALTER TABLE notes ADD COLUMN title_encrypted INTEGER DEFAULT 0;

-- Wrapped DEK (Data Encryption Key encrypted with user's KEK)
-- Stored as Base64 string
ALTER TABLE notes ADD COLUMN wrapped_dek TEXT;

-- Encryption version (0=plaintext, 1=encrypted with v1 scheme)
-- Allows future algorithm upgrades
ALTER TABLE notes ADD COLUMN encryption_version INTEGER DEFAULT 0;

-- Encryption metadata (JSON: algorithm, KDF params, IV, etc.)
ALTER TABLE notes ADD COLUMN encryption_metadata TEXT;

-- ============================================================
-- 3. Create keywords table (opt-in for searchable keywords)
-- ============================================================
-- WARNING: Keywords leak semantic information!
-- Only populated if user explicitly enables keyword extraction
CREATE TABLE IF NOT EXISTS note_keywords (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    note_id TEXT NOT NULL,
    keyword TEXT NOT NULL,
    created_at TEXT DEFAULT (datetime('now')),
    UNIQUE(note_id, keyword),
    FOREIGN KEY (note_id) REFERENCES notes(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_note_keywords_note_id ON note_keywords(note_id);
CREATE INDEX IF NOT EXISTS idx_note_keywords_keyword ON note_keywords(keyword);

-- ============================================================
-- 4. FTS5 index for keyword search
-- ============================================================
CREATE VIRTUAL TABLE IF NOT EXISTS note_keywords_fts USING fts5(
    note_id UNINDEXED,
    keyword,
    content=note_keywords,
    content_rowid=id
);

-- ============================================================
-- 5. Triggers to keep FTS in sync with keywords table
-- ============================================================

-- Trigger: INSERT
CREATE TRIGGER IF NOT EXISTS note_keywords_ai AFTER INSERT ON note_keywords BEGIN
    INSERT INTO note_keywords_fts(rowid, note_id, keyword)
    VALUES (new.id, new.note_id, new.keyword);
END;

-- Trigger: DELETE
CREATE TRIGGER IF NOT EXISTS note_keywords_ad AFTER DELETE ON note_keywords BEGIN
    DELETE FROM note_keywords_fts WHERE rowid = old.id;
END;

-- Trigger: UPDATE (CRITICAL - was missing in original plan)
CREATE TRIGGER IF NOT EXISTS note_keywords_au AFTER UPDATE ON note_keywords BEGIN
    DELETE FROM note_keywords_fts WHERE rowid = old.id;
    INSERT INTO note_keywords_fts(rowid, note_id, keyword)
    VALUES (new.id, new.note_id, new.keyword);
END;

-- ============================================================
-- 6. Encryption status view (for admin monitoring)
-- ============================================================
CREATE VIEW IF NOT EXISTS encryption_status AS
SELECT
    user_id,
    COUNT(*) as total_notes,
    SUM(CASE WHEN content_encrypted = 1 THEN 1 ELSE 0 END) as encrypted_notes,
    SUM(CASE WHEN content_encrypted = 0 THEN 1 ELSE 0 END) as plaintext_notes,
    ROUND(
        CAST(SUM(CASE WHEN content_encrypted = 1 THEN 1 ELSE 0 END) AS FLOAT) * 100.0 / COUNT(*),
        2
    ) as encryption_percentage
FROM notes
WHERE is_deleted = 0
GROUP BY user_id;

-- ============================================================
-- 7. Add user preference for keyword extraction
-- ============================================================
-- TODO: Add keywords_enabled column to user_preferences table in future migration
-- For now, keywords are opt-in only when explicitly requested via API
