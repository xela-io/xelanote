-- Skip FTS delete when a note was already soft-deleted.
DROP TRIGGER IF EXISTS notes_ad;

CREATE TRIGGER notes_ad AFTER DELETE ON notes WHEN OLD.is_deleted = 0 BEGIN
    INSERT INTO notes_fts(notes_fts, rowid, title, content)
    VALUES('delete', OLD.note_rowid, OLD.title, OLD.content);
END;
