-- Migration 021: Encryption User Preferences and Note Versioning Support
-- Resolves:
-- - Issue 5 (P1): Persist encrypt_titles and extract_keywords settings
-- - Issue 8 (P1): Complete Migration 020 TODO for keywords_enabled column
-- - Issue 3 (P0): Add encryption fields to note_versions table

-- ============================================================
-- 1. Add encryption preferences to user_preferences table
-- ============================================================

-- Enable/disable keyword extraction for encrypted notes
-- Default: 0 (disabled) for privacy-first approach
ALTER TABLE user_preferences ADD COLUMN keywords_enabled INTEGER DEFAULT 0;

-- Enable/disable title encryption
-- Default: 0 (titles remain plaintext for better UX)
ALTER TABLE user_preferences ADD COLUMN encrypt_titles INTEGER DEFAULT 0;

-- Recovery key hash (for password recovery without losing encrypted notes)
-- Stores bcrypt hash of recovery key (similar to password)
-- NULL = no recovery key set
ALTER TABLE user_preferences ADD COLUMN recovery_key_hash TEXT;

-- Recovery key salt (for client-side key derivation)
-- Used with Argon2id similar to encryption_salt
-- NULL = no recovery key set
ALTER TABLE user_preferences ADD COLUMN recovery_key_salt BLOB;

-- ============================================================
-- 2. Add encryption fields to note_versions table
-- ============================================================

-- Encrypted content (BLOB for binary data)
-- NULL for plaintext versions (backward compatibility)
ALTER TABLE note_versions ADD COLUMN encrypted_content BLOB;

-- Wrapped DEK (Data Encryption Key encrypted with user's KEK)
-- NULL for plaintext versions
ALTER TABLE note_versions ADD COLUMN wrapped_dek TEXT;

-- Flag: Is content encrypted? (0=plaintext, 1=encrypted)
ALTER TABLE note_versions ADD COLUMN content_encrypted INTEGER DEFAULT 0;

-- Flag: Is title encrypted? (0=plaintext, 1=encrypted)
ALTER TABLE note_versions ADD COLUMN title_encrypted INTEGER DEFAULT 0;

-- Encrypted title (JSON payload if title encrypted, NULL otherwise)
ALTER TABLE note_versions ADD COLUMN encrypted_title TEXT;

-- Encryption version (for future algorithm upgrades)
ALTER TABLE note_versions ADD COLUMN encryption_version INTEGER DEFAULT 0;

-- ============================================================
-- 3. Create index for recovery key lookups
-- ============================================================

-- Index for efficient recovery key validation
CREATE INDEX IF NOT EXISTS idx_user_preferences_recovery_key_hash
ON user_preferences(recovery_key_hash)
WHERE recovery_key_hash IS NOT NULL;

-- ============================================================
-- 4. Add view for encryption preferences monitoring
-- ============================================================

CREATE VIEW IF NOT EXISTS user_encryption_preferences AS
SELECT
    u.id as user_id,
    u.username,
    CASE WHEN u.encryption_salt IS NOT NULL THEN 1 ELSE 0 END as encryption_enabled,
    COALESCE(up.keywords_enabled, 0) as keywords_enabled,
    COALESCE(up.encrypt_titles, 0) as encrypt_titles,
    CASE WHEN up.recovery_key_hash IS NOT NULL THEN 1 ELSE 0 END as recovery_key_set,
    up.created_at as preferences_created_at,
    up.updated_at as preferences_updated_at
FROM users u
LEFT JOIN user_preferences up ON u.id = up.user_id;
