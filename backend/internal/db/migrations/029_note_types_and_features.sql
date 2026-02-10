-- =============================================================
-- MODULARES SYSTEM FÜR FEATURES (Journal, etc.)
-- =============================================================

-- 1. Note-Typen (Default 'note' für alle bestehenden Notes)
ALTER TABLE notes ADD COLUMN note_type TEXT DEFAULT 'note';

-- 2. Journal-Datum (unabhängig von Titel, encryption-safe)
-- NULL für normale Notes, YYYY-MM-DD für Journal
ALTER TABLE notes ADD COLUMN journal_date TEXT;

-- 3. User-Features Tabelle
CREATE TABLE IF NOT EXISTS user_features (
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    feature TEXT NOT NULL,
    enabled INTEGER DEFAULT 0,
    settings TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    PRIMARY KEY (user_id, feature)
);

CREATE TRIGGER IF NOT EXISTS user_features_updated
AFTER UPDATE ON user_features
BEGIN
    UPDATE user_features SET updated_at = datetime('now')
    WHERE user_id = NEW.user_id AND feature = NEW.feature;
END;

-- 4. Indexes für Performance
CREATE INDEX IF NOT EXISTS idx_notes_type
ON notes(user_id, note_type, created_at)
WHERE is_deleted = 0;

-- UNIQUE: Ein Journal pro Tag pro User (basiert auf journal_date, nicht title!)
CREATE UNIQUE INDEX IF NOT EXISTS idx_journal_unique_date
ON notes(user_id, journal_date)
WHERE note_type = 'journal' AND is_deleted = 0 AND journal_date IS NOT NULL;

-- Index für Kalender-Queries (Monat)
CREATE INDEX IF NOT EXISTS idx_journal_calendar
ON notes(user_id, journal_date)
WHERE note_type = 'journal' AND is_deleted = 0;

-- 5. Trigger für Datenintegrität
-- journal_date nur für Journal-Notes erlaubt

CREATE TRIGGER IF NOT EXISTS check_journal_date_insert
BEFORE INSERT ON notes
BEGIN
    SELECT CASE
        WHEN NEW.note_type = 'journal' AND NEW.journal_date IS NULL THEN
            RAISE(ABORT, 'journal_date required for note_type=journal')
        WHEN NEW.note_type != 'journal' AND NEW.journal_date IS NOT NULL THEN
            RAISE(ABORT, 'journal_date only allowed for note_type=journal')
    END;
END;

CREATE TRIGGER IF NOT EXISTS check_journal_date_update
BEFORE UPDATE ON notes
BEGIN
    SELECT CASE
        WHEN NEW.note_type = 'journal' AND NEW.journal_date IS NULL THEN
            RAISE(ABORT, 'journal_date required for note_type=journal')
        WHEN NEW.note_type != 'journal' AND NEW.journal_date IS NOT NULL THEN
            RAISE(ABORT, 'journal_date only allowed for note_type=journal')
    END;
END;

-- 6. Migriere existierende Notes (sollte bereits 'note' sein durch DEFAULT)
UPDATE notes SET note_type = 'note' WHERE note_type IS NULL;
