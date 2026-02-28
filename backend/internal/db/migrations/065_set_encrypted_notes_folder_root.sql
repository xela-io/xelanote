-- Migration 065: Reduce folder-path metadata leakage for encrypted notes
-- Normalize encrypted notes to root folder so plaintext folder taxonomy is not persisted.
UPDATE notes
SET folder_path = '/',
    updated_at = datetime('now')
WHERE (content_encrypted = 1 OR encryption_version IS NOT NULL)
  AND ifnull(folder_path, '/') != '/';
