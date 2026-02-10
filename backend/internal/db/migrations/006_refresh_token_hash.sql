-- Migration 006: Reset refresh tokens for hashed storage
-- Existing tokens were stored in plaintext; delete them so new hashed tokens are issued.
DELETE FROM refresh_tokens;
