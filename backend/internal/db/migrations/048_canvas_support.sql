-- Canvas note type support
-- Canvas notes use note_type = 'canvas' with JSON Canvas content in the content field.

CREATE INDEX IF NOT EXISTS idx_notes_canvas_type
  ON notes(user_id, note_type, updated_at)
  WHERE note_type = 'canvas' AND is_deleted = 0;
