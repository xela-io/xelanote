-- =============================================================
-- CHATGPT API KEY STORAGE + ACTIVE AI PROVIDER PREFERENCE
-- =============================================================

-- OpenAI API key (encrypted, BYOK)
ALTER TABLE user_preferences ADD COLUMN encrypted_openai_api_key TEXT DEFAULT NULL;
ALTER TABLE user_preferences ADD COLUMN openai_api_key_updated_at DATETIME DEFAULT NULL;

-- Active provider selection: auto | claude | gemini | chatgpt
ALTER TABLE user_preferences ADD COLUMN active_ai_provider TEXT DEFAULT 'auto';

-- Backfill existing rows
UPDATE user_preferences
SET active_ai_provider = 'auto'
WHERE active_ai_provider IS NULL OR active_ai_provider = '';

CREATE INDEX IF NOT EXISTS idx_user_preferences_active_ai_provider
ON user_preferences(active_ai_provider);
