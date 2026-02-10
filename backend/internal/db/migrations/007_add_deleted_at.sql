-- Migration 007: Add deleted_at timestamp for trash functionality
-- Enables audit trail for deleted notes and sorting by deletion time

-- Add deleted_at column (NULL for active notes, timestamp for deleted notes)
ALTER TABLE notes ADD COLUMN deleted_at TEXT DEFAULT NULL;

-- Update existing soft-deleted notes to have a deleted_at timestamp
UPDATE notes SET deleted_at = updated_at WHERE is_deleted = 1 AND deleted_at IS NULL;

-- Create index for efficient trash queries
CREATE INDEX IF NOT EXISTS idx_notes_deleted ON notes(is_deleted, deleted_at DESC)
  WHERE is_deleted = 1;
