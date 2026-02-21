-- F2-01: Prevent race condition on first-user admin creation.
-- Only one admin user may exist. This is a defense-in-depth measure;
-- MaxOpenConns(1) already serializes DB operations, but this constraint
-- guards against future architecture changes (e.g. PostgreSQL migration).
CREATE UNIQUE INDEX IF NOT EXISTS idx_single_admin ON users(is_admin) WHERE is_admin = 1;
