package service

import (
	"fmt"
	"log/slog"

	"github.com/xela-io/xelanote/internal/db"
)

// TelemetryService handles performance metrics and analytics events.
type TelemetryService struct {
	db  *db.DB
	log *slog.Logger
}

// NewTelemetryService creates a new TelemetryService.
func NewTelemetryService(database *db.DB, log *slog.Logger) *TelemetryService {
	return &TelemetryService{db: database, log: log}
}

// ValidPerfMetricNames is the set of accepted Web Vital metric names.
var ValidPerfMetricNames = map[string]bool{
	"LCP":  true,
	"INP":  true,
	"CLS":  true,
	"FCP":  true,
	"TTFB": true,
}

// ValidPerfMetricRatings is the set of accepted Web Vital ratings.
var ValidPerfMetricRatings = map[string]bool{
	"good":              true,
	"needs-improvement": true,
	"poor":              true,
}

// ValidAnalyticsEventNames is the whitelist of accepted analytics event names.
var ValidAnalyticsEventNames = map[string]bool{
	"ios_coach_shown":       true,
	"ios_step_changed":      true,
	"ios_snoozed":           true,
	"ios_dismissed":         true,
	"ios_installed_detected": true,
}

// RecordPerfMetric validates and stores a performance metric.
func (s *TelemetryService) RecordPerfMetric(userID int, metricName string, value float64, rating, sanitizedURL string) error {
	if !ValidPerfMetricNames[metricName] {
		return fmt.Errorf("invalid metric name: %s", metricName)
	}
	if !ValidPerfMetricRatings[rating] {
		return fmt.Errorf("invalid rating: %s", rating)
	}

	return s.db.RecordPerfMetric(db.PerfMetric{
		UserID:       userID,
		MetricName:   metricName,
		Value:        value,
		Rating:       rating,
		SanitizedURL: sanitizedURL,
	})
}

// RecordAnalyticsEvent validates and stores an analytics event.
func (s *TelemetryService) RecordAnalyticsEvent(userID int, eventName, dataJSON string) error {
	if !ValidAnalyticsEventNames[eventName] {
		return fmt.Errorf("invalid event name: %s", eventName)
	}

	return s.db.RecordAnalyticsEvent(db.AnalyticsEvent{
		UserID:    userID,
		EventName: eventName,
		DataJSON:  dataJSON,
	})
}
