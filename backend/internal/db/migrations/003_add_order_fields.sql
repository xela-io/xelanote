-- Migration 003: Add order fields for custom sorting
-- This allows users to reorder folders and notes via drag & drop

-- Add order column to folders table
ALTER TABLE folders ADD COLUMN display_order INTEGER NOT NULL DEFAULT 0;

-- Add order column to notes table
ALTER TABLE notes ADD COLUMN display_order INTEGER NOT NULL DEFAULT 0;

-- Initialize order values based on alphabetical order within each parent
-- Folders: order by name within each parent
UPDATE folders
SET display_order = (
    SELECT COUNT(*)
    FROM folders f2
    WHERE (f2.parent_id = folders.parent_id OR (f2.parent_id IS NULL AND folders.parent_id IS NULL))
    AND f2.name < folders.name
) * 100;

-- Notes: order by title within each folder
UPDATE notes
SET display_order = (
    SELECT COUNT(*)
    FROM notes n2
    WHERE n2.folder_path = notes.folder_path
    AND n2.is_deleted = 0
    AND n2.title < notes.title
) * 100
WHERE is_deleted = 0;

-- Create indexes for faster ordering queries
CREATE INDEX IF NOT EXISTS idx_folders_parent_order ON folders(parent_id, display_order);
CREATE INDEX IF NOT EXISTS idx_notes_folder_order ON notes(folder_path, display_order) WHERE is_deleted = 0;
