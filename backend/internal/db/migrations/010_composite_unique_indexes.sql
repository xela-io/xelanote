-- Migration 010: Make unique constraints user-specific
-- This migration changes the unique constraints on notes and folders
-- to be composite keys that include the user_id. This allows different
-- users to have notes with the same title and folders with the same path.

-- Step 1: Fix the 'notes' table
-- Drop the old global unique index on note titles
DROP INDEX IF EXISTS idx_notes_title_norm;

-- Create a new composite unique index for user_id and title_norm
CREATE UNIQUE INDEX IF NOT EXISTS idx_notes_user_title_norm ON notes(user_id, title_norm) WHERE is_deleted = 0;


-- Step 2: Fix the 'folders' table
-- SQLite doesn't allow dropping a UNIQUE constraint directly, so we have to recreate the table.

-- Rename the existing table
ALTER TABLE folders RENAME TO folders_old;

-- Create the new table without the UNIQUE constraint on 'path'
CREATE TABLE folders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    path TEXT NOT NULL,
    parent_id INTEGER REFERENCES folders(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    user_id INTEGER,
    display_order INTEGER NOT NULL DEFAULT 0
);

-- Copy the data from the old table to the new one
-- Note: This assumes the 'display_order' column was added in migration 003
INSERT INTO folders (id, path, parent_id, name, created_at, updated_at, user_id, display_order)
SELECT id, path, parent_id, name, created_at, updated_at, user_id, display_order
FROM folders_old;

-- Drop the old table
DROP TABLE folders_old;

-- Create the new composite unique index for user_id and path
CREATE UNIQUE INDEX IF NOT EXISTS idx_folders_user_path ON folders(user_id, path);

-- Re-create the other indexes that were on the original table
CREATE INDEX IF NOT EXISTS idx_folders_path ON folders(path);
CREATE INDEX IF NOT EXISTS idx_folders_parent ON folders(parent_id);
CREATE INDEX IF NOT EXISTS idx_folders_user_id ON folders(user_id);
