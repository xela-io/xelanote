-- Enable journal and recipe features for all existing users.
-- Uses INSERT OR REPLACE since (user_id, feature) is a composite primary key.
INSERT OR REPLACE INTO user_features (user_id, feature, enabled, updated_at)
SELECT id, 'journal', 1, datetime('now') FROM users;

INSERT OR REPLACE INTO user_features (user_id, feature, enabled, updated_at)
SELECT id, 'recipe', 1, datetime('now') FROM users;
