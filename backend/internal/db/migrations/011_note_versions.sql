-- Migration 011: Note versions table for version history
-- Stores snapshots of notes for timeline view and restore functionality

CREATE TABLE IF NOT EXISTS note_versions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    note_id TEXT NOT NULL,
    user_id INTEGER NOT NULL,
    version INTEGER NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    snapshot_at TEXT NOT NULL,
    FOREIGN KEY (note_id) REFERENCES notes(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Index for querying versions by note (most common query)
CREATE INDEX IF NOT EXISTS idx_note_versions_note_id ON note_versions(note_id);

-- Index for user-specific queries and pruning
CREATE INDEX IF NOT EXISTS idx_note_versions_user_id ON note_versions(user_id);

-- Index for time-based sorting and pagination
CREATE INDEX IF NOT EXISTS idx_note_versions_snapshot_at ON note_versions(snapshot_at);

-- Composite index for efficient version lookup and pagination
CREATE INDEX IF NOT EXISTS idx_note_versions_note_user ON note_versions(note_id, user_id, version DESC);
