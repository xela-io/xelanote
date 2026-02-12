-- Migration 044: Refresh token reuse detection support.
-- Adds token family lineage and token state markers for secure rotation.

ALTER TABLE refresh_tokens ADD COLUMN family_id TEXT;
ALTER TABLE refresh_tokens ADD COLUMN consumed_at TEXT;
ALTER TABLE refresh_tokens ADD COLUMN replaced_by TEXT;
ALTER TABLE refresh_tokens ADD COLUMN revoked_at TEXT;

-- Backfill existing rows so family logic is immediately usable.
UPDATE refresh_tokens
SET family_id = lower(hex(randomblob(16)))
WHERE family_id IS NULL OR family_id = '';

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_family_id ON refresh_tokens(family_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_consumed_at ON refresh_tokens(consumed_at);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_revoked_at ON refresh_tokens(revoked_at);
