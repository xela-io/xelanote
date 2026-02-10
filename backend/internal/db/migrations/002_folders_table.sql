-- Migration 002: Add folders table
-- Creates a separate folders table for proper folder management
-- Existing folder_path data will be migrated on-demand by the backend

-- Create folders table
CREATE TABLE IF NOT EXISTS folders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    path TEXT UNIQUE NOT NULL,
    parent_id INTEGER REFERENCES folders(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_folders_path ON folders(path);
CREATE INDEX IF NOT EXISTS idx_folders_parent ON folders(parent_id);

-- Insert root folder only
-- Other folders will be created on-demand when accessed
INSERT OR IGNORE INTO folders (id, path, parent_id, name)
VALUES (1, '/', NULL, 'Root');
