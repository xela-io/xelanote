-- Migration 022: KEK Persistence and Security Preferences
-- Adds security level settings and multi-device WebAuthn support

-- ============================================================
-- 1. Add security preferences to user_preferences
-- ============================================================

-- Security level: paranoid | balanced | convenient
ALTER TABLE user_preferences ADD COLUMN security_level TEXT DEFAULT 'balanced';

-- Auto-lock timeout in minutes (0 = never)
ALTER TABLE user_preferences ADD COLUMN auto_lock_timeout INTEGER DEFAULT 15;

-- Index
CREATE INDEX IF NOT EXISTS idx_user_preferences_security_level
ON user_preferences(security_level);

-- ============================================================
-- 2. Multi-Device WebAuthn Credentials
-- ============================================================

-- New table for multiple credentials per user
CREATE TABLE IF NOT EXISTS webauthn_credentials (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  credential_id TEXT NOT NULL,
  device_name TEXT,
  created_at TEXT NOT NULL,
  last_used_at TEXT,

  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  UNIQUE(user_id, credential_id)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_webauthn_user_id
ON webauthn_credentials(user_id);

CREATE INDEX IF NOT EXISTS idx_webauthn_credential_id
ON webauthn_credentials(credential_id);

-- ============================================================
-- 3. View for security monitoring
-- ============================================================

CREATE VIEW IF NOT EXISTS user_security_overview AS
SELECT
    u.id as user_id,
    u.username,
    CASE WHEN u.encryption_salt IS NOT NULL THEN 1 ELSE 0 END as encryption_enabled,
    COALESCE(up.security_level, 'balanced') as security_level,
    COALESCE(up.auto_lock_timeout, 15) as auto_lock_timeout_minutes,
    COUNT(wc.id) as webauthn_credential_count,
    up.updated_at as preferences_last_updated
FROM users u
LEFT JOIN user_preferences up ON u.id = up.user_id
LEFT JOIN webauthn_credentials wc ON u.id = wc.user_id
GROUP BY u.id;
