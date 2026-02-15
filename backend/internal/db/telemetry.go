package db

import (
	"fmt"
	"time"
)

// PerfMetric represents a Web Vitals performance metric record.
type PerfMetric struct {
	ID           int       `json:"id"`
	UserID       int       `json:"user_id"`
	MetricName   string    `json:"metric_name"`
	Value        float64   `json:"value"`
	Rating       string    `json:"rating"`
	SanitizedURL string    `json:"sanitized_url"`
	CreatedAt    time.Time `json:"created_at"`
}

// AnalyticsEvent represents a tracked analytics event (e.g. PWA funnel).
type AnalyticsEvent struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	EventName string    `json:"event_name"`
	DataJSON  string    `json:"data_json"`
	CreatedAt time.Time `json:"created_at"`
}

// RecordPerfMetric inserts a Web Vitals metric and cleans up records older than 90 days.
func (db *DB) RecordPerfMetric(metric PerfMetric) error {
	_, err := db.Exec(`
		INSERT INTO perf_metrics (user_id, metric_name, value, rating, sanitized_url)
		VALUES (?, ?, ?, ?, ?)`,
		metric.UserID, metric.MetricName, metric.Value, metric.Rating, metric.SanitizedURL,
	)
	if err != nil {
		return fmt.Errorf("failed to record perf metric: %w", err)
	}

	// Opportunistic 90-day cleanup (ignore errors)
	db.Exec("DELETE FROM perf_metrics WHERE created_at < datetime('now', '-90 days')") //nolint:errcheck
	return nil
}

// RecordAnalyticsEvent inserts an analytics event and cleans up records older than 90 days.
func (db *DB) RecordAnalyticsEvent(event AnalyticsEvent) error {
	_, err := db.Exec(`
		INSERT INTO analytics_events (user_id, event_name, data_json)
		VALUES (?, ?, ?)`,
		event.UserID, event.EventName, event.DataJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to record analytics event: %w", err)
	}

	// Opportunistic 90-day cleanup (ignore errors)
	db.Exec("DELETE FROM analytics_events WHERE created_at < datetime('now', '-90 days')") //nolint:errcheck
	return nil
}
