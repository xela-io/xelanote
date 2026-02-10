-- Migration 014: Create snippets table
CREATE TABLE IF NOT EXISTS snippets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    name_norm TEXT NOT NULL,
    description TEXT DEFAULT '',
    content TEXT NOT NULL,
    shortcut TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Index for efficient user filtering
CREATE INDEX IF NOT EXISTS idx_snippets_user_id ON snippets(user_id);

-- Composite unique index: each user has unique snippet names
CREATE UNIQUE INDEX IF NOT EXISTS idx_snippets_user_name_norm ON snippets(user_id, name_norm);

-- Trigger to update updated_at timestamp
CREATE TRIGGER IF NOT EXISTS update_snippets_timestamp
AFTER UPDATE ON snippets
FOR EACH ROW
WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE snippets SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
