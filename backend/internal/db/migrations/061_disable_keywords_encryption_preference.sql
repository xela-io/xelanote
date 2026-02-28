-- Deprecated: encrypted-note plaintext keyword indexing.
-- Force the stored preference off for all users to keep API state consistent.
UPDATE user_preferences
SET keywords_enabled = 0
WHERE COALESCE(keywords_enabled, 0) != 0;
