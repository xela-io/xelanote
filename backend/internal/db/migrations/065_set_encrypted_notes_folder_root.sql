-- Migration 065: Reduce folder-path metadata leakage for encrypted notes
-- Normalize encrypted notes to root folder so plaintext folder taxonomy is not persisted.
-- Appends hex(randomblob(4)) suffix to title_norm when a conflict would arise
-- with the UNIQUE(user_id, folder_path, title_norm) index.

UPDATE notes
SET folder_path = '/',
    title_norm = CASE
      WHEN EXISTS (
        SELECT 1 FROM notes AS other
        WHERE other.user_id = notes.user_id
          AND other.folder_path = '/'
          AND other.title_norm = notes.title_norm
          AND other.id != notes.id
      )
      THEN notes.title_norm || '-' || lower(hex(randomblob(4)))
      ELSE notes.title_norm
    END,
    updated_at = datetime('now')
WHERE (content_encrypted = 1 OR encryption_version IS NOT NULL)
  AND ifnull(folder_path, '/') != '/';
