CREATE TABLE IF NOT EXISTS note_due_dates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    note_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    line_text TEXT NOT NULL,
    line_index INTEGER NOT NULL,
    due_date TEXT NOT NULL,
    is_task_item INTEGER NOT NULL DEFAULT 0,
    is_completed INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_note_due_dates_user_date ON note_due_dates(user_id, due_date);
CREATE INDEX idx_note_due_dates_note ON note_due_dates(note_id);
CREATE INDEX idx_note_due_dates_user_incomplete ON note_due_dates(user_id, is_completed, due_date);
