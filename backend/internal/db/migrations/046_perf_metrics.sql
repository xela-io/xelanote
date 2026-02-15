-- Web Vitals performance metrics table
CREATE TABLE IF NOT EXISTS perf_metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    metric_name TEXT NOT NULL CHECK(metric_name IN ('LCP', 'INP', 'CLS', 'FCP', 'TTFB')),
    value REAL NOT NULL,
    rating TEXT NOT NULL CHECK(rating IN ('good', 'needs-improvement', 'poor')),
    sanitized_url TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_perf_metrics_user_created
    ON perf_metrics(user_id, created_at);

CREATE INDEX IF NOT EXISTS idx_perf_metrics_name_created
    ON perf_metrics(metric_name, created_at);
