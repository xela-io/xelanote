-- Migration 026: Add composite index for efficient ORDER BY queries
-- Optimizes: SELECT * FROM notes WHERE user_id=? AND is_deleted=0 ORDER BY updated_at DESC
--
-- Before: SEARCH notes USING INDEX idx_notes_user_id + TEMP B-TREE FOR ORDER BY
-- After:  SEARCH notes USING INDEX idx_notes_user_order (covering index, no temp sort)

-- Composite index for notes listing with updated_at sorting
-- Includes is_deleted to allow index-only filtering
CREATE INDEX IF NOT EXISTS idx_notes_user_order
ON notes(user_id, is_deleted, updated_at DESC);

-- Note: We keep the existing idx_notes_user_id for other queries that don't need sorting
-- SQLite query planner will choose the best index automatically
