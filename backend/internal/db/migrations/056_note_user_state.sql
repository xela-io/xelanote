CREATE TABLE IF NOT EXISTS note_user_state (
    note_id    TEXT    NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    state_data TEXT    NOT NULL DEFAULT '{}',
    updated_at TEXT    NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (note_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_note_user_state_user ON note_user_state(user_id);
