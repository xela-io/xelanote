-- =============================================================
-- AI MODEL PREFERENCES (PER USER)
-- =============================================================

ALTER TABLE user_preferences ADD COLUMN claude_model TEXT DEFAULT NULL;
ALTER TABLE user_preferences ADD COLUMN gemini_model TEXT DEFAULT NULL;
ALTER TABLE user_preferences ADD COLUMN openai_model TEXT DEFAULT NULL;
