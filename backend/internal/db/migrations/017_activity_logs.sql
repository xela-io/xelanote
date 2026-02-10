-- Activity logs table for audit trail
CREATE TABLE activity_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER,
    action TEXT NOT NULL,           -- 'login', 'logout', 'note_create', 'note_delete', etc.
    target_type TEXT,               -- 'note', 'folder', 'user', null
    target_id TEXT,                 -- ID of the affected object
    details TEXT,                   -- JSON with additional info
    ip_address TEXT,
    user_agent TEXT,                -- Browser/Client info for audit
    created_at TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_activity_logs_user ON activity_logs(user_id);
CREATE INDEX idx_activity_logs_action ON activity_logs(action);
CREATE INDEX idx_activity_logs_created ON activity_logs(created_at DESC);
CREATE INDEX idx_activity_logs_target ON activity_logs(target_type, target_id);
