-- Migration 064: Remove plaintext tag metadata from encrypted notes
DELETE FROM note_tags
WHERE note_id IN (
  SELECT id
  FROM notes
  WHERE content_encrypted = 1
     OR encryption_version IS NOT NULL
);

-- Cleanup orphan tags left after note_tags deletion
DELETE FROM tags
WHERE id NOT IN (SELECT DISTINCT tag_id FROM note_tags);
