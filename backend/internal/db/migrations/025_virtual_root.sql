-- Migration 025: Virtual Root - Eliminate hardcoded root folder
-- SAFETY: Non-destructive, fully reversible, with transaction wrapping

BEGIN TRANSACTION;

-- ============================================================================
-- SAFETY NOTES (automated checks not possible in pure SQL without triggers)
-- ============================================================================
-- This migration has been designed to be safe even if:
-- - Root folder (id=1) doesn't exist (DELETE will be no-op)
-- - Folders already have parent_id=NULL (UPDATE will be no-op)
-- - Notes are in root folder (moved to /Migrated)
--
-- Prerequisite: idx_folders_user_path UNIQUE index must exist (Migration 010)
-- - Required for INSERT OR IGNORE idempotency
-- - If missing, INSERT OR IGNORE may create duplicates
--
-- Schema assumption: Only folders.parent_id references folders.id
-- - notes table uses folder_path (string), not FK
-- - If future migrations add FK dependencies, review this migration first
--
-- Manual verification recommended after migration (see bottom of file)

-- ============================================================================
-- STEP 1: Handle edge case - notes in root folder
-- ============================================================================
-- Create /Migrated folder per user if any notes exist in root
-- UNIQUE constraint on (user_id, path) ensures INSERT OR IGNORE is idempotent
INSERT OR IGNORE INTO folders (user_id, path, parent_id, name, display_order)
SELECT DISTINCT user_id, '/Migrated', NULL, 'Migrated', 0
FROM notes
WHERE folder_path = '/' AND is_deleted = 0;

-- Move notes from root to /Migrated (safe even if /Migrated doesn't exist due to INSERT above)
UPDATE notes
SET folder_path = '/Migrated', updated_at = datetime('now')
WHERE folder_path = '/' AND is_deleted = 0;

-- ============================================================================
-- STEP 2: Update all top-level folders to parent_id=NULL
-- ============================================================================
-- This makes all folders currently pointing to root (id=1) truly top-level
-- Safe: If no folders have parent_id=1, this is a no-op
-- Safe: If root (id=1) doesn't exist, parent_id=1 is already orphaned and should be NULLed
UPDATE folders
SET parent_id = NULL, updated_at = datetime('now')
WHERE parent_id = 1;

-- ============================================================================
-- STEP 3: Delete root folder (id=1)
-- ============================================================================
-- Safe because:
-- 1. No notes are in '/' (moved to /Migrated in STEP 1)
-- 2. No folders reference id=1 as parent (updated to NULL in STEP 2)
-- 3. No other tables have foreign keys to folders.id (only self-reference)
-- 4. If id=1 doesn't exist, DELETE is a no-op
DELETE FROM folders WHERE id = 1;

COMMIT;

-- ============================================================================
-- Post-Migration Manual Verification Queries (run these after migration):
-- ============================================================================
-- 1. Confirm no root folder exists:
--    SELECT COUNT(*) FROM folders WHERE id = 1;
--    Expected: 0
--
-- 2. Confirm no folders reference deleted root:
--    SELECT COUNT(*) FROM folders WHERE parent_id = 1;
--    Expected: 0
--
-- 3. List top-level folders (should have parent_id=NULL):
--    SELECT id, path, parent_id, name FROM folders WHERE parent_id IS NULL LIMIT 10;
--
-- 4. Check if /Migrated was created (edge case):
--    SELECT user_id, COUNT(*) as note_count FROM notes
--    WHERE folder_path = '/Migrated' AND is_deleted = 0
--    GROUP BY user_id;
--    If any results: Review and redistribute these notes manually
--
-- 5. Verify folder hierarchy integrity (no orphans):
--    SELECT COUNT(*) FROM folders f1
--    WHERE f1.parent_id IS NOT NULL
--      AND NOT EXISTS (SELECT 1 FROM folders f2 WHERE f2.id = f1.parent_id);
--    Expected: 0 (no orphaned folders)
--
-- 6. Verify prerequisite (should have been checked before migration):
--    SELECT name FROM sqlite_master WHERE type='index' AND name='idx_folders_user_path';
--    Expected: idx_folders_user_path

-- ============================================================================
-- ROLLBACK Migration 025: Restore physical root folder
-- WARNING: Only use if migration caused critical issues
-- ============================================================================
-- Uncomment and run manually if rollback needed:
--
-- BEGIN TRANSACTION;
--
-- -- STEP 1: Recreate root folder
-- -- Note: We cannot force id=1 if AUTOINCREMENT has moved past it
-- -- Instead, we'll create a new root and update references
-- --
-- -- DESIGN DECISION: user_id for root folder
-- -- - Original Migration 002: Root created with no user_id (was added later)
-- -- - Migration 005: Existing folders (including root) given user_id=1
-- -- - Design: Root folder was ALWAYS user-scoped, just happened to be assigned to user 1
-- -- - After virtual root migration: Top-level folders belong to each user individually
-- -- - Rollback restores original behavior: root with user_id=1
-- -- - This is INTENTIONAL: Root folder was never "global/system", it was user 1's root
-- INSERT INTO folders (path, parent_id, name, user_id, display_order, created_at, updated_at)
-- VALUES ('/', NULL, 'Root', 1, 0, datetime('now'), datetime('now'));
--
-- -- Get the ID of the newly created root folder
-- -- (In SQLite, we can't directly set id=1 if AUTOINCREMENT has progressed)
-- -- This approach is safer: create new root, update all references
--
-- -- STEP 2: Update all top-level folders to point to the new root
-- -- Find folders with parent_id=NULL (except the root we just created)
-- -- LIMIT 1 ensures we don't fail if multiple "/" exist (bad data)
-- UPDATE folders
-- SET parent_id = (
--     SELECT id FROM folders
--     WHERE path = '/' AND parent_id IS NULL
--     ORDER BY id ASC
--     LIMIT 1
-- ),
--     updated_at = datetime('now')
-- WHERE parent_id IS NULL
--   AND path != '/';
--
-- -- STEP 3: Move notes from /Migrated back to root
-- UPDATE notes
-- SET folder_path = '/', updated_at = datetime('now')
-- WHERE folder_path = '/Migrated' AND is_deleted = 0;
--
-- -- STEP 4: Delete /Migrated folders per-user if they're now empty
-- -- Only delete /Migrated for users who have no notes there
-- -- This ensures we don't delete another user's /Migrated folder
-- DELETE FROM folders
-- WHERE path = '/Migrated'
--   AND NOT EXISTS (
--       SELECT 1 FROM notes
--       WHERE folder_path = '/Migrated'
--         AND is_deleted = 0
--         AND user_id = folders.user_id
--   );
--
-- COMMIT;
--
-- ============================================================================
-- IMPORTANT ROLLBACK NOTE:
-- ============================================================================
-- If the new root folder has a different ID than 1 (e.g., id=42), the code will
-- still work because we're removing all hardcoded rootID=1 references.
-- However, if you need to restore to EXACTLY id=1:
-- 1. Stop the application
-- 2. Restore from database backup
-- 3. Revert git commit
-- This is the safer approach for production rollback.
