CREATE TABLE IF NOT EXISTS fido2_credentials (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    credential_id BLOB NOT NULL,
    public_key BLOB NOT NULL,
    attestation_type TEXT NOT NULL DEFAULT 'none',
    aaguid BLOB,
    sign_count INTEGER NOT NULL DEFAULT 0,
    device_name TEXT NOT NULL DEFAULT '',
    transports TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    last_used_at TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(credential_id)
);

CREATE INDEX IF NOT EXISTS idx_fido2_user_id ON fido2_credentials(user_id);
CREATE INDEX IF NOT EXISTS idx_fido2_credential_id ON fido2_credentials(credential_id);
