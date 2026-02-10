-- Note-level sharing (ACL)
CREATE TABLE IF NOT EXISTS note_shares (
    id INTEGER PRIMARY KEY,
    note_id TEXT NOT NULL,
    owner_user_id INTEGER NOT NULL,
    shared_with_user_id INTEGER NOT NULL,
    role TEXT NOT NULL DEFAULT 'viewer' CHECK (role IN ('viewer', 'editor')),
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (note_id) REFERENCES notes(id) ON DELETE CASCADE,
    FOREIGN KEY (owner_user_id) REFERENCES users(id),
    FOREIGN KEY (shared_with_user_id) REFERENCES users(id),
    UNIQUE(note_id, shared_with_user_id)
);

CREATE INDEX IF NOT EXISTS idx_note_shares_shared_with ON note_shares(shared_with_user_id);
CREATE INDEX IF NOT EXISTS idx_note_shares_note ON note_shares(note_id);
CREATE INDEX IF NOT EXISTS idx_note_shares_owner ON note_shares(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_note_shares_access ON note_shares(shared_with_user_id, note_id);

-- Folder-level sharing (Phase 2, schema prepared now)
CREATE TABLE IF NOT EXISTS folder_shares (
    id INTEGER PRIMARY KEY,
    folder_id INTEGER NOT NULL,
    owner_user_id INTEGER NOT NULL,
    shared_with_user_id INTEGER NOT NULL,
    role TEXT NOT NULL DEFAULT 'viewer' CHECK (role IN ('viewer', 'editor')),
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (folder_id) REFERENCES folders(id) ON DELETE CASCADE,
    FOREIGN KEY (owner_user_id) REFERENCES users(id),
    FOREIGN KEY (shared_with_user_id) REFERENCES users(id),
    UNIQUE(folder_id, shared_with_user_id)
);

CREATE INDEX IF NOT EXISTS idx_folder_shares_shared_with ON folder_shares(shared_with_user_id);
CREATE INDEX IF NOT EXISTS idx_folder_shares_folder ON folder_shares(folder_id);
