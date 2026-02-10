-- Migration 005: Add user ownership to notes and folders
-- Each note and folder belongs to a specific user for multi-user support

-- Add user_id to notes table (no foreign key constraint - SQLite limitation)
ALTER TABLE notes ADD COLUMN user_id INTEGER;

-- Set user_id to 1 for all existing notes (assumes first user will have ID 1)
UPDATE notes SET user_id = 1 WHERE user_id IS NULL;

-- Add user_id to folders table (no foreign key constraint - SQLite limitation)
ALTER TABLE folders ADD COLUMN user_id INTEGER;

-- Set user_id to 1 for all existing folders (assumes first user will have ID 1)
UPDATE folders SET user_id = 1 WHERE user_id IS NULL;

-- Create indexes for efficient filtering by user
CREATE INDEX IF NOT EXISTS idx_notes_user_id ON notes(user_id);
CREATE INDEX IF NOT EXISTS idx_folders_user_id ON folders(user_id);

-- Note: Foreign key constraints will be enforced at the application level
-- SQLite's ALTER TABLE doesn't support adding foreign key constraints to existing tables
