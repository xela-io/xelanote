-- Analytics events table (PWA funnel, etc.)
CREATE TABLE IF NOT EXISTS analytics_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_name TEXT NOT NULL,
    data_json TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_analytics_events_user_created
    ON analytics_events(user_id, created_at);

CREATE INDEX IF NOT EXISTS idx_analytics_events_name_created
    ON analytics_events(event_name, created_at);
