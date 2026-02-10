CREATE TABLE IF NOT EXISTS task_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    note_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    actor_user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    task_text TEXT,
    task_index INTEGER NOT NULL,
    encrypted_task_text TEXT,
    wrapped_dek TEXT,
    encryption_metadata TEXT,
    text_encrypted INTEGER NOT NULL DEFAULT 0,
    event_type TEXT NOT NULL CHECK(event_type IN ('completed', 'reopened')),
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    CHECK (
        (text_encrypted = 0 AND task_text IS NOT NULL AND encrypted_task_text IS NULL AND wrapped_dek IS NULL)
        OR
        (text_encrypted = 1 AND task_text IS NULL AND encrypted_task_text IS NOT NULL AND wrapped_dek IS NOT NULL AND encryption_metadata IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_task_events_actor_date ON task_events(actor_user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_task_events_note_date ON task_events(note_id, created_at);
