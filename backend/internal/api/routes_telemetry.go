package api

import "github.com/go-chi/chi/v5"

func (s *Server) registerTelemetryRoutes(r chi.Router) {
	r.With(rateLimitMiddleware(s.perfMetricsLimiter)).Post("/perf-metrics", s.submitPerfMetric)
	r.Route("/analytics", func(r chi.Router) {
		r.With(rateLimitMiddleware(s.analyticsLimiter)).Post("/events", s.submitAnalyticsEvent)
	})
}
