-- Add 2FA fields to users table
ALTER TABLE users ADD COLUMN totp_secret TEXT DEFAULT NULL;
ALTER TABLE users ADD COLUMN totp_enabled INTEGER DEFAULT 0;
ALTER TABLE users ADD COLUMN totp_verified_at TEXT DEFAULT NULL;
ALTER TABLE users ADD COLUMN totp_disabled_at TEXT DEFAULT NULL;
ALTER TABLE users ADD COLUMN totp_setup_started_at TEXT DEFAULT NULL;
ALTER TABLE users ADD COLUMN last_totp_step INTEGER DEFAULT NULL;

-- Backup codes table
CREATE TABLE IF NOT EXISTS backup_codes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    code_hash TEXT NOT NULL,
    used INTEGER DEFAULT 0,
    used_at TEXT DEFAULT NULL,
    created_at TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_backup_codes_user_id ON backup_codes(user_id);
CREATE INDEX IF NOT EXISTS idx_backup_codes_used ON backup_codes(used);
