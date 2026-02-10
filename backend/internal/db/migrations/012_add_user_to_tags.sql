-- Migration 012: Add user_id to tags table for user-specific tags
-- This makes tags user-scoped, consistent with notes and folders

-- Add user_id column to tags table
ALTER TABLE tags ADD COLUMN user_id INTEGER;

-- Create index on user_id for efficient filtering
CREATE INDEX idx_tags_user_id ON tags(user_id);

-- Drop old unique constraint on name_norm (global)
DROP INDEX IF EXISTS idx_tags_name_norm;

-- Create new composite unique index on (user_id, name_norm)
-- This allows each user to have their own tags with the same names
CREATE UNIQUE INDEX idx_tags_user_name_norm ON tags(user_id, name_norm);
