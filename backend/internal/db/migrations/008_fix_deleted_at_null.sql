-- Fix inconsistent data: Set deleted_at for notes that are marked as deleted but have NULL deleted_at
-- This can happen if notes were deleted before the deleted_at column was added

UPDATE notes
SET deleted_at = updated_at
WHERE is_deleted = 1 AND deleted_at IS NULL;
