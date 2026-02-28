-- Invalidate legacy recovery keys for users with encrypted notes/versions.
-- Recovery-based DEK re-wrap is not implemented, so keeping these keys suggests
-- a recovery capability that does not exist for encrypted data.
UPDATE user_preferences
SET recovery_key_hash = NULL,
    recovery_key_salt = NULL,
    updated_at = datetime('now')
WHERE (recovery_key_hash IS NOT NULL OR recovery_key_salt IS NOT NULL)
  AND user_id IN (
    SELECT DISTINCT user_id
    FROM notes
    WHERE content_encrypted = 1
    UNION
    SELECT DISTINCT user_id
    FROM note_versions
    WHERE content_encrypted = 1
  );
