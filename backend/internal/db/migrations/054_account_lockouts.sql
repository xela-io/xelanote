-- F2-06: Persist account lockout state to DB so it survives server restarts.
-- In-memory map remains the fast-path; DB is the durable fallback.
CREATE TABLE IF NOT EXISTS account_lockouts (
    identifier_hash TEXT NOT NULL,
    ip              TEXT NOT NULL DEFAULT '',
    failed_attempts INTEGER NOT NULL DEFAULT 0,
    locked_until    TEXT NOT NULL DEFAULT '',
    last_attempt    TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (identifier_hash, ip)
);

CREATE INDEX IF NOT EXISTS idx_account_lockouts_locked_until ON account_lockouts(locked_until);
