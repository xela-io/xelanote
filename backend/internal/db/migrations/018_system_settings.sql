-- System settings table
CREATE TABLE system_settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TEXT DEFAULT (datetime('now'))
);

-- Default settings
INSERT INTO system_settings (key, value) VALUES
    ('registration_enabled', 'true'),
    ('max_notes_per_user', '0'),        -- 0 = unlimited
    ('max_storage_mb_per_user', '0'),   -- 0 = unlimited
    ('maintenance_mode', 'false'),
    ('activity_retention_days', '90');  -- Activity logs retention
