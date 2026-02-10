-- =============================================================
-- CLAUDE API KEY STORAGE (BYOK - Bring Your Own Key)
-- =============================================================
-- Speichert den verschlüsselten Claude API-Key pro Benutzer.
-- Der Key wird mit AES-256-GCM verschlüsselt (Server-Key).

-- 1. Encrypted API Key Feld zur user_preferences Tabelle
-- encrypted_claude_api_key: Base64-encoded AES-256-GCM ciphertext
-- Format: nonce (12 bytes) || ciphertext || tag (16 bytes)
ALTER TABLE user_preferences ADD COLUMN encrypted_claude_api_key TEXT DEFAULT NULL;

-- 2. Timestamp wann der Key zuletzt aktualisiert wurde
ALTER TABLE user_preferences ADD COLUMN claude_api_key_updated_at DATETIME DEFAULT NULL;
