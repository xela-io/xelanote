-- Shared note placements: allows recipients to place shared notes into their own folders
CREATE TABLE IF NOT EXISTS shared_note_placements (
    id INTEGER PRIMARY KEY,
    note_id TEXT NOT NULL,
    user_id INTEGER NOT NULL,
    folder_id INTEGER NOT NULL,
    created_at TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (note_id) REFERENCES notes(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (folder_id) REFERENCES folders(id) ON DELETE CASCADE,
    UNIQUE(note_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_placements_user_folder ON shared_note_placements(user_id, folder_id);
CREATE INDEX IF NOT EXISTS idx_placements_note ON shared_note_placements(note_id);

-- Index for the frequent permission-chain query (folder_shares access lookup)
CREATE INDEX IF NOT EXISTS idx_folder_shares_access ON folder_shares(shared_with_user_id, folder_id);
