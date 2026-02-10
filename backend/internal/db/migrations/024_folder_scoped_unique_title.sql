-- Migration 024: Make note title uniqueness folder-scoped
-- Allow notes with same title in different folders

-- STEP 1: Normalize folder_path (defense in depth)
-- Handle NULL, empty, and non-normalized paths

-- 1a. NULL, empty, '.' → '/'
UPDATE notes SET folder_path = '/'
WHERE folder_path IS NULL OR folder_path = '' OR folder_path = '.';

-- 1b. Remove double/triple slashes: '//work' → '/work', '///path' → '/path'
-- Note: SQLite doesn't have recursive REPLACE, so we do 3 passes (best effort)
-- This handles up to 6 consecutive slashes. If more exist, app-side normalization will catch them.
UPDATE notes SET folder_path = replace(folder_path, '//', '/') WHERE folder_path LIKE '%//%';
UPDATE notes SET folder_path = replace(folder_path, '//', '/') WHERE folder_path LIKE '%//%';
UPDATE notes SET folder_path = replace(folder_path, '//', '/') WHERE folder_path LIKE '%//%';

-- 1c. Remove trailing slashes (except for root '/')
UPDATE notes
SET folder_path = rtrim(folder_path, '/')
WHERE folder_path != '/' AND folder_path LIKE '%/';

-- 1d. Handle edge case: rtrim can produce empty string from '////'
UPDATE notes SET folder_path = '/' WHERE folder_path = '';

-- 1e. Ensure leading slash (defensive, should already be satisfied)
UPDATE notes
SET folder_path = '/' || folder_path
WHERE folder_path NOT LIKE '/%' AND folder_path != '';

-- STEP 2: Drop ALL old unique indexes (defense in depth)
-- Drop global index from schema.sql (very old, should have been removed in migration 010)
DROP INDEX IF EXISTS idx_notes_title_norm;
-- Drop user-scoped index from migration 010
DROP INDEX IF EXISTS idx_notes_user_title_norm;

-- STEP 3: Create new folder-scoped unique index
CREATE UNIQUE INDEX IF NOT EXISTS idx_notes_user_folder_title_norm
ON notes(user_id, folder_path, title_norm)
WHERE is_deleted = 0;
