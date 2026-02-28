-- Remove plaintext-derived metadata for encrypted notes.
-- Privacy hardening: encrypted note content must not retain server-side links or due dates.
DELETE FROM links
WHERE source_id IN (
    SELECT id
    FROM notes
    WHERE content_encrypted = 1
);

DELETE FROM unresolved_links
WHERE source_id IN (
    SELECT id
    FROM notes
    WHERE content_encrypted = 1
);

DELETE FROM note_due_dates
WHERE note_id IN (
    SELECT id
    FROM notes
    WHERE content_encrypted = 1
);
