-- Remove plaintext keyword metadata for encrypted notes.
-- Privacy hardening: encrypted note content must not retain server-searchable keywords.
DELETE FROM note_keywords
WHERE note_id IN (
    SELECT id
    FROM notes
    WHERE content_encrypted = 1
);
