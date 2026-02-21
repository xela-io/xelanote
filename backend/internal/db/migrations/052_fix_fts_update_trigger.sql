-- Fix: notes_au trigger unconditionally deletes OLD from FTS, even when
-- OLD.is_deleted=1 (meaning the row was already removed from FTS during
-- soft-delete). This causes "database disk image is malformed" when
-- restoring a soft-deleted note. Add WHERE OLD.is_deleted = 0 guard.

DROP TRIGGER IF EXISTS notes_au;

CREATE TRIGGER notes_au AFTER UPDATE ON notes BEGIN
    INSERT INTO notes_fts(notes_fts, rowid, title, content)
    SELECT 'delete', OLD.note_rowid, OLD.title, OLD.content
    WHERE OLD.is_deleted = 0;
    INSERT INTO notes_fts(rowid, title, content)
    SELECT NEW.note_rowid, NEW.title, NEW.content WHERE NEW.is_deleted = 0;
END;
