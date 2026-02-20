ALTER TABLE user_preferences ADD COLUMN dietary_preference TEXT DEFAULT 'none';

-- Backfill: normalize existing rows
UPDATE user_preferences SET dietary_preference = 'none'
WHERE dietary_preference IS NULL OR dietary_preference = '';
