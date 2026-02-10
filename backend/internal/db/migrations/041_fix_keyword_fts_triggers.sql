-- Migration 041: Fix keyword FTS triggers for external-content FTS5
-- The DELETE and UPDATE triggers used incorrect syntax (plain DELETE instead of FTS5 delete-command)

-- Fix DELETE trigger: Must use special FTS5 delete-command for external-content tables
DROP TRIGGER IF EXISTS note_keywords_ad;
CREATE TRIGGER note_keywords_ad AFTER DELETE ON note_keywords BEGIN
    INSERT INTO note_keywords_fts(note_keywords_fts, rowid, note_id, keyword)
    VALUES('delete', old.id, old.note_id, old.keyword);
END;

-- Fix UPDATE trigger: Same fix
DROP TRIGGER IF EXISTS note_keywords_au;
CREATE TRIGGER note_keywords_au AFTER UPDATE ON note_keywords BEGIN
    INSERT INTO note_keywords_fts(note_keywords_fts, rowid, note_id, keyword)
    VALUES('delete', old.id, old.note_id, old.keyword);
    INSERT INTO note_keywords_fts(rowid, note_id, keyword)
    VALUES (new.id, new.note_id, new.keyword);
END;

-- Rebuild FTS index for consistency
INSERT INTO note_keywords_fts(note_keywords_fts) VALUES('rebuild');
