-- Migration 045: Secure registration defaults.
-- If no users exist yet, disable public registration by default.

INSERT OR IGNORE INTO system_settings (key, value)
VALUES ('registration_enabled', 'false');

UPDATE system_settings
SET value = 'false', updated_at = datetime('now')
WHERE key = 'registration_enabled'
  AND (SELECT COUNT(*) FROM users) = 0;
