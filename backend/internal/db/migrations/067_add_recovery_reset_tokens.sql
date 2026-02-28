CREATE TABLE IF NOT EXISTS recovery_reset_tokens (
    token_hash TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL,
    expires_at TEXT NOT NULL,
    consumed_at TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_recovery_reset_tokens_user_id ON recovery_reset_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_recovery_reset_tokens_expires_at ON recovery_reset_tokens(expires_at);
